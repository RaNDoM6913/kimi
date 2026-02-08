package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	goredis "github.com/redis/go-redis/v9"

	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
)

const (
	sessionPrefix        = "sessions:"
	refreshPrefix        = "refresh:"
	sessionRefreshPrefix = "session_refresh:"
	userSessionsPrefix   = "user_sessions:"
)

type SessionRepo struct {
	client *goredis.Client
}

func NewSessionRepo(client *goredis.Client) *SessionRepo {
	return &SessionRepo{client: client}
}

func (r *SessionRepo) Create(ctx context.Context, session authsvc.SessionRecord, refreshToken string) error {
	if r.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	if strings.TrimSpace(session.SID) == "" || strings.TrimSpace(refreshToken) == "" || session.UserID <= 0 {
		return authsvc.ErrInvalidInput
	}

	ttl := ttlFor(session.ExpiresAt)
	fields := map[string]interface{}{
		"user_id":    session.UserID,
		"role":       session.Role,
		"expires_at": session.ExpiresAt.Unix(),
	}

	pipe := r.client.TxPipeline()
	pipe.HSet(ctx, sessionKey(session.SID), fields)
	pipe.Expire(ctx, sessionKey(session.SID), ttl)
	pipe.HSet(ctx, refreshKey(refreshToken), map[string]interface{}{
		"user_id":    session.UserID,
		"sid":        session.SID,
		"role":       session.Role,
		"expires_at": session.ExpiresAt.Unix(),
	})
	pipe.Expire(ctx, refreshKey(refreshToken), ttl)
	pipe.Set(ctx, sessionRefreshKey(session.SID), refreshToken, ttl)
	pipe.SAdd(ctx, userSessionsKey(session.UserID), session.SID)
	pipe.Expire(ctx, userSessionsKey(session.UserID), ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("create redis session: %w", err)
	}

	return nil
}

func (r *SessionRepo) GetSession(ctx context.Context, sid string) (authsvc.SessionRecord, error) {
	if r.client == nil {
		return authsvc.SessionRecord{}, fmt.Errorf("redis client is nil")
	}

	values, err := r.client.HGetAll(ctx, sessionKey(sid)).Result()
	if err != nil {
		return authsvc.SessionRecord{}, fmt.Errorf("get session hash: %w", err)
	}
	if len(values) == 0 {
		return authsvc.SessionRecord{}, authsvc.ErrSessionNotFound
	}

	session, err := parseSessionRecord(values)
	if err != nil {
		return authsvc.SessionRecord{}, err
	}
	session.SID = sid
	return session, nil
}

func (r *SessionRepo) GetByRefreshToken(ctx context.Context, refreshToken string) (authsvc.SessionRecord, error) {
	if r.client == nil {
		return authsvc.SessionRecord{}, fmt.Errorf("redis client is nil")
	}

	values, err := r.client.HGetAll(ctx, refreshKey(refreshToken)).Result()
	if err != nil {
		return authsvc.SessionRecord{}, fmt.Errorf("get refresh hash: %w", err)
	}
	if len(values) == 0 {
		return authsvc.SessionRecord{}, authsvc.ErrRefreshNotFound
	}

	session, err := parseSessionRecord(values)
	if err != nil {
		return authsvc.SessionRecord{}, err
	}

	sid := strings.TrimSpace(values["sid"])
	if sid == "" {
		return authsvc.SessionRecord{}, authsvc.ErrRefreshNotFound
	}
	session.SID = sid

	return session, nil
}

func (r *SessionRepo) RotateRefresh(ctx context.Context, sid, oldRefreshToken, newRefreshToken string, expiresAt time.Time) error {
	if r.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	session, err := r.GetByRefreshToken(ctx, oldRefreshToken)
	if err != nil {
		return err
	}
	if sid != "" && sid != session.SID {
		return authsvc.ErrRefreshNotFound
	}

	session.ExpiresAt = expiresAt
	ttl := ttlFor(expiresAt)
	fields := map[string]interface{}{
		"user_id":    session.UserID,
		"role":       session.Role,
		"expires_at": expiresAt.Unix(),
	}

	pipe := r.client.TxPipeline()
	pipe.Del(ctx, refreshKey(oldRefreshToken))
	pipe.HSet(ctx, refreshKey(newRefreshToken), map[string]interface{}{
		"user_id":    session.UserID,
		"sid":        session.SID,
		"role":       session.Role,
		"expires_at": expiresAt.Unix(),
	})
	pipe.Expire(ctx, refreshKey(newRefreshToken), ttl)
	pipe.HSet(ctx, sessionKey(session.SID), fields)
	pipe.Expire(ctx, sessionKey(session.SID), ttl)
	pipe.Set(ctx, sessionRefreshKey(session.SID), newRefreshToken, ttl)
	pipe.SAdd(ctx, userSessionsKey(session.UserID), session.SID)
	pipe.Expire(ctx, userSessionsKey(session.UserID), ttl)

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("rotate refresh token: %w", err)
	}

	return nil
}

func (r *SessionRepo) DeleteSession(ctx context.Context, sid string) error {
	if r.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	if strings.TrimSpace(sid) == "" {
		return nil
	}

	sessionValues, err := r.client.HGetAll(ctx, sessionKey(sid)).Result()
	if err != nil {
		return fmt.Errorf("load session for delete: %w", err)
	}

	refreshToken, err := r.client.Get(ctx, sessionRefreshKey(sid)).Result()
	if err != nil && err != goredis.Nil {
		return fmt.Errorf("load session refresh pointer: %w", err)
	}

	var userID int64
	if value, ok := sessionValues["user_id"]; ok {
		parsed, parseErr := strconv.ParseInt(value, 10, 64)
		if parseErr == nil && parsed > 0 {
			userID = parsed
		}
	}

	pipe := r.client.TxPipeline()
	pipe.Del(ctx, sessionKey(sid))
	pipe.Del(ctx, sessionRefreshKey(sid))
	if refreshToken != "" {
		pipe.Del(ctx, refreshKey(refreshToken))
	}
	if userID > 0 {
		pipe.SRem(ctx, userSessionsKey(userID), sid)
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

func (r *SessionRepo) DeleteAllForUser(ctx context.Context, userID int64) error {
	if r.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	if userID <= 0 {
		return authsvc.ErrInvalidInput
	}

	sids, err := r.client.SMembers(ctx, userSessionsKey(userID)).Result()
	if err != nil {
		return fmt.Errorf("list user sessions: %w", err)
	}

	for _, sid := range sids {
		if err := r.DeleteSession(ctx, sid); err != nil {
			return err
		}
	}

	if err := r.client.Del(ctx, userSessionsKey(userID)).Err(); err != nil {
		return fmt.Errorf("delete user sessions key: %w", err)
	}

	return nil
}

func parseSessionRecord(values map[string]string) (authsvc.SessionRecord, error) {
	userID, err := strconv.ParseInt(values["user_id"], 10, 64)
	if err != nil || userID <= 0 {
		return authsvc.SessionRecord{}, authsvc.ErrUnauthorized
	}

	expiresUnix, err := strconv.ParseInt(values["expires_at"], 10, 64)
	if err != nil {
		return authsvc.SessionRecord{}, authsvc.ErrUnauthorized
	}

	return authsvc.SessionRecord{
		UserID:    userID,
		Role:      values["role"],
		ExpiresAt: time.Unix(expiresUnix, 0).UTC(),
	}, nil
}

func ttlFor(expiresAt time.Time) time.Duration {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return time.Second
	}
	return ttl
}

func sessionKey(sid string) string {
	return sessionPrefix + sid
}

func refreshKey(token string) string {
	return refreshPrefix + token
}

func sessionRefreshKey(sid string) string {
	return sessionRefreshPrefix + sid
}

func userSessionsKey(userID int64) string {
	return userSessionsPrefix + strconv.FormatInt(userID, 10)
}
