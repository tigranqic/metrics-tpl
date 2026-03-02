package logger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestInit(t *testing.T) {
	Init("debug", "text")
	assert.NotNil(t, Get())
	Init("info", "json")
	assert.NotNil(t, Get())
}

func TestParseLevel(t *testing.T) {
	assert.Equal(t, zapcore.DebugLevel, parseLevel("debug"))
	assert.Equal(t, zapcore.InfoLevel, parseLevel("info"))
	assert.Equal(t, zapcore.WarnLevel, parseLevel("warn"))
	assert.Equal(t, zapcore.ErrorLevel, parseLevel("error"))
	assert.Equal(t, zapcore.InfoLevel, parseLevel("unknown"))
}

func TestFormatOrDefault(t *testing.T) {
	assert.Equal(t, "text", formatOrDefault(""))
	assert.Equal(t, "json", formatOrDefault("json"))
}

func TestGet_Nil(t *testing.T) {
	log = nil
	assert.NotNil(t, Get())
}
