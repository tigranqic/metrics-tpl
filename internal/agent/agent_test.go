package agent

import (
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
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodPost {
			t.Errorf("expected POST method, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "text/plain" {
			t.Errorf("expected Content-Type text/plain, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := NewAgent(server.URL, 2, 10)
	err := a.sendMetric("gauge", "Alloc", "123.45")
	if err != nil {
		t.Fatalf("sendMetric failed: %v", err)
	}

	if !strings.HasPrefix(gotPath, "/update/gauge/Alloc/123.45") {
		t.Errorf("unexpected request path: %s", gotPath)
	}
}

func TestMain(m *testing.M) {
	logger.Init("debug", "json")
	code := m.Run()
	os.Exit(code)
}
