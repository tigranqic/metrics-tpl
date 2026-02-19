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
	chooseString := func(flagVal string, flagSet bool, envVal string, envSet bool, fileVal string, def string) string {
		if envSet && envVal != "" {
			return envVal
		}
		if flagSet && flagVal != "" && flagVal != def {
			return flagVal
		}
		if fileVal != "" {
			return fileVal
		}
		return def
	}

	chooseInt := func(flagVal int, flagSet bool, envVal int, envSet bool, fileVal int, def int) int {
		if envSet && envVal != def {
			return envVal
		}
		if flagSet && flagVal != def {
			return flagVal
		}
		if fileVal != 0 {
			return fileVal
		}
		return def
	}

	chooseBool := func(flagVal bool, flagSet bool, envVal bool, envSet bool, fileVal *bool, def bool) bool {
		if envSet && envVal != def {
			return envVal
		}
		if flagSet && flagVal != def {
			return flagVal
		}
		if fileVal != nil {
			return *fileVal
		}
		return def
	}

	configPath := ""
	if configEnv, ok := os.LookupEnv("CONFIG"); ok && configEnv != "" {
		configPath = configEnv
	}

	for i, arg := range os.Args[1:] {
		if (arg == "-c" || arg == "-config") && i+1 < len(os.Args)-1 {
			configPath = os.Args[i+2]
			break
		}
	}

	// Load config file if specified
	var fileConfig map[string]interface{}
	if configPath != "" {
		var err error
		fileConfig, err = LoadFileConfig(configPath, isAgent)
		if err != nil {
			return nil, err
		}
	}
	if fileConfig == nil {
		fileConfig = make(map[string]interface{})
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
	_ = flag.String("c", "", "Path to configuration file (JSON format)")
	_ = flag.String("config", "", "Path to configuration file (JSON format) - alternative to -c")

	var reportFlag *int
	var restoreFlag *bool
	var reportVal int
	var reportFlagSet bool

	if isAgent {
		reportFlag = flag.Int("r", DefaultReportInterval, "Report interval in seconds (agent)")
		reportVal = *reportFlag
	} else {
		reportVal = DefaultReportInterval
	}

	var restoreVal bool
	var restoreFlagSet bool
	if !isAgent {
		restoreFlag = flag.Bool("r", DefaultRestore, "Restore metrics from file on startup (server)")
		restoreVal = *restoreFlag
	} else {
		restoreVal = DefaultRestore
	}

	flag.Parse()

	// Determine which flags were actually set
	flagSet := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		flagSet[f.Name] = true
	})

	reportFlagSet = flagSet["r"]
	restoreFlagSet = flagSet["r"]

	fileAddr := getStringFromConfig(fileConfig, "address")
	fileReportStr := getStringFromConfig(fileConfig, "report_interval")
	filePollStr := getStringFromConfig(fileConfig, "poll_interval")
	fileStoreStr := getStringFromConfig(fileConfig, "store_interval")
	fileStoreFile := getStringFromConfig(fileConfig, "store_file")
	fileRestore := getBoolFromConfig(fileConfig, "restore")
	fileDBDSN := getStringFromConfig(fileConfig, "database_dsn")
	fileCryptoKey := getStringFromConfig(fileConfig, "crypto_key")

	fileReport := parseIntervalFromConfig(fileReportStr)
	filePoll := parseIntervalFromConfig(filePollStr)
	fileStore := parseIntervalFromConfig(fileStoreStr)

	serverAddr := chooseString(*serverAddrFlag, flagSet["a"], envAddr, envAddrSet, fileAddr, DefaultServerAddr)
	reportInterval := chooseInt(reportVal, reportFlagSet, envReport, envReportSet, fileReport, DefaultReportInterval)
	pollInterval := chooseInt(*pollFlag, flagSet["p"], envPoll, envPollSet, filePoll, DefaultPollInterval)
	storeInterval := chooseInt(*storeFlag, flagSet["i"], envStore, envStoreSet, fileStore, DefaultStoreInterval)

	fileStorage := chooseString(*fileFlag, flagSet["f"], envFile, envFileSet, fileStoreFile, DefaultFileStoragePath)
	if fileStorage == DefaultFileStoragePath && fileStoreFile != "" && !flagSet["f"] && !envFileSet {
		fileStorage = fileStoreFile
	}

	restorePtr := (*bool)(nil)
	if fileRestore {
		restorePtr = &fileRestore
	}
	restore := chooseBool(restoreVal, restoreFlagSet, envRestore, envRestoreSet, restorePtr, DefaultRestore)

	databaseDSN := chooseString(*dbDSNFlag, flagSet["d"], envDBDSN, envDBDSNSet, fileDBDSN, "")
	key := chooseString(*keyFlag, flagSet["k"], envKey, envKeySet, "", "")
	rateLimit := chooseInt(*rateLimitFlag, flagSet["l"], envRateLimit, envRateLimitSet, 0, DefaultRateLimit)
	auditFile := chooseString(*auditFileFlag, flagSet["audit-file"], envAuditFile, envAuditFileSet, "", "")
	auditURL := chooseString(*auditURLFlag, flagSet["audit-url"], envAuditURL, envAuditURLSet, "", "")
	cryptoKey := chooseString(*cryptoKeyFlag, flagSet["crypto-key"], envCryptoKey, envCryptoKeySet, fileCryptoKey, "")

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
