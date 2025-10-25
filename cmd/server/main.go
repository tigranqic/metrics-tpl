package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/tigranqic/metrics-tpl/internal/config"
	"github.com/tigranqic/metrics-tpl/internal/handler"
	"github.com/tigranqic/metrics-tpl/internal/repository"
	"github.com/tigranqic/metrics-tpl/pkg/logger"
)

func main() {
	cfg, err := config.Load(false)
	if err != nil {
		slog.Error("failed to load server config", "err", err)
		os.Exit(1)
	}

	logger.Init(cfg.LogLevel, cfg.LogFormat)

	if len(flag.Args()) > 0 {
		slog.Error("unknown arguments", "args", flag.Args())
		os.Exit(1)
	}

	store := repository.NewMemStorage()
	h := handler.NewHandler(store)

	slog.Info("starting HTTP server", "address", cfg.ServerAddr)

	if err := http.ListenAndServe(cfg.ServerAddr, h.Router()); err != nil {
		slog.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}
