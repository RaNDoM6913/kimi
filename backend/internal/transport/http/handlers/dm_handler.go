package handlers

import "net/http"

type DMHandler struct{}

func NewDMHandler() *DMHandler {
	return &DMHandler{}
}

func (h *DMHandler) Handle(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w, "DM_NOT_IMPLEMENTED", "dm endpoint is not implemented yet")
}
