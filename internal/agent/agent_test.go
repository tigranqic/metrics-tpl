package agent

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	models "github.com/tigranqic/metrics-tpl/internal/model"
	"github.com/tigranqic/metrics-tpl/internal/proto"
	"github.com/tigranqic/metrics-tpl/pkg/hashutil"
	"github.com/tigranqic/metrics-tpl/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
)

func TestCollectMetrics(t *testing.T) {
	a := NewAgent("http://localhost:8080", 2, 10, "", 5)
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

	a := NewAgent(server.URL, 2, 10, "", 5)
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

	a := NewAgent(server.URL, 2*time.Second, 10*time.Second, "", 5)

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

	a := NewAgent(server.URL, 2*time.Second, 10*time.Second, "", 5)

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

	a := NewAgent(server.URL, 2, 10, key, 5)

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

	a := NewAgent(server.URL, 2, 10, "", 5)

	_ = a.sendMetric("gauge", "Alloc", "1")

	if gotHash != "" {
		t.Errorf("did not expect Hash header, got %s", gotHash)
	}
}

func TestAgentRunStopsGracefully(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := NewAgent(server.URL, 100*time.Millisecond, 100*time.Millisecond, "", 2)

	stop := make(chan struct{})

	go a.Run(stop)

	time.Sleep(300 * time.Millisecond)

	a.mu.Lock()
	metricsCount := len(a.metrics)
	a.mu.Unlock()

	if metricsCount == 0 {
		t.Errorf("expected some metrics to be collected, got %d", metricsCount)
	}

	close(stop)

	time.Sleep(100 * time.Millisecond)
}

func TestSendWorkerProcessesMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := NewAgent(server.URL, 1, 1, "", 1)
	stop := make(chan struct{})
	a.workerWg.Add(1)
	go a.sendWorker(stop)

	metric := models.Metrics{ID: "TestMetric", MType: "gauge", Value: ptrFloat64(123)}
	a.sendCh <- []models.Metrics{metric}

	time.Sleep(50 * time.Millisecond)
	close(stop)
}

func TestCollectSystemMetrics(t *testing.T) {
	a := NewAgent("", 10*time.Millisecond, 10*time.Millisecond, "", 1)
	stop := make(chan struct{})
	go a.collectSystemMetrics(stop)
	time.Sleep(50 * time.Millisecond)
	close(stop)

	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.metrics["TotalMemory"]; !ok {
		t.Error("expected TotalMemory metric to be collected")
	}
	if _, ok := a.metrics["CPUutilization1"]; !ok {
		t.Error("expected CPUutilization1 metric to be collected")
	}
}

func TestSendMetricInvalidValue(t *testing.T) {
	a := NewAgent("", 1, 1, "", 1)
	err := a.sendMetric("gauge", "Alloc", "notanumber")
	if err == nil {
		t.Error("expected error for invalid gauge value")
	}
	err = a.sendMetric("counter", "PollCount", "abc")
	if err == nil {
		t.Error("expected error for invalid counter value")
	}
}

func TestSendBatchSuccess(t *testing.T) {
	sent := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sent = true
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := NewAgent(server.URL, 1, 1, "", 5)

	metrics := []models.Metrics{
		{ID: "Alloc", MType: "gauge", Value: ptrFloat64(100)},
	}

	err := a.sendBatch(metrics)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !sent {
		t.Errorf("expected batch to be sent")
	}
}

func TestRunServerNotAvailable(t *testing.T) {
	a := NewAgent("http://127.0.0.1:59999", 10*time.Millisecond, 10*time.Millisecond, "", 1)
	stop := make(chan struct{})
	go a.Run(stop)

	time.Sleep(50 * time.Millisecond)
	close(stop)
}

func TestSendMetricGzipWriteError(t *testing.T) {
	a := NewAgent("", 1, 1, "", 1)

	err := a.sendMetric("gauge", "Alloc", "NaN")
	if err == nil {
		t.Errorf("expected error due to invalid value")
	}
}

func TestSendWorkerClosedChannel(t *testing.T) {
	a := NewAgent("", 1, 1, "", 1)
	stop := make(chan struct{})

	a.workerWg.Add(1)
	go a.sendWorker(stop)
	close(stop)

	a.workerWg.Wait()
}

func ptrFloat64(f float64) *float64 { return &f }

type mockMetricsServer struct {
	proto.UnimplementedMetricsServer
	gotReq *proto.UpdateMetricsRequest
	gotIP  string
}

func (m *mockMetricsServer) UpdateMetrics(ctx context.Context, req *proto.UpdateMetricsRequest) (*proto.UpdateMetricsResponse, error) {
	m.gotReq = req
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		ips := md.Get("x-real-ip")
		if len(ips) > 0 {
			m.gotIP = ips[0]
		}
	}
	return &proto.UpdateMetricsResponse{}, nil
}

func TestAgent_reportgRPC(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	mock := &mockMetricsServer{}
	proto.RegisterMetricsServer(s, mock)
	go func() {
		if err := s.Serve(lis); err != nil && err != grpc.ErrServerStopped {
			log.Printf("gRPC server error: %v", err)
		}
	}()
	defer s.Stop()

	conn, err := grpc.NewClient("passthrough://bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("failed to close connection: %v", err)
		}
	}()
	a := NewAgent("http://localhost:8080", 2, 10, "", 1)
	a.grpcClient = proto.NewMetricsClient(conn)

	gaugeVal := 123.45
	metrics := []models.Metrics{
		{ID: "Alloc", MType: "gauge", Value: &gaugeVal},
	}

	err = a.reportgRPC(metrics)
	require.NoError(t, err)

	assert.NotNil(t, mock.gotReq)
	assert.Len(t, mock.gotReq.Metrics, 1)
	assert.Equal(t, "Alloc", mock.gotReq.Metrics[0].Id)
	assert.Equal(t, gaugeVal, mock.gotReq.Metrics[0].Value)
	assert.NotEmpty(t, mock.gotIP)
}

func TestMain(m *testing.M) {
	logger.Init("debug", "json")
	code := m.Run()
	os.Exit(code)
}
