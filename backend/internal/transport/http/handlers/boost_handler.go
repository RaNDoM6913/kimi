package handlers

import "net/http"

type BoostHandler struct{}

func NewBoostHandler() *BoostHandler {
	return &BoostHandler{}
}

func (h *BoostHandler) Handle(w http.ResponseWriter, _ *http.Request) {
	writeNotImplemented(w, "BOOST_NOT_IMPLEMENTED", "boost endpoint is not implemented yet")
}
