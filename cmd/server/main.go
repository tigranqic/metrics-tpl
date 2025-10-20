package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/tigranqic/metrics-tpl/internal/handler"
	"github.com/tigranqic/metrics-tpl/internal/repository"
	"github.com/tigranqic/metrics-tpl/pkg/logger"
)

func main() {
	addr := flag.String("a", "localhost:8080", "HTTP server address")
	logLevel := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	logFormat := flag.String("log-format", "text", "Log format: text or json")

	flag.Parse()

	logger.Init(*logLevel, *logFormat)

	if len(flag.Args()) > 0 {
		slog.Error("unknown arguments", "args", flag.Args())
		os.Exit(1)
	}

	store := repository.NewMemStorage()
	h := handler.NewHandler(store)

	slog.Info("starting HTTP server", "address", *addr)

	if err := http.ListenAndServe(*addr, h.Router()); err != nil {
		slog.Error("server stopped with error", "err", err)
		os.Exit(1)
	}
}
