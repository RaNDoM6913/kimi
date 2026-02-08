package model

import (
	"time"

	"github.com/ivankudzin/tgapp/backend/internal/domain/enums"
)

type Purchase struct {
	ID          int64             `json:"id"`
	UserID      int64             `json:"user_id"`
	SKU         enums.PurchaseSKU `json:"sku"`
	Provider    string            `json:"provider"`
	ReceiptID   string            `json:"receipt_id"`
	Status      string            `json:"status"`
	PurchasedAt time.Time         `json:"purchased_at"`
}
