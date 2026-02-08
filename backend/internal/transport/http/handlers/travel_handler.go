package handlers

import "net/http"

type TravelHandler struct{}

func NewTravelHandler() *TravelHandler {
	return &TravelHandler{}
}

func (h *TravelHandler) Handle(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w, "TRAVEL_NOT_IMPLEMENTED", "travel endpoint is not implemented yet")
}
