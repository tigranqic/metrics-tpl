// Package config provides loading and parsing of application configuration
// from environment variables, command-line flags, and default values.
// It supports both agent and server modes and handles intervals, storage paths,
// logging settings, database DSNs, audit configuration, and rate limits.
package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

const (
	DefaultServerAddr     = "localhost:8080"
	DefaultReportInterval = 10
	DefaultPollInterval   = 2
	DefaultRateLimit      = 5
)

type Config struct {
	LogLevel          string        `yaml:"log_level" env:"LOG_LEVEL" env-default:"info"`
	LogFormat         string        `yaml:"log_format" env:"LOG_FORMAT" env-default:"json"`
	ServerAddr        string        `yaml:"address" env:"ADDRESS" env-default:"localhost:8080"`
	ReportInterval    time.Duration `yaml:"report_interval" env:"REPORT_INTERVAL" env-default:"10s"`
	PollInterval      time.Duration `yaml:"poll_interval" env:"POLL_INTERVAL" env-default:"2s"`
	StoreInterval     time.Duration `yaml:"store_interval" env:"STORE_INTERVAL" env-default:"15s"`
	FileStoragePath   string        `yaml:"store_file" env:"FILE_STORAGE_PATH" env-default:"metrics.json"`
	Restore           bool          `yaml:"restore" env:"RESTORE" env-default:"false"`
	DatabaseDSN       string        `yaml:"database_dsn" env:"DATABASE_DSN" env-default:"postgres://postgres:postgres@localhost:15449/metrics-tpl?sslmode=disable"`
	Key               string        `yaml:"key" env:"KEY"`
	RateLimit         int           `yaml:"rate_limit" env:"RATE_LIMIT" env-default:"5"`
	AuditFile         string        `yaml:"audit_file" env:"AUDIT_FILE"`
	AuditURL          string        `yaml:"audit_url" env:"AUDIT_URL"`
	CryptoKey         string        `yaml:"crypto_key" env:"CRYPTO_KEY"`
	ReadTimeout       time.Duration `yaml:"read_timeout" env:"READ_TIMEOUT" env-default:"10s"`
	WriteTimeout      time.Duration `yaml:"write_timeout" env:"WRITE_TIMEOUT" env-default:"10s"`
	IdleTimeout       time.Duration `yaml:"idle_timeout" env:"IDLE_TIMEOUT" env-default:"60s"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout" env:"READ_HEADER_TIMEOUT" env-default:"5s"`
	TrustedSubnet     string        `yaml:"trusted_subnet" env:"TRUSTED_SUBNET"`
}

func Load(isAgent bool) (*Config, error) {
	fixEnvDurations()

	var cfg Config

	configPath := os.Getenv("CONFIG")
	if configPath == "" {
		for i, arg := range os.Args {
			if (arg == "-c" || arg == "-config") && i+1 < len(os.Args) {
				configPath = os.Args[i+1]
				break
			}
		}
	}

	if configPath != "" {
		if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
			return nil, fmt.Errorf("config file error: %w", err)
		}
	} else {
		if err := cleanenv.ReadEnv(&cfg); err != nil {
			return nil, err
		}
	}

	fs := flag.NewFlagSet("metrics", flag.ContinueOnError)
	fAddr := fs.String("a", cfg.ServerAddr, "")
	fPoll := fs.Int("p", int(cfg.PollInterval.Seconds()), "")
	fStore := fs.Int("i", int(cfg.StoreInterval.Seconds()), "")
	fFile := fs.String("f", cfg.FileStoragePath, "")
	fDSN := fs.String("d", cfg.DatabaseDSN, "")
	fKey := fs.String("k", cfg.Key, "")
	fLimit := fs.Int("l", cfg.RateLimit, "")
	fCrypto := fs.String("crypto-key", cfg.CryptoKey, "")
	fTrustedSubnet := fs.String("t", cfg.TrustedSubnet, "")

	var fReport *int
	var fRestore *bool
	if isAgent {
		fReport = fs.Int("r", int(cfg.ReportInterval.Seconds()), "")
	} else {
		fRestore = fs.Bool("r", cfg.Restore, "")
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		return nil, err
	}

	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "a":
			cfg.ServerAddr = *fAddr
		case "p":
			cfg.PollInterval = time.Duration(*fPoll) * time.Second
		case "i":
			cfg.StoreInterval = time.Duration(*fStore) * time.Second
		case "f":
			cfg.FileStoragePath = *fFile
		case "d":
			cfg.DatabaseDSN = *fDSN
		case "k":
			cfg.Key = *fKey
		case "l":
			cfg.RateLimit = *fLimit
		case "crypto-key":
			cfg.CryptoKey = *fCrypto
		case "t":
			cfg.TrustedSubnet = *fTrustedSubnet
		case "r":
			if isAgent {
				cfg.ReportInterval = time.Duration(*fReport) * time.Second
			} else {
				cfg.Restore = *fRestore
			}
		}
	})

	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil, err
	}

	if cfg.ReportInterval <= 0 || cfg.PollInterval <= 0 {
		return nil, fmt.Errorf("intervals must be positive")
	}

	if isAgent {
		if !strings.HasPrefix(cfg.ServerAddr, "http") {
			cfg.ServerAddr = "http://" + cfg.ServerAddr
		}
	} else {
		cfg.ServerAddr = strings.TrimPrefix(strings.TrimPrefix(cfg.ServerAddr, "https://"), "http://")
	}

	return &cfg, nil
}

func fixEnvDurations() {
	envVars := []string{
		"REPORT_INTERVAL", "POLL_INTERVAL", "STORE_INTERVAL",
		"READ_TIMEOUT", "WRITE_TIMEOUT", "IDLE_TIMEOUT", "READ_HEADER_TIMEOUT",
	}
	isNumeric := regexp.MustCompile(`^\d+$`)
	for _, v := range envVars {
		val := os.Getenv(v)
		if val != "" && isNumeric.MatchString(val) {
			err := os.Setenv(v, val+"s")
			if err != nil {
				log.Printf("failed to set env %s: %v", v, err)
			}
		}
	}
}

func parseIntervalFromConfig(val string) int {
	if val == "" {
		return 0
	}

	// Try to parse as duration (e.g., "1s", "10s", "1m")
	duration, err := time.ParseDuration(val)
	if err == nil {
		return int(duration.Seconds())
	}

	// Try to parse as plain number (seconds)
	if n, err := strconv.Atoi(val); err == nil && n > 0 {
		return n
	}

	return 0
}
