package handlers

import (
	"errors"
	"net/http"

	analyticsvc "github.com/ivankudzin/tgapp/backend/internal/services/analytics"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type EventsHandler struct {
	service *analyticsvc.Service
}

func NewEventsHandler(service *analyticsvc.Service) *EventsHandler {
	return &EventsHandler{service: service}
}

func (h *EventsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	h.Batch(w, r)
}

func (h *EventsHandler) Batch(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeInternal(w, "EVENTS_SERVICE_UNAVAILABLE", "events service is unavailable")
		return
	}

	var req dto.EventsBatchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid request body")
		return
	}

	input := make([]analyticsvc.BatchEvent, 0, len(req))
	for _, item := range req {
		input = append(input, analyticsvc.BatchEvent{
			Name:  item.Name,
			TS:    item.TS,
			Props: item.Props,
		})
	}

	var userID *int64
	if identity, ok := authsvc.IdentityFromContext(r.Context()); ok && identity.UserID > 0 {
		uid := identity.UserID
		userID = &uid
	}

	if err := h.service.IngestBatch(r.Context(), userID, input); err != nil {
		switch {
		case errors.Is(err, analyticsvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid events batch: max 100 events, each with non-empty name")
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to ingest events")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, dto.EventsBatchResponse{
		OK:       true,
		Accepted: len(input),
	})
}
