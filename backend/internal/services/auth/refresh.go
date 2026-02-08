package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func NewOpaqueToken(byteLen int) (string, error) {
	if byteLen <= 0 {
		return "", fmt.Errorf("invalid token size")
	}

	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}

func NewRefreshToken() (string, error) {
	return NewOpaqueToken(32)
}

func NewSessionID() (string, error) {
	return NewOpaqueToken(20)
}
