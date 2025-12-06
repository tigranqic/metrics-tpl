package repository

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	models "github.com/tigranqic/metrics-tpl/internal/model"
	"go.uber.org/zap"
)

type PostgresStorage struct {
	dbExec DBExecutor
	db     *sql.DB
	log    *zap.Logger
}

func NewPostgresStorage(db *sql.DB, log *zap.Logger) *PostgresStorage {
	return &PostgresStorage{dbExec: db, db: db, log: log}
}

func (s *PostgresStorage) Update(metricType, name, value string) error {
	ctx := context.Background()

	switch metricType {

	case models.Gauge:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}

		return ExecWithRetry(ctx, s.db, s.log, `
			INSERT INTO metrics (id, mtype, value)
			VALUES ($1, 'gauge', $2)
			ON CONFLICT (id)
			DO UPDATE SET value = EXCLUDED.value
		`, name, v)

	case models.Counter:
		delta, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}

		return ExecWithRetry(ctx, s.db, s.log, `
			INSERT INTO metrics (id, mtype, delta)
			VALUES ($1, 'counter', $2)
			ON CONFLICT (id)
			DO UPDATE SET delta = metrics.delta + EXCLUDED.delta
		`, name, delta)

	default:
		return errors.New("unsupported metric type")
	}
}

func (s *PostgresStorage) GetGauge(name string) (float64, error) {
	var v float64
	err := s.db.QueryRow(`
		SELECT value FROM metrics WHERE id=$1 AND mtype='gauge'
	`, name).Scan(&v)
	return v, err
}

func (s *PostgresStorage) GetCounter(name string) (int64, error) {
	var v int64
	err := s.db.QueryRow(`
		SELECT delta FROM metrics WHERE id=$1 AND mtype='counter'
	`, name).Scan(&v)
	return v, err
}

func (s *PostgresStorage) GetAll() map[string]*models.Metrics {
	rows, err := s.db.Query(`SELECT id, mtype, delta, value FROM metrics`)
	if err != nil {
		return nil
	}
	defer func() {
		_ = rows.Close()
	}()

	result := make(map[string]*models.Metrics)

	for rows.Next() {
		var id, mtype string
		var delta sql.NullInt64
		var value sql.NullFloat64

		if err := rows.Scan(&id, &mtype, &delta, &value); err != nil {
			continue
		}

		m := &models.Metrics{
			ID:    id,
			MType: mtype,
		}

		if delta.Valid {
			d := delta.Int64
			m.Delta = &d
		}
		if value.Valid {
			v := value.Float64
			m.Value = &v
		}

		result[id] = m
	}

	if err := rows.Err(); err != nil {
		return nil
	}

	return result
}

func (s *PostgresStorage) UpdateBatch(batch []models.Metrics) error {
	ctx := context.Background()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	for _, m := range batch {
		switch m.MType {

		case models.Gauge:
			if m.Value == nil {
				continue
			}
			err = ExecWithRetry(ctx, s.db, s.log, `
				INSERT INTO metrics (id, mtype, value)
				VALUES ($1, 'gauge', $2)
				ON CONFLICT (id)
				DO UPDATE SET value = EXCLUDED.value
			`, m.ID, *m.Value)

			if err != nil {
				return err
			}

		case models.Counter:
			if m.Delta == nil {
				continue
			}
			err = ExecWithRetry(ctx, s.db, s.log, `
				INSERT INTO metrics (id, mtype, delta)
				VALUES ($1, 'counter', $2)
				ON CONFLICT (id)
				DO UPDATE SET delta = metrics.delta + EXCLUDED.delta
			`, m.ID, *m.Delta)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
