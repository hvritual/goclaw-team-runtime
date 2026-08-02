// Package checksum provides content-digest adapters for Space.
package checksum

import (
	"crypto/sha256"
	"encoding/hex"
)

// SHA256 computes SHA-256 content digests.
type SHA256 struct{}

// Sum returns a lowercase sha256-prefixed digest.
func (SHA256) Sum(data []byte) string {
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:])
}
