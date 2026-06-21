package log

import (
	"fmt"
	"os"
	"time"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config 日志配置
type Config struct {
	Level      string // debug, info, warn, error
	Console    string // json, console
	LogPath    string // 日志文件路径，空表示不输出到文件
	MaxSize    int    // 单个日志文件最大大小（MB），默认100
	MaxBackups int    // 保留的旧日志文件最大数量，默认7
	MaxAge     int    // 保留旧日志文件的最大天数，默认30
}

var (
	logger    *zap.Logger
	levelCtrl zap.AtomicLevel
)

// Init 初始化全局 logger
func Init(cfg Config) {
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.Console == "" {
		cfg.Console = "console"
	}
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 100
	}
	if cfg.MaxBackups == 0 {
		cfg.MaxBackups = 7
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 30
	}

	config := zapcore.EncoderConfig{
		MessageKey: "msg",
		LevelKey:   "level",
		TimeKey:    "ts",
		CallerKey:  "file",
		EncodeLevel: func(level zapcore.Level, encoder zapcore.PrimitiveArrayEncoder) {
			encoder.AppendString(level.CapitalString())
		},
		EncodeCaller: func(caller zapcore.EntryCaller, encoder zapcore.PrimitiveArrayEncoder) {
			encoder.AppendString(caller.TrimmedPath())
		},
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02T15:04:05.000Z"))
		},
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendInt64(int64(d) / 1000000)
		},
	}

	levelCtrl = zap.NewAtomicLevel()

	var writers []zapcore.WriteSyncer
	if cfg.LogPath == "" {
		writers = append(writers, zapcore.AddSync(os.Stderr))
	} else {
		ljWriter := &lumberjack.Logger{
			Filename:   cfg.LogPath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			LocalTime:  true,
		}
		writers = append(writers, zapcore.AddSync(ljWriter))
	}

	ws := zapcore.NewMultiWriteSyncer(writers...)

	var core zapcore.Core
	if cfg.Console == "json" {
		core = zapcore.NewCore(zapcore.NewJSONEncoder(config), ws, levelCtrl)
	} else {
		core = zapcore.NewCore(zapcore.NewConsoleEncoder(config), ws, levelCtrl)
	}

	logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.WarnLevel), zap.AddCallerSkip(1))

	SetLevel(cfg.Level)
}

// SetLevel 动态修改日志级别
func SetLevel(level string) {
	levelCtrl.SetLevel(ParseLevel(level))
}

// GetLevel 获取当前日志级别
func GetLevel() zapcore.Level {
	return levelCtrl.Level()
}

// ParseLevel 解析日志级别字符串
func ParseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}

// GetLogInst 获取底层 zap.Logger 实例
func GetLogInst() *zap.Logger {
	return logger
}

// Sync 刷新日志缓冲区
func Sync() error {
	return logger.Sync()
}

// ----- 对外公开 API -----

func Debug(format string, v ...any) {
	logger.Sugar().Debugf(format, v...)
}

func Info(format string, v ...any) {
	logger.Sugar().Infof(format, v...)
}

func Warn(format string, v ...any) {
	logger.Sugar().Warnf(format, v...)
}

func Error(format string, v ...any) {
	logger.Sugar().Errorf(format, v...)
}

func Fatal(format string, v ...any) {
	logger.Sugar().Errorf(format, v...)
	_ = logger.Sync()
	os.Exit(1)
}

func Panic(format string, v ...any) {
	s := fmt.Sprintf(format, v...)
	logger.Sugar().Errorf(s)
	_ = logger.Sync()
	panic(s)
}

func With(args ...any) *zap.SugaredLogger {
	return logger.Sugar().With(args...)
}
