package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type AuthHandler struct {
	service *authsvc.Service
}

func NewAuthHandler(service *authsvc.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	h.Telegram(w, r)
}

func (h *AuthHandler) Telegram(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeInternal(w, "AUTH_SERVICE_UNAVAILABLE", "auth service is unavailable")
		return
	}
	deviceID, ok := requiredDeviceID(r)
	if !ok {
		writeBadRequest(w, "VALIDATION_ERROR", "X-Device-Id header is required")
		return
	}

	var req dto.TelegramAuthRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "INVALID_REQUEST", "invalid request body")
		return
	}

	res, err := h.service.LoginTelegram(r.Context(), req.InitData, deviceID)
	if err != nil {
		handleAuthError(w, err)
		return
	}

	httperrors.Write(w, http.StatusOK, dto.AuthTokensResponse{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		ExpiresInSec: maxInt64(0, int64(time.Until(res.AccessExpires).Seconds())),
		Me: dto.AuthMeResponse{
			ID:   res.Me.ID,
			Role: res.Me.Role,
		},
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeInternal(w, "AUTH_SERVICE_UNAVAILABLE", "auth service is unavailable")
		return
	}

	var req dto.RefreshRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "INVALID_REQUEST", "invalid request body")
		return
	}

	res, err := h.service.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		handleAuthError(w, err)
		return
	}

	httperrors.Write(w, http.StatusOK, dto.AuthTokensResponse{
		AccessToken:  res.AccessToken,
		RefreshToken: res.RefreshToken,
		ExpiresInSec: maxInt64(0, int64(time.Until(res.AccessExpires).Seconds())),
		Me: dto.AuthMeResponse{
			ID:   res.Me.ID,
			Role: res.Me.Role,
		},
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeInternal(w, "AUTH_SERVICE_UNAVAILABLE", "auth service is unavailable")
		return
	}

	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}

	if err := h.service.Logout(r.Context(), identity.SID); err != nil {
		handleAuthError(w, err)
		return
	}

	httperrors.Write(w, http.StatusOK, dto.LogoutResponse{OK: true})
}

func (h *AuthHandler) LogoutAll(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeInternal(w, "AUTH_SERVICE_UNAVAILABLE", "auth service is unavailable")
		return
	}

	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}

	if err := h.service.LogoutAll(r.Context(), identity.UserID); err != nil {
		handleAuthError(w, err)
		return
	}

	httperrors.Write(w, http.StatusOK, dto.LogoutResponse{OK: true})
}

func handleAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, authsvc.ErrInvalidInput):
		writeBadRequest(w, "INVALID_REQUEST", "request validation failed")
	case errors.Is(err, authsvc.ErrUnauthorized):
		writeUnauthorized(w, "UNAUTHORIZED", "authentication failed")
	default:
		writeInternal(w, "INTERNAL_ERROR", "internal server error")
	}
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeBadRequest(w http.ResponseWriter, code, message string) {
	httperrors.Write(w, http.StatusBadRequest, httperrors.APIError{Code: code, Message: message})
}

func writeUnauthorized(w http.ResponseWriter, code, message string) {
	httperrors.Write(w, http.StatusUnauthorized, httperrors.APIError{Code: code, Message: message})
}

func writeInternal(w http.ResponseWriter, code, message string) {
	httperrors.Write(w, http.StatusInternalServerError, httperrors.APIError{Code: code, Message: message})
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func requiredDeviceID(r *http.Request) (string, bool) {
	if r == nil {
		return "", false
	}
	deviceID, ok := authsvc.DeviceIDFromContext(r.Context())
	if !ok || deviceID == "" {
		return "", false
	}
	return deviceID, true
}
