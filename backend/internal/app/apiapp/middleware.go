package apiapp

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/ivankudzin/tgapp/backend/internal/config"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

func ApplyMiddlewares(r chiRouter, log *zap.Logger) {
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(deviceIDMiddleware())
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(60 * time.Second))
	r.Use(requestLogger(log))
}

func AuthMiddleware(authService *authsvc.Service, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authService == nil {
				httperrors.Write(w, http.StatusInternalServerError, httperrors.APIError{
					Code:    "AUTH_SERVICE_UNAVAILABLE",
					Message: "auth service is unavailable",
				})
				return
			}

			accessToken, ok := extractBearerToken(r.Header.Get("Authorization"))
			if !ok {
				httperrors.Write(w, http.StatusUnauthorized, httperrors.APIError{
					Code:    "UNAUTHORIZED",
					Message: "missing bearer token",
				})
				return
			}

			claims, err := authService.ValidateAccessToken(r.Context(), accessToken)
			if err != nil {
				if log != nil {
					log.Debug("auth middleware validation failed", zap.Error(err))
				}
				httperrors.Write(w, http.StatusUnauthorized, httperrors.APIError{
					Code:    "UNAUTHORIZED",
					Message: "invalid access token",
				})
				return
			}

			ctx := authsvc.WithIdentity(r.Context(), authsvc.Identity{
				UserID: claims.UserID,
				SID:    claims.SID,
				Role:   claims.Role,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(value string) (string, bool) {
	parts := strings.SplitN(strings.TrimSpace(value), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
		return "", false
	}
	return parts[1], true
}

func requestLogger(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			if log != nil {
				log.Info("http_request",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Duration("duration", time.Since(start)),
				)
			}
		})
	}
}

func deviceIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := strings.TrimSpace(r.Header.Get("X-Device-Id"))
			if raw == "" {
				next.ServeHTTP(w, r)
				return
			}

			parsed, err := uuid.Parse(raw)
			if err != nil {
				httperrors.Write(w, http.StatusBadRequest, httperrors.APIError{
					Code:    "VALIDATION_ERROR",
					Message: "invalid X-Device-Id header",
				})
				return
			}

			ctx := authsvc.WithDeviceID(r.Context(), parsed.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		normalized := normalizeRole(role)
		if normalized == "" {
			continue
		}
		allowed[normalized] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := authsvc.IdentityFromContext(r.Context())
			if !ok || strings.TrimSpace(identity.Role) == "" {
				httperrors.Write(w, http.StatusUnauthorized, httperrors.APIError{
					Code:    "UNAUTHORIZED",
					Message: "authentication required",
				})
				return
			}

			if len(allowed) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			if _, exists := allowed[normalizeRole(identity.Role)]; !exists {
				httperrors.Write(w, http.StatusForbidden, httperrors.APIError{
					Code:    "FORBIDDEN",
					Message: "insufficient role",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func AdminBotAuthMiddleware(cfg config.AdminConfig, log *zap.Logger) func(http.Handler) http.Handler {
	expectedToken := strings.TrimSpace(cfg.BotToken)
	role := strings.TrimSpace(cfg.BotRole)
	if role == "" {
		role = "MODERATOR"
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if expectedToken == "" {
				httperrors.Write(w, http.StatusInternalServerError, httperrors.APIError{
					Code:    "ADMIN_BOT_AUTH_UNAVAILABLE",
					Message: "admin bot auth is not configured",
				})
				return
			}

			incomingToken := strings.TrimSpace(r.Header.Get("X-Admin-Bot-Token"))
			if incomingToken != expectedToken {
				httperrors.Write(w, http.StatusUnauthorized, httperrors.APIError{
					Code:    "UNAUTHORIZED",
					Message: "invalid admin bot token",
				})
				return
			}

			rawActorTGID := strings.TrimSpace(r.Header.Get("X-Actor-Tg-Id"))
			actorTGID, err := strconv.ParseInt(rawActorTGID, 10, 64)
			if err != nil || actorTGID == 0 {
				httperrors.Write(w, http.StatusBadRequest, httperrors.APIError{
					Code:    "VALIDATION_ERROR",
					Message: "invalid X-Actor-Tg-Id header",
				})
				return
			}

			ctx := authsvc.WithActorIsBot(r.Context(), true)
			ctx = authsvc.WithActorRole(ctx, role)
			ctx = authsvc.WithActorTGID(ctx, actorTGID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type chiRouter interface {
	Use(middlewares ...func(http.Handler) http.Handler)
}

func normalizeRole(role string) string {
	return strings.ToUpper(strings.TrimSpace(role))
}
