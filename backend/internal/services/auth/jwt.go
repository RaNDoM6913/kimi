package auth

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secret    []byte
	accessTTL time.Duration
	now       func() time.Time
}

type tokenClaims struct {
	SID  string `json:"sid"`
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func NewJWTManager(secret string, accessTTL time.Duration) *JWTManager {
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}

	return &JWTManager{
		secret:    []byte(secret),
		accessTTL: accessTTL,
		now:       time.Now,
	}
}

func (m *JWTManager) GenerateAccessToken(userID int64, sid, role string) (string, time.Time, error) {
	if len(m.secret) == 0 {
		return "", time.Time{}, fmt.Errorf("jwt secret is empty")
	}
	if userID <= 0 || strings.TrimSpace(sid) == "" {
		return "", time.Time{}, fmt.Errorf("invalid access token payload")
	}

	now := m.now().UTC()
	expiresAt := now.Add(m.accessTTL)
	claims := tokenClaims{
		SID:  sid,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign access token: %w", err)
	}

	return signed, expiresAt, nil
}

func (m *JWTManager) ParseAccessToken(raw string) (AccessClaims, error) {
	if strings.TrimSpace(raw) == "" {
		return AccessClaims{}, ErrUnauthorized
	}

	claims := &tokenClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(_ *jwt.Token) (interface{}, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil || token == nil || !token.Valid {
		return AccessClaims{}, ErrUnauthorized
	}

	userID, parseErr := strconv.ParseInt(claims.Subject, 10, 64)
	if parseErr != nil || userID <= 0 {
		return AccessClaims{}, ErrUnauthorized
	}
	if strings.TrimSpace(claims.SID) == "" {
		return AccessClaims{}, ErrUnauthorized
	}
	if claims.ExpiresAt == nil {
		return AccessClaims{}, ErrUnauthorized
	}

	return AccessClaims{
		UserID:    userID,
		SID:       claims.SID,
		Role:      claims.Role,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}
