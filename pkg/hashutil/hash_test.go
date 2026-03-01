package hashutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalcSHA256(t *testing.T) {
	tests := []struct {
		name     string
		body     []byte
		key      string
		expected string
	}{
		{
			name:     "Empty body and key",
			body:     []byte(""),
			key:      "",
			expected: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:     "Simple text",
			body:     []byte("hello"),
			key:      "world",
			expected: "936a185caaa266bb9cbe981e9e05cb78cd732b0b3280eb944412bb6f8f8f07af",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalcSHA256(tt.body, tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}
