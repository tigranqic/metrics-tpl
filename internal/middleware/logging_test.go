package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestLoggingMiddleware_DefaultStatusOK(t *testing.T) {
	logger := zap.NewNop()

	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, _ = w.Write([]byte("ok"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	LoggingMiddleware(logger)(handler).ServeHTTP(rec, req)

	assert.True(t, handlerCalled, "handler should be called")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestLoggingMiddleware_CustomStatusAndSize(t *testing.T) {
	logger := zap.NewNop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("hello"))
	})

	req := httptest.NewRequest(http.MethodPost, "/create", nil)
	rec := httptest.NewRecorder()

	LoggingMiddleware(logger)(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())
	assert.Len(t, rec.Body.Bytes(), 5)
}

func TestLoggingMiddleware_ZeroBody(t *testing.T) {
	logger := zap.NewNop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/empty", nil)
	rec := httptest.NewRecorder()

	LoggingMiddleware(logger)(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestLoggingMiddleware_MultipleWrites(t *testing.T) {
	logger := zap.NewNop()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("he"))
		_, _ = w.Write([]byte("llo"))
	})

	req := httptest.NewRequest(http.MethodGet, "/multi", nil)
	rec := httptest.NewRecorder()

	LoggingMiddleware(logger)(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())
	assert.Len(t, rec.Body.Bytes(), 5)
}
