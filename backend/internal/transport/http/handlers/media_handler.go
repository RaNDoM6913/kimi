package handlers

import (
	"errors"
	"fmt"
	"net/http"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	mediasvc "github.com/ivankudzin/tgapp/backend/internal/services/media"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

const maxPhotoUploadSize = 20 << 20 // 20 MiB

type MediaHandler struct {
	service *mediasvc.Service
}

func NewMediaHandler(service *mediasvc.Service) *MediaHandler {
	return &MediaHandler{service: service}
}

func (h *MediaHandler) Handle(w http.ResponseWriter, r *http.Request) {
	h.PhotoUpload(w, r)
}

func (h *MediaHandler) PhotoUpload(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "MEDIA_SERVICE_UNAVAILABLE", "media service is unavailable")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPhotoUploadSize)
	if err := r.ParseMultipartForm(maxPhotoUploadSize); err != nil {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeBadRequest(w, "VALIDATION_ERROR", "file is required")
		return
	}
	defer file.Close()

	if header == nil || header.Size <= 0 {
		writeBadRequest(w, "VALIDATION_ERROR", "file is empty")
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	photo, err := h.service.UploadPhoto(r.Context(), identity.UserID, header.Filename, contentType, file, header.Size)
	if err != nil {
		handleMediaError(w, err)
		return
	}

	httperrors.Write(w, http.StatusOK, dto.MediaPhotoResponse{
		ID:       photo.ID,
		Position: photo.Position,
		URL:      photo.URL,
	})
}

func (h *MediaHandler) PhotosList(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.service == nil {
		writeInternal(w, "MEDIA_SERVICE_UNAVAILABLE", "media service is unavailable")
		return
	}

	photos, err := h.service.ListPhotos(r.Context(), identity.UserID)
	if err != nil {
		handleMediaError(w, err)
		return
	}

	items := make([]dto.MediaPhotoResponse, 0, len(photos))
	for _, photo := range photos {
		items = append(items, dto.MediaPhotoResponse{
			ID:       photo.ID,
			Position: photo.Position,
			URL:      photo.URL,
		})
	}

	httperrors.Write(w, http.StatusOK, dto.MediaPhotosListResponse{Items: items})
}

func handleMediaError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, mediasvc.ErrValidation):
		writeBadRequest(w, "VALIDATION_ERROR", "invalid media request")
	case errors.Is(err, mediasvc.ErrPhotoLimitReached):
		httperrors.Write(w, http.StatusConflict, httperrors.APIError{
			Code:    "PHOTO_LIMIT_REACHED",
			Message: fmt.Sprintf("maximum %d active photos allowed", mediasvc.MaxActivePhotos()),
		})
	default:
		writeInternal(w, "INTERNAL_ERROR", "media operation failed")
	}
}
