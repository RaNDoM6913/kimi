package handlers

import (
	"errors"
	"net/http"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	likessvc "github.com/ivankudzin/tgapp/backend/internal/services/likes"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type LikesHandler struct {
	service *likessvc.Service
}

func NewLikesHandler(service *likessvc.Service) *LikesHandler {
	return &LikesHandler{service: service}
}

func (h *LikesHandler) Handle(w http.ResponseWriter, r *http.Request) {
	h.Incoming(w, r)
}

func (h *LikesHandler) Incoming(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "LIKES_SERVICE_UNAVAILABLE", "likes service is unavailable")
		return
	}

	result, err := h.service.GetIncoming(r.Context(), identity.UserID)
	if err != nil {
		switch {
		case errors.Is(err, likessvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid likes request")
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to load incoming likes")
		}
		return
	}

	preview := make([]dto.LikesIncomingPreviewResponse, 0, len(result.Preview))
	for _, row := range result.Preview {
		preview = append(preview, dto.LikesIncomingPreviewResponse{
			UserID:  row.UserID,
			LikedAt: row.LikedAt,
		})
	}

	profiles := make([]dto.LikesIncomingProfileResponse, 0, len(result.Profiles))
	for _, row := range result.Profiles {
		profiles = append(profiles, mapLikesIncomingProfile(row))
	}

	httperrors.Write(w, http.StatusOK, dto.LikesIncomingResponse{
		Blurred:    result.Blurred,
		TotalCount: result.TotalCount,
		Preview:    preview,
		Profiles:   profiles,
	})
}

func (h *LikesHandler) RevealOne(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "LIKES_SERVICE_UNAVAILABLE", "likes service is unavailable")
		return
	}

	profile, err := h.service.RevealOne(r.Context(), identity.UserID)
	if err != nil {
		switch {
		case errors.Is(err, likessvc.ErrRevealRequired):
			httperrors.Write(w, http.StatusConflict, httperrors.APIError{
				Code:    "REVEAL_CREDIT_REQUIRED",
				Message: "reveal credit is required",
			})
		case errors.Is(err, likessvc.ErrNothingToReveal):
			httperrors.Write(w, http.StatusConflict, httperrors.APIError{
				Code:    "NOTHING_TO_REVEAL",
				Message: "no incoming likes to reveal",
			})
		case errors.Is(err, likessvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid reveal request")
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to reveal like")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, dto.LikesRevealOneResponse{
		Profile: mapLikesIncomingProfile(profile),
	})
}

func mapLikesIncomingProfile(profile likessvc.IncomingProfile) dto.LikesIncomingProfileResponse {
	return dto.LikesIncomingProfileResponse{
		UserID:      profile.UserID,
		DisplayName: profile.DisplayName,
		Age:         profile.Age,
		CityID:      profile.CityID,
		City:        profile.City,
		LikedAt:     profile.LikedAt,
	}
}
