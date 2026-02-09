package rate

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	likes1SecWindow   = time.Second
	likes10SecWindow  = 10 * time.Second
	likesMinuteWindow = time.Minute
)

const likeRateCheckScript = `
local function hit(key, limit, window_ms)
	if limit <= 0 then
		return 0, 0, false
	end

	local c = redis.call("INCR", key)
	if c == 1 then
		redis.call("PEXPIRE", key, window_ms)
	end

	local ttl = redis.call("PTTL", key)
	if ttl < 0 then
		redis.call("PEXPIRE", key, window_ms)
		ttl = window_ms
	end

	if c > limit then
		local retry = math.floor((ttl + 999) / 1000)
		if retry < 1 then
			retry = 1
		end
		return c, retry, true
	end

	return c, 0, false
end

local _, retry, blocked = hit(KEYS[1], tonumber(ARGV[1]), tonumber(ARGV[2]))
if blocked then
	return {0, retry, "user_1s"}
end

_, retry, blocked = hit(KEYS[2], tonumber(ARGV[3]), tonumber(ARGV[4]))
if blocked then
	return {0, retry, "user_10s"}
end

_, retry, blocked = hit(KEYS[3], tonumber(ARGV[5]), tonumber(ARGV[6]))
if blocked then
	return {0, retry, "user_1m"}
end

_, retry, blocked = hit(KEYS[4], tonumber(ARGV[7]), tonumber(ARGV[8]))
if blocked then
	return {0, retry, "sid_1m"}
end

return {1, 0, ""}
`

type Store interface {
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)
	WindowState(ctx context.Context, key string) (int64, time.Duration, error)
}

type Limiter struct {
	store        Store
	perSec       int
	per10Sec     int
	perMinute    int
	sidPerMinute int
}

func NewLimiter(store Store, perSec, per10Sec, perMinute int) *Limiter {
	if perSec < 0 {
		perSec = 0
	}
	if per10Sec < 0 {
		per10Sec = 0
	}
	if perMinute < 0 {
		perMinute = 0
	}

	return &Limiter{
		store:        store,
		perSec:       perSec,
		per10Sec:     per10Sec,
		perMinute:    perMinute,
		sidPerMinute: perMinute,
	}
}

// CheckLikeRate checks LIKE+SUPERLIKE anti-abuse windows.
// Returns allowed=false with retry_after and reason when blocked.
func (l *Limiter) CheckLikeRate(ctx context.Context, userID int64, sid, ip string) (bool, int, string) {
	if userID <= 0 || l.store == nil {
		return false, 1, "invalid_input"
	}
	_ = ip

	sidLimit := l.sidPerMinute
	normalizedSID := strings.TrimSpace(sid)
	if normalizedSID == "" {
		sidLimit = 0
		normalizedSID = "_"
	}

	result, err := l.store.Eval(
		ctx,
		likeRateCheckScript,
		[]string{
			user1SecKey(userID),
			user10SecKey(userID),
			userMinuteKey(userID),
			sidMinuteKey(normalizedSID),
		},
		l.perSec,
		int64(likes1SecWindow/time.Millisecond),
		l.per10Sec,
		int64(likes10SecWindow/time.Millisecond),
		l.perMinute,
		int64(likesMinuteWindow/time.Millisecond),
		sidLimit,
		int64(likesMinuteWindow/time.Millisecond),
	)
	if err != nil {
		// Fail-open on Redis errors to avoid hard outage.
		return true, 0, ""
	}

	allowed, retryAfter, reason, ok := parseCheckResult(result)
	if !ok {
		return true, 0, ""
	}
	return allowed, retryAfter, reason
}

// AllowLike is a backward-compatible wrapper for old callers.
func (l *Limiter) AllowLike(ctx context.Context, userID int64) (int64, bool, error) {
	if userID <= 0 {
		return 0, false, fmt.Errorf("invalid user id")
	}
	if l.store == nil {
		return 0, false, fmt.Errorf("rate limiter store is nil")
	}

	allowed, retryAfter, _ := l.CheckLikeRate(ctx, userID, "", "")
	if !allowed {
		return int64(retryAfter), false, nil
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

	if l.perSec > 0 {
		count, ttl, err := l.store.WindowState(ctx, user1SecKey(userID))
		if err != nil {
			return 0, err
		}
		if count > int64(l.perSec) {
			retryAfterSec = maxInt64(retryAfterSec, ceilSeconds(ttl))
		}
	}

	if l.per10Sec > 0 {
		count, ttl, err := l.store.WindowState(ctx, user10SecKey(userID))
		if err != nil {
			return 0, err
		}
		if count > int64(l.per10Sec) {
			retryAfterSec = maxInt64(retryAfterSec, ceilSeconds(ttl))
		}
	}

	if l.perMinute > 0 {
		count, ttl, err := l.store.WindowState(ctx, userMinuteKey(userID))
		if err != nil {
			return 0, err
		}
		if count > int64(l.perMinute) {
			retryAfterSec = maxInt64(retryAfterSec, ceilSeconds(ttl))
		}
	}

	return retryAfterSec, nil
}

func user1SecKey(userID int64) string {
	return "rl:like:user:" + strconv.FormatInt(userID, 10) + ":1s"
}

func user10SecKey(userID int64) string {
	return "rl:like:user:" + strconv.FormatInt(userID, 10) + ":10s"
}

func userMinuteKey(userID int64) string {
	return "rl:like:user:" + strconv.FormatInt(userID, 10) + ":1m"
}

func sidMinuteKey(sid string) string {
	return "rl:like:sid:" + sid + ":1m"
}

func parseCheckResult(raw interface{}) (allowed bool, retryAfter int, reason string, ok bool) {
	arr, ok := raw.([]interface{})
	if !ok || len(arr) < 3 {
		return false, 0, "", false
	}

	allowedInt, ok := asInt64(arr[0])
	if !ok {
		return false, 0, "", false
	}
	retryInt, ok := asInt64(arr[1])
	if !ok {
		return false, 0, "", false
	}

	reason, ok = arr[2].(string)
	if !ok {
		reason = ""
	}

	if retryInt < 0 {
		retryInt = 0
	}
	return allowedInt == 1, int(retryInt), reason, true
}

func asInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case string:
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
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
