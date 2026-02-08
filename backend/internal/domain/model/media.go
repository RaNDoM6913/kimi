package model

import (
	"time"

	"github.com/ivankudzin/tgapp/backend/internal/domain/enums"
)

type Media struct {
	ID         int64           `json:"id"`
	UserID     int64           `json:"user_id"`
	Kind       enums.MediaKind `json:"kind"`
	S3Key      string          `json:"s3_key"`
	SignedURL  string          `json:"signed_url"`
	Position   int             `json:"position"`
	IsApproved bool            `json:"is_approved"`
	CreatedAt  time.Time       `json:"created_at"`
}
