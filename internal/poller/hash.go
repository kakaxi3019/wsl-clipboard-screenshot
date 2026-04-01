package poller

import (
	"crypto/sha256"
	"fmt"
)

// HashBytes computes SHA256 hash of data and returns hex string
func HashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}
