package model

import "time"

type Entitlement struct {
	UserID                int64      `json:"user_id"`
	PlusActive            bool       `json:"plus_active"`
	PlusExpiresAt         *time.Time `json:"plus_expires_at"`
	BoostActiveUntil      *time.Time `json:"boost_active_until"`
	SuperLikeCredits      int        `json:"superlike_credits"`
	RevealCredits         int        `json:"reveal_credits"`
	MessageWoMatchCredits int        `json:"message_wo_match_credits"`
	LikeTokens            int        `json:"like_tokens"`
	IncognitoUntil        *time.Time `json:"incognito_until"`
}
