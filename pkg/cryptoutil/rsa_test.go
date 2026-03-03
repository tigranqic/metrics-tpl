package cryptoutil

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEncryptDecrypt tests the RSA encryption and decryption functionalities
func TestEncryptDecrypt(t *testing.T) {
	// Generate a test RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate RSA private key: %v", err)
	}

	publicKey := &privateKey.PublicKey

	// Test data
	originalData := []byte("test metric data for encryption")

	// Encrypt
	encrypted, err := Encrypt(originalData, publicKey)
	if err != nil {
		t.Fatalf("failed to encrypt data: %v", err)
	}

	// Verify that encrypted data is different from original
	if string(encrypted) == string(originalData) {
		t.Error("encrypted data should be different from original data")
	}

	// Decrypt
	decrypted, err := Decrypt(encrypted, privateKey)
	if err != nil {
		t.Fatalf("failed to decrypt data: %v", err)
	}

	// Verify that decrypted data matches original
	if string(decrypted) != string(originalData) {
		t.Errorf("decrypted data does not match original.\nExpected: %s\nGot: %s", originalData, decrypted)
	}
}

// TestEncryptEmpty tests encryption with empty data
func TestEncryptEmpty(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	_, err := Encrypt([]byte{}, publicKey)
	if err == nil {
		t.Error("encryption of empty data should fail")
	}
}

// TestDecryptEmpty tests decryption with empty data
func TestDecryptEmpty(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	_, err := Decrypt([]byte{}, privateKey)
	if err == nil {
		t.Error("decryption of empty data should fail")
	}
}

// TestEncryptNilKey tests encryption with nil public key
func TestEncryptNilKey(t *testing.T) {
	_, err := Encrypt([]byte("test"), nil)
	if err == nil {
		t.Error("encryption with nil key should fail")
	}
}

// TestDecryptNilKey tests decryption with nil private key
func TestDecryptNilKey(t *testing.T) {
	_, err := Decrypt([]byte("test"), nil)
	if err == nil {
		t.Error("decryption with nil key should fail")
	}
}

// TestEncryptLargeData tests encryption with large data
func TestEncryptLargeData(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	// Create data close to the maximum size for RSA-2048 OAEP
	largeData := make([]byte, 100)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	encrypted, err := Encrypt(largeData, publicKey)
	if err != nil {
		t.Fatalf("failed to encrypt large data: %v", err)
	}

	decrypted, err := Decrypt(encrypted, privateKey)
	if err != nil {
		t.Fatalf("failed to decrypt large data: %v", err)
	}

	if string(decrypted) != string(largeData) {
		t.Error("decrypted large data does not match original")
	}
}

func TestDecryptWithWrongPrivateKey(t *testing.T) {
	privateKey1, _ := rsa.GenerateKey(rand.Reader, 2048)
	privateKey2, _ := rsa.GenerateKey(rand.Reader, 2048)

	data := []byte("secret data")

	encrypted, err := Encrypt(data, &privateKey1.PublicKey)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	_, err = Decrypt(encrypted, privateKey2)
	if err == nil {
		t.Error("decryption with wrong private key should fail")
	}
}

func TestEncryptTooLargeData(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey

	// 191 байт — больше допустимого лимита
	tooLarge := make([]byte, 191)

	_, err := Encrypt(tooLarge, publicKey)
	if err == nil {
		t.Error("encryption of too large data should fail")
	}
}

func TestLoadKeysPKCS1(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	dir := t.TempDir()

	// --- PRIVATE KEY (PKCS1)
	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	})

	privPath := filepath.Join(dir, "private.pem")
	if err := os.WriteFile(privPath, privPem, 0600); err != nil {
		t.Fatalf("failed to write private key: %v", err)
	}

	// --- PUBLIC KEY
	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}

	pubPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	pubPath := filepath.Join(dir, "public.pem")
	if err := os.WriteFile(pubPath, pubPem, 0644); err != nil {
		t.Fatalf("failed to write public key: %v", err)
	}

	// --- LOAD
	loadedPriv, err := LoadPrivateKey(privPath)
	if err != nil {
		t.Fatalf("failed to load private key: %v", err)
	}

	loadedPub, err := LoadPublicKey(pubPath)
	if err != nil {
		t.Fatalf("failed to load public key: %v", err)
	}

	data := []byte("test")
	encrypted, err := Encrypt(data, loadedPub)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, loadedPriv)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if string(decrypted) != string(data) {
		t.Error("loaded keys do not match original")
	}
}

func TestLoadPrivateKeyPKCS8(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	dir := t.TempDir()

	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("failed to marshal pkcs8: %v", err)
	}

	pemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Bytes,
	})

	path := filepath.Join(dir, "pkcs8.pem")
	if err := os.WriteFile(path, pemBytes, 0600); err != nil {
		t.Fatalf("failed to write key: %v", err)
	}

	_, err = LoadPrivateKey(path)
	if err != nil {
		t.Fatalf("failed to load pkcs8 private key: %v", err)
	}
}

func TestLoadInvalidPEM(t *testing.T) {
	dir := t.TempDir()

	path := filepath.Join(dir, "bad.pem")
	if err := os.WriteFile(path, []byte("not a pem file"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	_, err := LoadPublicKey(path)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse PEM block")
}

func TestLoadPrivateKey_Errors(t *testing.T) {
	dir := t.TempDir()

	t.Run("Missing file", func(t *testing.T) {
		_, err := LoadPrivateKey(filepath.Join(dir, "missing"))
		assert.Error(t, err)
	})

	t.Run("Invalid block type", func(t *testing.T) {
		path := filepath.Join(dir, "badtype.pem")
		err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("abc")}), 0600)
		assert.NoError(t, err)
		_, err = LoadPrivateKey(path)
		assert.Error(t, err)
	})

	t.Run("Malformed key bytes", func(t *testing.T) {
		path := filepath.Join(dir, "malformed.pem")
		err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("not-key")}), 0600)
		assert.NoError(t, err)
		_, err = LoadPrivateKey(path)
		assert.Error(t, err)
	})
}

func TestLoadPublicKey_Errors(t *testing.T) {
	dir := t.TempDir()

	t.Run("Missing file", func(t *testing.T) {
		_, err := LoadPublicKey(filepath.Join(dir, "missing"))
		assert.Error(t, err)
	})

	t.Run("Invalid block type", func(t *testing.T) {
		path := filepath.Join(dir, "badtype.pem")
		err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: []byte("abc")}), 0600)
		assert.NoError(t, err)
		_, err = LoadPublicKey(path)
		assert.Error(t, err)
	})

	t.Run("Malformed key bytes", func(t *testing.T) {
		path := filepath.Join(dir, "malformed.pem")
		err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte("not-key")}), 0600)
		assert.NoError(t, err)
		_, err = LoadPublicKey(path)
		assert.Error(t, err)
	})
}
