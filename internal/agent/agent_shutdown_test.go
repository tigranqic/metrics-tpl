package agent

import (
	"context"
	"strconv"
	"testing"
	"time"

	models "github.com/tigranqic/metrics-tpl/internal/model"
	"github.com/tigranqic/metrics-tpl/pkg/logger"
)

// TestGracefulShutdown verifies that agent gracefully shuts down and drains remaining metrics
func TestGracefulShutdown(t *testing.T) {
	logger.Init("debug", "text")

	t.Run("Agent drains metrics during shutdown", func(t *testing.T) {
		agent := NewAgent("http://localhost:9999", 1*time.Second, 2*time.Second, "", 2)
		// Reduce retries for faster test execution
		agent.client.RetryMax = 0

		// Add some metrics to the send channel
		stop := make(chan struct{})
		agent.workerWg.Add(2)

		// Start two workers
		go agent.sendWorker(stop)
		go agent.sendWorker(stop)

		// Add metrics
		m1 := models.Metrics{ID: "test1", MType: "gauge", Value: func() *float64 { v := 1.0; return &v }()}
		m2 := models.Metrics{ID: "test2", MType: "gauge", Value: func() *float64 { v := 2.0; return &v }()}
		m3 := models.Metrics{ID: "test3", MType: "gauge", Value: func() *float64 { v := 3.0; return &v }()}

		agent.sendCh <- []models.Metrics{m1, m2, m3}

		time.Sleep(100 * time.Millisecond)

		// Close stop channel to signal graceful shutdown
		close(stop)

		// Wait for shutdown with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		agent.WaitForShutdown(ctx)

		// Verify workers have finished
		select {
		case <-ctx.Done():
			t.Fatal("shutdown timeout exceeded")
		default:
			// Workers should be done
		}
	})

	t.Run("Agent shutdown with context timeout", func(t *testing.T) {
		agent := NewAgent("http://localhost:9999", 1*time.Second, 2*time.Second, "", 1)
		agent.client.RetryMax = 0

		stop := make(chan struct{})
		agent.workerWg.Add(1)
		go agent.sendWorker(stop)

		// Close stop to signal shutdown
		close(stop)

		// Create a very short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Should handle timeout gracefully
		agent.WaitForShutdown(ctx)
	})

	t.Run("Multiple workers drain queue", func(t *testing.T) {
		agent := NewAgent("http://localhost:9999", 1*time.Second, 2*time.Second, "", 3)
		agent.client.RetryMax = 0

		stop := make(chan struct{})
		agent.workerWg.Add(3)

		// Start three workers
		for i := 0; i < 3; i++ {
			go agent.sendWorker(stop)
		}

		// Add metrics
		for i := 0; i < 10; i++ {
			m := models.Metrics{
				ID:    "test" + strconv.Itoa(i),
				MType: "gauge",
				Value: func(v float64) *float64 { return &v }(float64(i)),
			}
			agent.sendCh <- []models.Metrics{m}
		}

		// Give workers time to process some metrics
		time.Sleep(100 * time.Millisecond)

		// Signal shutdown
		close(stop)

		// Wait for all workers to drain
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		agent.WaitForShutdown(ctx)
	})
}

// TestAgentWorkerDrain tests that sendWorker properly drains remaining metrics
func TestAgentWorkerDrain(t *testing.T) {
	logger.Init("debug", "text")

	agent := NewAgent("http://localhost:9999", 1*time.Second, 2*time.Second, "", 1)
	// Disable retries for faster test execution
	agent.client.RetryMax = 0

	stop := make(chan struct{})
	agent.workerWg.Add(1)

	// Channel to track when worker exits
	done := make(chan struct{})

	go func() {
		agent.sendWorker(stop)
		close(done)
	}()

	// Add some metrics
	m1 := models.Metrics{ID: "test1", MType: "gauge", Value: func() *float64 { v := 1.0; return &v }()}
	agent.sendCh <- []models.Metrics{m1}

	// Give worker time to process
	time.Sleep(50 * time.Millisecond)

	// Add more metrics and close stop channel
	m2 := models.Metrics{ID: "test2", MType: "gauge", Value: func() *float64 { v := 2.0; return &v }()}
	agent.sendCh <- []models.Metrics{m2}

	close(stop)

	// Wait for worker to drain and exit (with longer timeout for safety)
	select {
	case <-done:
		// Worker exited as expected
	case <-time.After(5 * time.Second):
		t.Fatal("worker did not exit after drain")
	}
}
