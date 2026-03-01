package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/tigranqic/metrics-tpl/internal/repository/pgerrors"
)

func newPQError(code, msg string) *pq.Error {
	return &pq.Error{
		Code:    pq.ErrorCode(code),
		Message: msg,
	}
}

func TestPostgresErrorClassifier_Retriable(t *testing.T) {
	classifier := pgerrors.NewPostgresErrorClassifier()

	retriableCodes := []string{
		pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure,
		pgerrcode.TransactionRollback,
		pgerrcode.SerializationFailure,
		pgerrcode.DeadlockDetected,
		pgerrcode.CannotConnectNow,
	}

	for _, code := range retriableCodes {
		err := newPQError(code, "some retriable error")
		class := classifier.Classify(err)
		assert.Equal(t, pgerrors.Retriable, class, "code %s will be Retriable", code)
	}
}

func TestPostgresErrorClassifier_NonRetriable(t *testing.T) {
	classifier := pgerrors.NewPostgresErrorClassifier()

	nonRetriableCodes := []string{
		pgerrcode.UniqueViolation,
		pgerrcode.ForeignKeyViolation,
		pgerrcode.CheckViolation,
	}

	for _, code := range nonRetriableCodes {
		err := newPQError(code, "some non-retriable error")
		class := classifier.Classify(err)
		assert.Equal(t, pgerrors.NonRetriable, class, "code %s wiil be NonRetriable", code)
	}
}

func TestPostgresErrorClassifier_OtherErrors(t *testing.T) {
	classifier := pgerrors.NewPostgresErrorClassifier()

	err := errors.New("normal error")
	class := classifier.Classify(err)
	assert.Equal(t, pgerrors.NonRetriable, class)

	class = classifier.Classify(nil)
	assert.Equal(t, pgerrors.NonRetriable, class)
}

type mockExecutor struct {
	calls int
	err   error
}

func (m *mockExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	m.calls++
	return nil, m.err
}

func TestExecWithRetry(t *testing.T) {
	log := zap.NewNop()
	ctx := context.Background()

	t.Run("Success first attempt", func(t *testing.T) {
		m := &mockExecutor{}
		err := ExecWithRetry(ctx, m, log, "SELECT 1")
		assert.NoError(t, err)
		assert.Equal(t, 1, m.calls)
	})

	t.Run("Non-retriable error", func(t *testing.T) {
		m := &mockExecutor{err: errors.New("fatal")}
		err := ExecWithRetry(ctx, m, log, "SELECT 1")
		assert.Error(t, err)
		assert.Equal(t, 1, m.calls)
	})

	t.Run("PQ Non-retriable error", func(t *testing.T) {
		m := &mockExecutor{err: &pq.Error{Code: pgerrcode.UniqueViolation}}
		err := ExecWithRetry(ctx, m, log, "SELECT 1")
		assert.Error(t, err)
		assert.Equal(t, 1, m.calls)
	})

	t.Run("Retriable exhaustion", func(t *testing.T) {
		// Mock a retriable error
		m := &mockExecutor{err: &pq.Error{Code: pgerrcode.ConnectionFailure}}

		if testing.Short() {
			t.Skip("skipping test in short mode")
		}

		err := ExecWithRetry(ctx, m, log, "SELECT 1")
		assert.Error(t, err)
		assert.Equal(t, 4, m.calls) // 1 initial + 3 retries
	})
}
