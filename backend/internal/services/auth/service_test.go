package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	redrepo "github.com/ivankudzin/tgapp/backend/internal/repo/redis"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
)

func TestRefreshRotation(t *testing.T) {
	svc, cleanup := newAuthServiceForTest(t)
	defer cleanup()

	ctx := context.Background()
	loginRes, err := svc.LoginTelegram(ctx, "user_id=1001")
	if err != nil {
		t.Fatalf("login telegram: %v", err)
	}

	refreshRes, err := svc.Refresh(ctx, loginRes.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}

	if refreshRes.RefreshToken == loginRes.RefreshToken {
		t.Fatalf("refresh token was not rotated")
	}

	if _, err := svc.Refresh(ctx, loginRes.RefreshToken); !errors.Is(err, authsvc.ErrUnauthorized) {
		t.Fatalf("old refresh token should be unauthorized, got err=%v", err)
	}

	if _, err := svc.ValidateAccessToken(ctx, refreshRes.AccessToken); err != nil {
		t.Fatalf("new access token validation failed: %v", err)
	}
}

func TestLogoutInvalidatesSession(t *testing.T) {
	svc, cleanup := newAuthServiceForTest(t)
	defer cleanup()

	ctx := context.Background()
	loginRes, err := svc.LoginTelegram(ctx, "user_id=2002")
	if err != nil {
		t.Fatalf("login telegram: %v", err)
	}

	claims, err := svc.ValidateAccessToken(ctx, loginRes.AccessToken)
	if err != nil {
		t.Fatalf("validate access token before logout: %v", err)
	}

	if err := svc.Logout(ctx, claims.SID); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if _, err := svc.ValidateAccessToken(ctx, loginRes.AccessToken); !errors.Is(err, authsvc.ErrUnauthorized) {
		t.Fatalf("access token should be unauthorized after logout, got err=%v", err)
	}
}

func newAuthServiceForTest(t *testing.T) (*authsvc.Service, func()) {
	t.Helper()

	mini, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}

	client := goredis.NewClient(&goredis.Options{Addr: mini.Addr()})
	repo := redrepo.NewSessionRepo(client)
	jwtManager := authsvc.NewJWTManager("test-secret", 15*time.Minute)
	svc := authsvc.NewService(jwtManager, repo, 45*24*time.Hour)

	cleanup := func() {
		_ = client.Close()
		mini.Close()
	}

	return svc, cleanup
}
