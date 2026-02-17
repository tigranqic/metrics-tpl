// Package logger provides initialization and access to a global Zap logger.
// It supports configurable log levels and output formats (JSON or text),
// and ensures a fallback to a no-op logger if initialization fails.
package logger

import (
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

func Init(levelStr, formatStr string) {
	level := parseLevel(levelStr)
	format := strings.ToLower(strings.TrimSpace(formatStr))

	var cfg zap.Config

	switch format {
	case "json":
		cfg = zap.NewProductionConfig()
	default:
		cfg = zap.NewDevelopmentConfig()
	}

	cfg.Level = zap.NewAtomicLevelAt(level)

	var err error
	log, err = cfg.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %v\n", err)
		log = zap.NewNop()
		return
	}

	log.Info("logger initialized",
		zap.String("level", level.String()),
		zap.String("format", formatOrDefault(format)))
}

func parseLevel(levelStr string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func formatOrDefault(format string) string {
	if format == "" {
		return "text"
	}
	return format
}

func Get() *zap.Logger {
	if log == nil {
		return zap.NewNop()
	}
	return log
}
