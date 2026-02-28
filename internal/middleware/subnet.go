// Package middleware provides HTTP middleware for request and response processing.
package middleware

import (
	"net"
	"net/http"

	"go.uber.org/zap"
)

// SubnetMiddleware creates a middleware that checks if the request's X-Real-IP
// belongs to the trusted subnet.
func SubnetMiddleware(trustedSubnet string, log *zap.Logger) func(http.Handler) http.Handler {
	var subnet *net.IPNet
	if trustedSubnet != "" {
		_, ipNet, err := net.ParseCIDR(trustedSubnet)
		if err != nil {
			log.Error("failed to parse trusted subnet", zap.String("subnet", trustedSubnet), zap.Error(err))
		} else {
			subnet = ipNet
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no subnet is configured, allow all requests
			if subnet == nil {
				next.ServeHTTP(w, r)
				return
			}

			ipStr := r.Header.Get("X-Real-IP")
			if ipStr == "" {
				log.Debug("access denied: X-Real-IP header is missing")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			ip := net.ParseIP(ipStr)
			if ip == nil || !subnet.Contains(ip) {
				log.Debug("access denied: IP not in trusted subnet", zap.String("ip", ipStr))
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
