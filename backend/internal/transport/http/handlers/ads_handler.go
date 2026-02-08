package handlers

import (
	"errors"
	"net/http"

	adssvc "github.com/ivankudzin/tgapp/backend/internal/services/ads"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type AdsHandler struct {
	service *adssvc.Service
}

func NewAdsHandler(service *adssvc.Service) *AdsHandler {
	return &AdsHandler{service: service}
}

func (h *AdsHandler) Impression(w http.ResponseWriter, r *http.Request) {
	h.handleEvent(w, r, adssvc.EventTypeImpression)
}

func (h *AdsHandler) Click(w http.ResponseWriter, r *http.Request) {
	h.handleEvent(w, r, adssvc.EventTypeClick)
}

func (h *AdsHandler) handleEvent(w http.ResponseWriter, r *http.Request, eventType string) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "ADS_SERVICE_UNAVAILABLE", "ads service is unavailable")
		return
	}

	var req dto.AdEventRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid request body")
		return
	}

	var err error
	switch eventType {
	case adssvc.EventTypeClick:
		err = h.service.Click(r.Context(), identity.UserID, req.AdID, req.Meta)
	default:
		err = h.service.Impression(r.Context(), identity.UserID, req.AdID, req.Meta)
	}
	if err != nil {
		switch {
		case errors.Is(err, adssvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid ad event payload")
		case errors.Is(err, adssvc.ErrAdNotFound):
			httperrors.Write(w, http.StatusNotFound, httperrors.APIError{
				Code:    "AD_NOT_FOUND",
				Message: "ad card not found or inactive",
			})
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to save ad event")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, dto.LogoutResponse{OK: true})
}
