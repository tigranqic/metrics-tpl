package middleware

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tigranqic/metrics-tpl/pkg/hashutil"
	"go.uber.org/zap"
)

func TestHashMiddleware_SkipsIfNoKey(t *testing.T) {
	logger := zap.NewNop()
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, err := w.Write([]byte("ok"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("body"))
	rec := httptest.NewRecorder()

	m := NewHashMiddleware("", logger)
	m.Handle(handler).ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestHashMiddleware_RejectsInvalidHash(t *testing.T) {
	logger := zap.NewNop()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("body"))
	req.Header.Set("Hash", "invalidhash")
	rec := httptest.NewRecorder()

	m := NewHashMiddleware("secret", logger)
	m.Handle(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid hash")
}

func TestHashMiddleware_AllowsCorrectHash(t *testing.T) {
	logger := zap.NewNop()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})

	body := []byte("body")
	hash := hashutil.CalcSHA256(body, "secret")

	req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewReader(body))
	req.Header.Set("Hash", hash)
	rec := httptest.NewRecorder()

	m := NewHashMiddleware("secret", logger)
	m.Handle(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "ok", rec.Body.String())
	assert.NotEmpty(t, rec.Header().Get("Hash"))
}

func TestHashMiddleware_SetsResponseHash(t *testing.T) {
	logger := zap.NewNop()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("response"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	m := NewHashMiddleware("secret", logger)
	m.Handle(handler).ServeHTTP(rec, req)

	expectedHash := hashutil.CalcSHA256([]byte("response"), "secret")
	assert.Equal(t, expectedHash, rec.Header().Get("Hash"))
	assert.Equal(t, "response", rec.Body.String())
}

func TestHashMiddleware_BypassHashNone(t *testing.T) {
	logger := zap.NewNop()
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		_, err := w.Write([]byte("ok"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/test", strings.NewReader("body"))
	req.Header.Set("Hash", "none")
	rec := httptest.NewRecorder()

	m := NewHashMiddleware("secret", logger)
	m.Handle(handler).ServeHTTP(rec, req)

	assert.True(t, handlerCalled)
	assert.Equal(t, "ok", rec.Body.String())
}

func TestHashMiddleware_ReadBodyError(t *testing.T) {
	logger := zap.NewNop()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("ok"))
		if err != nil {
			t.Fatalf("failed to write response: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/test", errorReader{})
	rec := httptest.NewRecorder()

	m := NewHashMiddleware("secret", logger)
	m.Handle(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to read body")
}

type errorReader struct{}

func (e errorReader) Read(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (e errorReader) Close() error               { return nil }
