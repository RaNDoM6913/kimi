package rate

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

const (
	likesMinuteWindow = time.Minute
	likes10SecWindow  = 10 * time.Second
)

type WindowStore interface {
	IncrementWindow(ctx context.Context, key string, window time.Duration) (int64, time.Duration, error)
	WindowState(ctx context.Context, key string) (int64, time.Duration, error)
}

type Limiter struct {
	store     WindowStore
	perMinute int
	per10Sec  int
}

func NewLimiter(store WindowStore, perMinute, per10Sec int) *Limiter {
	if perMinute < 0 {
		perMinute = 0
	}
	if per10Sec < 0 {
		per10Sec = 0
	}

	return &Limiter{
		store:     store,
		perMinute: perMinute,
		per10Sec:  per10Sec,
	}
}

func (l *Limiter) AllowLike(ctx context.Context, userID int64) (int64, bool, error) {
	if userID <= 0 {
		return 0, false, fmt.Errorf("invalid user id")
	}
	if l.store == nil {
		return 0, false, fmt.Errorf("rate limiter store is nil")
	}

	retryAfterSec := int64(0)

	if l.perMinute > 0 {
		count, ttl, err := l.store.IncrementWindow(ctx, minuteKey(userID), likesMinuteWindow)
		if err != nil {
			return 0, false, err
		}
		if count > int64(l.perMinute) {
			retryAfterSec = maxInt64(retryAfterSec, ceilSeconds(ttl))
		}
	}

	if l.per10Sec > 0 {
		count, ttl, err := l.store.IncrementWindow(ctx, tenSecKey(userID), likes10SecWindow)
		if err != nil {
			return 0, false, err
		}
		if count > int64(l.per10Sec) {
			retryAfterSec = maxInt64(retryAfterSec, ceilSeconds(ttl))
		}
	}

	if retryAfterSec > 0 {
		return retryAfterSec, false, nil
	}

	return 0, true, nil
}

func (l *Limiter) RetryAfterLike(ctx context.Context, userID int64) (int64, error) {
	if userID <= 0 {
		return 0, fmt.Errorf("invalid user id")
	}
	if l.store == nil {
		return 0, fmt.Errorf("rate limiter store is nil")
	}

	retryAfterSec := int64(0)

	if l.perMinute > 0 {
		count, ttl, err := l.store.WindowState(ctx, minuteKey(userID))
		if err != nil {
			return 0, err
		}
		if count >= int64(l.perMinute) {
			retryAfterSec = maxInt64(retryAfterSec, ceilSeconds(ttl))
		}
	}

	if l.per10Sec > 0 {
		count, ttl, err := l.store.WindowState(ctx, tenSecKey(userID))
		if err != nil {
			return 0, err
		}
		if count >= int64(l.per10Sec) {
			retryAfterSec = maxInt64(retryAfterSec, ceilSeconds(ttl))
		}
	}

	return retryAfterSec, nil
}

func minuteKey(userID int64) string {
	return "rate:likes:min:" + strconv.FormatInt(userID, 10)
}

func tenSecKey(userID int64) string {
	return "rate:likes:10s:" + strconv.FormatInt(userID, 10)
}

func ceilSeconds(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	sec := int64(d / time.Second)
	if d%time.Second != 0 {
		sec++
	}
	if sec <= 0 {
		sec = 1
	}
	return sec
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
