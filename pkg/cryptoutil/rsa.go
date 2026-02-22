// Package cryptoutil provides RSA encryption and decryption utilities
// for asymmetric cryptographic operations.
package cryptoutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

// LoadPublicKey reads a PEM-encoded RSA public key from a file.
// The file should contain a PEM block with type "PUBLIC KEY".
func LoadPublicKey(keyPath string) (*rsa.PublicKey, error) {
	if keyPath == "" {
		return nil, fmt.Errorf("key path is empty")
	}

	absPath, err := filepath.Abs(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	keyData, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block from public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	pubKey, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not an RSA public key")
	}

	return pubKey, nil
}

// LoadPrivateKey reads a PEM-encoded RSA private key from a file.
// The file should contain a PEM block with type "RSA PRIVATE KEY" or "PRIVATE KEY".
func LoadPrivateKey(keyPath string) (*rsa.PrivateKey, error) {
	if keyPath == "" {
		return nil, fmt.Errorf("key path is empty")
	}

	absPath, err := filepath.Abs(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	keyData, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block from private key")
	}

	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try parsing as PKCS8
		key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}

		privKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("key is not an RSA private key")
		}

		return privKey, nil
	}

	return privKey, nil
}

// Encrypt encrypts data using RSA OAEP with SHA256 hash with the provided public key.
// Returns the encrypted data as bytes.
func Encrypt(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data is empty")
	}

	if publicKey == nil {
		return nil, fmt.Errorf("public key is nil")
	}

	encrypted, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, data, nil)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	return encrypted, nil
}

// Decrypt decrypts data using RSA OAEP with SHA256 hash with the provided private key.
// Returns the decrypted data as bytes.
func Decrypt(encryptedData []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	if len(encryptedData) == 0 {
		return nil, fmt.Errorf("encrypted data is empty")
	}

	if privateKey == nil {
		return nil, fmt.Errorf("private key is nil")
	}

	decrypted, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return decrypted, nil
}
