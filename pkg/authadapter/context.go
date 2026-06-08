package authadapter

import (
	"context"
	"errors"
)

var ErrUnauthenticated = errors.New("authenticated user is required")

type AuthenticatedUser struct {
	AppUserID   int64
	Email       string
	DisplayName string
	Scopes      []string
}

type userContextKey struct{}

func ContextWithUser(ctx context.Context, user *AuthenticatedUser) context.Context {
	return context.WithValue(ctx, userContextKey{}, user)
}

func UserFromContext(ctx context.Context) (*AuthenticatedUser, bool) {
	user, ok := ctx.Value(userContextKey{}).(*AuthenticatedUser)
	return user, ok && user != nil && user.AppUserID > 0
}

func RequireUser(ctx context.Context) (*AuthenticatedUser, error) {
	user, ok := UserFromContext(ctx)
	if !ok {
		return nil, ErrUnauthenticated
	}
	return user, nil
}

func (u *AuthenticatedUser) HasScope(scope string) bool {
	for _, candidate := range u.Scopes {
		if candidate == scope || candidate == "notes:admin" {
			return true
		}
	}
	return false
}
