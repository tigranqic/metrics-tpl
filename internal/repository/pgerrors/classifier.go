package pgerrors

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
)

type PGErrorClassification int

const (
	NonRetriable PGErrorClassification = iota
	Retriable
)

type PostgresErrorClassifier struct{}

func NewPostgresErrorClassifier() *PostgresErrorClassifier {
	return &PostgresErrorClassifier{}
}

func (c *PostgresErrorClassifier) Classify(err error) PGErrorClassification {
	if err == nil {
		return NonRetriable
	}

	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return classifyPQError(pgErr)
	}

	return NonRetriable
}

func classifyPQError(pgErr *pq.Error) PGErrorClassification {
	switch pgErr.Code {
	// Class 08 — Connection Exception
	case pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure:
		return Retriable

	// Class 40 — Transaction Rollback
	case pgerrcode.TransactionRollback,
		pgerrcode.SerializationFailure,
		pgerrcode.DeadlockDetected:
		return Retriable

	// Class 57
	case pgerrcode.CannotConnectNow:
		return Retriable
	}

	switch pgErr.Code {
	// Class 23 — Integrity constraints
	case pgerrcode.UniqueViolation,
		pgerrcode.ForeignKeyViolation,
		pgerrcode.CheckViolation:
		return NonRetriable
	}

	return NonRetriable
}
