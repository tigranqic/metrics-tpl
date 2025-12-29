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
