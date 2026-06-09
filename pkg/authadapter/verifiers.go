package authadapter

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"slices"

	"github.com/lao-tseu-is-alive/go-cloud-k8s-common-libs/pkg/goHttpEcho"
)

var ErrInvalidToken = errors.New("invalid bearer token")

// JWTVerifier adapts the JWT format issued by go-cloud-k8s-auth to the notes
// service's deliberately small authentication contract.
type JWTVerifier struct {
	checker       goHttpEcho.JwtChecker
	defaultScopes []string
}

func NewJWTVerifier(checker goHttpEcho.JwtChecker, defaultScopes []string) (*JWTVerifier, error) {
	if checker == nil {
		return nil, errors.New("JWT checker is required")
	}
	return &JWTVerifier{
		checker:       checker,
		defaultScopes: slices.Clone(defaultScopes),
	}, nil
}

func (v *JWTVerifier) VerifyBearerToken(_ context.Context, token string) (user *AuthenticatedUser, err error) {
	defer func() {
		if recover() != nil {
			user = nil
			err = ErrInvalidToken
		}
	}()
	claims, err := v.checker.ParseToken(token)
	if err != nil || claims == nil || claims.User == nil || claims.User.UserId <= 0 {
		return nil, ErrInvalidToken
	}
	scopes := slices.Clone(v.defaultScopes)
	if claims.User.IsAdmin && !slices.Contains(scopes, "notes:admin") {
		scopes = append(scopes, "notes:admin")
	}
	return &AuthenticatedUser{
		AppUserID:   int64(claims.User.UserId),
		Email:       claims.User.Email,
		DisplayName: claims.User.Name,
		Scopes:      scopes,
	}, nil
}

// DevTokenVerifier is an explicit local-development verifier. It must only be
// selected when NOTES_AUTH_MODE=dev.
type DevTokenVerifier struct {
	token string
	user  AuthenticatedUser
}

func NewDevTokenVerifier(token string, user AuthenticatedUser) (*DevTokenVerifier, error) {
	if token == "" {
		return nil, errors.New("development token is required")
	}
	if user.AppUserID <= 0 {
		return nil, fmt.Errorf("development user ID must be positive")
	}
	user.Scopes = slices.Clone(user.Scopes)
	return &DevTokenVerifier{token: token, user: user}, nil
}

func (v *DevTokenVerifier) VerifyBearerToken(_ context.Context, token string) (*AuthenticatedUser, error) {
	if subtle.ConstantTimeCompare([]byte(token), []byte(v.token)) != 1 {
		return nil, ErrInvalidToken
	}
	user := v.user
	user.Scopes = slices.Clone(v.user.Scopes)
	return &user, nil
}
