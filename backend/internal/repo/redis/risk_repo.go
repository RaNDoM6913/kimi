package redis

import (
	"context"
	"fmt"
	"strconv"

	goredis "github.com/redis/go-redis/v9"
)

type RiskRepo struct {
	client *goredis.Client
}

type RiskStateRecord struct {
	RiskScore       int
	CooldownUntil   int64
	LastViolationAt int64
	Exists          bool
}

func NewRiskRepo(client *goredis.Client) *RiskRepo {
	return &RiskRepo{client: client}
}

func (r *RiskRepo) Get(ctx context.Context, userID int64) (RiskStateRecord, error) {
	if r.client == nil {
		return RiskStateRecord{}, fmt.Errorf("redis client is nil")
	}
	if userID <= 0 {
		return RiskStateRecord{}, fmt.Errorf("invalid user id")
	}

	values, err := r.client.HGetAll(ctx, riskKey(userID)).Result()
	if err != nil {
		return RiskStateRecord{}, fmt.Errorf("get risk state: %w", err)
	}
	if len(values) == 0 {
		return RiskStateRecord{}, nil
	}

	riskScore, err := parseInt(values["risk_score"])
	if err != nil {
		return RiskStateRecord{}, fmt.Errorf("parse risk_score: %w", err)
	}
	cooldownUntil, err := parseInt64(values["cooldown_until"])
	if err != nil {
		return RiskStateRecord{}, fmt.Errorf("parse cooldown_until: %w", err)
	}
	lastViolationAt, err := parseInt64(values["last_violation_at"])
	if err != nil {
		return RiskStateRecord{}, fmt.Errorf("parse last_violation_at: %w", err)
	}

	if riskScore < 0 {
		riskScore = 0
	}
	if cooldownUntil < 0 {
		cooldownUntil = 0
	}
	if lastViolationAt < 0 {
		lastViolationAt = 0
	}

	return RiskStateRecord{
		RiskScore:       riskScore,
		CooldownUntil:   cooldownUntil,
		LastViolationAt: lastViolationAt,
		Exists:          true,
	}, nil
}

func (r *RiskRepo) EvalForUser(ctx context.Context, userID int64, script string, args ...interface{}) (interface{}, error) {
	if r.client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	if userID <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}
	if script == "" {
		return nil, fmt.Errorf("script is required")
	}

	result, err := r.client.Eval(ctx, script, []string{riskKey(userID)}, args...).Result()
	if err != nil {
		return nil, fmt.Errorf("eval risk script: %w", err)
	}
	return result, nil
}

func riskKey(userID int64) string {
	return "risk:user:" + strconv.FormatInt(userID, 10)
}

func parseInt(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	return v, nil
}

func parseInt64(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}
