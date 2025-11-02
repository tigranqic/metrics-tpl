package agent

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"time"

	models "github.com/tigranqic/metrics-tpl/internal/model"
	"github.com/tigranqic/metrics-tpl/pkg/logger"
	"go.uber.org/zap"
)

type Agent struct {
	ServerURL      string
	PollInterval   time.Duration
	ReportInterval time.Duration

	pollCount             int64
	lastReportedPollCount int64
	client                *http.Client

	metrics map[string]string
	log     *zap.Logger
}

func NewAgent(serverURL string, pollInterval, reportInterval time.Duration) *Agent {
	return &Agent{
		ServerURL:      serverURL,
		PollInterval:   pollInterval,
		ReportInterval: reportInterval,
		client:         &http.Client{Timeout: 5 * time.Second},
		metrics:        make(map[string]string),
		log:            logger.Get(),
	}
}

func (a *Agent) collectMetrics() {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	a.metrics["Alloc"] = strconv.FormatFloat(float64(memStats.Alloc), 'f', -1, 64)
	a.metrics["BuckHashSys"] = strconv.FormatFloat(float64(memStats.BuckHashSys), 'f', -1, 64)
	a.metrics["Frees"] = strconv.FormatFloat(float64(memStats.Frees), 'f', -1, 64)
	a.metrics["GCCPUFraction"] = strconv.FormatFloat(memStats.GCCPUFraction, 'f', -1, 64)
	a.metrics["GCSys"] = strconv.FormatFloat(float64(memStats.GCSys), 'f', -1, 64)
	a.metrics["HeapAlloc"] = strconv.FormatFloat(float64(memStats.HeapAlloc), 'f', -1, 64)
	a.metrics["HeapIdle"] = strconv.FormatFloat(float64(memStats.HeapIdle), 'f', -1, 64)
	a.metrics["HeapInuse"] = strconv.FormatFloat(float64(memStats.HeapInuse), 'f', -1, 64)
	a.metrics["HeapObjects"] = strconv.FormatFloat(float64(memStats.HeapObjects), 'f', -1, 64)
	a.metrics["HeapReleased"] = strconv.FormatFloat(float64(memStats.HeapReleased), 'f', -1, 64)
	a.metrics["HeapSys"] = strconv.FormatFloat(float64(memStats.HeapSys), 'f', -1, 64)
	a.metrics["LastGC"] = strconv.FormatFloat(float64(memStats.LastGC), 'f', -1, 64)
	a.metrics["Lookups"] = strconv.FormatFloat(float64(memStats.Lookups), 'f', -1, 64)
	a.metrics["MCacheInuse"] = strconv.FormatFloat(float64(memStats.MCacheInuse), 'f', -1, 64)
	a.metrics["MCacheSys"] = strconv.FormatFloat(float64(memStats.MCacheSys), 'f', -1, 64)
	a.metrics["MSpanInuse"] = strconv.FormatFloat(float64(memStats.MSpanInuse), 'f', -1, 64)
	a.metrics["MSpanSys"] = strconv.FormatFloat(float64(memStats.MSpanSys), 'f', -1, 64)
	a.metrics["Mallocs"] = strconv.FormatFloat(float64(memStats.Mallocs), 'f', -1, 64)
	a.metrics["NextGC"] = strconv.FormatFloat(float64(memStats.NextGC), 'f', -1, 64)
	a.metrics["NumForcedGC"] = strconv.FormatInt(int64(memStats.NumForcedGC), 10)
	a.metrics["NumGC"] = strconv.FormatInt(int64(memStats.NumGC), 10)
	a.metrics["OtherSys"] = strconv.FormatFloat(float64(memStats.OtherSys), 'f', -1, 64)
	a.metrics["PauseTotalNs"] = strconv.FormatFloat(float64(memStats.PauseTotalNs), 'f', -1, 64)
	a.metrics["StackInuse"] = strconv.FormatFloat(float64(memStats.StackInuse), 'f', -1, 64)
	a.metrics["StackSys"] = strconv.FormatFloat(float64(memStats.StackSys), 'f', -1, 64)
	a.metrics["Sys"] = strconv.FormatFloat(float64(memStats.Sys), 'f', -1, 64)
	a.metrics["TotalAlloc"] = strconv.FormatFloat(float64(memStats.TotalAlloc), 'f', -1, 64)

	// Special metrics
	a.pollCount++
	delta := a.pollCount - a.lastReportedPollCount
	a.metrics["PollCount"] = strconv.FormatInt(delta, 10)
	a.metrics["RandomValue"] = strconv.FormatFloat(rand.Float64()*1000, 'f', 3, 64)
}

func (a *Agent) sendMetric(metricType, name, value string) error {
	fullURL := a.ServerURL + "/update/"

	var m models.Metrics
	m.ID = name
	m.MType = metricType

	switch metricType {
	case "gauge":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		m.Value = &v
	case "counter":
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		m.Delta = &v
	}

	body, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal metric: %w", err)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(body); err != nil {
		return fmt.Errorf("gzip write failed: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("gzip close failed: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fullURL, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Accept-Encoding", "gzip")

	start := time.Now()
	resp, err := a.client.Do(req)
	duration := time.Since(start)

	if err != nil {
		a.log.Error("failed to send request", zap.String("url", fullURL), zap.Error(err), zap.Duration("duration", duration))
		return fmt.Errorf("failed to send request to %q: %w", fullURL, err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	a.log.Info("metric sent",
		zap.String("url", fullURL),
		zap.Int("status", resp.StatusCode),
		zap.Duration("duration", duration),
	)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s", resp.Status)
	}

	return nil
}

func waitForServer(baseURL string, timeout time.Duration, log *zap.Logger) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	log.Info("waiting for server", zap.String("url", baseURL), zap.Duration("timeout", timeout))

	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/ping")
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			log.Info("server is ready", zap.String("url", baseURL))
			return nil
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("server %s not responding within %s", baseURL, timeout)
}

func (a *Agent) Run(stop <-chan struct{}) {
	a.log.Info("starting agent loop", zap.String("server", a.ServerURL))

	if err := waitForServer(a.ServerURL, 10*time.Second, a.log); err != nil {
		a.log.Error("server not available", zap.String("url", a.ServerURL), zap.Error(err))
		return
	}

	pollTicker := time.NewTicker(a.PollInterval)
	reportTicker := time.NewTicker(a.ReportInterval)

	defer pollTicker.Stop()
	defer reportTicker.Stop()

	a.collectMetrics()

	for {
		select {
		case <-pollTicker.C:
			a.log.Debug("collecting metrics")
			a.collectMetrics()

		case <-reportTicker.C:
			a.log.Debug("sending metrics batch", zap.Int("count", len(a.metrics)))
			for name, val := range a.metrics {
				var metricType string
				if name == "PollCount" {
					metricType = "counter"
				} else {
					metricType = "gauge"
				}
				if err := a.sendMetric(metricType, name, val); err != nil {
					a.log.Error("failed to send metric", zap.String("name", name), zap.Error(err))
				}
			}
			a.lastReportedPollCount = a.pollCount

		case <-stop:
			a.log.Info("agent stopped gracefully")
			return
		}
	}
}
