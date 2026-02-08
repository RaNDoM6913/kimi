package handlers

import (
	"net/http"
	"time"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	likessvc "github.com/ivankudzin/tgapp/backend/internal/services/likes"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type QuotaHandler struct {
	service *likessvc.Service
}

type quotaResponsePayload struct {
	LikesLeft         int       `json:"likes_left"`
	ResetAt           time.Time `json:"reset_at"`
	TooFastRetryAfter *int64    `json:"too_fast_retry_after,omitempty"`
}

func NewQuotaHandler(service *likessvc.Service) *QuotaHandler {
	return &QuotaHandler{service: service}
}

func (h *QuotaHandler) Handle(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "QUOTA_SERVICE_UNAVAILABLE", "quota service is unavailable")
		return
	}

	snapshot, err := h.service.GetSnapshot(r.Context(), identity.UserID, timezoneFromRequest(r))
	if err != nil {
		writeInternal(w, "INTERNAL_ERROR", "failed to load quota")
		return
	}

	httperrors.Write(w, http.StatusOK, mapQuotaSnapshot(snapshot))
}

func mapQuotaSnapshot(snapshot likessvc.Snapshot) quotaResponsePayload {
	return quotaResponsePayload{
		LikesLeft:         snapshot.LikesLeft,
		ResetAt:           snapshot.ResetAt.UTC(),
		TooFastRetryAfter: snapshot.TooFastRetryAfter,
	}
}
