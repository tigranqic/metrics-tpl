package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/tigranqic/metrics-tpl/internal/agent"
	"github.com/tigranqic/metrics-tpl/internal/config"
	"github.com/tigranqic/metrics-tpl/pkg/logger"
)

func main() {
	cfg, err := config.Load(true)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	logger.Init(cfg.LogLevel, cfg.LogFormat)
	slog.Info("starting agent", "server", cfg.ServerAddr)

	a := agent.NewAgent(cfg.ServerAddr, cfg.PollInterval, cfg.ReportInterval)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	agentStop := make(chan struct{})

	go a.Run(agentStop)

	<-ctx.Done()
	slog.Info("received termination signal, shutting down")

	close(agentStop)

	slog.Info("agent stopped gracefully")
}
