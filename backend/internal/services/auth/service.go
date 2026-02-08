package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ivankudzin/tgapp/backend/internal/domain/enums"
)

const (
	MinRefreshTTL = 30 * 24 * time.Hour
	MaxRefreshTTL = 90 * 24 * time.Hour
)

type SessionStore interface {
	Create(ctx context.Context, session SessionRecord, refreshToken string) error
	GetSession(ctx context.Context, sid string) (SessionRecord, error)
	GetByRefreshToken(ctx context.Context, refreshToken string) (SessionRecord, error)
	RotateRefresh(ctx context.Context, sid, oldRefreshToken, newRefreshToken string, expiresAt time.Time) error
	DeleteSession(ctx context.Context, sid string) error
	DeleteAllForUser(ctx context.Context, userID int64) error
}

type Service struct {
	jwt        *JWTManager
	sessions   SessionStore
	refreshTTL time.Duration
	now        func() time.Time
}

func NewService(jwtManager *JWTManager, sessions SessionStore, refreshTTL time.Duration) *Service {
	if refreshTTL < MinRefreshTTL {
		refreshTTL = MinRefreshTTL
	}
	if refreshTTL > MaxRefreshTTL {
		refreshTTL = MaxRefreshTTL
	}

	return &Service{
		jwt:        jwtManager,
		sessions:   sessions,
		refreshTTL: refreshTTL,
		now:        time.Now,
	}
}

func (s *Service) LoginTelegram(ctx context.Context, initData string) (AuthResult, error) {
	if err := ValidateTelegramInitData(initData); err != nil {
		return AuthResult{}, err
	}

	userID, err := ResolveTelegramUserID(initData)
	if err != nil {
		return AuthResult{}, fmt.Errorf("resolve telegram user id: %w", err)
	}

	return s.issueForUser(ctx, userID, string(enums.RoleUser))
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (AuthResult, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return AuthResult{}, ErrInvalidInput
	}

	session, err := s.sessions.GetByRefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, ErrRefreshNotFound) {
			return AuthResult{}, ErrUnauthorized
		}
		return AuthResult{}, fmt.Errorf("get refresh token session: %w", err)
	}
	if s.now().After(session.ExpiresAt) {
		return AuthResult{}, ErrUnauthorized
	}

	newRefreshToken, err := NewRefreshToken()
	if err != nil {
		return AuthResult{}, fmt.Errorf("generate refresh token: %w", err)
	}

	newExpiresAt := s.now().Add(s.refreshTTL)
	if err := s.sessions.RotateRefresh(ctx, session.SID, refreshToken, newRefreshToken, newExpiresAt); err != nil {
		if errors.Is(err, ErrRefreshNotFound) {
			return AuthResult{}, ErrUnauthorized
		}
		return AuthResult{}, fmt.Errorf("rotate refresh token: %w", err)
	}

	accessToken, accessExpires, err := s.jwt.GenerateAccessToken(session.UserID, session.SID, session.Role)
	if err != nil {
		return AuthResult{}, fmt.Errorf("generate access token: %w", err)
	}

	return AuthResult{
		AccessToken:   accessToken,
		RefreshToken:  newRefreshToken,
		AccessExpires: accessExpires,
		Me: Me{
			ID:   session.UserID,
			Role: session.Role,
		},
	}, nil
}

func (s *Service) Logout(ctx context.Context, sid string) error {
	if strings.TrimSpace(sid) == "" {
		return ErrInvalidInput
	}
	if err := s.sessions.DeleteSession(ctx, sid); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *Service) LogoutAll(ctx context.Context, userID int64) error {
	if userID <= 0 {
		return ErrInvalidInput
	}
	if err := s.sessions.DeleteAllForUser(ctx, userID); err != nil {
		return fmt.Errorf("delete all sessions: %w", err)
	}
	return nil
}

func (s *Service) ValidateAccessToken(ctx context.Context, accessToken string) (AccessClaims, error) {
	claims, err := s.jwt.ParseAccessToken(accessToken)
	if err != nil {
		return AccessClaims{}, ErrUnauthorized
	}

	session, err := s.sessions.GetSession(ctx, claims.SID)
	if err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			return AccessClaims{}, ErrUnauthorized
		}
		return AccessClaims{}, fmt.Errorf("get session: %w", err)
	}

	if session.UserID != claims.UserID || session.Role != claims.Role {
		return AccessClaims{}, ErrUnauthorized
	}
	if s.now().After(session.ExpiresAt) {
		return AccessClaims{}, ErrUnauthorized
	}

	return claims, nil
}

func (s *Service) issueForUser(ctx context.Context, userID int64, role string) (AuthResult, error) {
	sessionID, err := NewSessionID()
	if err != nil {
		return AuthResult{}, fmt.Errorf("generate session id: %w", err)
	}
	refreshToken, err := NewRefreshToken()
	if err != nil {
		return AuthResult{}, fmt.Errorf("generate refresh token: %w", err)
	}

	sessionExpiresAt := s.now().Add(s.refreshTTL)
	session := SessionRecord{
		SID:       sessionID,
		UserID:    userID,
		Role:      role,
		ExpiresAt: sessionExpiresAt,
	}
	if err := s.sessions.Create(ctx, session, refreshToken); err != nil {
		return AuthResult{}, fmt.Errorf("create session: %w", err)
	}

	accessToken, accessExpires, err := s.jwt.GenerateAccessToken(userID, sessionID, role)
	if err != nil {
		return AuthResult{}, fmt.Errorf("generate access token: %w", err)
	}

	return AuthResult{
		AccessToken:   accessToken,
		RefreshToken:  refreshToken,
		AccessExpires: accessExpires,
		Me: Me{
			ID:   userID,
			Role: role,
		},
	}, nil
}
