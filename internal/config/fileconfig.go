package config

import (
	"encoding/json"
	"errors"
	"os"
)

// FileConfigServer represents the server configuration from JSON file
type FileConfigServer struct {
	Address       string `json:"address"`
	Restore       bool   `json:"restore"`
	StoreInterval string `json:"store_interval"`
	StoreFile     string `json:"store_file"`
	DatabaseDSN   string `json:"database_dsn"`
	CryptoKey     string `json:"crypto_key"`
}

// FileConfigAgent represents the agent configuration from JSON file
type FileConfigAgent struct {
	Address        string `json:"address"`
	ReportInterval string `json:"report_interval"`
	PollInterval   string `json:"poll_interval"`
	CryptoKey      string `json:"crypto_key"`
}

// LoadFileConfig loads configuration from a JSON file
// If isAgent is true, parses as agent config, otherwise as server config
func LoadFileConfig(path string, isAgent bool) (map[string]interface{}, error) {
	if path == "" {
		return nil, errors.New("config file path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := make(map[string]interface{})
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return config, nil
}

// getStringFromConfig extracts a string value from the config map
func getStringFromConfig(config map[string]interface{}, key string) string {
	if val, ok := config[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// getBoolFromConfig extracts a boolean value from the config map
func getBoolFromConfig(config map[string]interface{}, key string) bool {
	if val, ok := config[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}
