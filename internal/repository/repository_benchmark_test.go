package repository

import (
	"regexp"
	"strconv"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	models "github.com/tigranqic/metrics-tpl/internal/model"
	"go.uber.org/zap"
)

func setupBenchmarkStorage(b *testing.B) (*PostgresStorage, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		b.Fatalf("failed to open sqlmock: %v", err)
	}

	logger := zap.NewNop()
	store := NewPostgresStorage(db, logger)

	return store, mock
}

func BenchmarkUpdate_Gauge(b *testing.B) {
	store, mock := setupBenchmarkStorage(b)

	name := "temperature"
	value := "42.5"
	v, _ := strconv.ParseFloat(value, 64)

	for i := 0; i < b.N; i++ {
		mock.ExpectExec(regexp.QuoteMeta(`
			INSERT INTO metrics (id, mtype, value)
			VALUES ($1, 'gauge', $2)
			ON CONFLICT (id)
			DO UPDATE SET value = EXCLUDED.value
		`)).
			WithArgs(name, v).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	b.ResetTimer()
	for b.Loop() {
		_ = store.Update(models.Gauge, name, value)
	}
}

func BenchmarkUpdate_Counter(b *testing.B) {
	store, mock := setupBenchmarkStorage(b)

	name := "requests"
	value := "100"
	delta, _ := strconv.ParseInt(value, 10, 64)

	for i := 0; i < b.N; i++ {
		mock.ExpectExec(regexp.QuoteMeta(`
			INSERT INTO metrics (id, mtype, delta)
			VALUES ($1, 'counter', $2)
			ON CONFLICT (id)
			DO UPDATE SET delta = metrics.delta + EXCLUDED.delta
		`)).
			WithArgs(name, delta).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	b.ResetTimer()
	for b.Loop() {
		_ = store.Update(models.Counter, name, value)
	}
}

func BenchmarkGetGauge(b *testing.B) {
	store, mock := setupBenchmarkStorage(b)

	name := "temperature"
	value := 42.5

	for i := 0; i < b.N; i++ {
		rows := sqlmock.NewRows([]string{"value"}).
			AddRow(value)
		mock.ExpectQuery(`SELECT value FROM metrics WHERE id=\$1 AND mtype='gauge'`).
			WithArgs(name).
			WillReturnRows(rows)
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = store.GetGauge(name)
	}
}

func BenchmarkGetCounter(b *testing.B) {
	store, mock := setupBenchmarkStorage(b)

	name := "requests"
	value := int64(1000)

	for i := 0; i < b.N; i++ {
		rows := sqlmock.NewRows([]string{"delta"}).
			AddRow(value)
		mock.ExpectQuery(`SELECT delta FROM metrics WHERE id=\$1 AND mtype='counter'`).
			WithArgs(name).
			WillReturnRows(rows)
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = store.GetCounter(name)
	}
}

func BenchmarkUpdateBatch_Small(b *testing.B) {
	store, mock := setupBenchmarkStorage(b)

	metrics := make([]models.Metrics, 10)
	for i := 0; i < 10; i++ {
		v := float64(i)
		d := int64(i)
		if i%2 == 0 {
			metrics[i] = models.Metrics{
				ID:    "gauge_" + strconv.Itoa(i),
				MType: "gauge",
				Value: &v,
			}
		} else {
			metrics[i] = models.Metrics{
				ID:    "counter_" + strconv.Itoa(i),
				MType: "counter",
				Delta: &d,
			}
		}
	}

	for i := 0; i < b.N; i++ {
		mock.ExpectBegin()
		for j := 0; j < len(metrics); j++ {
			mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO metrics`)).
				WillReturnResult(sqlmock.NewResult(1, 1))
		}
		mock.ExpectCommit()
	}

	b.ResetTimer()
	for b.Loop() {
		_ = store.UpdateBatch(metrics)
	}
}

func BenchmarkUpdateBatch_Large(b *testing.B) {
	store, mock := setupBenchmarkStorage(b)

	metrics := make([]models.Metrics, 100)
	for i := 0; i < 100; i++ {
		v := float64(i)
		d := int64(i)
		if i%2 == 0 {
			metrics[i] = models.Metrics{
				ID:    "gauge_" + strconv.Itoa(i),
				MType: "gauge",
				Value: &v,
			}
		} else {
			metrics[i] = models.Metrics{
				ID:    "counter_" + strconv.Itoa(i),
				MType: "counter",
				Delta: &d,
			}
		}
	}

	for i := 0; i < b.N; i++ {
		mock.ExpectBegin()
		for j := 0; j < len(metrics); j++ {
			mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO metrics`)).
				WillReturnResult(sqlmock.NewResult(1, 1))
		}
		mock.ExpectCommit()
	}

	b.ResetTimer()
	for b.Loop() {
		_ = store.UpdateBatch(metrics)
	}
}

func BenchmarkGetAll_Small(b *testing.B) {
	store, mock := setupBenchmarkStorage(b)

	for i := 0; i < b.N; i++ {
		rows := sqlmock.NewRows([]string{"id", "mtype", "delta", "value"})
		for j := 0; j < 10; j++ {
			if j%2 == 0 {
				rows.AddRow("gauge_"+strconv.Itoa(j), "gauge", nil, float64(j))
			} else {
				rows.AddRow("counter_"+strconv.Itoa(j), "counter", int64(j), nil)
			}
		}
		mock.ExpectQuery(`SELECT id, mtype, delta, value FROM metrics`).
			WillReturnRows(rows)
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = store.GetAll()
	}
}

func BenchmarkGetAll_Large(b *testing.B) {
	store, mock := setupBenchmarkStorage(b)

	for i := 0; i < b.N; i++ {
		rows := sqlmock.NewRows([]string{"id", "mtype", "delta", "value"})
		for j := 0; j < 1000; j++ {
			if j%2 == 0 {
				rows.AddRow("gauge_"+strconv.Itoa(j), "gauge", nil, float64(j))
			} else {
				rows.AddRow("counter_"+strconv.Itoa(j), "counter", int64(j), nil)
			}
		}
		mock.ExpectQuery(`SELECT id, mtype, delta, value FROM metrics`).
			WillReturnRows(rows)
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = store.GetAll()
	}
}

func BenchmarkParsing_MixedMetrics(b *testing.B) {
	gaugeValues := []string{"42.5", "100.25", "0.001", "9999.99"}
	counterValues := []string{"100", "1000", "50", "10000"}

	b.ResetTimer()
	for b.Loop() {
		for _, v := range gaugeValues {
			_, _ = strconv.ParseFloat(v, 64)
		}
		for _, v := range counterValues {
			_, _ = strconv.ParseInt(v, 10, 64)
		}
	}
}
