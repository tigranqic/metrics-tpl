package logger

import (
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
		panic(err)
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
