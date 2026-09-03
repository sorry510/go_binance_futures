package contextengine

import (
	"crypto/sha256"
	"encoding/hex"
)

func ContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
