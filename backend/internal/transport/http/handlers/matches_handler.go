package handlers

import (
	"errors"
	"net/http"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	matchessvc "github.com/ivankudzin/tgapp/backend/internal/services/matches"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type MatchesHandler struct {
	service *matchessvc.Service
}

func NewMatchesHandler(service *matchessvc.Service) *MatchesHandler {
	return &MatchesHandler{service: service}
}

func (h *MatchesHandler) Handle(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "MATCHES_SERVICE_UNAVAILABLE", "matches service is unavailable")
		return
	}

	items, err := h.service.List(r.Context(), identity.UserID, parseIntOrDefault(r.URL.Query().Get("limit"), 100))
	if err != nil {
		switch {
		case errors.Is(err, matchessvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid matches request")
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to load matches")
		}
		return
	}

	responseItems := make([]dto.MatchItemResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, dto.MatchItemResponse{
			ID:           item.ID,
			TargetUserID: item.TargetUserID,
			DisplayName:  item.DisplayName,
			Age:          item.Age,
			CityID:       item.CityID,
			City:         item.City,
			CreatedAt:    item.CreatedAt,
		})
	}

	httperrors.Write(w, http.StatusOK, dto.MatchesResponse{Items: responseItems})
}

func (h *MatchesHandler) Unmatch(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "MATCHES_SERVICE_UNAVAILABLE", "matches service is unavailable")
		return
	}

	var req dto.UnmatchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid request body")
		return
	}

	deleted, err := h.service.Unmatch(r.Context(), identity.UserID, req.TargetID)
	if err != nil {
		switch {
		case errors.Is(err, matchessvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid unmatch request")
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to unmatch")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, struct {
		OK      bool `json:"ok"`
		Deleted bool `json:"deleted"`
	}{
		OK:      true,
		Deleted: deleted,
	})
}

func (h *MatchesHandler) Block(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "MATCHES_SERVICE_UNAVAILABLE", "matches service is unavailable")
		return
	}

	var req dto.BlockRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if err := h.service.Block(r.Context(), identity.UserID, req.TargetID, req.Reason); err != nil {
		switch {
		case errors.Is(err, matchessvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid block request")
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to block user")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, dto.LogoutResponse{OK: true})
}

func (h *MatchesHandler) Report(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "MATCHES_SERVICE_UNAVAILABLE", "matches service is unavailable")
		return
	}

	var req dto.ReportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if err := h.service.Report(r.Context(), identity.UserID, req.TargetID, req.Reason, req.Details); err != nil {
		switch {
		case errors.Is(err, matchessvc.ErrValidation), errors.Is(err, matchessvc.ErrInvalidReportReason):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid report request")
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to report user")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, dto.LogoutResponse{OK: true})
}
