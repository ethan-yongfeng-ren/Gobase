package log

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Conf struct {
	LogWay     string //console or 日志文件,默认为日志文件
	EncoderWay string //json or console，默认为console
	LogLevel   string
	LogPath    string //文件日志路径，默认需要为./
	MaxDays    int
	MaxSize    int  //单位M
	MaxBackups int  //最多保留多少个文件
	Compress   bool //是否将轮转后的历史日志压缩为gzip，默认不压缩
}

var logger *zap.Logger
var levelCtrl zap.AtomicLevel
var infoWriter io.Writer = os.Stdout

func InitLog(logFile string, conf Conf) {
	config := zapcore.EncoderConfig{
		MessageKey: "msg",   //结构化（json）输出：msg的key
		LevelKey:   "level", //结构化（json）输出：日志级别的key（INFO，WARN，ERROR等）
		TimeKey:    "ts",    //结构化（json）输出：时间的key（INFO，WARN，ERROR等）
		CallerKey:  "file",  //结构化（json）输出：打印日志的文件对应的Key
		EncodeLevel: func(level zapcore.Level, encoder zapcore.PrimitiveArrayEncoder) {
			encoder.AppendString(fmt.Sprintf("[%s]", level.CapitalString()))
		},
		EncodeCaller: func(caller zapcore.EntryCaller, encoder zapcore.PrimitiveArrayEncoder) {
			idx := strings.LastIndexByte(caller.File, '/')
			if idx == -1 {
				encoder.AppendString(fmt.Sprintf("[%s]", caller.FullPath()))
				return
			}
			encoder.AppendString(fmt.Sprintf("[%s:%d]", caller.File[idx+1:], caller.Line))
		},
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02T15:04:05.999Z")) // 日志采集需要精确到毫秒
		}, //输出的时间格式
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendInt64(int64(d) / 1000000)
		},
	}
	// 获取io.Writer的实现
	if conf.LogWay == "" {
		infoWriter = &lumberjack.Logger{
			Filename:   fmt.Sprintf("%s/%s", conf.LogPath, logFile),
			MaxSize:    conf.MaxSize,    //最大M数，超过则切割
			MaxBackups: conf.MaxBackups, //最大文件保留数，超过就删除最老的日志文件
			MaxAge:     conf.MaxDays,    //保存30天
			Compress:   conf.Compress,   //是否压缩轮转后的历史日志
		}
	}
	levelCtrl = zap.NewAtomicLevel()
	ReloadLogLevel(conf.LogLevel)
	// 实现多个输出
	var core zapcore.Core
	if conf.EncoderWay == "" || conf.EncoderWay == "console" {
		core = zapcore.NewTee(
			zapcore.NewCore(zapcore.NewConsoleEncoder(config), zapcore.AddSync(infoWriter), levelCtrl), //将info及以下写入logPath，NewConsoleEncoder 是非结构化输出
		)
	} else {
		core = zapcore.NewTee(
			zapcore.NewCore(zapcore.NewJSONEncoder(config), zapcore.AddSync(infoWriter), levelCtrl),
		)
	}
	//logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.WarnLevel), zap.AddCallerSkip(1))
	logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.WarnLevel), zap.AddCallerSkip(1))
}

func ReloadLogLevel(level string) {
	levelCtrl.SetLevel(ParseLogLevel(level))
}

func GetLogLevel() zapcore.Level {
	return levelCtrl.Level()
}

func ParseLogLevel(level string) zapcore.Level {
	switch level {
	case "error":
		return zap.ErrorLevel
	case "warn":
		return zap.WarnLevel
	case "info":
		return zap.InfoLevel
	case "debug":
		return zap.DebugLevel
	default:
		return zap.DPanicLevel
	}
}

func GetLogWriter() io.Writer {
	return infoWriter
}

func GetLogInst() *zap.Logger {
	return logger
}

func Error(format string, v ...interface{}) {
	logger.Sugar().Errorf(format, v...)
}

func Warn(format string, v ...interface{}) {
	logger.Sugar().Warnf(format, v...)
}

func Info(format string, v ...interface{}) {
	logger.Sugar().Infof(format, v...)
}

func Debug(format string, v ...interface{}) {
	logger.Sugar().Debugf(format, v...)
}
