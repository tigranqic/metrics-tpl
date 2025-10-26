package agent

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"time"
)

type Agent struct {
	ServerURL      string
	PollInterval   time.Duration
	ReportInterval time.Duration

	pollCount             int64
	lastReportedPollCount int64
	client                *http.Client

	metrics map[string]string
}

func NewAgent(serverURL string, pollInterval, reportInterval time.Duration) *Agent {
	return &Agent{
		ServerURL:      serverURL,
		PollInterval:   pollInterval,
		ReportInterval: reportInterval,
		client:         &http.Client{Timeout: 5 * time.Second},
		metrics:        make(map[string]string),
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
	slog.Info("url from iter test", "url", a.ServerURL)
	fullURL, err := url.JoinPath(a.ServerURL, "update", metricType, name, value)
	if err != nil {
		return fmt.Errorf("failed to build URL: %w", err)
	}

	slog.Info(fullURL, "full url", fullURL)
	req, err := http.NewRequest(http.MethodPost, fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for %q: %w", fullURL, err)
	}
	req.Header.Set("Content-Type", "text/plain")
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to %q: %w", fullURL, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Warn("error closing response body", "error", err)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s", resp.Status)
	}
	return nil
}

func waitForServer(url string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{
		Timeout: 2 * time.Second,
	}
	slog.Info("waitf or service url", url, url)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url + "/ping")
		if err == nil && resp.StatusCode == http.StatusOK {
			defer func() {
				if err := resp.Body.Close(); err != nil {
					slog.Warn("error closing response body", "error", err)
				}
			}()
			return nil
		}
		if resp != nil {
			defer func() {
				if err := resp.Body.Close(); err != nil {
					slog.Warn("error closing response body", "error", err)
				}
			}()
		}
		slog.Debug("waiting for server", "url", url)
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("server %s not responding within %s", url, timeout)
}

func (a *Agent) Run(stop <-chan struct{}) {
	slog.Info("starting agent loop", "server", a.ServerURL)

	if err := waitForServer(a.ServerURL, 10*time.Second); err != nil {
		slog.Error("server not available", "url", a.ServerURL, "error", err)
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
			slog.Debug("collecting metrics")
			a.collectMetrics()

		case <-reportTicker.C:
			slog.Debug("sending metrics batch", "count", len(a.metrics))
			for name, val := range a.metrics {
				var metricType string
				if name == "NumForcedGC" || name == "NumGC" || name == "PollCount" {
					metricType = "counter"
				} else {
					metricType = "gauge"
				}
				if err := a.sendMetric(metricType, name, val); err != nil {
					slog.Error("failed to send metric", "name", name, "error", err)
				}
			}
			a.lastReportedPollCount = a.pollCount

		case <-stop:
			slog.Info("agent stopped gracefully")
			return
		}
	}
}
