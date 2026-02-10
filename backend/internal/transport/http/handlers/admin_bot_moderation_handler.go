package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
	analyticsvc "github.com/ivankudzin/tgapp/backend/internal/services/analytics"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	modsvc "github.com/ivankudzin/tgapp/backend/internal/services/moderation"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type AdminBotModerationHandler struct {
	service   *modsvc.Service
	telemetry *analyticsvc.Service
}

func NewAdminBotModerationHandler(service *modsvc.Service, telemetry *analyticsvc.Service) *AdminBotModerationHandler {
	return &AdminBotModerationHandler{
		service:   service,
		telemetry: telemetry,
	}
}

func (h *AdminBotModerationHandler) QueueAcquire(w http.ResponseWriter, r *http.Request) {
	actorTGID, ok := adminBotActorTGID(r)
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "admin bot authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "MODERATION_SERVICE_UNAVAILABLE", "moderation service is unavailable")
		return
	}

	item, err := h.service.GetNextQueueItem(r.Context(), actorTGID)
	if err != nil {
		switch {
		case errors.Is(err, modsvc.ErrQueueEmpty):
			w.WriteHeader(http.StatusNoContent)
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to acquire moderation item")
		}
		return
	}

	var birthdate *string
	if item.Profile.Birthdate != nil {
		v := item.Profile.Birthdate.UTC().Format("2006-01-02")
		birthdate = &v
	}

	httperrors.Write(w, http.StatusOK, dto.AdminBotModQueueAcquireResponse{
		ModerationItem: dto.AdminBotModerationItem{
			ID:           item.ItemID,
			UserID:       item.UserID,
			Status:       item.Status,
			ETABucket:    item.ETABucket,
			CreatedAt:    item.CreatedAt,
			LockedByTGID: item.LockedByTGID,
			LockedAt:     item.LockedAt,
			LockedUntil:  item.LockedUntil,
		},
		Profile: dto.AdminBotProfileCard{
			UserID:      item.Profile.UserID,
			DisplayName: item.Profile.DisplayName,
			CityID:      item.Profile.CityID,
			Gender:      item.Profile.Gender,
			LookingFor:  item.Profile.LookingFor,
			Goals:       item.Profile.Goals,
			Occupation:  item.Profile.Occupation,
			Education:   item.Profile.Education,
			Birthdate:   birthdate,
		},
		Media: dto.AdminBotProfileMedia{
			Photos: append([]string(nil), item.PhotoURLs...),
			Circle: item.CircleURL,
		},
	})
}

func (h *AdminBotModerationHandler) Approve(w http.ResponseWriter, r *http.Request) {
	actorTGID, ok := adminBotActorTGID(r)
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "admin bot authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "MODERATION_SERVICE_UNAVAILABLE", "moderation service is unavailable")
		return
	}

	itemID, ok := moderationItemIDFromRequest(r)
	if !ok {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid moderation item id")
		return
	}

	if err := h.service.Approve(r.Context(), itemID, actorTGID); err != nil {
		switch {
		case errors.Is(err, pgrepo.ErrModerationItemNotFound):
			httperrors.Write(w, http.StatusNotFound, httperrors.APIError{
				Code:    "NOT_FOUND",
				Message: "moderation item not found",
			})
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to approve moderation item")
		}
		return
	}

	h.logModerationAudit(r, "MODERATION_APPROVE", actorTGID, itemID, nil)
	httperrors.Write(w, http.StatusOK, dto.LogoutResponse{OK: true})
}

func (h *AdminBotModerationHandler) Reject(w http.ResponseWriter, r *http.Request) {
	actorTGID, ok := adminBotActorTGID(r)
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "admin bot authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "MODERATION_SERVICE_UNAVAILABLE", "moderation service is unavailable")
		return
	}

	itemID, ok := moderationItemIDFromRequest(r)
	if !ok {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid moderation item id")
		return
	}

	var req dto.AdminBotModerationRejectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if err := h.service.Reject(
		r.Context(),
		itemID,
		actorTGID,
		req.ReasonCode,
		req.ReasonText,
		req.RequiredFixStep,
	); err != nil {
		switch {
		case errors.Is(err, modsvc.ErrInvalidReasonCode):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid reason_code")
		case errors.Is(err, pgrepo.ErrModerationItemNotFound):
			httperrors.Write(w, http.StatusNotFound, httperrors.APIError{
				Code:    "NOT_FOUND",
				Message: "moderation item not found",
			})
		default:
			if strings.Contains(strings.ToLower(err.Error()), "required") {
				writeBadRequest(w, "VALIDATION_ERROR", "invalid reject payload")
				return
			}
			writeInternal(w, "INTERNAL_ERROR", "failed to reject moderation item")
		}
		return
	}

	h.logModerationAudit(r, "MODERATION_REJECT", actorTGID, itemID, map[string]any{
		"reason_code":       strings.ToUpper(strings.TrimSpace(req.ReasonCode)),
		"required_fix_step": strings.TrimSpace(req.RequiredFixStep),
	})
	httperrors.Write(w, http.StatusOK, dto.LogoutResponse{OK: true})
}

func (h *AdminBotModerationHandler) logModerationAudit(
	r *http.Request,
	action string,
	actorTGID int64,
	itemID int64,
	extra map[string]any,
) {
	if h.telemetry == nil || r == nil {
		return
	}

	props := map[string]any{
		"action":             action,
		"actor_tg_id":        actorTGID,
		"moderation_item_id": itemID,
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

func adminBotActorTGID(r *http.Request) (int64, bool) {
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

func moderationItemIDFromRequest(r *http.Request) (int64, bool) {
	if r == nil {
		return 0, false
	}
	rawID := strings.TrimSpace(chi.URLParam(r, "id"))
	if rawID == "" {
		return 0, false
	}
	itemID, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || itemID <= 0 {
		return 0, false
	}
	return itemID, true
}
