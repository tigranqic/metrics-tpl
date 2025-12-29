package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

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
			return fmt.Errorf("failed to parse gauge metric %q: %w", name, err)
		}

		if err := ExecWithRetry(ctx, s.db, s.log, `INSERT INTO metrics (id, mtype, value)
			VALUES ($1, 'gauge', $2)
			ON CONFLICT (id)
			DO UPDATE SET value = EXCLUDED.value`, name, v); err != nil {
			return fmt.Errorf("failed to update gauge metric %q: %w", name, err)
		}
		return nil
	case models.Counter:
		delta, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("failed to parse counter metric %q: %w", name, err)
		}

		if err := ExecWithRetry(ctx, s.db, s.log, `INSERT INTO metrics (id, mtype, delta)
			VALUES ($1, 'counter', $2)
			ON CONFLICT (id)
			DO UPDATE SET delta = metrics.delta + EXCLUDED.delta`, name, delta); err != nil {
			return fmt.Errorf("failed to update counter metric %q: %w", name, err)
		}
		return nil

	default:
		return errors.New("unsupported metric type")
	}
}

func (s *PostgresStorage) GetGauge(name string) (float64, error) {
	var v float64
	err := s.db.QueryRow(`
		SELECT value FROM metrics WHERE id=$1 AND mtype='gauge'
	`, name).Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("failed to get gauge metric %q: %w", name, err)
	}
	return v, err
}

func (s *PostgresStorage) GetCounter(name string) (int64, error) {
	var v int64
	err := s.db.QueryRow(`
		SELECT delta FROM metrics WHERE id=$1 AND mtype='counter'
	`, name).Scan(&v)
	if err != nil {
		return 0, fmt.Errorf("failed to get counter metric %q: %w", name, err)
	}
	return v, err
}

func (s *PostgresStorage) GetAll() (map[string]*models.Metrics, error) {
	rows, err := s.db.Query(`SELECT id, mtype, delta, value FROM metrics`)
	if err != nil {
		return nil, fmt.Errorf("failed to query all metrics: %w", err)
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
		return nil, fmt.Errorf("failed to query all metrics: %w", err)
	}

	return result, nil
}

func (s *PostgresStorage) UpdateBatch(batch []models.Metrics) error {
	if len(batch) == 0 {
		return nil
	}

	ctx := context.Background()
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			s.log.Error("failed to rollback transaction:", zap.Error(err))
		}
	}()

	uniqueGauges := make(map[string]models.Metrics)
	counterSums := make(map[string]int64)

	for _, m := range batch {
		switch m.MType {
		case models.Gauge:
			if m.Value != nil {
				uniqueGauges[m.ID] = m
			}
		case models.Counter:
			if m.Delta != nil {
				counterSums[m.ID] += *m.Delta
			}
		}
	}

	if len(uniqueGauges) > 0 {
		var placeholders []string
		var values []interface{}
		i := 1
		for _, m := range uniqueGauges {
			placeholders = append(placeholders, fmt.Sprintf("($%d, 'gauge', $%d)", i, i+1))
			values = append(values, m.ID, *m.Value)
			i += 2
		}

		query := fmt.Sprintf(`
			INSERT INTO metrics (id, mtype, value)
			VALUES %s
			ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value
		`, strings.Join(placeholders, ","))

		if _, err := tx.ExecContext(ctx, query, values...); err != nil {
			return fmt.Errorf("failed to update gauge batch: %w", err)
		}
	}

	if len(counterSums) > 0 {
		var placeholders []string
		var values []interface{}
		i := 1
		for id, delta := range counterSums {
			placeholders = append(placeholders, fmt.Sprintf("($%d, 'counter', $%d)", i, i+1))
			values = append(values, id, delta)
			i += 2
		}

		query := fmt.Sprintf(`
			INSERT INTO metrics (id, mtype, delta)
			VALUES %s
			ON CONFLICT (id) DO UPDATE SET delta = metrics.delta + EXCLUDED.delta
		`, strings.Join(placeholders, ","))

		if _, err := tx.ExecContext(ctx, query, values...); err != nil {
			return fmt.Errorf("failed to update counter batch: %w", err)
		}
	}

	return tx.Commit()
}
