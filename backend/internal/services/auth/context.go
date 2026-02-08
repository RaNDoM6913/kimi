package auth

import "context"

type identityContextKey string

const identityKey identityContextKey = "auth_identity"

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
