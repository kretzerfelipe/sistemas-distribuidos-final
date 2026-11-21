package logger

import (
	"log/slog"
	"os"
	"strings"
)

const serviceName = "main"

var levelsByName = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

func New(levelName string) *slog.Logger {
	level, levelIsKnown := levelsByName[strings.ToLower(levelName)]
	if !levelIsKnown {
		level = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler).With("service", serviceName)
}
