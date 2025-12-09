package main

import (
	"flag"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/tigranqic/metrics-tpl/internal/config"
	"github.com/tigranqic/metrics-tpl/internal/handler"
	"github.com/tigranqic/metrics-tpl/internal/middleware"
	"github.com/tigranqic/metrics-tpl/internal/repository"
	"github.com/tigranqic/metrics-tpl/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load(false)
	if err != nil {
		println("failed to load server config:", err.Error())
		os.Exit(1)
	}

	logger.Init(cfg.LogLevel, cfg.LogFormat)
	log := logger.Get()

	if len(flag.Args()) > 0 {
		log.Error("unknown arguments", zap.Strings("args", flag.Args()))
		os.Exit(1)
	}

	db, store, err := repository.InitStorage(cfg, log)
	if err != nil {
		log.Fatal("failed to initialize storage", zap.Error(err))
	}

	h := handler.NewHandler(store, db)
	loggedHandler := middleware.LoggingMiddleware(log)(h.Router())

	log.Info("starting HTTP server", zap.String("address", cfg.ServerAddr))

	if err := http.ListenAndServe(cfg.ServerAddr, loggedHandler); err != nil {
		log.Error("server stopped with error", zap.Error(err))
		os.Exit(1)
	}
}
