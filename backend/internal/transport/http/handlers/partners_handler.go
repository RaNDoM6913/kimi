package handlers

import "net/http"

type PartnersHandler struct{}

func NewPartnersHandler() *PartnersHandler {
	return &PartnersHandler{}
}

func (h *PartnersHandler) Handle(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w, "PARTNERS_NOT_IMPLEMENTED", "partners endpoint is not implemented yet")
}
