package repository

import (
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/pressly/goose"
	"github.com/tigranqic/metrics-tpl/internal/config"
	"go.uber.org/zap"
)

func InitStorage(cfg *config.Config, log *zap.Logger) (*sql.DB, Storage, error) {
	if cfg.DatabaseDSN != "" {
		db, err := connectPostgres(cfg)
		if err != nil {
			return nil, nil, err
		}
		store := NewPostgresStorage(db, log)
		log.Info("using PostgreSQL storage")
		return db, store, nil
	}

	if cfg.FileStoragePath != "" {
		mem := NewMemStorage(cfg.FileStoragePath, cfg.StoreInterval)
		log.Info("using file-backed memory storage", zap.String("file", cfg.FileStoragePath))

		if cfg.Restore {
			if err := mem.LoadFromFile(cfg.FileStoragePath); err != nil {
				log.Error("failed to restore metrics", zap.Error(err))
			} else {
				log.Info("metrics restored successfully", zap.String("file", cfg.FileStoragePath))
			}
		}

		if cfg.StoreInterval > 0 {
			stopCh := make(chan struct{})
			mem.StartAutoSave(cfg.FileStoragePath, cfg.StoreInterval, stopCh)
		}

		return nil, mem, nil
	}

	mem := NewMemStorage("", 0)
	log.Info("using In-memory storage")

	return nil, mem, nil
}

func connectPostgres(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DatabaseDSN)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return nil, err
	}

	return db, nil
}
