// Package config provides loading and parsing of application configuration
// from environment variables, command-line flags, and default values.
// It supports both agent and server modes and handles intervals, storage paths,
// logging settings, database DSNs, audit configuration, and rate limits.
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
	LogLevel        string
	LogFormat       string
	ServerAddr      string
	ReportInterval  time.Duration
	PollInterval    time.Duration
	IsAgent         bool
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
	Key             string
	RateLimit       int
	AuditFile       string
	AuditURL        string
	CryptoKey       string
}

const (
	DefaultLogLevel        = "info"
	DefaultLogFormat       = "json"
	DefaultServerAddr      = "localhost:8080"
	DefaultReportInterval  = 10
	DefaultPollInterval    = 2
	DefaultStoreInterval   = 15
	DefaultFileStoragePath = "metrics.json"
	DefaultRestore         = false
	DefaultDBDSN           = "postgres://postgres:postgres@localhost:15449/metrics-tpl?sslmode=disable"
	DefaultRateLimit       = 5
)

func Load(isAgent bool) (*Config, error) {
	chooseString := func(envVal string, envSet bool, flagVal string, def string) string {
		if envSet {
			return envVal
		} else if flagVal != "" && flagVal != def {
			return flagVal
		}
		return def
	}

	chooseInt := func(envVal int, envSet bool, flagVal int, def int) int {
		if envSet {
			return envVal
		} else if flagVal != def {
			return flagVal
		}
		return def
	}

	chooseBool := func(envVal bool, envSet bool, flagVal bool, def bool) bool {
		if envSet {
			return envVal
		} else if flagVal != def {
			return flagVal
		}
		return def
	}

	envAddr, envAddrSet := getenvString("ADDRESS", DefaultServerAddr)
	envReport, envReportSet := getenvInt("REPORT_INTERVAL", DefaultReportInterval)
	envPoll, envPollSet := getenvInt("POLL_INTERVAL", DefaultPollInterval)
	envStore, envStoreSet := getenvInt("STORE_INTERVAL", DefaultStoreInterval)
	envFile, envFileSet := getenvString("FILE_STORAGE_PATH", DefaultFileStoragePath)
	envRestore, envRestoreSet := getenvBool("RESTORE", DefaultRestore)
	envDBDSN, envDBDSNSet := getenvString("DATABASE_DSN", "")
	envKey, envKeySet := getenvString("KEY", "")
	envRateLimit, envRateLimitSet := getenvInt("RATE_LIMIT", DefaultRateLimit)
	envAuditFile, envAuditFileSet := getenvString("AUDIT_FILE", "")
	envAuditURL, envAuditURLSet := getenvString("AUDIT_URL", "")
	envCryptoKey, envCryptoKeySet := getenvString("CRYPTO_KEY", "")

	logLevel := flag.String("log-level", DefaultLogLevel, "Log level: debug, info, warn, error")
	logFormat := flag.String("log-format", DefaultLogFormat, "Log format: text or json")
	serverAddrFlag := flag.String("a", DefaultServerAddr, "HTTP server address")
	pollFlag := flag.Int("p", DefaultPollInterval, "Poll interval in seconds")
	storeFlag := flag.Int("i", DefaultStoreInterval, "Interval in seconds to store metrics (0 = sync)")
	fileFlag := flag.String("f", DefaultFileStoragePath, "File path for metrics storage")
	dbDSNFlag := flag.String("d", "", "Database DSN connection string")
	keyFlag := flag.String("k", "", "Key for hash")
	rateLimitFlag := flag.Int("l", DefaultRateLimit, "Rate limit")
	auditFileFlag := flag.String("audit-file", "", "file to write audit")
	auditURLFlag := flag.String("audit-url", "", "url to send audit")
	cryptoKeyFlag := flag.String("crypto-key", "", "Path to crypto key file (public key for agent, private key for server)")

	var reportFlag *int
	var restoreFlag *bool
	var reportVal int
	if isAgent {
		reportFlag = flag.Int("r", DefaultReportInterval, "Report interval in seconds (agent)")
		reportVal = *reportFlag
	} else {
		reportVal = DefaultReportInterval
	}

	var restoreVal bool
	if !isAgent {
		restoreFlag = flag.Bool("r", DefaultRestore, "Restore metrics from file on startup (server)")
		restoreVal = *restoreFlag
	} else {
		restoreVal = DefaultRestore
	}
	flag.Parse()

	serverAddr := chooseString(envAddr, envAddrSet, *serverAddrFlag, DefaultServerAddr)
	reportInterval := chooseInt(envReport, envReportSet, reportVal, DefaultReportInterval)
	pollInterval := chooseInt(envPoll, envPollSet, *pollFlag, DefaultPollInterval)
	storeInterval := chooseInt(envStore, envStoreSet, *storeFlag, DefaultStoreInterval)
	fileStorage := chooseString(envFile, envFileSet, *fileFlag, DefaultFileStoragePath)
	restore := chooseBool(envRestore, envRestoreSet, restoreVal, DefaultRestore)
	databaseDSN := chooseString(envDBDSN, envDBDSNSet, *dbDSNFlag, "")
	key := chooseString(envKey, envKeySet, *keyFlag, "")
	rateLimit := chooseInt(envRateLimit, envRateLimitSet, *rateLimitFlag, DefaultRateLimit)
	auditFile := chooseString(envAuditFile, envAuditFileSet, *auditFileFlag, "")
	auditURL := chooseString(envAuditURL, envAuditURLSet, *auditURLFlag, "")
	cryptoKey := chooseString(envCryptoKey, envCryptoKeySet, *cryptoKeyFlag, "")

	if reportInterval <= 0 {
		return nil, errors.New("report interval must be greater than zero")
	}
	if pollInterval <= 0 {
		return nil, errors.New("poll interval must be greater than zero")
	}

	if isAgent && !strings.HasPrefix(serverAddr, "http://") && !strings.HasPrefix(serverAddr, "https://") {
		serverAddr = "http://" + serverAddr
	}
	if !isAgent {
		serverAddr = strings.TrimPrefix(serverAddr, "http://")
		serverAddr = strings.TrimPrefix(serverAddr, "https://")
	}

	return &Config{
		LogLevel:        *logLevel,
		LogFormat:       *logFormat,
		ServerAddr:      serverAddr,
		ReportInterval:  time.Duration(reportInterval) * time.Second,
		PollInterval:    time.Duration(pollInterval) * time.Second,
		StoreInterval:   time.Duration(storeInterval) * time.Second,
		FileStoragePath: fileStorage,
		Restore:         restore,
		DatabaseDSN:     databaseDSN,
		Key:             key,
		RateLimit:       rateLimit,
		AuditFile:       auditFile,
		AuditURL:        auditURL,
		CryptoKey:       cryptoKey,
	}, nil
}

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

func getenvBool(key string, def bool) (bool, bool) {
	if val := os.Getenv(key); val != "" {
		v := strings.ToLower(val)
		if v == "true" || v == "1" {
			return true, true
		}
		if v == "false" || v == "0" {
			return false, true
		}
	}
	return def, false
}
