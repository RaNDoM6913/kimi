package auth

import (
	"errors"
	"time"
)

var (
	ErrInvalidInput    = errors.New("invalid input")
	ErrUnauthorized    = errors.New("unauthorized")
	ErrSessionNotFound = errors.New("session not found")
	ErrRefreshNotFound = errors.New("refresh token not found")
)

type SessionRecord struct {
	SID       string
	UserID    int64
	Role      string
	ExpiresAt time.Time
}

type AccessClaims struct {
	UserID    int64
	SID       string
	Role      string
	ExpiresAt time.Time
}

type Me struct {
	ID   int64
	Role string
}

type AuthResult struct {
	AccessToken   string
	RefreshToken  string
	AccessExpires time.Time
	Me            Me
}
