package handlers

import "net/http"

type SettingsHandler struct{}

func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{}
}

func (h *SettingsHandler) Handle(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w, "SETTINGS_NOT_IMPLEMENTED", "settings endpoint is not implemented yet")
}
