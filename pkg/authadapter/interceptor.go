package authadapter

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"connectrpc.com/connect"
)

type TokenVerifier interface {
	VerifyBearerToken(context.Context, string) (*AuthenticatedUser, error)
}

func NewInterceptor(verifier TokenVerifier, log *slog.Logger) connect.UnaryInterceptorFunc {
	if log == nil {
		log = slog.Default()
	}
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if verifier == nil {
				return nil, connect.NewError(connect.CodeInternal, errors.New("token verifier is not configured"))
			}
			authorization := req.Header().Get("Authorization")
			parts := strings.Fields(authorization)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("valid bearer token is required"))
			}
			user, err := verifier.VerifyBearerToken(ctx, parts[1])
			if err != nil || user == nil || user.AppUserID <= 0 {
				log.Warn("bearer token verification failed", "error", err)
				return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid bearer token"))
			}
			return next(ContextWithUser(ctx, user), req)
		}
	}
}
