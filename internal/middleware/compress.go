package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

func GzipCompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gzw := gzip.NewWriter(w)
		defer func() {
			_ = gzw.Close()
		}()

		gzrw := &gzipResponseWriter{
			ResponseWriter: w,
			Writer:         gzw,
		}

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Del("Content-Length")

		next.ServeHTTP(gzrw, r)

		defer func() {
			_ = gzw.Flush()
		}()
	})
}

type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Header() http.Header {
	return w.ResponseWriter.Header()
}

func GzipDecompress(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "failed to read gzip body", http.StatusBadRequest)
				return
			}
			defer func() {
				_ = gz.Close()
			}()
			r.Body = io.NopCloser(gz)
		}
		next.ServeHTTP(w, r)
	})
}
