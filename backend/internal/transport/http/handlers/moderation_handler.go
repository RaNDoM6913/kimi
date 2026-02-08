package handlers

import (
	"errors"
	"net/http"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	modsvc "github.com/ivankudzin/tgapp/backend/internal/services/moderation"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type ModerationHandler struct {
	service *modsvc.Service
}

func NewModerationHandler(service *modsvc.Service) *ModerationHandler {
	return &ModerationHandler{service: service}
}

func (h *ModerationHandler) Handle(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "MODERATION_SERVICE_UNAVAILABLE", "moderation service is unavailable")
		return
	}

	status, err := h.service.GetUserStatus(r.Context(), identity.UserID)
	if err != nil {
		switch {
		case errors.Is(err, modsvc.ErrQueueEmpty):
			httperrors.Write(w, http.StatusOK, dto.ModerationStatusResponse{
				Status:    "PENDING",
				ETABucket: "up_to_10",
			})
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to load moderation status")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, dto.ModerationStatusResponse{
		Status:          status.Status,
		ReasonText:      status.ReasonText,
		RequiredFixStep: status.RequiredFixStep,
		ETABucket:       status.ETABucket,
	})
}
