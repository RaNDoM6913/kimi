package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const (
	CounterTooFast1hKey         = "cnt:too_fast:1h"
	CounterCooldownApplied1hKey = "cnt:cooldown_applied:1h"
	CounterShadowEnabled24hKey  = "cnt:shadow_enabled:24h"

	OffendersUser24hKey   = "zset:offenders:user:24h"
	OffendersDevice24hKey = "zset:offenders:device:24h"
)

type AntiAbuseDashboardRepo struct {
	client *goredis.Client
}

type AntiAbuseSummary struct {
	TooFast1h         int64 `json:"too_fast_1h"`
	CooldownApplied1h int64 `json:"cooldown_applied_1h"`
	ShadowEnabled24h  int64 `json:"shadow_enabled_24h"`
}

type OffenderItem struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

func NewAntiAbuseDashboardRepo(client *goredis.Client) *AntiAbuseDashboardRepo {
	return &AntiAbuseDashboardRepo{client: client}
}

func (r *AntiAbuseDashboardRepo) ObserveEvent(ctx context.Context, userID *int64, name string, props map[string]any) error {
	if r.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	eventName := strings.ToLower(strings.TrimSpace(name))
	if !strings.HasPrefix(eventName, "antiabuse_") {
		return nil
	}

	if userID != nil && *userID > 0 {
		if err := r.incrementOffender(ctx, OffendersUser24hKey, strconv.FormatInt(*userID, 10), 24*time.Hour); err != nil {
			return err
		}
	}

	if deviceID := normalizeDeviceID(props); deviceID != "" {
		if err := r.incrementOffender(ctx, OffendersDevice24hKey, deviceID, 24*time.Hour); err != nil {
			return err
		}
	}

	switch eventName {
	case "antiabuse_too_fast":
		return r.incrementCounter(ctx, CounterTooFast1hKey, time.Hour)
	case "antiabuse_cooldown_applied":
		return r.incrementCounter(ctx, CounterCooldownApplied1hKey, time.Hour)
	case "antiabuse_shadow_enabled":
		return r.incrementCounter(ctx, CounterShadowEnabled24hKey, 24*time.Hour)
	default:
		return nil
	}
}

func (r *AntiAbuseDashboardRepo) Summary(ctx context.Context) (AntiAbuseSummary, error) {
	if r.client == nil {
		return AntiAbuseSummary{}, fmt.Errorf("redis client is nil")
	}

	tooFast, err := r.counterValue(ctx, CounterTooFast1hKey)
	if err != nil {
		return AntiAbuseSummary{}, err
	}
	cooldownApplied, err := r.counterValue(ctx, CounterCooldownApplied1hKey)
	if err != nil {
		return AntiAbuseSummary{}, err
	}
	shadowEnabled, err := r.counterValue(ctx, CounterShadowEnabled24hKey)
	if err != nil {
		return AntiAbuseSummary{}, err
	}

	return AntiAbuseSummary{
		TooFast1h:         tooFast,
		CooldownApplied1h: cooldownApplied,
		ShadowEnabled24h:  shadowEnabled,
	}, nil
}

func (r *AntiAbuseDashboardRepo) Top(ctx context.Context, kind string, limit int64) ([]OffenderItem, error) {
	if r.client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	key, ok := offendersKeyByKind(kind)
	if !ok {
		return nil, fmt.Errorf("invalid offenders kind")
	}

	pairs, err := r.client.ZRevRangeWithScores(ctx, key, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("read top offenders: %w", err)
	}

	items := make([]OffenderItem, 0, len(pairs))
	for _, pair := range pairs {
		member, ok := pair.Member.(string)
		if !ok {
			member = fmt.Sprint(pair.Member)
		}
		items = append(items, OffenderItem{
			ID:    member,
			Score: pair.Score,
		})
	}
	return items, nil
}

func (r *AntiAbuseDashboardRepo) incrementCounter(ctx context.Context, key string, ttl time.Duration) error {
	pipe := r.client.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("increment counter %s: %w", key, err)
	}
	return nil
}

func (r *AntiAbuseDashboardRepo) incrementOffender(ctx context.Context, key, member string, ttl time.Duration) error {
	member = strings.TrimSpace(member)
	if member == "" {
		return nil
	}
	pipe := r.client.Pipeline()
	pipe.ZIncrBy(ctx, key, 1, member)
	pipe.Expire(ctx, key, ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("increment offenders zset %s: %w", key, err)
	}
	return nil
}

func (r *AntiAbuseDashboardRepo) counterValue(ctx context.Context, key string) (int64, error) {
	value, err := r.client.Get(ctx, key).Int64()
	if err == goredis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read counter %s: %w", key, err)
	}
	return value, nil
}

func offendersKeyByKind(kind string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "user":
		return OffendersUser24hKey, true
	case "device":
		return OffendersDevice24hKey, true
	default:
		return "", false
	}
}

func normalizeDeviceID(props map[string]any) string {
	if len(props) == 0 {
		return ""
	}
	raw, ok := props["device_id"]
	if !ok {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
