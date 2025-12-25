package agent

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	models "github.com/tigranqic/metrics-tpl/internal/model"
	"github.com/tigranqic/metrics-tpl/pkg/hashutil"
	"github.com/tigranqic/metrics-tpl/pkg/logger"
)

func TestCollectMetrics(t *testing.T) {
	a := NewAgent("http://localhost:8080", 2, 10, "")
	a.collectMetrics()

	if len(a.metrics) == 0 {
		t.Error("expected some metrics to be collected")
	}
	if _, ok := a.metrics["Alloc"]; !ok {
		t.Error("expected 'Alloc' metric to be present")
	}
	if _, ok := a.metrics["PollCount"]; !ok {
		t.Error("expected 'PollCount' metric to be present")
	}
}

func TestSendMetric(t *testing.T) {
	var gotBody string
	var gotHeader string
	var gotMethod string
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotHeader = r.Header.Get("Content-Type")

		var bodyBytes []byte
		var err error

		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				t.Fatalf("failed to create gzip reader: %v", err)
			}
			defer func() {
				_ = gz.Close()
			}()
			bodyBytes, err = io.ReadAll(gz)
			if err != nil {
				t.Fatalf("failed to read gzip body: %v", err)
			}
		} else {
			bodyBytes, err = io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}
		}

		gotBody = string(bodyBytes)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := NewAgent(server.URL, 2, 10, "")
	err := a.sendMetric("gauge", "Alloc", "123.45")
	if err != nil {
		t.Fatalf("sendMetric failed: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("expected POST method, got %s", gotMethod)
	}
	if gotPath != "/update/" {
		t.Errorf("expected path /update/, got %s", gotPath)
	}

	if gotHeader != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", gotHeader)
	}

	expectedSubstrings := []string{
		`"id":"Alloc"`,
		`"type":"gauge"`,
		`"value":123.45`,
	}
	for _, substr := range expectedSubstrings {
		if !strings.Contains(gotBody, substr) {
			t.Errorf("expected body to contain %q, got %q", substr, gotBody)
		}
	}
}

func TestSendMetricRetryableHTTP(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			http.Error(w, "temporary error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := NewAgent(server.URL, 2*time.Second, 10*time.Second, "")

	err := a.sendMetric("gauge", "Alloc", "42")
	if err != nil {
		t.Fatalf("expected metric to succeed eventually, got error: %v", err)
	}

	if callCount != 3 {
		t.Errorf("expected 3 attempts, got %d", callCount)
	}
}

func TestSendBatchFallbackRetryableHTTP(t *testing.T) {
	callCount := 0
	sentMetrics := []string{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Path == "/updates/" {
			http.NotFound(w, r)
			return
		}

		var bodyBytes []byte
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, _ := gzip.NewReader(r.Body)
			bodyBytes, _ = io.ReadAll(gz)
			_ = gz.Close()
		} else {
			bodyBytes, _ = io.ReadAll(r.Body)
		}

		var m models.Metrics
		if err := json.Unmarshal(bodyBytes, &m); err == nil {
			sentMetrics = append(sentMetrics, m.ID)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := NewAgent(server.URL, 2*time.Second, 10*time.Second, "")

	metrics := []models.Metrics{
		{ID: "Alloc", MType: "gauge", Value: ptrFloat64(100)},
		{ID: "Random", MType: "gauge", Value: ptrFloat64(42)},
	}

	err := a.sendBatch(metrics)
	if err != nil {
		t.Fatalf("sendBatch failed: %v", err)
	}

	if len(sentMetrics) != 2 {
		t.Errorf("expected 2 metrics sent individually, got %d", len(sentMetrics))
	}
}

func TestSendMetricWithHash(t *testing.T) {
	const key = "secret"

	var gotHash string
	var gotBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHash = r.Header.Get("Hash")

		var body []byte
		if r.Header.Get("Content-Encoding") == "gzip" {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				t.Fatal(err)
			}
			body, _ = io.ReadAll(gz)
			_ = gz.Close()
		} else {
			body, _ = io.ReadAll(r.Body)
		}

		gotBody = body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := NewAgent(server.URL, 2, 10, key)

	err := a.sendMetric("gauge", "Alloc", "123.45")
	if err != nil {
		t.Fatalf("sendMetric failed: %v", err)
	}

	expectedHash := hashutil.CalcSHA256(gotBody, key)

	if gotHash == "" {
		t.Fatal("expected Hash header to be set")
	}

	if gotHash != expectedHash {
		t.Fatalf("invalid hash: expected %s, got %s", expectedHash, gotHash)
	}
}

func TestSendMetricWithoutKey_NoHash(t *testing.T) {
	var gotHash string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHash = r.Header.Get("Hash")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := NewAgent(server.URL, 2, 10, "")

	_ = a.sendMetric("gauge", "Alloc", "1")

	if gotHash != "" {
		t.Errorf("did not expect Hash header, got %s", gotHash)
	}
}

func ptrFloat64(f float64) *float64 { return &f }

func TestMain(m *testing.M) {
	logger.Init("debug", "json")
	code := m.Run()
	os.Exit(code)
}
