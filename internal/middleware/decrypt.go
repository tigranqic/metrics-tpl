// Package middleware provides HTTP middleware for request and response processing.
// It includes gzip compression, decompression, and RSA decryption handlers.
package middleware

import (
	"bytes"
	"crypto/rsa"
	"io"
	"net/http"

	"github.com/tigranqic/metrics-tpl/pkg/cryptoutil"
	"go.uber.org/zap"
)

// NewDecryptionMiddleware creates a middleware that decrypts request bodies
// if the X-Encrypted header is present.
func NewDecryptionMiddleware(privKey *rsa.PrivateKey, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If no private key is configured, skip decryption
			if privKey == nil {
				next.ServeHTTP(w, r)
				return
			}

			// Check if the request is encrypted
			if r.Header.Get("X-Encrypted") != "true" {
				next.ServeHTTP(w, r)
				return
			}

			// Read the encrypted body
			encryptedBody, err := io.ReadAll(r.Body)
			if err != nil {
				log.Error("failed to read encrypted body", zap.Error(err))
				http.Error(w, "failed to read body", http.StatusBadRequest)
				return
			}
			defer func() {
				_ = r.Body.Close()
			}()

			// Decrypt the body
			decryptedBody, err := cryptoutil.Decrypt(encryptedBody, privKey)
			if err != nil {
				log.Error("failed to decrypt body", zap.Error(err))
				http.Error(w, "decryption failed", http.StatusBadRequest)
				return
			}

			// Replace the request body with the decrypted data
			r.Body = io.NopCloser(bytes.NewReader(decryptedBody))

			next.ServeHTTP(w, r)
		})
	}
}
