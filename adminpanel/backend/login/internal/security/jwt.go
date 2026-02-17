package security

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

type AdminClaims struct {
	UserID     int64  `json:"uid"`
	TelegramID int64  `json:"tid"`
	Role       string `json:"role"`
	Username   string `json:"username,omitempty"`
	SID        string `json:"sid"`
	jwt.RegisteredClaims
}

func NewTokenManager(secret string, ttl time.Duration) *TokenManager {
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	return &TokenManager{secret: []byte(secret), ttl: ttl}
}

func (m *TokenManager) Issue(userID, telegramID int64, role, username, sessionID string) (string, time.Time, error) {
	now := time.Now().UTC()
	expires := now.Add(m.ttl)
	claims := AdminClaims{
		UserID:     userID,
		TelegramID: telegramID,
		Role:       role,
		Username:   username,
		SID:        sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign jwt: %w", err)
	}
	return signed, expires, nil
}

func (m *TokenManager) Parse(tokenString string) (AdminClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AdminClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil {
		return AdminClaims{}, ErrInvalidToken
	}
	claims, ok := token.Claims.(*AdminClaims)
	if !ok || !token.Valid {
		return AdminClaims{}, ErrInvalidToken
	}
	if claims.UserID <= 0 || claims.SID == "" {
		return AdminClaims{}, ErrInvalidToken
	}
	return *claims, nil
}
