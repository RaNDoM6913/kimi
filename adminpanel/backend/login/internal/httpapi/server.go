package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ivankudzin/tgapp/adminpanel/backend/login/internal/service"
)

type Server struct {
	httpServer   *http.Server
	svc          *service.Service
	bootstrapKey string
}

func NewServer(addr string, svc *service.Service, bootstrapKey string) *Server {
	r := chi.NewRouter()
	s := &Server{svc: svc, bootstrapKey: bootstrapKey}
	s.registerRoutes(r)

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return s
}

func (s *Server) Start() error {
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) registerRoutes(r chi.Router) {
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
	})

	r.Route("/v1", func(r chi.Router) {
		r.Post("/auth/telegram/start", s.handleTelegramStart)
		r.Post("/auth/2fa/verify", s.handle2FAVerify)
		r.Post("/auth/password/verify", s.handlePasswordVerify)
		r.Get("/auth/me", s.handleMe)
		r.Post("/auth/logout", s.handleLogout)

		r.Post("/admin/2fa/setup/start", s.withBootstrapKey(s.handleStartTOTPSetup))
		r.Post("/admin/2fa/setup/confirm", s.withBootstrapKey(s.handleConfirmTOTPSetup))
	})
}

type telegramStartRequest struct {
	InitData string `json:"init_data"`
}

type verify2FARequest struct {
	ChallengeID string `json:"challenge_id"`
	Code        string `json:"code"`
}

type verifyPasswordRequest struct {
	ChallengeID string `json:"challenge_id"`
	Password    string `json:"password"`
}

type startTOTPSetupRequest struct {
	TelegramID  int64  `json:"telegram_id"`
	AccountName string `json:"account_name"`
}

type confirmTOTPSetupRequest struct {
	SetupID string `json:"setup_id"`
	Code    string `json:"code"`
}

func (s *Server) handleTelegramStart(w http.ResponseWriter, r *http.Request) {
	var req telegramStartRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	result, err := s.svc.TelegramStart(r.Context(), service.TelegramStartInput{
		InitData:  req.InitData,
		IP:        clientIP(r),
		UserAgent: r.UserAgent(),
	})
	if err != nil {
		s.handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handle2FAVerify(w http.ResponseWriter, r *http.Request) {
	var req verify2FARequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}
	result, err := s.svc.VerifyTOTP(r.Context(), req.ChallengeID, req.Code)
	if err != nil {
		s.handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handlePasswordVerify(w http.ResponseWriter, r *http.Request) {
	var req verifyPasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}
	result, err := s.svc.VerifyPassword(r.Context(), req.ChallengeID, req.Password)
	if err != nil {
		s.handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleStartTOTPSetup(w http.ResponseWriter, r *http.Request) {
	var req startTOTPSetupRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	result, err := s.svc.StartTOTPSetup(r.Context(), req.TelegramID, req.AccountName)
	if err != nil {
		s.handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleConfirmTOTPSetup(w http.ResponseWriter, r *http.Request) {
	var req confirmTOTPSetupRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON body")
		return
	}

	if err := s.svc.ConfirmTOTPSetup(r.Context(), req.SetupID, req.Code); err != nil {
		s.handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing Bearer token")
		return
	}
	claims, err := s.svc.ValidateAccessToken(r.Context(), token)
	if err != nil {
		s.handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":          claims.UserID,
		"telegram_id": claims.TelegramID,
		"role":        claims.Role,
		"username":    claims.Username,
	})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing Bearer token")
		return
	}
	if err := s.svc.Logout(r.Context(), token); err != nil {
		s.handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) withBootstrapKey(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.TrimSpace(s.bootstrapKey) == "" {
			writeError(w, http.StatusNotImplemented, "bootstrap_disabled", "Bootstrap key is not configured")
			return
		}
		provided := strings.TrimSpace(r.Header.Get("X-Bootstrap-Key"))
		if provided == "" || provided != s.bootstrapKey {
			writeError(w, http.StatusForbidden, "forbidden", "Invalid bootstrap key")
			return
		}
		next(w, r)
	}
}

func (s *Server) handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "invalid_input", "Invalid input")
	case errors.Is(err, service.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication failed")
	case errors.Is(err, service.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Access denied")
	case errors.Is(err, service.ErrAccountLocked):
		writeError(w, http.StatusLocked, "account_locked", "Account is temporarily locked")
	case errors.Is(err, service.ErrChallengeExpired):
		writeError(w, http.StatusUnauthorized, "challenge_expired", "Challenge expired")
	case errors.Is(err, service.ErrChallengeStep):
		writeError(w, http.StatusConflict, "invalid_step", "Invalid challenge step")
	case errors.Is(err, service.ErrTOTPNotConfigured):
		writeError(w, http.StatusPreconditionRequired, "totp_not_configured", "2FA is not configured")
	case errors.Is(err, service.ErrSessionExpired):
		writeError(w, http.StatusUnauthorized, "session_expired", "Session expired. Sign in again")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func decodeJSON(r *http.Request, out any) error {
	if r.Body == nil {
		return fmt.Errorf("empty body")
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, code int, errCode, msg string) {
	writeJSON(w, code, map[string]any{
		"error": map[string]any{
			"code":    errCode,
			"message": msg,
		},
	})
}

func extractBearerToken(authHeader string) string {
	parts := strings.Fields(strings.TrimSpace(authHeader))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			candidate := strings.TrimSpace(parts[0])
			if candidate != "" {
				return candidate
			}
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}
