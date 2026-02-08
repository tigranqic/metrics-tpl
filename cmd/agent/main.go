package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "net/http/pprof"

	"github.com/tigranqic/metrics-tpl/internal/agent"
	"github.com/tigranqic/metrics-tpl/internal/config"
	"github.com/tigranqic/metrics-tpl/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load(true)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	logger.Init(cfg.LogLevel, cfg.LogFormat)
	log := logger.Get()

	log.Info("starting agent", zap.String("server", cfg.ServerAddr))

	a := agent.NewAgent(cfg.ServerAddr, cfg.PollInterval, cfg.ReportInterval, cfg.Key, cfg.RateLimit)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	agentStop := make(chan struct{})

	go func() {
		log.Info("starting pprof server", zap.String("address", "localhost:6060"))
		err := http.ListenAndServe(":6060", nil)
		if err != nil {
			log.Error("pprof server failed", zap.Error(err))
		}
	}()

	go a.Run(agentStop)

	<-ctx.Done()
	log.Info("received termination signal, shutting down")

	close(agentStop)

	log.Info("agent stopped gracefully")
}
