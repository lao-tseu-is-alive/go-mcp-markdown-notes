package authadapter

import (
	"context"
	"errors"
)

// ErrUnauthenticated is returned when no valid identity is present in the request context.
var ErrUnauthenticated = errors.New("authenticated user is required")

// AuthenticatedUser holds the identity and granted scopes extracted from a verified bearer token.
type AuthenticatedUser struct {
	AppUserID   int64
	Email       string
	DisplayName string
	Scopes      []string
}

type userContextKey struct{}

// ContextWithUser stores the authenticated user in ctx for downstream handlers to retrieve.
func ContextWithUser(ctx context.Context, user *AuthenticatedUser) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

// UserFromContext returns the authenticated user from ctx, returning false when the value is absent, nil, or has no positive AppUserID.
func UserFromContext(ctx context.Context) (*AuthenticatedUser, bool) {
	user, ok := ctx.Value(userContextKey{}).(*AuthenticatedUser)
	return user, ok && user != nil && user.AppUserID > 0
}

// RequireUser returns the authenticated user or ErrUnauthenticated when none is present.
func RequireUser(ctx context.Context) (*AuthenticatedUser, error) {
	user, ok := UserFromContext(ctx)
	if !ok {
		return nil, ErrUnauthenticated
	}
	return user, nil
}

// HasScope reports whether the user holds the given scope. The "notes:admin" scope acts as a wildcard and satisfies any scope check.
func (u *AuthenticatedUser) HasScope(scope string) bool {
	for _, candidate := range u.Scopes {
		if candidate == scope || candidate == "notes:admin" {
			return true
		}
	}
	return false
}
