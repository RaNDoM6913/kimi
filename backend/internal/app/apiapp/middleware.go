package apiapp

import (
	"net/http"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

func ApplyMiddlewares(r chiRouter, log *zap.Logger) {
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
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

type chiRouter interface {
	Use(middlewares ...func(http.Handler) http.Handler)
}
