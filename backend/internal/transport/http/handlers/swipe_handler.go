package handlers

import (
	"errors"
	"net/http"
	"strings"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	likessvc "github.com/ivankudzin/tgapp/backend/internal/services/likes"
	swipesvc "github.com/ivankudzin/tgapp/backend/internal/services/swipes"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type SwipeHandler struct {
	service *swipesvc.Service
}

func NewSwipeHandler(service *swipesvc.Service) *SwipeHandler {
	return &SwipeHandler{service: service}
}

func (h *SwipeHandler) Handle(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "SWIPE_SERVICE_UNAVAILABLE", "swipe service is unavailable")
		return
	}

	var req dto.SwipeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if req.TargetID <= 0 || strings.TrimSpace(req.Action) == "" {
		writeBadRequest(w, "VALIDATION_ERROR", "target_id and action are required")
		return
	}

	result, err := h.service.Swipe(r.Context(), identity.UserID, req.TargetID, req.Action, timezoneFromRequest(r))
	if err != nil {
		switch {
		case errors.Is(err, swipesvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid swipe request")
		case errors.Is(err, swipesvc.ErrUnsupportedAction):
			writeBadRequest(w, "VALIDATION_ERROR", "unsupported action")
		case errors.Is(err, swipesvc.ErrSuperLikeRequirements):
			httperrors.Write(w, http.StatusConflict, httperrors.APIError{
				Code:    "SUPERLIKE_REQUIREMENTS_NOT_MET",
				Message: "superlike requires like_token=1 and superlike_credit=1",
			})
		case errors.Is(err, likessvc.ErrDailyLimit):
			httperrors.Write(w, http.StatusTooManyRequests, httperrors.APIError{
				Code:    "LIKE_LIMIT_REACHED",
				Message: "daily likes limit reached",
			})
		default:
			if tf, ok := likessvc.IsTooFast(err); ok {
				httperrors.Write(w, http.StatusTooManyRequests, struct {
					Code          string `json:"code"`
					Message       string `json:"message"`
					RetryAfterSec int64  `json:"retry_after_sec"`
				}{
					Code:          "TOO_FAST",
					Message:       "too many like actions, slow down",
					RetryAfterSec: tf.RetryAfter(),
				})
				return
			}
			writeInternal(w, "INTERNAL_ERROR", "failed to process swipe")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, struct {
		OK           bool                 `json:"ok"`
		MatchCreated bool                 `json:"match_created"`
		Quota        quotaResponsePayload `json:"quota"`
	}{
		OK:           true,
		MatchCreated: result.MatchCreated,
		Quota:        mapQuotaSnapshot(result.Quota),
	})
}

func timezoneFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v := strings.TrimSpace(r.Header.Get("X-Timezone")); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.URL.Query().Get("tz")); v != "" {
		return v
	}
	return ""
}
