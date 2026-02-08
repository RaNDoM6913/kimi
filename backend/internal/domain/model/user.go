package model

import (
	"time"

	"github.com/ivankudzin/tgapp/backend/internal/domain/enums"
)

type User struct {
	ID         int64      `json:"id"`
	TelegramID int64      `json:"telegram_id"`
	Username   string     `json:"username"`
	Role       enums.Role `json:"role"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}
