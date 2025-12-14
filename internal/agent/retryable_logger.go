package agent

import (
	"go.uber.org/zap"
)

type RetryableLogger struct {
	log *zap.Logger
}

func (l *RetryableLogger) Error(msg string, keysAndValues ...interface{}) {
	l.log.Sugar().Errorw(msg, keysAndValues...)
}

func (l *RetryableLogger) Info(msg string, keysAndValues ...interface{}) {
	l.log.Sugar().Infow(msg, keysAndValues...)
}

func (l *RetryableLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.log.Sugar().Debugw(msg, keysAndValues...)
}

func (l *RetryableLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.log.Sugar().Warnw(msg, keysAndValues...)
}
