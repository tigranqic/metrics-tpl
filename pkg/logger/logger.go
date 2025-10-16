package logger

import (
	"log/slog"
	"os"
	"strings"
)

func Init(levelStr, formatStr string) {
	level := parseLevel(levelStr)
	format := strings.ToLower(strings.TrimSpace(formatStr))

	var handler slog.Handler

	switch format {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	default:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	slog.Info("logger initialized", "level", level.String(), "format", formatOrDefault(format))
}

func parseLevel(levelStr string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func formatOrDefault(format string) string {
	if format == "" {
		return "text"
	}
	return format
}
