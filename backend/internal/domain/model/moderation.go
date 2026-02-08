package model

import (
	"time"

	"github.com/ivankudzin/tgapp/backend/internal/domain/enums"
)

type ModerationItem struct {
	ID              int64                  `json:"id"`
	UserID          int64                  `json:"user_id"`
	Status          enums.ModerationStatus `json:"status"`
	ETABucket       string                 `json:"eta_bucket"`
	ReasonText      string                 `json:"reason_text"`
	RequiredFixStep string                 `json:"required_fix_step"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}
