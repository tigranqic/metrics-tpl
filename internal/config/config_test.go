package config

import (
	"flag"
	"io"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)

	os.Args = []string{os.Args[0]}
}

func clearEnv(keys ...string) {
	for _, k := range keys {
		_ = os.Unsetenv(k)
	}
}

func TestLoadDefaultsAgent(t *testing.T) {
	resetFlags()
	clearEnv(
		"ADDRESS", "REPORT_INTERVAL", "POLL_INTERVAL",
		"RATE_LIMIT", "KEY", "AUDIT_FILE", "AUDIT_URL",
	)

	cfg, err := Load(true)
	require.NoError(t, err)

	assert.Equal(t, "http://"+DefaultServerAddr, cfg.ServerAddr)
	assert.Equal(t, time.Duration(DefaultReportInterval)*time.Second, cfg.ReportInterval)
	assert.Equal(t, time.Duration(DefaultPollInterval)*time.Second, cfg.PollInterval)
	assert.Equal(t, DefaultRateLimit, cfg.RateLimit)
}

func TestEnvOverridesFlags(t *testing.T) {
	resetFlags()
	clearEnv()

	require.NoError(t, os.Setenv("ADDRESS", "env-host:9999"))
	require.NoError(t, os.Setenv("POLL_INTERVAL", "5"))

	os.Args = []string{
		"cmd",
		"-a=flag-host:1111",
		"-p=2",
	}

	cfg, err := Load(true)
	require.NoError(t, err)

	assert.Equal(t, "http://env-host:9999", cfg.ServerAddr)
	assert.Equal(t, 5*time.Second, cfg.PollInterval)
}

func TestAgentAddsHTTPPrefix(t *testing.T) {
	resetFlags()
	clearEnv()

	require.NoError(t, os.Setenv("ADDRESS", "localhost:1234"))

	cfg, err := Load(true)
	require.NoError(t, err)

	assert.Equal(t, "http://localhost:1234", cfg.ServerAddr)
}

func TestServerStripsHTTPPrefix(t *testing.T) {
	resetFlags()
	clearEnv()

	require.NoError(t, os.Setenv("ADDRESS", "localhost:8080"))

	cfg, err := Load(false)
	require.NoError(t, err)

	assert.Equal(t, "localhost:8080", cfg.ServerAddr)
}

func TestRateLimitFromEnv(t *testing.T) {
	resetFlags()
	clearEnv()

	require.NoError(t, os.Setenv("RATE_LIMIT", "10"))

	cfg, err := Load(true)
	require.NoError(t, err)

	assert.Equal(t, 10, cfg.RateLimit)
}

func TestAuditConfigFromEnv(t *testing.T) {
	resetFlags()
	clearEnv()

	require.NoError(t, os.Setenv("AUDIT_FILE", "/tmp/audit.log"))
	require.NoError(t, os.Setenv("AUDIT_URL", "http://audit"))

	cfg, err := Load(true)
	require.NoError(t, err)

	assert.Equal(t, "/tmp/audit.log", cfg.AuditFile)
	assert.Equal(t, "http://audit", cfg.AuditURL)
}

func TestKeyFromEnv(t *testing.T) {
	resetFlags()
	clearEnv()

	require.NoError(t, os.Setenv("KEY", "super-secret"))

	cfg, err := Load(true)
	require.NoError(t, err)

	assert.Equal(t, "super-secret", cfg.Key)
}
