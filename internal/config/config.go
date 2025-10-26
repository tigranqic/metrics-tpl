package config

import (
	"errors"
	"flag"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	LogLevel       string
	LogFormat      string
	ServerAddr     string
	ReportInterval time.Duration
	PollInterval   time.Duration
	IsAgent        bool
}

const (
	DefaultLogLevel       = "info"
	DefaultLogFormat      = "json"
	DefaultServerAddr     = "localhost:8080"
	DefaultReportInterval = 10
	DefaultPollInterval   = 2
)

func getenvInt(key string, def int) (int, bool) {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			return n, true
		}
	}
	return def, false
}

func getenvString(key string, def string) (string, bool) {
	if val := os.Getenv(key); val != "" {
		return val, true
	}
	return def, false
}

func Load(isAgent bool) (*Config, error) {
	envAddr, envAddrSet := getenvString("ADDRESS", DefaultServerAddr)
	envReport, envReportSet := getenvInt("REPORT_INTERVAL", DefaultReportInterval)
	envPoll, envPollSet := getenvInt("POLL_INTERVAL", DefaultPollInterval)

	logLevel := flag.String("log-level", DefaultLogLevel, "Log level: debug, info, warn, error")
	logFormat := flag.String("log-format", DefaultLogFormat, "Log format: text or json")
	var serverAddrDefault string
	if envAddrSet {
		serverAddrDefault = envAddr
	} else {
		serverAddrDefault = DefaultServerAddr
	}
	serverAddr := flag.String("a", serverAddrDefault, "HTTP server address")

	var reportDefault int
	if envReportSet {
		reportDefault = envReport
	} else {
		reportDefault = DefaultReportInterval
	}
	reportInterval := flag.Int("r", reportDefault, "Report interval in seconds")

	var pollDefault int
	if envPollSet {
		pollDefault = envPoll
	} else {
		pollDefault = DefaultPollInterval
	}
	pollInterval := flag.Int("p", pollDefault, "Poll interval in seconds")

	flag.Parse()

	if *reportInterval <= 0 {
		return nil, errors.New("report interval must be greater than zero")
	}
	if *pollInterval <= 0 {
		return nil, errors.New("poll interval must be greater than zero")
	}

	if isAgent && !strings.HasPrefix(*serverAddr, "http://") && !strings.HasPrefix(*serverAddr, "https://") {
		*serverAddr = "http://" + *serverAddr
	}

	if !isAgent {
		*serverAddr = strings.TrimPrefix(*serverAddr, "http://")
		*serverAddr = strings.TrimPrefix(*serverAddr, "https://")
	}

	return &Config{
		LogLevel:       *logLevel,
		LogFormat:      *logFormat,
		ServerAddr:     *serverAddr,
		ReportInterval: time.Duration(*reportInterval) * time.Second,
		PollInterval:   time.Duration(*pollInterval) * time.Second,
	}, nil
}
