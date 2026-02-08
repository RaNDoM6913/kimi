package handlers

import (
	"errors"
	"net/http"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	geosvc "github.com/ivankudzin/tgapp/backend/internal/services/geo"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type LocationHandler struct {
	service *geosvc.Service
}

func NewLocationHandler(service *geosvc.Service) *LocationHandler {
	return &LocationHandler{service: service}
}

func (h *LocationHandler) Handle(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "LOCATION_SERVICE_UNAVAILABLE", "location service is unavailable")
		return
	}

	var req dto.ProfileLocationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid request body")
		return
	}
	if req.Lat == nil || req.Lon == nil {
		writeBadRequest(w, "VALIDATION_ERROR", "lat and lon are required")
		return
	}

	city, err := h.service.UpdateProfileLocation(r.Context(), identity.UserID, *req.Lat, *req.Lon)
	if err != nil {
		switch {
		case errors.Is(err, geosvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid lat/lon")
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to update profile location")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, dto.ProfileLocationResponse{
		CityID:   city.ID,
		CityName: city.Name,
	})
}
