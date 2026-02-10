package antiabuse

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	redrepo "github.com/ivankudzin/tgapp/backend/internal/repo/redis"
	analyticsvc "github.com/ivankudzin/tgapp/backend/internal/services/analytics"
)

const (
	defaultRiskDecayHours = 6
)

const violationScript = `
local key = KEYS[1]
local weight = tonumber(ARGV[1])
local now = tonumber(ARGV[2])
local decay_sec = tonumber(ARGV[3])
local steps_count = tonumber(ARGV[4])
local forced_step = tonumber(ARGV[5 + steps_count])

if weight == nil or weight < 1 then
	weight = 1
end
if now == nil or now < 0 then
	now = 0
end
if decay_sec == nil or decay_sec < 0 then
	decay_sec = 0
end
if steps_count == nil or steps_count < 0 then
	steps_count = 0
end

local risk = tonumber(redis.call("HGET", key, "risk_score")) or 0
local cooldown_until = tonumber(redis.call("HGET", key, "cooldown_until")) or 0
local last_violation = tonumber(redis.call("HGET", key, "last_violation_at")) or 0

if decay_sec > 0 and risk > 0 and last_violation > 0 and now > last_violation then
	local elapsed = now - last_violation
	local decays = math.floor(elapsed / decay_sec)
	if decays > 0 then
		if decays > risk then
			decays = risk
		end
		risk = risk - decays
		last_violation = last_violation + decays * decay_sec
	end
end

risk = risk + weight

local step = 0
if forced_step ~= nil and forced_step >= 0 then
	step = forced_step
elseif steps_count > 0 then
	local idx = risk
	if idx < 1 then
		idx = 1
	end
	if idx > steps_count then
		idx = steps_count
	end
	step = tonumber(ARGV[4 + idx]) or 0
end

local candidate = now + step
if candidate > cooldown_until then
	cooldown_until = candidate
end

last_violation = now

redis.call("HSET", key,
	"risk_score", risk,
	"cooldown_until", cooldown_until,
	"last_violation_at", last_violation)

return {risk, cooldown_until, last_violation}
`

const decayScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local decay_sec = tonumber(ARGV[2])

if now == nil or now < 0 then
	now = 0
end
if decay_sec == nil or decay_sec < 0 then
	decay_sec = 0
end

local risk = tonumber(redis.call("HGET", key, "risk_score")) or 0
local cooldown_until = tonumber(redis.call("HGET", key, "cooldown_until")) or 0
local last_violation = tonumber(redis.call("HGET", key, "last_violation_at")) or 0

if decay_sec > 0 and risk > 0 and last_violation > 0 and now > last_violation then
	local elapsed = now - last_violation
	local decays = math.floor(elapsed / decay_sec)
	if decays > 0 then
		if decays > risk then
			decays = risk
		end
		risk = risk - decays
		last_violation = last_violation + decays * decay_sec
		redis.call("HSET", key,
			"risk_score", risk,
			"last_violation_at", last_violation)
	end
end

return {risk, cooldown_until, last_violation}
`

var ErrValidation = errors.New("validation error")

type Store interface {
	Get(ctx context.Context, userID int64) (redrepo.RiskStateRecord, error)
	EvalForUser(ctx context.Context, userID int64, script string, args ...interface{}) (interface{}, error)
}

type Config struct {
	RiskDecayHours   int
	CooldownStepsSec []int
	ShadowThreshold  int
}

type State struct {
	RiskScore       int
	CooldownUntil   *time.Time
	LastViolationAt *time.Time
	ShadowEnabled   bool
	Exists          bool
}

type Service struct {
	store     Store
	cfg       Config
	telemetry TelemetryService
	now       func() time.Time
}

type TelemetryService interface {
	IngestBatch(ctx context.Context, userID *int64, events []analyticsvc.BatchEvent) error
}

func NewService(store Store, cfg Config) *Service {
	if cfg.RiskDecayHours <= 0 {
		cfg.RiskDecayHours = defaultRiskDecayHours
	}
	if cfg.ShadowThreshold <= 0 {
		cfg.ShadowThreshold = 5
	}
	if len(cfg.CooldownStepsSec) == 0 {
		cfg.CooldownStepsSec = []int{30, 60, 300, 1800, 86400}
	}

	return &Service{
		store: store,
		cfg:   cfg,
		now:   time.Now,
	}
}

func (s *Service) AttachTelemetry(telemetry TelemetryService) {
	s.telemetry = telemetry
}

func (s *Service) GetState(ctx context.Context, userID int64) (State, error) {
	if userID <= 0 {
		return State{}, ErrValidation
	}
	if s.store == nil {
		return State{}, fmt.Errorf("risk store is nil")
	}

	rec, err := s.store.Get(ctx, userID)
	if err != nil {
		return State{}, err
	}
	return s.mapRecord(rec), nil
}

func (s *Service) ApplyViolation(ctx context.Context, userID int64, weight int, now time.Time) (State, error) {
	return s.applyViolation(ctx, userID, weight, nil, now)
}

func (s *Service) ApplyViolationWithCooldown(ctx context.Context, userID int64, weight int, cooldownSec int, now time.Time) (State, error) {
	if cooldownSec < 0 {
		cooldownSec = 0
	}
	return s.applyViolation(ctx, userID, weight, &cooldownSec, now)
}

func (s *Service) applyViolation(ctx context.Context, userID int64, weight int, cooldownSec *int, now time.Time) (State, error) {
	if userID <= 0 {
		return State{}, ErrValidation
	}
	if weight <= 0 {
		weight = 1
	}
	if s.store == nil {
		return State{}, fmt.Errorf("risk store is nil")
	}
	if now.IsZero() {
		now = s.now().UTC()
	}

	args := make([]interface{}, 0, 4+len(s.cfg.CooldownStepsSec))
	args = append(args,
		weight,
		now.UTC().Unix(),
		s.decaySeconds(),
		len(s.cfg.CooldownStepsSec),
	)
	for _, step := range s.cfg.CooldownStepsSec {
		if step < 0 {
			step = 0
		}
		args = append(args, step)
	}
	forcedStep := -1
	if cooldownSec != nil {
		forcedStep = *cooldownSec
		if forcedStep < 0 {
			forcedStep = 0
		}
	}
	args = append(args, forcedStep)

	wasShadow := false
	if prev, err := s.store.Get(ctx, userID); err == nil {
		wasShadow = prev.RiskScore >= s.cfg.ShadowThreshold
	}

	rec, err := s.execStateScript(ctx, userID, violationScript, args...)
	if err != nil {
		return State{}, err
	}
	state := s.mapRecord(rec)
	if !wasShadow && state.ShadowEnabled {
		s.emitShadowEnabled(ctx, userID, state, now)
	}
	return state, nil
}

func (s *Service) ApplyDecay(ctx context.Context, userID int64, now time.Time) (State, error) {
	if userID <= 0 {
		return State{}, ErrValidation
	}
	if s.store == nil {
		return State{}, fmt.Errorf("risk store is nil")
	}
	if now.IsZero() {
		now = s.now().UTC()
	}

	rec, err := s.execStateScript(ctx, userID, decayScript,
		now.UTC().Unix(),
		s.decaySeconds(),
	)
	if err != nil {
		return State{}, err
	}
	return s.mapRecord(rec), nil
}

func (s *Service) execStateScript(ctx context.Context, userID int64, script string, args ...interface{}) (redrepo.RiskStateRecord, error) {
	raw, err := s.store.EvalForUser(ctx, userID, script, args...)
	if err != nil {
		return redrepo.RiskStateRecord{}, err
	}

	arr, ok := raw.([]interface{})
	if !ok || len(arr) < 3 {
		return redrepo.RiskStateRecord{}, fmt.Errorf("unexpected risk script result")
	}

	risk, ok := asInt(arr[0])
	if !ok {
		return redrepo.RiskStateRecord{}, fmt.Errorf("unexpected risk score value")
	}
	cooldownUntil, ok := asInt64(arr[1])
	if !ok {
		return redrepo.RiskStateRecord{}, fmt.Errorf("unexpected cooldown value")
	}
	lastViolationAt, ok := asInt64(arr[2])
	if !ok {
		return redrepo.RiskStateRecord{}, fmt.Errorf("unexpected last_violation value")
	}

	if risk < 0 {
		risk = 0
	}
	if cooldownUntil < 0 {
		cooldownUntil = 0
	}
	if lastViolationAt < 0 {
		lastViolationAt = 0
	}

	return redrepo.RiskStateRecord{
		RiskScore:       risk,
		CooldownUntil:   cooldownUntil,
		LastViolationAt: lastViolationAt,
		Exists:          true,
	}, nil
}

func (s *Service) mapRecord(rec redrepo.RiskStateRecord) State {
	state := State{
		RiskScore:     rec.RiskScore,
		ShadowEnabled: rec.RiskScore >= s.cfg.ShadowThreshold,
		Exists:        rec.Exists,
	}
	if rec.CooldownUntil > 0 {
		v := time.Unix(rec.CooldownUntil, 0).UTC()
		state.CooldownUntil = &v
	}
	if rec.LastViolationAt > 0 {
		v := time.Unix(rec.LastViolationAt, 0).UTC()
		state.LastViolationAt = &v
	}
	return state
}

func (s *Service) decaySeconds() int {
	hours := s.cfg.RiskDecayHours
	if hours <= 0 {
		hours = defaultRiskDecayHours
	}
	return hours * int(time.Hour/time.Second)
}

func asInt(value interface{}) (int, bool) {
	switch v := value.(type) {
	case int64:
		return int(v), true
	case int:
		return v, true
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func asInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

func (s *Service) emitShadowEnabled(ctx context.Context, userID int64, state State, now time.Time) {
	if s.telemetry == nil || userID <= 0 {
		return
	}

	props := map[string]any{
		"risk_score":        state.RiskScore,
		"shadow_threshold":  s.cfg.ShadowThreshold,
		"cooldown_until_ts": int64(0),
	}
	if state.CooldownUntil != nil {
		props["cooldown_until_ts"] = state.CooldownUntil.UTC().Unix()
	}

	uid := userID
	_ = s.telemetry.IngestBatch(ctx, &uid, []analyticsvc.BatchEvent{
		{
			Name:  "antiabuse_shadow_enabled",
			TS:    now.UTC().UnixMilli(),
			Props: props,
		},
	})
}
