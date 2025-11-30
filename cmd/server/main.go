package main

import (
	"flag"
	"net/http"
	"os"

	"database/sql"

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

	var db *sql.DB
	if cfg.DatabaseDSN != "" {
		db, err = sql.Open("postgres", cfg.DatabaseDSN)
		if err != nil {
			log.Error("failed to open DB connection", zap.Error(err))
			os.Exit(1)
		}

		if err := db.Ping(); err != nil {
			log.Error("failed to ping DB", zap.Error(err))
			os.Exit(1)
		}
		log.Info("DB connection established")
	} else {
		log.Info("no DB configured – running in memory mode")
	}

	store := repository.NewMemStorage(cfg.FileStoragePath, cfg.StoreInterval)

	if cfg.Restore {
		if err := store.LoadFromFile(cfg.FileStoragePath); err != nil {
			log.Error("failed to restore metrics", zap.Error(err))
		} else {
			log.Info("metrics restored successfully", zap.String("file", cfg.FileStoragePath))
		}
	}

	stopCh := make(chan struct{})
	if cfg.StoreInterval > 0 {
		store.StartAutoSave(cfg.FileStoragePath, cfg.StoreInterval, stopCh)
		defer close(stopCh)
	}

	h := handler.NewHandler(store, db)
	loggedHandler := middleware.LoggingMiddleware(log)(h.Router())

	log.Info("starting HTTP server", zap.String("address", cfg.ServerAddr))

	if err := http.ListenAndServe(cfg.ServerAddr, loggedHandler); err != nil {
		log.Error("server stopped with error", zap.Error(err))
		os.Exit(1)
	}
}
