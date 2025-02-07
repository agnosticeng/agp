package signer

import (
	"crypto/hmac"
	"crypto/sha256"
)

type Signer func(data []byte) []byte

func HMAC256Signer(secret []byte) func([]byte) []byte {
	return func(data []byte) []byte {
		var hmac = hmac.New(sha256.New, secret)
		hmac.Write([]byte(data))
		return hmac.Sum(nil)
	}
}
