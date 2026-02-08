package handlers

import (
	"errors"
	"net/http"
	"time"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	profilesvc "github.com/ivankudzin/tgapp/backend/internal/services/profiles"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type ProfileHandler struct {
	service *profilesvc.Service
}

func NewProfileHandler(service *profilesvc.Service) *ProfileHandler {
	return &ProfileHandler{service: service}
}

func (h *ProfileHandler) Handle(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w, "PROFILE_NOT_IMPLEMENTED", "profile endpoint is not implemented yet")
}

func (h *ProfileHandler) Core(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "PROFILE_SERVICE_UNAVAILABLE", "profile service is unavailable")
		return
	}

	var req dto.ProfileCoreRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid request body")
		return
	}

	birthdate, err := time.Parse("2006-01-02", req.Birthdate)
	if err != nil {
		writeBadRequest(w, "VALIDATION_ERROR", "birthdate must be YYYY-MM-DD")
		return
	}

	completed, err := h.service.UpdateCore(r.Context(), identity.UserID, profilesvc.CoreInput{
		Birthdate:  birthdate,
		Gender:     req.Gender,
		LookingFor: req.LookingFor,
		Occupation: req.Occupation,
		Education:  req.Education,
		HeightCM:   req.HeightCM,
		EyeColor:   req.EyeColor,
		Zodiac:     req.Zodiac,
		Languages:  req.Languages,
		Goals:      req.Goals,
	})
	if err != nil {
		switch {
		case errors.Is(err, profilesvc.ErrAgeRejected):
			httperrors.Write(w, http.StatusBadRequest, httperrors.APIError{
				Code:    "AGE_REJECTED",
				Message: "user must be at least 18 years old",
			})
		case errors.Is(err, profilesvc.ErrValidation):
			httperrors.Write(w, http.StatusBadRequest, httperrors.APIError{
				Code:    "VALIDATION_ERROR",
				Message: "profile core validation failed",
			})
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to save profile core")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, dto.ProfileCoreResponse{ProfileCompleted: completed})
}
