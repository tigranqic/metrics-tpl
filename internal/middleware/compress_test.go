package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGzipCompress_WritesCompressed(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello gzip"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	GzipCompress(handler).ServeHTTP(rec, req)

	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"))

	gr, err := gzip.NewReader(rec.Body)
	assert.NoError(t, err)
	defer gr.Close()

	body, err := io.ReadAll(gr)
	assert.NoError(t, err)
	assert.Equal(t, "hello gzip", string(body))
}

func TestGzipCompress_SkipsWithoutAcceptEncoding(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plain body"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	GzipCompress(handler).ServeHTTP(rec, req)

	assert.Equal(t, "", rec.Header().Get("Content-Encoding"))
	assert.Equal(t, "plain body", rec.Body.String())
}

func TestGzipDecompress_HandlesGzip(t *testing.T) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	_, _ = gzw.Write([]byte("compressed request"))
	_ = gzw.Close()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
	})

	req := httptest.NewRequest(http.MethodPost, "/", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	GzipDecompress(handler).ServeHTTP(rec, req)

	assert.Equal(t, "compressed request", rec.Body.String())
}

func TestGzipDecompress_PassesThroughNonGzip(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("plain request"))
	rec := httptest.NewRecorder()

	GzipDecompress(handler).ServeHTTP(rec, req)

	assert.Equal(t, "plain request", rec.Body.String())
}

func TestGzipDecompress_InvalidGzip(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("should not reach"))
	})

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("invalid gzip"))
	req.Header.Set("Content-Encoding", "gzip")
	rec := httptest.NewRecorder()

	GzipDecompress(handler).ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to read gzip body")
}
