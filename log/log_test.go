package log

import (
	`testing`
)

func Test_Init(t *testing.T) {
	Init(
		Config{
			Format:     "json",
			Level:      "debug",
			LogPath:    "./log/test.log",
			MaxSize:    100,
			MaxBackups: 7,
			MaxAge:     30,
		})
	Info("value：%d", 1234)
	Debug("value：%d", 1234)
	Warn("value：%d", 1234)
	Error("value：%d", 1234)
	Fatal("value：%d", 1234)
	Panic("value：%d", 1234)
}

func Test_Init2(t *testing.T) {
	Init(
		Config{
			Format:     "console",
			Level:      "debug",
			LogPath:    "",
			MaxSize:    100,
			MaxBackups: 7,
			MaxAge:     30,
		})
	Info("value：%d", 1234)
	Debug("value：%d", 1234)
}
