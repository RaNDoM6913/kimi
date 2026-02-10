package auth

import "context"

type identityContextKey string

const identityKey identityContextKey = "auth_identity"
const deviceIDKey identityContextKey = "device_id"
const actorIsBotKey identityContextKey = "actor_is_bot"
const actorRoleKey identityContextKey = "actor_role"
const actorTGIDKey identityContextKey = "actor_tg_id"

type Identity struct {
	UserID int64
	SID    string
	Role   string
}

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey, identity)
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey).(Identity)
	return identity, ok
}

func WithDeviceID(ctx context.Context, deviceID string) context.Context {
	return context.WithValue(ctx, deviceIDKey, deviceID)
}

func DeviceIDFromContext(ctx context.Context) (string, bool) {
	deviceID, ok := ctx.Value(deviceIDKey).(string)
	return deviceID, ok
}

func WithActorIsBot(ctx context.Context, isBot bool) context.Context {
	return context.WithValue(ctx, actorIsBotKey, isBot)
}

func ActorIsBotFromContext(ctx context.Context) (bool, bool) {
	value, ok := ctx.Value(actorIsBotKey).(bool)
	return value, ok
}

func WithActorRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, actorRoleKey, role)
}

func ActorRoleFromContext(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(actorRoleKey).(string)
	return value, ok
}

func WithActorTGID(ctx context.Context, actorTGID int64) context.Context {
	return context.WithValue(ctx, actorTGIDKey, actorTGID)
}

func ActorTGIDFromContext(ctx context.Context) (int64, bool) {
	value, ok := ctx.Value(actorTGIDKey).(int64)
	return value, ok
}
