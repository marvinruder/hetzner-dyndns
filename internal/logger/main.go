package logger

import (
	"log/slog"
	"os"
	"strings"
)

func getLogLevel(level string) slog.Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

var logger = *slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	Level: getLogLevel(os.Getenv("LOG_LEVEL")),
}))

var (
	Error = logger.Error
	Warn  = logger.Warn
	Info  = logger.Info
	Debug = logger.Debug
)
