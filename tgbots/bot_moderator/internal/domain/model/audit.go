package model

import (
	"encoding/json"
	"time"

	"bot_moderator/internal/domain/enums"
)

type Audit struct {
	ID        string
	ActorTGID int64
	Action    enums.AuditAction
	Payload   json.RawMessage
	CreatedAt time.Time
}
