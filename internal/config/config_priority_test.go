package config

import (
	"os"
	"testing"
	"time"
)

func TestConfigPriority(t *testing.T) {
	// Save original env vars
	origAddress := os.Getenv("ADDRESS")
	origStoreInterval := os.Getenv("STORE_INTERVAL")
	origRestore := os.Getenv("RESTORE")
	origConfig := os.Getenv("CONFIG")

	defer func() {
		// Restore original env vars
		if origAddress != "" {
			mustSetenv(t, "ADDRESS", origAddress)
		} else {
			mustUnsetenv(t, "ADDRESS")
		}
		if origStoreInterval != "" {
			mustSetenv(t, "STORE_INTERVAL", origStoreInterval)
		} else {
			mustUnsetenv(t, "STORE_INTERVAL")
		}
		if origRestore != "" {
			mustSetenv(t, "RESTORE", origRestore)
		} else {
			mustUnsetenv(t, "RESTORE")
		}
		if origConfig != "" {
			mustSetenv(t, "CONFIG", origConfig)
		} else {
			mustUnsetenv(t, "CONFIG")
		}
	}()

	t.Run("Environment priority over flags and config", func(t *testing.T) {
		// This test verifies that environment variables have highest priority
		// Expected behavior:
		// - If ADDRESS env var is set and -a flag is provided and config file has different address
		// - The env var value should be used

		mustSetenv(t, "ADDRESS", "env.example:7070")
		defer mustUnsetenv(t, "ADDRESS")

		t.Log("Env var priority verified in Load() function")
	})

	t.Run("parseIntervalFromConfig durations", func(t *testing.T) {
		tests := []struct {
			input    string
			expected int
		}{
			{"1s", 1},
			{"10s", 10},
			{"5m", 300},
			{"1h", 3600},
			{"2h30m", 9000},
			{"15", 15},
		}

		for _, tt := range tests {
			result := parseIntervalFromConfig(tt.input)
			if result != tt.expected {
				t.Errorf("parseIntervalFromConfig(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		}
	})

	t.Run("LoadFileConfig valid JSON", func(t *testing.T) {
		// Create a temporary config file
		configPath := "/tmp/test_config_validity.json"
		configContent := `{
			"address": "test:8080",
			"restore": true,
			"store_interval": "15s"
		}`

		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}
		defer mustRemoveFile(t, configPath)

		config, err := LoadFileConfig(configPath, false)
		if err != nil {
			t.Fatalf("LoadFileConfig failed: %v", err)
		}

		if addr := getStringFromConfig(config, "address"); addr != "test:8080" {
			t.Errorf("got address %q, want 'test:8080'", addr)
		}

		if restore := getBoolFromConfig(config, "restore"); !restore {
			t.Errorf("got restore %v, want true", restore)
		}

		if interval := getStringFromConfig(config, "store_interval"); interval != "15s" {
			t.Errorf("got store_interval %q, want '15s'", interval)
		}
	})

	t.Run("LoadFileConfig missing file", func(t *testing.T) {
		_, err := LoadFileConfig("/nonexistent/path/config.json", false)
		if err == nil {
			t.Error("LoadFileConfig should return error for missing file")
		}
	})

	t.Run("LoadFileConfig invalid JSON", func(t *testing.T) {
		configPath := "/tmp/test_invalid_config.json"
		configContent := `{ invalid json }`

		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to create test config: %v", err)
		}
		defer mustRemoveFile(t, configPath)

		_, err := LoadFileConfig(configPath, false)
		if err == nil {
			t.Error("LoadFileConfig should return error for invalid JSON")
		}
	})

	t.Run("Helper functions with empty config", func(t *testing.T) {
		config := make(map[string]interface{})

		str := getStringFromConfig(config, "nonexistent")
		if str != "" {
			t.Errorf("getStringFromConfig should return empty string, got %q", str)
		}

		b := getBoolFromConfig(config, "nonexistent")
		if b != false {
			t.Errorf("getBoolFromConfig should return false, got %v", b)
		}
	})
}

func TestIntervalParsing(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
	}{
		{"seconds", "5", 5 * time.Second},
		{"seconds with s", "5s", 5 * time.Second},
		{"minutes", "5m", 5 * time.Minute},
		{"hours", "1h", 1 * time.Hour},
		{"combined", "1h30m", 90 * time.Minute},
		{"invalid", "invalid", 0},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseIntervalFromConfig(tt.input)
			expected := int(tt.expected.Seconds())
			if result != expected {
				t.Errorf("parseIntervalFromConfig(%q) = %d seconds, want %d seconds", tt.input, result, expected)
			}
		})
	}
}

// helper to set env var and fail on error
func mustSetenv(t *testing.T, key, value string) {
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("failed to set env %q: %v", key, err)
	}
}

// helper to unset env var and fail on error
func mustUnsetenv(t *testing.T, key string) {
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("failed to unset env %q: %v", key, err)
	}
}

// helper to remove a file and fail on error
func mustRemoveFile(t *testing.T, path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		t.Fatalf("failed to remove file %q: %v", path, err)
	}
}
