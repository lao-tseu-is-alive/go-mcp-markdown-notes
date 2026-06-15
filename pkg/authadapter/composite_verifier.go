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

// NewCompositeVerifier constructs a CompositeVerifier from a JWT verifier and a PAT verifier.
func NewCompositeVerifier(jwt, pat TokenVerifier) (*CompositeVerifier, error) {
	if jwt == nil || pat == nil {
		return nil, errors.New("both jwt and pat verifiers are required")
	}
	return &CompositeVerifier{jwt: jwt, pat: pat}, nil
}

// VerifyBearerToken dispatches to the PAT verifier for "pat_..." tokens and to the JWT verifier for all others.
func (v *CompositeVerifier) VerifyBearerToken(ctx context.Context, token string) (*AuthenticatedUser, error) {
	if strings.HasPrefix(token, PatTokenPrefix) {
		return v.pat.VerifyBearerToken(ctx, token)
	}
	return v.jwt.VerifyBearerToken(ctx, token)
}
