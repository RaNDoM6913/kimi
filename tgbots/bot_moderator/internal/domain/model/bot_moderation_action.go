package model

import "time"

type BotModerationAction struct {
	ActorTGID        int64
	ActorRole        string
	TargetUserID     int64
	ModerationItemID int64
	Decision         string
	ReasonCode       string
	DurationSec      *int
	CreatedAt        time.Time
}
