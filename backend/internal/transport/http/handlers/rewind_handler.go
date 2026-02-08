package handlers

import (
	"errors"
	"net/http"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	swipesvc "github.com/ivankudzin/tgapp/backend/internal/services/swipes"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type RewindHandler struct {
	service *swipesvc.Service
}

func NewRewindHandler(service *swipesvc.Service) *RewindHandler {
	return &RewindHandler{service: service}
}

func (h *RewindHandler) Handle(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "REWIND_SERVICE_UNAVAILABLE", "rewind service is unavailable")
		return
	}

	result, err := h.service.Rewind(r.Context(), identity.UserID, timezoneFromRequest(r))
	if err != nil {
		switch {
		case errors.Is(err, swipesvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid rewind request")
		case errors.Is(err, swipesvc.ErrNoActionsToRewind):
			httperrors.Write(w, http.StatusConflict, httperrors.APIError{
				Code:    "NOTHING_TO_REWIND",
				Message: "no actions to rewind",
			})
		case errors.Is(err, swipesvc.ErrRewindLimitReached):
			httperrors.Write(w, http.StatusTooManyRequests, httperrors.APIError{
				Code:    "REWIND_LIMIT_REACHED",
				Message: "daily rewind limit reached",
			})
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to process rewind")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, struct {
		OK             bool                 `json:"ok"`
		UndoneAction   string               `json:"undone_action"`
		UndoneTargetID int64                `json:"undone_target_id"`
		Quota          quotaResponsePayload `json:"quota"`
	}{
		OK:             true,
		UndoneAction:   result.UndoneAction,
		UndoneTargetID: result.UndoneTargetID,
		Quota:          mapQuotaSnapshot(result.Quota),
	})
}
