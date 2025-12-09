package repository

import (
	"context"
	"database/sql"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	models "github.com/tigranqic/metrics-tpl/internal/model"
	"go.uber.org/zap"
)

func newTestStorage(t *testing.T) (*PostgresStorage, sqlmock.Sqlmock, *zap.Logger) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open sqlmock: %v", err)
	}

	logger, _ := zap.NewDevelopment()
	store := NewPostgresStorage(db, logger)

	return store, mock, logger
}

func TestPostgresStorage_UpdateGauge(t *testing.T) {
	store, mock, _ := newTestStorage(t)

	name := "temperature"
	value := "42.5"

	v, _ := strconv.ParseFloat(value, 64)

	mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO metrics (id, mtype, value)
        VALUES ($1, 'gauge', $2)
        ON CONFLICT (id)
        DO UPDATE SET value = EXCLUDED.value
    `)).
		WithArgs(name, v).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.Update(models.Gauge, name, value)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStorage_UpdateCounter(t *testing.T) {
	store, mock, _ := newTestStorage(t)

	name := "requests"
	value := "10"

	delta, _ := strconv.ParseInt(value, 10, 64)

	mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO metrics (id, mtype, delta)
        VALUES ($1, 'counter', $2)
        ON CONFLICT (id)
        DO UPDATE SET delta = metrics.delta + EXCLUDED.delta
    `)).
		WithArgs(name, delta).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := store.Update(models.Counter, name, value)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresStorage_GetGauge(t *testing.T) {
	store, mock, _ := newTestStorage(t)

	rows := sqlmock.NewRows([]string{"value"}).AddRow(55.5)
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT value FROM metrics WHERE id=$1 AND mtype='gauge'
    `)).
		WithArgs("temperature").
		WillReturnRows(rows)

	val, err := store.GetGauge("temperature")
	assert.NoError(t, err)
	assert.Equal(t, 55.5, val)
}

func TestPostgresStorage_GetCounter(t *testing.T) {
	store, mock, _ := newTestStorage(t)

	rows := sqlmock.NewRows([]string{"delta"}).AddRow(123)
	mock.ExpectQuery(regexp.QuoteMeta(`
        SELECT delta FROM metrics WHERE id=$1 AND mtype='counter'
    `)).
		WithArgs("requests").
		WillReturnRows(rows)

	val, err := store.GetCounter("requests")
	assert.NoError(t, err)
	assert.Equal(t, int64(123), val)
}

func TestPostgresStorage_UpdateBatch(t *testing.T) {
	store, mock, _ := newTestStorage(t)

	batch := []models.Metrics{
		{ID: "g1", MType: models.Gauge, Value: ptrFloat64(10.1)},
		{ID: "c1", MType: models.Counter, Delta: ptrInt64(5)},
	}

	mock.ExpectBegin()

	mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO metrics (id, mtype, value)
        VALUES ($1, 'gauge', $2)
        ON CONFLICT (id)
        DO UPDATE SET value = EXCLUDED.value
    `)).WithArgs("g1", 10.1).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec(regexp.QuoteMeta(`
        INSERT INTO metrics (id, mtype, delta)
        VALUES ($1, 'counter', $2)
        ON CONFLICT (id)
        DO UPDATE SET delta = metrics.delta + EXCLUDED.delta
    `)).WithArgs("c1", int64(5)).WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectCommit()

	err := store.UpdateBatch(batch)
	assert.NoError(t, err)
}

func TestPostgresStorage_GetAll(t *testing.T) {
	store, mock, _ := newTestStorage(t)

	rows := sqlmock.NewRows([]string{"id", "mtype", "delta", "value"}).
		AddRow("g1", models.Gauge, nil, 10.5).
		AddRow("c1", models.Counter, 7, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, mtype, delta, value FROM metrics`)).
		WillReturnRows(rows)

	result, err := store.GetAll()
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 10.5, *result["g1"].Value)
	assert.Equal(t, int64(7), *result["c1"].Delta)
}

type mockDB struct {
	calls    int
	failures int
	records  map[string]any
}

func (m *mockDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	m.calls++
	if m.calls <= m.failures {
		return nil, &pq.Error{
			Code: "08006",
		}
	}

	if len(args) >= 2 {
		key := args[0].(string)
		m.records[key] = args[1]
	}

	return sqlmock.NewResult(1, 1), nil
}

func (m *mockDB) QueryRow(query string, args ...any) *sql.Row {
	key := args[0].(string)
	val := m.records[key]

	row := &sql.Row{}
	_ = val
	return row
}

func ptrFloat64(v float64) *float64 { return &v }
func ptrInt64(v int64) *int64       { return &v }

func TestExecWithRetry(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mock := &mockDB{
		failures: 2,
		records:  make(map[string]any),
	}

	start := time.Now()
	err := ExecWithRetry(context.Background(), mock, logger, "INSERT INTO metrics ...", "g1", 42.5)
	elapsed := time.Since(start)

	assert.NoError(t, err)
	assert.Equal(t, 3, mock.calls, "Exec called 3 times")
	assert.GreaterOrEqual(t, elapsed.Seconds(), 4.0, "time between retry should be keep") // 1+3 sec
}

func TestPostgresStorage_UpdateBatchWithRetry(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mock := &mockDB{
		records:  make(map[string]any),
		failures: 2, // simulate retries
	}

	batch := []models.Metrics{
		{ID: "g1", MType: models.Gauge, Value: ptrFloat64(10.1)},
		{ID: "c1", MType: models.Counter, Delta: ptrInt64(5)},
	}

	for i, m := range batch {
		var val any
		switch m.MType {
		case models.Gauge:
			val = *m.Value
		case models.Counter:
			val = *m.Delta
		}

		start := time.Now()
		err := ExecWithRetry(context.Background(), mock, logger,
			"INSERT OR UPDATE ...",
			m.ID, val,
		)
		duration := time.Since(start)

		assert.NoError(t, err)

		// Only check retries for first metric which has failures
		if i == 0 {
			assert.Equal(t, 3, mock.calls, "Exec should be called 3 times for first metric")
			assert.GreaterOrEqual(t, duration.Seconds(), 4.0, "time between retries should be respected")
		}
	}

	assert.Equal(t, 10.1, mock.records["g1"])
	assert.Equal(t, int64(5), mock.records["c1"])
}
