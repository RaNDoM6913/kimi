package handlers

import (
	"net/http"

	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

func writeNotImplemented(w http.ResponseWriter, code, message string) {
	httperrors.Write(w, http.StatusNotImplemented, httperrors.APIError{
		Code:    code,
		Message: message,
	})
}
