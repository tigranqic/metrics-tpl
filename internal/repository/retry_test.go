package repository

import (
	"errors"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"

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
