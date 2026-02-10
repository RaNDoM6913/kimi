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
	loginRes, err := svc.LoginTelegram(ctx, "user_id=1001", "7adf6f94-8cd6-4f6f-9b54-c8eaf9f19fbf")
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
	loginRes, err := svc.LoginTelegram(ctx, "user_id=2002", "8f69f668-18ad-4ff1-b2ea-a42f99772c34")
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

func TestLoginTelegramUsesRoleFromUserStore(t *testing.T) {
	svc, cleanup := newAuthServiceForTest(t)
	defer cleanup()

	svc.AttachUsers(fakeUserStore{
		userID: 5001,
		role:   "OWNER",
	})

	ctx := context.Background()
	loginRes, err := svc.LoginTelegram(ctx, "user_id=3003", "39a60e35-3d38-4cf8-a69f-d8bc7c6076e4")
	if err != nil {
		t.Fatalf("login telegram: %v", err)
	}

	if loginRes.Me.ID != 5001 {
		t.Fatalf("unexpected user id: got %d want %d", loginRes.Me.ID, 5001)
	}
	if loginRes.Me.Role != "OWNER" {
		t.Fatalf("unexpected role in login response: got %q want %q", loginRes.Me.Role, "OWNER")
	}

	claims, err := svc.ValidateAccessToken(ctx, loginRes.AccessToken)
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	if claims.UserID != 5001 {
		t.Fatalf("unexpected user id in claims: got %d want %d", claims.UserID, 5001)
	}
	if claims.Role != "OWNER" {
		t.Fatalf("unexpected role in claims: got %q want %q", claims.Role, "OWNER")
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

type fakeUserStore struct {
	userID int64
	role   string
}

func (f fakeUserStore) GetOrCreateByTelegramID(context.Context, int64) (authsvc.UserRecord, error) {
	return authsvc.UserRecord{
		UserID: f.userID,
		Role:   f.role,
	}, nil
}
