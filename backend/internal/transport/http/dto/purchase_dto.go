package dto

import "time"

type PurchaseCreateRequest struct {
	SKU      string `json:"sku"`
	Provider string `json:"provider"`
}

type PurchaseCreateResponse struct {
	PurchaseID int64  `json:"purchase_id"`
	SKU        string `json:"sku"`
	Provider   string `json:"provider"`
	Status     string `json:"status"`
}

type PurchaseWebhookRequest struct {
	PurchaseID   int64                  `json:"purchase_id"`
	Provider     string                 `json:"provider"`
	ProviderTxID string                 `json:"provider_tx_id"`
	Status       string                 `json:"status,omitempty"`
	Payload      map[string]interface{} `json:"payload,omitempty"`
}

type PurchaseWebhookResponse struct {
	OK         bool   `json:"ok"`
	PurchaseID int64  `json:"purchase_id"`
	UserID     int64  `json:"user_id"`
	SKU        string `json:"sku"`
	Status     string `json:"status"`
	Idempotent bool   `json:"idempotent"`
}

type EntitlementsResponse struct {
	IsPlus                bool       `json:"is_plus"`
	PlusUntil             *time.Time `json:"plus_until,omitempty"`
	BoostUntil            *time.Time `json:"boost_until,omitempty"`
	SuperLikeCredits      int        `json:"superlike_credits"`
	RevealCredits         int        `json:"reveal_credits"`
	MessageWoMatchCredits int        `json:"message_wo_match_credits"`
	LikeTokens            int        `json:"like_tokens"`
	IncognitoUntil        *time.Time `json:"incognito_until,omitempty"`
}

// Backward aliases for old skeleton handlers.
type PurchaseRequest = PurchaseCreateRequest

type PurchaseResponse = PurchaseCreateResponse
