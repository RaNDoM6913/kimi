package model

import (
	"encoding/json"
	"time"
)

type ExportQueueItem struct {
	ID        string
	Kind      string
	Payload   json.RawMessage
	Status    string
	CreatedAt time.Time
}
