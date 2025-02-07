package query_hasher

import (
	"crypto/sha256"
	"encoding/hex"
)

type QuerHashFunc func(query string) string

func SHA256QueryHasher(query string) string {
	var h = sha256.New()
	h.Write([]byte(query))
	return hex.EncodeToString(h.Sum(nil))
}
