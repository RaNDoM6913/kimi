package model

import (
	"time"
)

type BotUser struct {
	TgID       int64
	Username   string
	FirstName  string
	LastName   string
	LastSeenAt time.Time
}
