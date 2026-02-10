package apiapp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"

	"github.com/ivankudzin/tgapp/backend/internal/config"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
)

func TestRequireRoleAllowsCaseInsensitiveMatch(t *testing.T) {
	mw := RequireRole("OWNER", "SUPPORT", "MODERATOR")

	req := httptest.NewRequest(http.MethodGet, "/admin/health", nil)
	req = req.WithContext(authsvc.WithIdentity(context.Background(), authsvc.Identity{
		UserID: 1,
		SID:    "sid-1",
		Role:   "moderator",
	}))
	rr := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusNoContent)
	}
}

func TestRequireRoleRejectsForbiddenRole(t *testing.T) {
	mw := RequireRole("OWNER", "SUPPORT", "MODERATOR")

	req := httptest.NewRequest(http.MethodGet, "/admin/health", nil)
	req = req.WithContext(authsvc.WithIdentity(context.Background(), authsvc.Identity{
		UserID: 2,
		SID:    "sid-2",
		Role:   "user",
	}))
	rr := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatalf("handler must not be called for forbidden role")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusForbidden)
	}
}

func TestAdminBotAuthMiddlewareRejectsInvalidToken(t *testing.T) {
	mw := AdminBotAuthMiddleware(config.AdminConfig{
		BotToken: "secret-token",
		BotRole:  "MODERATOR",
	}, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/admin/bot/queue", nil)
	req.Header.Set("X-Admin-Bot-Token", "bad-token")
	req.Header.Set("X-Actor-Tg-Id", "123")
	rr := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatalf("handler must not be called on invalid token")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestAdminBotAuthMiddlewareRejectsInvalidActorHeader(t *testing.T) {
	mw := AdminBotAuthMiddleware(config.AdminConfig{
		BotToken: "secret-token",
		BotRole:  "MODERATOR",
	}, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/admin/bot/queue", nil)
	req.Header.Set("X-Admin-Bot-Token", "secret-token")
	req.Header.Set("X-Actor-Tg-Id", "abc")
	rr := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatalf("handler must not be called on invalid actor header")
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestAdminBotAuthMiddlewareSetsActorContext(t *testing.T) {
	mw := AdminBotAuthMiddleware(config.AdminConfig{
		BotToken: "secret-token",
		BotRole:  "MODERATOR",
	}, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/admin/bot/queue", nil)
	req.Header.Set("X-Admin-Bot-Token", "secret-token")
	req.Header.Set("X-Actor-Tg-Id", "987654321")
	rr := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isBot, ok := authsvc.ActorIsBotFromContext(r.Context())
		if !ok || !isBot {
			t.Fatalf("actor_is_bot missing in context")
		}
		role, ok := authsvc.ActorRoleFromContext(r.Context())
		if !ok || role != "MODERATOR" {
			t.Fatalf("actor_role mismatch: %q", role)
		}
		tgID, ok := authsvc.ActorTGIDFromContext(r.Context())
		if !ok || tgID != 987654321 {
			t.Fatalf("actor_tg_id mismatch: %d", tgID)
		}
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d want %d", rr.Code, http.StatusNoContent)
	}
}
