package model

import "time"

type ModerationStatus string

const (
	ModerationStatusPending  ModerationStatus = "PENDING"
	ModerationStatusApproved ModerationStatus = "APPROVED"
	ModerationStatusRejected ModerationStatus = "REJECTED"
)

type ModerationItem struct {
	ID            int64
	UserID        int64
	Status        ModerationStatus
	ETABucket     string
	RejectNote    string
	CreatedAt     time.Time
	LockedByTGID  *int64
	LockedAt      *time.Time
	LockedUntil   *time.Time
	UpdatedAt     time.Time
	TargetType    string
	TargetID      *int64
	ModeratorTGID *int64
}

type ModerationProfile struct {
	UserID      int64
	TGID        int64
	Username    string
	DisplayName string
	CityID      string
	Birthdate   *time.Time
	Age         int
	Gender      string
	LookingFor  string
	Goals       []string
	Languages   []string
	Occupation  string
	Education   string
}

type ModerationQueueItem struct {
	ModerationItemID int64
	TargetUserID     int64
	ETABucket        string
	CreatedAt        time.Time
	LockedAt         time.Time
	Profile          ModerationProfile
	PhotoURLs        []string
	CircleURL        string
}
