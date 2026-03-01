package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/tigranqic/metrics-tpl/internal/agent"
	"github.com/tigranqic/metrics-tpl/internal/config"
	"github.com/tigranqic/metrics-tpl/pkg/logger"
	"go.uber.org/zap"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

// main initializes and runs the metrics agent, handling graceful shutdown.
// It sets up logging, configuration, the agent instance, and optional pprof profiling.
func main() {
	// Print build information
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	// Load agent configuration
	cfg, err := config.Load(true)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	// Initialize structured logger
	logger.Init(cfg.LogLevel, cfg.LogFormat)
	log := logger.Get()

	log.Info("starting agent", zap.String("server", cfg.ServerAddr))

	// Create a new agent instance
	a := agent.NewAgent(cfg.ServerAddr, cfg.PollInterval, cfg.ReportInterval, cfg.Key, cfg.RateLimit)

	// Set gRPC address if provided
	if err := a.SetGRPC(cfg.GRPCAddr); err != nil {
		log.Fatal("failed to set gRPC address", zap.Error(err))
	}

	// Load crypto key if provided
	if cfg.CryptoKey != "" {
		if err := a.SetCryptoKey(cfg.CryptoKey); err != nil {
			log.Fatal("failed to load crypto key", zap.Error(err))
		}
	}

	// Context to handle OS signals for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	// Channel to signal agent to stop
	agentStop := make(chan struct{})

	// Start pprof server in a separate goroutine
	go func() {
		log.Info("starting pprof server", zap.String("address", "localhost:6060"))
		err := http.ListenAndServe(":6060", nil)
		if err != nil {
			log.Error("pprof server failed", zap.Error(err))
		}
	}()

	// Start the agent in a separate goroutine
	go a.Run(agentStop)

	// Wait for termination signal
	<-ctx.Done()
	log.Info("received termination signal, starting graceful shutdown")

	// Signal agent to stop collecting and reporting new metrics
	close(agentStop)

	// Give workers time to drain the metrics channel and send all pending metrics
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Wait for all workers to finish or timeout
	a.WaitForShutdown(shutdownCtx)

	log.Info("agent stopped gracefully")
}
