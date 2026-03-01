package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	_ "github.com/lib/pq"

	"github.com/tigranqic/metrics-tpl/internal/audit"
	"github.com/tigranqic/metrics-tpl/internal/config"
	"github.com/tigranqic/metrics-tpl/internal/handler"
	"github.com/tigranqic/metrics-tpl/internal/middleware"
	"github.com/tigranqic/metrics-tpl/internal/proto"
	"github.com/tigranqic/metrics-tpl/internal/repository"
	"github.com/tigranqic/metrics-tpl/pkg/cryptoutil"
	"github.com/tigranqic/metrics-tpl/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
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
	h := handler.NewHandler(store, db, log, cfg.Key, auditPublisher, cfg.TrustedSubnet)

	// Load crypto key if provided
	if cfg.CryptoKey != "" {
		privKey, err := cryptoutil.LoadPrivateKey(cfg.CryptoKey)
		if err != nil {
			log.Fatal("failed to load private key", zap.Error(err))
		}
		h.SetPrivateKey(privKey)
	}

	loggedHandler := middleware.LoggingMiddleware(log)(h.Router())

	// Create HTTP server with proper shutdown configuration
	server := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           loggedHandler,
		ReadTimeout:       time.Duration(cfg.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(cfg.WriteTimeout) * time.Second,
		IdleTimeout:       time.Duration(cfg.IdleTimeout) * time.Second,
		ReadHeaderTimeout: time.Duration(cfg.ReadHeaderTimeout) * time.Second,
	}

	// Create gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(handler.SubnetInterceptor(cfg.TrustedSubnet, log)),
	)
	proto.RegisterMetricsServer(grpcServer, handler.NewMetricsServer(store, log))

	// Start pprof server in a separate goroutine
	go func() {
		log.Info("starting pprof server", zap.String("address", "localhost:6065"))
		if err := http.ListenAndServe(":6065", nil); err != nil && err != http.ErrServerClosed {
			log.Error("pprof server failed", zap.Error(err))
		}
	}()

	// Set up signal handling for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	// Start HTTP server in a separate goroutine
	go func() {
		log.Info("starting HTTP server", zap.String("address", cfg.ServerAddr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", zap.Error(err))
		}
	}()

	// Start gRPC server in a separate goroutine
	go func() {
		listen, err := net.Listen("tcp", cfg.GRPCAddr)
		if err != nil {
			log.Fatal("failed to listen for gRPC", zap.Error(err))
		}
		log.Info("starting gRPC server", zap.String("address", cfg.GRPCAddr))
		if err := grpcServer.Serve(listen); err != nil {
			log.Error("gRPC server failed", zap.Error(err))
		}
	}()

	// Wait for termination signal
	<-ctx.Done()
	log.Info("received termination signal, starting graceful shutdown")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Gracefully shutdown the HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown error", zap.Error(err))
	}

	// Gracefully stop the gRPC server
	grpcServer.GracefulStop()

	// Ensure all unsaved data is persisted
	if err := store.Shutdown(); err != nil {
		log.Error("failed to shutdown storage", zap.Error(err))
	}

	log.Info("server stopped gracefully")
}
