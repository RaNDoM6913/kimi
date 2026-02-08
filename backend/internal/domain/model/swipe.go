package model

import (
	"time"

	"github.com/ivankudzin/tgapp/backend/internal/domain/enums"
)

type Swipe struct {
	ID           int64             `json:"id"`
	ActorUserID  int64             `json:"actor_user_id"`
	TargetUserID int64             `json:"target_user_id"`
	Action       enums.SwipeAction `json:"action"`
	CreatedAt    time.Time         `json:"created_at"`
}
