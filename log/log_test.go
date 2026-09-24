package log

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

// 包内使用全局日志状态，测试必须串行执行并在结束时恢复状态。
func preserveLogState(t *testing.T) {
	t.Helper()
	oldLogger, oldLevel, oldWriter := logger, levelCtrl, infoWriter
	t.Cleanup(func() {
		logger, levelCtrl, infoWriter = oldLogger, oldLevel, oldWriter
	})
}

func TestParseLogLevel(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"", zapcore.DPanicLevel},
		{"unknown", zapcore.DPanicLevel},
	} {
		t.Run(tc.input, func(t *testing.T) {
			if got := ParseLogLevel(tc.input); got != tc.want {
				t.Fatalf("ParseLogLevel(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestInitLog(t *testing.T) {
	for _, encoder := range []string{"", "console", "json"} {
		t.Run("encoder="+encoder, func(t *testing.T) {
			preserveLogState(t)
			dir := t.TempDir()
			InitLog("test.log", Conf{LogPath: dir, EncoderWay: encoder, LogLevel: "debug"})
			writer := GetLogWriter()
			closer, ok := writer.(io.Closer)
			if !ok {
				t.Fatalf("file writer %T does not implement io.Closer", writer)
			}
			t.Cleanup(func() {
				if err := closer.Close(); err != nil {
					t.Errorf("close log: %v", err)
				}
			})
			if GetLogInst() == nil {
				t.Fatal("logger is nil")
			}
			Debug("debug %d", 1)
			Info("info %d", 2)
			Warn("warn %d", 3)
			Error("error %d", 4)
			data, err := os.ReadFile(filepath.Join(dir, "test.log"))
			if err != nil {
				t.Fatal(err)
			}
			messages := []string{"debug 1", "info 2", "warn 3", "error 4"}
			levels := []string{"[DEBUG]", "[INFO]", "[WARN]", "[ERROR]"}
			if encoder == "json" {
				lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
				if len(lines) != len(messages) {
					t.Fatalf("got %d JSON entries, want %d", len(lines), len(messages))
				}
				for i, line := range lines {
					var entry map[string]interface{}
					if err := json.Unmarshal(line, &entry); err != nil {
						t.Fatal(err)
					}
					if entry["msg"] != messages[i] || entry["level"] != levels[i] {
						t.Errorf("unexpected entry: %v", entry)
					}
					caller, _ := entry["file"].(string)
					if !strings.HasPrefix(caller, "[log_test.go:") {
						t.Errorf("caller should identify the logging call: %q", caller)
					}
				}
			} else {
				for i, message := range messages {
					if !bytes.Contains(data, []byte(message)) || !bytes.Contains(data, []byte(levels[i])) {
						t.Errorf("missing %s entry %q in %q", levels[i], message, data)
					}
				}
			}
		})
	}
}

func TestReloadLogLevel(t *testing.T) {
	preserveLogState(t)
	var output bytes.Buffer
	infoWriter = &output
	InitLog("", Conf{LogWay: "console", LogLevel: "info"})
	if GetLogWriter() != &output {
		t.Fatal("console writer was replaced")
	}
	if got := GetLogLevel(); got != zapcore.InfoLevel {
		t.Fatalf("initial level = %v, want info", got)
	}
	Debug("hidden debug")
	Info("visible info")
	ReloadLogLevel("debug")
	if got := GetLogLevel(); got != zapcore.DebugLevel {
		t.Fatalf("reloaded level = %v, want debug", got)
	}
	Debug("visible debug")
	ReloadLogLevel("error")
	Warn("hidden warn")
	Error("visible error")
	for _, message := range []string{"visible info", "visible debug", "visible error"} {
		if !strings.Contains(output.String(), message) {
			t.Errorf("missing message %q", message)
		}
	}
	for _, message := range []string{"hidden debug", "hidden warn"} {
		if strings.Contains(output.String(), message) {
			t.Errorf("filtered message was logged: %q", message)
		}
	}
}

func TestLogRotationCompression(t *testing.T) {
	for _, compress := range []bool{false, true} {
		name := "uncompressed"
		if compress {
			name = "gzip"
		}
		t.Run(name, func(t *testing.T) {
			preserveLogState(t)
			dir := t.TempDir()
			InitLog("test.log", Conf{LogPath: dir, LogLevel: "info", MaxSize: 1, Compress: compress})
			writer := GetLogWriter()
			closer, ok := writer.(io.Closer)
			if !ok {
				t.Fatalf("file writer %T does not implement io.Closer", writer)
			}
			t.Cleanup(func() {
				if err := closer.Close(); err != nil {
					t.Errorf("close log: %v", err)
				}
			})
			// 两次写入合计超过1 MiB，验证真实的自动轮转。
			first := strings.Repeat("a", 600*1024) + "\n"
			second := strings.Repeat("b", 600*1024) + "\n"
			for _, content := range []string{first, second} {
				if n, err := io.WriteString(writer, content); err != nil || n != len(content) {
					t.Fatalf("write: n=%d, err=%v", n, err)
				}
			}
			current, err := os.ReadFile(filepath.Join(dir, "test.log"))
			if err != nil {
				t.Fatal(err)
			}
			if string(current) != second {
				t.Fatal("active log does not contain the second write")
			}
			pattern := filepath.Join(dir, "test-*.log")
			if compress {
				pattern += ".gz"
			}
			deadline := time.Now().Add(5 * time.Second)
			var backup string
			for {
				matches, err := filepath.Glob(pattern)
				if err != nil {
					t.Fatal(err)
				}
				if len(matches) == 1 {
					backup = matches[0]
					if !compress {
						break
					}
					// gzip异步写入，源文件删除后才表示压缩完成。
					if _, err := os.Stat(strings.TrimSuffix(backup, ".gz")); os.IsNotExist(err) {
						break
					}
				}
				if time.Now().After(deadline) {
					t.Fatalf("rotation did not finish: matching backups %v", matches)
				}
				time.Sleep(10 * time.Millisecond)
			}
			data, err := os.ReadFile(backup)
			if err != nil {
				t.Fatal(err)
			}
			if compress {
				reader, err := gzip.NewReader(bytes.NewReader(data))
				if err != nil {
					t.Fatal(err)
				}
				defer reader.Close()
				data, err = io.ReadAll(reader)
				if err != nil {
					t.Fatal(err)
				}
			}
			if string(data) != first {
				t.Fatal("rotated log content differs from the first write")
			}
		})
	}
}
