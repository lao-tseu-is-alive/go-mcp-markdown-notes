package authadapter

import (
	"context"
	"errors"
	"strings"
)

// CompositeVerifier routes bearer tokens by shape: opaque "pat_..." personal
// access tokens go to the introspection-based verifier, everything else is
// parsed locally as a JWT.
type CompositeVerifier struct {
	jwt TokenVerifier
	pat TokenVerifier
}

func NewCompositeVerifier(jwt, pat TokenVerifier) (*CompositeVerifier, error) {
	if jwt == nil || pat == nil {
		return nil, errors.New("both jwt and pat verifiers are required")
	}
	return &CompositeVerifier{jwt: jwt, pat: pat}, nil
}

func (v *CompositeVerifier) VerifyBearerToken(ctx context.Context, token string) (*AuthenticatedUser, error) {
	if strings.HasPrefix(token, PatTokenPrefix) {
		return v.pat.VerifyBearerToken(ctx, token)
	}
	return v.jwt.VerifyBearerToken(ctx, token)
}
