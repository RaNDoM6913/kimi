package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

const encryptedValuePrefix = "enc:v1:"

var ErrInvalidSecretCipherKey = errors.New("invalid secret cipher key")

type SecretCipher struct {
	aead cipher.AEAD
}

func NewSecretCipher(rawKey string) (*SecretCipher, error) {
	key, err := decodeKey(rawKey)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create aes cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}
	return &SecretCipher{aead: aead}, nil
}

func (c *SecretCipher) Encrypt(plain string) (string, error) {
	if c == nil || c.aead == nil {
		return "", ErrInvalidSecretCipherKey
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := c.aead.Seal(nil, nonce, []byte(plain), nil)
	payload := append(nonce, ciphertext...)
	return encryptedValuePrefix + base64.StdEncoding.EncodeToString(payload), nil
}

func (c *SecretCipher) Decrypt(value string) (string, error) {
	if c == nil || c.aead == nil {
		return "", ErrInvalidSecretCipherKey
	}
	trimmed := strings.TrimSpace(value)
	if !strings.HasPrefix(trimmed, encryptedValuePrefix) {
		// Backward compatibility for previously stored plaintext values.
		return trimmed, nil
	}

	encoded := strings.TrimPrefix(trimmed, encryptedValuePrefix)
	payload, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode encrypted secret: %w", err)
	}
	nonceSize := c.aead.NonceSize()
	if len(payload) <= nonceSize {
		return "", fmt.Errorf("invalid encrypted secret payload")
	}
	nonce := payload[:nonceSize]
	ciphertext := payload[nonceSize:]
	plain, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}
	return string(plain), nil
}

func decodeKey(raw string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, ErrInvalidSecretCipherKey
	}

	if decoded, err := base64.StdEncoding.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := base64.RawStdEncoding.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded, nil
	}
	if decoded, err := hex.DecodeString(trimmed); err == nil && len(decoded) == 32 {
		return decoded, nil
	}

	if len(trimmed) == 32 {
		return []byte(trimmed), nil
	}
	return nil, ErrInvalidSecretCipherKey
}
