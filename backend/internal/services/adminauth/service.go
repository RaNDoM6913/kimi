package adminauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
)

var (
	ErrUnauthorized   = errors.New("unauthorized")
	ErrSessionExpired = errors.New("session expired")
	ErrUnavailable    = errors.New("admin auth is unavailable")
)

type SessionStore interface {
	Touch(ctx context.Context, sid uuid.UUID, adminUserID int64, idleTimeout time.Duration) (string, error)
}

type Service struct {
	secret      []byte
	sessions    SessionStore
	idleTimeout time.Duration
	configured  bool
}

type Claims struct {
	UserID     int64
	TelegramID int64
	Role       string
	Username   string
	SID        string
}

type tokenClaims struct {
	UserID     int64  `json:"uid"`
	TelegramID int64  `json:"tid"`
	Role       string `json:"role"`
	Username   string `json:"username,omitempty"`
	SID        string `json:"sid"`
	jwt.RegisteredClaims
}

func NewService(jwtSecret string, idleTimeout time.Duration, sessions SessionStore) *Service {
	secret := strings.TrimSpace(jwtSecret)
	if idleTimeout <= 0 {
		idleTimeout = 30 * time.Minute
	}
	return &Service{
		secret:      []byte(secret),
		sessions:    sessions,
		idleTimeout: idleTimeout,
		configured:  secret != "" && sessions != nil,
	}
}

func (s *Service) IsConfigured() bool {
	return s != nil && s.configured
}

func (s *Service) ValidateAccessToken(ctx context.Context, accessToken string) (Claims, error) {
	if !s.IsConfigured() {
		return Claims{}, ErrUnavailable
	}

	claims, err := s.parse(accessToken)
	if err != nil {
		return Claims{}, ErrUnauthorized
	}

	sid, err := uuid.Parse(strings.TrimSpace(claims.SID))
	if err != nil {
		return Claims{}, ErrUnauthorized
	}
	role, err := s.sessions.Touch(ctx, sid, claims.UserID, s.idleTimeout)
	if err != nil {
		if errors.Is(err, pgrepo.ErrAdminSessionNotFound) {
			return Claims{}, ErrSessionExpired
		}
		return Claims{}, fmt.Errorf("touch admin session: %w", err)
	}
	if strings.TrimSpace(role) != "" {
		claims.Role = role
	}
	return claims, nil
}

func (s *Service) parse(accessToken string) (Claims, error) {
	token, err := jwt.ParseWithClaims(accessToken, &tokenClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, ErrUnauthorized
		}
		return s.secret, nil
	})
	if err != nil {
		return Claims{}, ErrUnauthorized
	}
	tc, ok := token.Claims.(*tokenClaims)
	if !ok || !token.Valid {
		return Claims{}, ErrUnauthorized
	}
	if tc.UserID <= 0 || strings.TrimSpace(tc.SID) == "" {
		return Claims{}, ErrUnauthorized
	}
	return Claims{
		UserID:     tc.UserID,
		TelegramID: tc.TelegramID,
		Role:       strings.TrimSpace(tc.Role),
		Username:   strings.TrimSpace(tc.Username),
		SID:        strings.TrimSpace(tc.SID),
	}, nil
}
