package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	_ "net/http/pprof"

	_ "github.com/lib/pq"

	"github.com/tigranqic/metrics-tpl/internal/audit"
	"github.com/tigranqic/metrics-tpl/internal/config"
	"github.com/tigranqic/metrics-tpl/internal/handler"
	"github.com/tigranqic/metrics-tpl/internal/middleware"
	"github.com/tigranqic/metrics-tpl/internal/repository"
	"github.com/tigranqic/metrics-tpl/pkg/cryptoutil"
	"github.com/tigranqic/metrics-tpl/pkg/logger"
	"go.uber.org/zap"
)

var (
	buildVersion string = "N/A"
	buildDate    string = "N/A"
	buildCommit  string = "N/A"
)

// main initializes configuration, logging, storage, and starts the HTTP server.
func main() {
	// Print build information
	fmt.Printf("Build version: %s\n", buildVersion)
	fmt.Printf("Build date: %s\n", buildDate)
	fmt.Printf("Build commit: %s\n", buildCommit)

	// Load server configuration
	cfg, err := config.Load(false)
	if err != nil {
		println("failed to load server config:", err.Error())
		os.Exit(1)
	}

	// Initialize logger
	logger.Init(cfg.LogLevel, cfg.LogFormat)
	log := logger.Get()

	// Check for unknown command-line arguments
	if len(flag.Args()) > 0 {
		log.Error("unknown arguments", zap.Strings("args", flag.Args()))
		os.Exit(1)
	}

	// Initialize database and storage
	db, store, err := repository.InitStorage(cfg, log)
	if err != nil {
		log.Fatal("failed to initialize storage", zap.Error(err))
	}

	// Setup audit observers
	var observers []audit.Observer

	if cfg.AuditFile != "" {
		fo, err := audit.NewFileObserver(cfg.AuditFile)
		if err != nil {
			log.Fatal("failed to init file audit observer", zap.Error(err))
		}
		observers = append(observers, fo)
	}

	if cfg.AuditURL != "" {
		observers = append(observers, audit.NewHTTPObserver(cfg.AuditURL))
	}

	// Initialize audit publisher if any observers exist
	var auditPublisher *audit.Publisher
	if len(observers) > 0 {
		auditPublisher = audit.NewPublisherWithPool(log, observers, 3)
	}

	// Create HTTP handler with middleware
	h := handler.NewHandler(store, db, log, cfg.Key, auditPublisher)

	// Load crypto key if provided
	if cfg.CryptoKey != "" {
		privKey, err := cryptoutil.LoadPrivateKey(cfg.CryptoKey)
		if err != nil {
			log.Fatal("failed to load private key", zap.Error(err))
		}
		h.SetPrivateKey(privKey)
	}

	loggedHandler := middleware.LoggingMiddleware(log)(h.Router())

	// Start pprof server in a separate goroutine
	go func() {
		log.Info("starting pprof server", zap.String("address", "localhost:6060"))
		if err := http.ListenAndServe(":6065", nil); err != nil {
			log.Error("pprof server failed", zap.Error(err))
		}
	}()

	// Start main HTTP server
	log.Info("starting HTTP server", zap.String("address", cfg.ServerAddr))
	if err := http.ListenAndServe(cfg.ServerAddr, loggedHandler); err != nil {
		log.Error("server stopped with error", zap.Error(err))
		os.Exit(1)
	}
}
