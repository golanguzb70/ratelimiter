package ratelimiter

import (
	"crypto/sha256"
	"encoding/hex"
)

func Hash(key string) string {
	hasher := sha256.New()

	hasher.Write([]byte(key))

	hashSum := hasher.Sum(nil)

	return hex.EncodeToString(hashSum)
}
