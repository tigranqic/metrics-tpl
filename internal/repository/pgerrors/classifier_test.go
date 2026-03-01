package pgerrors

import (
	"errors"
	"testing"

	"github.com/jackc/pgerrcode"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestPostgresErrorClassifier_Classify(t *testing.T) {
	c := NewPostgresErrorClassifier()

	tests := []struct {
		name     string
		err      error
		expected PGErrorClassification
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: NonRetriable,
		},
		{
			name:     "Generic error",
			err:      errors.New("standard error"),
			expected: NonRetriable,
		},
		{
			name:     "Retriable connection error",
			err:      &pq.Error{Code: pgerrcode.ConnectionException},
			expected: Retriable,
		},
		{
			name:     "Retriable deadlock",
			err:      &pq.Error{Code: pgerrcode.DeadlockDetected},
			expected: Retriable,
		},
		{
			name:     "Non-retriable unique violation",
			err:      &pq.Error{Code: pgerrcode.UniqueViolation},
			expected: NonRetriable,
		},
		{
			name:     "Retriable cannot connect now",
			err:      &pq.Error{Code: pgerrcode.CannotConnectNow},
			expected: Retriable,
		},
		{
			name:     "Unknown PQ error",
			err:      &pq.Error{Code: "99999"},
			expected: NonRetriable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Classify(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
