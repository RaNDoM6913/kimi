package model

import (
	"encoding/json"
	"time"
)

type LookupUser struct {
	UserID           int64
	TGID             int64
	Username         string
	CityID           string
	Birthdate        *time.Time
	Age              int
	Gender           string
	LookingFor       string
	Goals            []string
	Languages        []string
	Occupation       string
	Education        string
	ModerationStatus string
	Approved         bool
	PhotoKeys        []string
	CircleKey        string
	PhotoURLs        []string
	CircleURL        string
	PlusExpiresAt    *time.Time
	BoostUntil       *time.Time
	SuperlikeCredits int
	RevealCredits    int
	LikeTokens       int
	IsBanned         bool
	BanReason        string
}

type BotLookupAction struct {
	ActorTGID   int64
	ActorRole   string
	Query       string
	FoundUserID *int64
	Action      string
	Payload     json.RawMessage
	CreatedAt   time.Time
}
