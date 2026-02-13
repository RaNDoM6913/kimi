package handlers

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	analyticsvc "github.com/ivankudzin/tgapp/backend/internal/services/analytics"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	feedsvc "github.com/ivankudzin/tgapp/backend/internal/services/feed"
	mediasvc "github.com/ivankudzin/tgapp/backend/internal/services/media"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type CandidateHandler struct {
	service   *feedsvc.Service
	media     *mediasvc.Service
	telemetry *analyticsvc.Service
	now       func() time.Time
}

func NewCandidateHandler(service *feedsvc.Service, media *mediasvc.Service, telemetry *analyticsvc.Service) *CandidateHandler {
	return &CandidateHandler{
		service:   service,
		media:     media,
		telemetry: telemetry,
		now:       time.Now,
	}
}

func (h *CandidateHandler) Profile(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "FEED_SERVICE_UNAVAILABLE", "feed service is unavailable")
		return
	}

	candidateUserID, ok := candidateUserIDFromRequest(r)
	if !ok {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid candidate user id")
		return
	}

	candidate, err := h.service.GetCandidateProfile(r.Context(), identity.UserID, candidateUserID)
	if err != nil {
		switch {
		case errors.Is(err, feedsvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid candidate user id")
		case errors.Is(err, feedsvc.ErrNotFound):
			httperrors.Write(w, http.StatusNotFound, httperrors.APIError{
				Code:    "NOT_FOUND",
				Message: "candidate not found",
			})
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to load candidate profile")
		}
		return
	}

	h.logCandidateProfileOpen(r.Context(), identity.UserID, candidateUserID)

	httperrors.Write(w, http.StatusOK, dto.CandidateProfileResponse{
		UserID:      candidate.UserID,
		DisplayName: candidate.DisplayName,
		Age:         candidate.Age,
		Zodiac:      candidate.Zodiac,
		CityID:      candidate.CityID,
		City:        candidate.City,
		DistanceKM:  candidate.DistanceKM,
		Bio:         candidate.Bio,
		Occupation:  candidate.Occupation,
		Education:   candidate.Education,
		HeightCM:    candidate.HeightCM,
		EyeColor:    candidate.EyeColor,
		Languages:   append([]string(nil), candidate.Languages...),
		Goals:       append([]string(nil), candidate.Goals...),
		IsTravel:    candidate.IsTravel,
		TravelCity:  candidate.TravelCity,
		Badges: dto.CandidateBadgesResponse{
			IsPlus: candidate.Badges.IsPlus,
		},
	})
}

func (h *CandidateHandler) Photos(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "FEED_SERVICE_UNAVAILABLE", "feed service is unavailable")
		return
	}
	if h.media == nil {
		writeInternal(w, "MEDIA_SERVICE_UNAVAILABLE", "media service is unavailable")
		return
	}

	candidateUserID, ok := candidateUserIDFromRequest(r)
	if !ok {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid candidate user id")
		return
	}

	if _, err := h.service.GetCandidateProfile(r.Context(), identity.UserID, candidateUserID); err != nil {
		switch {
		case errors.Is(err, feedsvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid candidate user id")
		case errors.Is(err, feedsvc.ErrNotFound):
			httperrors.Write(w, http.StatusNotFound, httperrors.APIError{
				Code:    "NOT_FOUND",
				Message: "candidate not found",
			})
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to load candidate profile")
		}
		return
	}

	photos, err := h.media.ListPhotos(r.Context(), candidateUserID)
	if err != nil {
		writeInternal(w, "INTERNAL_ERROR", "failed to load candidate photos")
		return
	}
	if len(photos) > 3 {
		photos = photos[:3]
	}

	responsePhotos := make([]dto.CandidatePhotoResponse, 0, len(photos))
	for _, photo := range photos {
		responsePhotos = append(responsePhotos, dto.CandidatePhotoResponse{
			Slot: photo.Position,
			URL:  photo.URL,
			W:    nil,
			H:    nil,
		})
	}

	h.logCandidatePhotosFetch(r.Context(), identity.UserID, candidateUserID, len(responsePhotos))

	httperrors.Write(w, http.StatusOK, dto.CandidatePhotosResponse{
		UserID: candidateUserID,
		Photos: responsePhotos,
	})
}

func (h *CandidateHandler) logCandidateProfileOpen(ctx context.Context, viewerUserID, candidateUserID int64) {
	if h.telemetry == nil || viewerUserID <= 0 || candidateUserID <= 0 {
		return
	}

	uid := viewerUserID
	_ = h.telemetry.IngestBatch(ctx, &uid, []analyticsvc.BatchEvent{
		{
			Name: "candidate_profile_open",
			TS:   h.now().UTC().UnixMilli(),
			Props: map[string]any{
				"candidate_user_id": candidateUserID,
			},
		},
	})
}

func (h *CandidateHandler) logCandidatePhotosFetch(ctx context.Context, viewerUserID, candidateUserID int64, photosCount int) {
	if h.telemetry == nil || viewerUserID <= 0 || candidateUserID <= 0 {
		return
	}

	uid := viewerUserID
	_ = h.telemetry.IngestBatch(ctx, &uid, []analyticsvc.BatchEvent{
		{
			Name: "candidate_photos_fetch",
			TS:   h.now().UTC().UnixMilli(),
			Props: map[string]any{
				"candidate_user_id": candidateUserID,
				"photos_count":      photosCount,
			},
		},
	})
}

func candidateUserIDFromRequest(r *http.Request) (int64, bool) {
	if r == nil {
		return 0, false
	}
	raw := strings.TrimSpace(chi.URLParam(r, "user_id"))
	if raw == "" {
		return 0, false
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}
