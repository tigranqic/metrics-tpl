package agent

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/tigranqic/metrics-tpl/pkg/logger"
)

func TestCollectMetrics(t *testing.T) {
	a := NewAgent("http://localhost:8080", 2, 10)
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

	// сервер распаковывает gzip, если Content-Encoding: gzip
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

	a := NewAgent(server.URL, 2, 10)
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

func TestMain(m *testing.M) {
	logger.Init("debug", "json")
	code := m.Run()
	os.Exit(code)
}
