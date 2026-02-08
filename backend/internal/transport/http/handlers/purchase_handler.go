package handlers

import (
	"errors"
	"net/http"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	entsvc "github.com/ivankudzin/tgapp/backend/internal/services/entitlements"
	paymentsvc "github.com/ivankudzin/tgapp/backend/internal/services/payments"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/dto"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
)

type PurchaseHandler struct {
	payments     *paymentsvc.Service
	entitlements *entsvc.Service
}

func NewPurchaseHandler(payments *paymentsvc.Service, entitlements *entsvc.Service) *PurchaseHandler {
	return &PurchaseHandler{
		payments:     payments,
		entitlements: entitlements,
	}
}

func (h *PurchaseHandler) Handle(w http.ResponseWriter, r *http.Request) {
	h.Create(w, r)
}

func (h *PurchaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.payments == nil {
		writeInternal(w, "PAYMENTS_SERVICE_UNAVAILABLE", "payments service is unavailable")
		return
	}

	var req dto.PurchaseCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid request body")
		return
	}

	result, err := h.payments.Create(r.Context(), identity.UserID, paymentsvc.CreateInput{
		SKU:      req.SKU,
		Provider: req.Provider,
	})
	if err != nil {
		switch {
		case errors.Is(err, paymentsvc.ErrValidation), errors.Is(err, paymentsvc.ErrUnsupportedSKU):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid purchase create payload")
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to create purchase")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, dto.PurchaseCreateResponse{
		PurchaseID: result.PurchaseID,
		SKU:        result.SKU,
		Provider:   result.Provider,
		Status:     result.Status,
	})
}

func (h *PurchaseHandler) Webhook(w http.ResponseWriter, r *http.Request) {
	if h.payments == nil {
		writeInternal(w, "PAYMENTS_SERVICE_UNAVAILABLE", "payments service is unavailable")
		return
	}

	var req dto.PurchaseWebhookRequest
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "VALIDATION_ERROR", "invalid request body")
		return
	}

	result, err := h.payments.ConfirmWebhook(r.Context(), paymentsvc.WebhookInput{
		PurchaseID:   req.PurchaseID,
		Provider:     req.Provider,
		ProviderTxID: req.ProviderTxID,
		Status:       req.Status,
		Payload:      req.Payload,
	})
	if err != nil {
		switch {
		case errors.Is(err, paymentsvc.ErrValidation), errors.Is(err, paymentsvc.ErrUnsupportedSKU):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid webhook payload")
		case errors.Is(err, paymentsvc.ErrPurchaseNotFound):
			httperrors.Write(w, http.StatusNotFound, httperrors.APIError{
				Code:    "PURCHASE_NOT_FOUND",
				Message: "purchase not found",
			})
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to process webhook")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, dto.PurchaseWebhookResponse{
		OK:         true,
		PurchaseID: result.PurchaseID,
		UserID:     result.UserID,
		SKU:        result.SKU,
		Status:     result.Status,
		Idempotent: result.AlreadyProcessed,
	})
}

func (h *PurchaseHandler) Entitlements(w http.ResponseWriter, r *http.Request) {
	identity, ok := authsvc.IdentityFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, "UNAUTHORIZED", "authentication required")
		return
	}
	if h.entitlements == nil {
		writeInternal(w, "ENTITLEMENTS_SERVICE_UNAVAILABLE", "entitlements service is unavailable")
		return
	}

	snapshot, err := h.entitlements.Get(r.Context(), identity.UserID)
	if err != nil {
		switch {
		case errors.Is(err, entsvc.ErrValidation):
			writeBadRequest(w, "VALIDATION_ERROR", "invalid entitlements request")
		default:
			writeInternal(w, "INTERNAL_ERROR", "failed to load entitlements")
		}
		return
	}

	httperrors.Write(w, http.StatusOK, dto.EntitlementsResponse{
		IsPlus:                snapshot.IsPlus,
		PlusUntil:             snapshot.PlusUntil,
		BoostUntil:            snapshot.BoostUntil,
		SuperLikeCredits:      snapshot.SuperLikeCredits,
		RevealCredits:         snapshot.RevealCredits,
		MessageWoMatchCredits: snapshot.MessageWoMatchCredits,
		LikeTokens:            snapshot.LikeTokens,
		IncognitoUntil:        snapshot.IncognitoUntil,
	})
}
