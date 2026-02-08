package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	feedsvc "github.com/ivankudzin/tgapp/backend/internal/services/feed"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type FeedHandler struct {
	service *feedsvc.Service
}

func NewFeedHandler(service *feedsvc.Service) *FeedHandler {
	return &FeedHandler{service: service}
}

func (h *FeedHandler) Handle(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "FEED_SERVICE_UNAVAILABLE", "feed service is unavailable")
		return
	}

	cursor := strings.TrimSpace(r.URL.Query().Get("cursor"))
	limit := parseIntOrDefault(r.URL.Query().Get("limit"), 20)
	if limit > 50 {
		limit = 50
	}
	if limit <= 0 {
		limit = 20
	}

	result, err := h.service.Get(r.Context(), identity.UserID, cursor, limit)
	if err != nil {
		switch {
		case errors.Is(err, feedsvc.ErrInvalidCursor):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid cursor")
		case errors.Is(err, feedsvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid feed request")
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to load feed")
		}
		return
	}

	items := make([]dto.FeedItemResponse, 0, len(result.Items))
	for _, item := range result.Items {
		responseItem := dto.FeedItemResponse{
			IsAd: item.IsAd,
		}
		if item.IsAd && item.Ad != nil {
			responseItem.Ad = &dto.FeedAdCardResponse{
				ID:       item.Ad.ID,
				Kind:     item.Ad.Kind,
				Title:    item.Ad.Title,
				AssetURL: item.Ad.AssetURL,
				ClickURL: item.Ad.ClickURL,
			}
		} else {
			responseItem.UserID = item.UserID
			responseItem.DisplayName = item.DisplayName
			responseItem.Age = item.Age
			responseItem.CityID = item.CityID
			responseItem.City = item.City
			responseItem.DistanceKM = item.DistanceKM
		}
		items = append(items, responseItem)
	}

	httperrors.Write(w, http.StatusOK, dto.FeedResponse{
		Items:      items,
		NextCursor: result.NextCursor,
	})
}

func parseIntOrDefault(raw string, fallback int) int {
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return fallback
	}
	return value
}
