package handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	analyticsvc "github.com/ivankudzin/tgapp/backend/internal/services/analytics"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	userssvc "github.com/ivankudzin/tgapp/backend/internal/services/users"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type AdminBotUsersHandler struct {
	users     *userssvc.Service
	telemetry *analyticsvc.Service
}

func NewAdminBotUsersHandler(users *userssvc.Service, telemetry *analyticsvc.Service) *AdminBotUsersHandler {
	return &AdminBotUsersHandler{
		users:     users,
		telemetry: telemetry,
	}
}

func (h *AdminBotUsersHandler) LookupUser(w http.ResponseWriter, r *http.Request) {
	actorTGID, ok := adminBotActorTGIDForUsers(r)
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "admin bot authentication required")
		return
	}
	if h.users == nil {
		writeInternal(w, "USERS_SERVICE_UNAVAILABLE", "users service is unavailable")
		return
	}

	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		writeBadRequest(w, "VALIDATION_ERROR", "query is required")
		return
	}

	found, err := h.users.LookupUser(r.Context(), query)
	if err != nil {
		switch {
		case errors.Is(err, userssvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid lookup query")
		case errors.Is(err, userssvc.ErrNotFound):
			httperrors.Write(w, http.StatusNotFound, httperrors.APIError{
				Code:    "NOT_FOUND",
				Message: "user not found",
			})
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to lookup user")
		}
		return
	}

	h.logAuditAction(r, "LOOKUP_USER", actorTGID, found.UserID, map[string]any{
		"query":        query,
		"target_tg_id": found.TGID,
	})

	httperrors.Write(w, http.StatusOK, dto.AdminBotLookupUserResponse{
		User: dto.AdminBotLookupUser{
			UserID:           found.UserID,
			TGID:             found.TGID,
			Username:         found.Username,
			CityID:           found.CityID,
			Birthdate:        found.Birthdate,
			Age:              found.Age,
			Gender:           found.Gender,
			LookingFor:       found.LookingFor,
			Goals:            append([]string(nil), found.Goals...),
			Languages:        append([]string(nil), found.Languages...),
			Occupation:       found.Occupation,
			Education:        found.Education,
			ModerationStatus: found.ModerationStatus,
			Approved:         found.Approved,
			PhotoKeys:        append([]string(nil), found.PhotoKeys...),
			CircleKey:        found.CircleKey,
			PhotoURLs:        append([]string(nil), found.PhotoURLs...),
			CircleURL:        found.CircleURL,
			PlusExpiresAt:    found.PlusExpiresAt,
			BoostUntil:       found.BoostUntil,
			SuperlikeCredits: found.SuperlikeCredits,
			RevealCredits:    found.RevealCredits,
			LikeTokens:       found.LikeTokens,
			IsBanned:         found.IsBanned,
			BanReason:        found.BanReason,
		},
	})
}

func (h *AdminBotUsersHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	actorTGID, ok := adminBotActorTGIDForUsers(r)
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "admin bot authentication required")
		return
	}
	if h.users == nil {
		writeInternal(w, "USERS_SERVICE_UNAVAILABLE", "users service is unavailable")
		return
	}

	userID, ok := adminBotUserIDFromRequest(r)
	if !ok {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid user id")
		return
	}

	var req dto.AdminBotBanRequest
	if err := decodeJSON(r, &req); err != nil && !errors.Is(err, io.EOF) {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if err := h.users.SetBan(r.Context(), userID, true, req.Reason, actorTGID); err != nil {
		switch {
		case errors.Is(err, userssvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid ban request")
		case errors.Is(err, userssvc.ErrNotFound):
			httperrors.Write(w, http.StatusNotFound, httperrors.APIError{
				Code:    "NOT_FOUND",
				Message: "user not found",
			})
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to ban user")
		}
		return
	}

	h.logAuditAction(r, "BAN_USER", actorTGID, userID, map[string]any{
		"reason": strings.TrimSpace(req.Reason),
	})
	httperrors.Write(w, http.StatusOK, dto.LogoutResponse{OK: true})
}

func (h *AdminBotUsersHandler) UnbanUser(w http.ResponseWriter, r *http.Request) {
	actorTGID, ok := adminBotActorTGIDForUsers(r)
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "admin bot authentication required")
		return
	}
	if h.users == nil {
		writeInternal(w, "USERS_SERVICE_UNAVAILABLE", "users service is unavailable")
		return
	}

	userID, ok := adminBotUserIDFromRequest(r)
	if !ok {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid user id")
		return
	}

	if err := h.users.SetBan(r.Context(), userID, false, "", actorTGID); err != nil {
		switch {
		case errors.Is(err, userssvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid unban request")
		case errors.Is(err, userssvc.ErrNotFound):
			httperrors.Write(w, http.StatusNotFound, httperrors.APIError{
				Code:    "NOT_FOUND",
				Message: "user not found",
			})
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to unban user")
		}
		return
	}

	h.logAuditAction(r, "UNBAN_USER", actorTGID, userID, nil)
	httperrors.Write(w, http.StatusOK, dto.LogoutResponse{OK: true})
}

func (h *AdminBotUsersHandler) ForceReview(w http.ResponseWriter, r *http.Request) {
	actorTGID, ok := adminBotActorTGIDForUsers(r)
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "admin bot authentication required")
		return
	}
	if h.users == nil {
		writeInternal(w, "USERS_SERVICE_UNAVAILABLE", "users service is unavailable")
		return
	}

	userID, ok := adminBotUserIDFromRequest(r)
	if !ok {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid user id")
		return
	}

	if err := h.users.ForceReview(r.Context(), userID); err != nil {
		switch {
		case errors.Is(err, userssvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid force-review request")
		case errors.Is(err, userssvc.ErrNotFound):
			httperrors.Write(w, http.StatusNotFound, httperrors.APIError{
				Code:    "NOT_FOUND",
				Message: "user not found",
			})
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to force review")
		}
		return
	}

	h.logAuditAction(r, "FORCE_REVIEW", actorTGID, userID, nil)
	httperrors.Write(w, http.StatusOK, dto.LogoutResponse{OK: true})
}

func (h *AdminBotUsersHandler) logAuditAction(
	r *http.Request,
	action string,
	actorTGID int64,
	targetUserID int64,
	extra map[string]any,
) {
	if h.telemetry == nil || r == nil {
		return
	}

	props := map[string]any{
		"action":         action,
		"actor_tg_id":    actorTGID,
		"target_user_id": targetUserID,
	}
	for key, value := range extra {
		props[key] = value
	}

	_ = h.telemetry.IngestBatch(r.Context(), nil, []analyticsvc.BatchEvent{
		{
			Name:  "audit_log",
			TS:    time.Now().UTC().UnixMilli(),
			Props: props,
		},
	})
}

func adminBotActorTGIDForUsers(r *http.Request) (int64, bool) {
	if r == nil {
		return 0, false
	}

	isBot, ok := authsvc.ActorIsBotFromContext(r.Context())
	if !ok || !isBot {
		return 0, false
	}

	actorTGID, ok := authsvc.ActorTGIDFromContext(r.Context())
	if !ok || actorTGID == 0 {
		return 0, false
	}
	return actorTGID, true
}

func adminBotUserIDFromRequest(r *http.Request) (int64, bool) {
	if r == nil {
		return 0, false
	}
	rawID := strings.TrimSpace(chi.URLParam(r, "id"))
	if rawID == "" {
		return 0, false
	}
	userID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || userID <= 0 {
		return 0, false
	}
	return userID, true
}
