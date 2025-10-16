package config

import (
	"errors"
	"flag"
	"time"
)

type Config struct {
	LogLevel       string
	LogFormat      string
	ServerAddr     string
	ReportInterval time.Duration
	PollInterval   time.Duration
}

func Load() (*Config, error) {
	logLevel := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	logFormat := flag.String("log-format", "text", "Log format: text or json")

	serverAddr := flag.String("a", "localhost:8080", "HTTP server address")
	reportInterval := flag.Int("r", 10, "Report interval in seconds")
	pollInterval := flag.Int("p", 2, "Poll interval in seconds")

	flag.Parse()

	if *reportInterval <= 0 {
		return nil, errors.New("report interval must be greater than zero")
	}
	if *pollInterval <= 0 {
		return nil, errors.New("poll interval must be greater than zero")
	}

	return &Config{
		LogLevel:       *logLevel,
		LogFormat:      *logFormat,
		ServerAddr:     "http://" + *serverAddr,
		ReportInterval: time.Duration(*reportInterval) * time.Second,
		PollInterval:   time.Duration(*pollInterval) * time.Second,
	}, nil
}
