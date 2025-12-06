package repository

import (
	"context"
	"time"

	"github.com/lib/pq"
	"github.com/tigranqic/metrics-tpl/internal/repository/pgerrors"
	"go.uber.org/zap"
)

func ExecWithRetry(ctx context.Context, db DBExecutor, log *zap.Logger, query string, args ...any) error {
	classifier := pgerrors.NewPostgresErrorClassifier()

	intervals := []time.Duration{
		1 * time.Second,
		3 * time.Second,
		5 * time.Second,
	}

	var lastErr error

	for attempt := 0; attempt <= len(intervals); attempt++ {
		_, err := db.ExecContext(ctx, query, args...)
		if err == nil {

			if attempt > 0 {
				log.Info("successful PostgreSQL retry",
					zap.Int("attempt", attempt+1),
				)
			}
			return nil
		}

		lastErr = err

		pgCode := ""
		if pgErr, ok := err.(*pq.Error); ok {
			pgCode = string(pgErr.Code)
		}

		if classifier.Classify(err) == pgerrors.NonRetriable {
			log.Error("non-retriable PostgreSQL error",
				zap.Error(err),
				zap.String("pg_code", pgCode),
			)
			return err
		}

		if attempt < len(intervals) {
			wait := intervals[attempt]
			log.Warn("PostgreSQL operation failed, retrying",
				zap.Int("attempt", attempt+1),
				zap.Error(err),
				zap.String("pg_code", pgCode),
				zap.Duration("sleep", wait),
			)
			time.Sleep(wait)
		}
	}

	log.Error("PostgreSQL retry attempts exhausted",
		zap.Error(lastErr),
	)
	return lastErr
}
