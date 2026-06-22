package log

import (
	`testing`
)

func TestInitLog(t *testing.T) {

	InitLog("test.log", Conf{
		LogWay:     "console",
		EncoderWay: "json",
		LogLevel:   "debug",
		LogPath:    "./logs/",
		MaxDays:    7,
		MaxSize:    100,
	})
	Info("hello world")
	Debug("hello world")
	Error("hello world")
	Warn("hello world")
}
