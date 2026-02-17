// Package hashutil provides utilities for hashing data.
// It includes functions to compute SHA256 hashes with an optional secret key
// for integrity verification or signature purposes.
package hashutil

import (
	"crypto/sha256"
	"encoding/hex"
)

func CalcSHA256(body []byte, key string) string {
	h := sha256.New()
	h.Write(body)
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}
