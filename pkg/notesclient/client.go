package notesclient

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1/notesv1connect"
)

// Option configures a notes client created by New.
type Option func(*options)

type options struct {
	httpClient *http.Client
	timeout    time.Duration
}

// WithHTTPClient overrides the underlying *http.Client used by the Connect client.
// If not provided (or nil is passed), http.DefaultClient is used.
func WithHTTPClient(c *http.Client) Option {
	return func(o *options) {
		o.httpClient = c
	}
}

// WithTimeout sets a per-request timeout for all calls made by the client.
// The timeout is enforced via a client-side interceptor. If not set, no
// additional timeout is applied (callers can still use context deadlines).
func WithTimeout(d time.Duration) Option {
	return func(o *options) {
		o.timeout = d
	}
}

// New builds a Connect client for the notes-server that sends the
// given bearer token (a personal access token "pat_..." or a dev token) on
// every RPC call.
//
// Additional behavior can be configured with options such as WithHTTPClient
// and WithTimeout.
func New(baseURL, token string, opts ...Option) (notesv1connect.NotesServiceClient, error) {
	cfg := options{}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.httpClient == nil {
		cfg.httpClient = http.DefaultClient
	}

	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return nil, errors.New("notes server URL is required")
	}
	if token == "" {
		return nil, errors.New("bearer token is required")
	}

	var interceptors []connect.Interceptor

	if cfg.timeout > 0 {
		interceptors = append(interceptors, timeoutInterceptor(cfg.timeout))
	}

	interceptors = append(interceptors, bearerTokenInterceptor(token))

	return notesv1connect.NewNotesServiceClient(
		cfg.httpClient,
		baseURL,
		connect.WithInterceptors(interceptors...),
	), nil
}

// bearerTokenInterceptor attaches the Authorization header to every request.
func bearerTokenInterceptor(token string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("Authorization", "Bearer "+token)
			return next(ctx, req)
		}
	}
}

// timeoutInterceptor adds a client-side per-request timeout.
func timeoutInterceptor(d time.Duration) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			ctx, cancel := context.WithTimeout(ctx, d)
			defer cancel()
			return next(ctx, req)
		}
	}
}
