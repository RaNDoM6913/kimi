package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"

	"github.com/ivankudzin/tgapp/backend/internal/config"
	redrepo "github.com/ivankudzin/tgapp/backend/internal/repo/redis"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
)

func TestMeReturnsOwnerRoleFromAuthIdentity(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("run miniredis: %v", err)
	}
	defer mr.Close()

	redisClient := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	defer func() { _ = redisClient.Close() }()

	sessionRepo := redrepo.NewSessionRepo(redisClient)
	jwtManager := authsvc.NewJWTManager("test-secret", 15*time.Minute)
	authService := authsvc.NewService(jwtManager, sessionRepo, 45*24*time.Hour)
	authService.AttachUsers(meTestUserStore{
		userID: 7001,
		role:   "OWNER",
	})

	loginRes, err := authService.LoginTelegram(context.Background(), "user_id=777", "c4cc1deb-9f95-4e40-952c-8ea393f56e00")
	if err != nil {
		t.Fatalf("login telegram: %v", err)
	}

	claims, err := authService.ValidateAccessToken(context.Background(), loginRes.AccessToken)
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}

	handler := NewMeHandler(config.Default().Remote, nil)
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req = req.WithContext(authsvc.WithIdentity(req.Context(), authsvc.Identity{
		UserID: claims.UserID,
		SID:    claims.SID,
		Role:   claims.Role,
	}))

	rr := httptest.NewRecorder()
	handler.Handle(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusOK)
	}

	var payload dto.MeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload.User.Role != "OWNER" {
		t.Fatalf("unexpected me.user.role: got %q want %q", payload.User.Role, "OWNER")
	}

	var raw map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &raw); err != nil {
		t.Fatalf("decode raw response: %v", err)
	}
	userRaw, ok := raw["user"].(map[string]any)
	if !ok {
		t.Fatalf("response.user is not an object")
	}
	if _, exists := userRaw["lat"]; exists {
		t.Fatalf("/me must not expose exact lat")
	}
	if _, exists := userRaw["lon"]; exists {
		t.Fatalf("/me must not expose exact lon")
	}
}

type meTestUserStore struct {
	userID int64
	role   string
}

func (s meTestUserStore) GetOrCreateByTelegramID(context.Context, int64) (authsvc.UserRecord, error) {
	return authsvc.UserRecord{
		UserID: s.userID,
		Role:   s.role,
	}, nil
}
