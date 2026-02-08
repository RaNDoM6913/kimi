package model

type FeedItem struct {
	CandidateUserID int64   `json:"candidate_user_id"`
	Score           float64 `json:"score"`
	GoalsPriority   bool    `json:"goals_priority"`
	DistanceKM      float64 `json:"distance_km"`
	Age             int     `json:"age"`
	IsAd            bool    `json:"is_ad"`
}
