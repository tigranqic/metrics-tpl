package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestSubnetMiddleware(t *testing.T) {
	logger := zap.NewNop()
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name          string
		trustedSubnet string
		xRealIP       string
		expectedCode  int
	}{
		{
			name:          "Empty subnet - allow all",
			trustedSubnet: "",
			xRealIP:       "192.168.1.1",
			expectedCode:  http.StatusOK,
		},
		{
			name:          "Valid IP in subnet",
			trustedSubnet: "192.168.1.0/24",
			xRealIP:       "192.168.1.15",
			expectedCode:  http.StatusOK,
		},
		{
			name:          "Invalid IP in subnet",
			trustedSubnet: "192.168.1.0/24",
			xRealIP:       "10.0.0.1",
			expectedCode:  http.StatusForbidden,
		},
		{
			name:          "Missing X-Real-IP header",
			trustedSubnet: "192.168.1.0/24",
			xRealIP:       "",
			expectedCode:  http.StatusForbidden,
		},
		{
			name:          "Malformed IP",
			trustedSubnet: "192.168.1.0/24",
			xRealIP:       "not-an-ip",
			expectedCode:  http.StatusForbidden,
		},
		{
			name:          "Invalid trusted subnet CIDR",
			trustedSubnet: "invalid-cidr",
			xRealIP:       "192.168.1.1",
			expectedCode:  http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := SubnetMiddleware(tt.trustedSubnet, logger)
			handler := middleware(nextHandler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.xRealIP != "" {
				req.Header.Set("X-Real-IP", tt.xRealIP)
			}
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedCode, rec.Code)
		})
	}
}
