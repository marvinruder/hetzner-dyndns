package logger

import (
	"log/slog"
	"os"
)

var logger = *slog.New(slog.NewTextHandler(os.Stdout, nil))

var (
	Error = logger.Error
	Warn  = logger.Warn
	Info  = logger.Info
	Debug = logger.Debug
)
