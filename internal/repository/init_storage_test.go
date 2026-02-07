package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tigranqic/metrics-tpl/internal/config"
	"go.uber.org/zap"
)

func TestInitStorage_ChooseMemStorageByDefault(t *testing.T) {
	cfg := &config.Config{}
	logger, _ := zap.NewDevelopment()

	db, store, err := InitStorage(cfg, logger)
	assert.NoError(t, err)
	assert.Nil(t, db)
	assert.NotNil(t, store)
}

func TestInitStorage_ChooseFileStorage(t *testing.T) {
	tmpFile := t.TempDir() + "/metrics.json"
	cfg := &config.Config{
		FileStoragePath: tmpFile,
	}
	logger, _ := zap.NewDevelopment()

	db, store, err := InitStorage(cfg, logger)
	assert.NoError(t, err)
	assert.Nil(t, db)
	assert.NotNil(t, store)
}

func TestInitStorage_InvalidPostgresDSN(t *testing.T) {
	cfg := &config.Config{
		DatabaseDSN: "invalid_dsn",
	}
	logger, _ := zap.NewDevelopment()

	db, store, err := InitStorage(cfg, logger)
	assert.Error(t, err)
	assert.Nil(t, db)
	assert.Nil(t, store)
}
