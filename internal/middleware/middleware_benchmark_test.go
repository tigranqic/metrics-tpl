package middleware

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"testing"
)

func BenchmarkGzipCompress_Small(b *testing.B) {
	data := []byte(`{"id":"test_metric","type":"gauge","value":123.45}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		_, _ = w.Write(data)
		_ = w.Close()
	}
}

func BenchmarkGzipCompress_Medium(b *testing.B) {
	response := `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>Metrics</title></head><body><h1>Metrics</h1><ul>`
	for i := 0; i < 100; i++ {
		response += `<li>metric_` + string(rune(i%10)) + ` (gauge): ` + string(rune(i%100)) + `.5</li>`
	}
	response += `</ul></body></html>`
	data := []byte(response)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		_, _ = w.Write(data)
		_ = w.Close()
	}
}

func BenchmarkGzipCompress_Large(b *testing.B) {
	response := `[`
	for i := 0; i < 1000; i++ {
		if i > 0 {
			response += `,`
		}
		response += `{"id":"metric_` + string(rune(i%10)) + `","type":"gauge","value":` + string(rune(i%100)) + `.5}`
	}
	response += `]`
	data := []byte(response)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		_, _ = w.Write(data)
		_ = w.Close()
	}
}

func BenchmarkGzipDecompress_Small(b *testing.B) {
	data := []byte(`{"id":"test_metric","type":"gauge","value":123.45}`)

	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	_, _ = w.Write(data)
	_ = w.Close()
	compressedData := compressed.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, _ := gzip.NewReader(bytes.NewReader(compressedData))
		_, _ = io.ReadAll(r)
		_ = r.Close()
	}
}

func BenchmarkGzipDecompress_Medium(b *testing.B) {
	response := `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>Metrics</title></head><body><h1>Metrics</h1><ul>`
	for i := 0; i < 100; i++ {
		response += `<li>metric_` + string(rune(i%10)) + ` (gauge): ` + string(rune(i%100)) + `.5</li>`
	}
	response += `</ul></body></html>`
	data := []byte(response)

	var compressed bytes.Buffer
	w := gzip.NewWriter(&compressed)
	_, _ = w.Write(data)
	_ = w.Close()
	compressedData := compressed.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r, _ := gzip.NewReader(bytes.NewReader(compressedData))
		_, _ = io.ReadAll(r)
		_ = r.Close()
	}
}

func BenchmarkHash_Small(b *testing.B) {
	data := []byte(`{"id":"test","type":"gauge","value":123.45}`)
	key := "test_key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateHashBench(data, key)
	}
}

func BenchmarkHash_Medium(b *testing.B) {
	data := []byte(`[{"id":"metric_0","type":"gauge","value":0.5},{"id":"metric_1","type":"gauge","value":1.5},{"id":"metric_2","type":"gauge","value":2.5},{"id":"metric_3","type":"gauge","value":3.5},{"id":"metric_4","type":"gauge","value":4.5},{"id":"metric_5","type":"gauge","value":5.5},{"id":"metric_6","type":"gauge","value":6.5},{"id":"metric_7","type":"gauge","value":7.5},{"id":"metric_8","type":"gauge","value":8.5},{"id":"metric_9","type":"gauge","value":9.5}]`)
	key := "test_key"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = generateHashBench(data, key)
	}
}

func generateHashBench(data []byte, key string) string {
	h := sha256.New()
	h.Write(data)
	h.Write([]byte(key))
	return fmt.Sprintf("%x", h.Sum(nil))
}
