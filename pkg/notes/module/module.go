// Package module provides an importable Module for the notes domain.
//
// It encapsulates the domain wiring (repository + service) and transport
// wiring (Connect interceptors + handler) so that the same code can be
// used both in the standalone notes-server binary and in a multi-service
// bundle that shares one http.Server, one database pool, and one auth verifier.
//
// Standalone mode:
//
//	mod, _ := module.New(ctx, cfg, deps)
//	module.Migrate(ctx, pool)
//	mod.RegisterRoutes(mux)
//
// Bundle mode — collect handlers from all modules and mount on a shared mux:
//
//	for _, ch := range mod.ConnectHandlers() {
//	    sharedMux.Handle(ch.Pattern, ch.Handler)
//	}
package module

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/authadapter"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/notes"
)

const defaultRequestTimeout = 10 * time.Second

// Config holds notes-module-specific configuration.
type Config struct {
	// RequestTimeout is the per-RPC deadline enforced by the server-side timeout interceptor.
	// Defaults to 10 s when zero.
	RequestTimeout time.Duration
}

func (c Config) requestTimeout() time.Duration {
	if c.RequestTimeout <= 0 {
		return defaultRequestTimeout
	}
	return c.RequestTimeout
}

// Deps holds cross-cutting dependencies injected by the main binary or a bundle.
// All fields are required except Logger, which falls back to slog.Default.
type Deps struct {
	Pool     *pgxpool.Pool
	Verifier authadapter.TokenVerifier
	Logger   *slog.Logger
}

// Module encapsulates the notes domain: repository, business service, and Connect handler.
// It holds no listener and never starts an HTTP server itself.
type Module struct {
	cfg     Config
	deps    Deps
	service *notes.Service
	connect *notes.ConnectServer
}

// New creates a fully wired notes Module ready to register routes.
// ctx is used only for any synchronous initialisation that may be added in the future;
// it does not need to outlive the call.
func New(_ context.Context, cfg Config, deps Deps) (*Module, error) {
	if deps.Pool == nil {
		return nil, fmt.Errorf("notes module: database pool is required")
	}
	if deps.Verifier == nil {
		return nil, fmt.Errorf("notes module: token verifier is required")
	}
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}

	repo, err := notes.NewPostgresRepository(deps.Pool, deps.Logger)
	if err != nil {
		return nil, fmt.Errorf("notes module: storage init: %w", err)
	}
	svc, err := notes.NewService(repo, deps.Logger)
	if err != nil {
		return nil, fmt.Errorf("notes module: service init: %w", err)
	}
	cs, err := notes.NewConnectServer(svc, deps.Logger)
	if err != nil {
		return nil, fmt.Errorf("notes module: connect server init: %w", err)
	}

	return &Module{cfg: cfg, deps: deps, service: svc, connect: cs}, nil
}

// Start is a placeholder for future background workers (e.g. event consumers).
func (m *Module) Start(_ context.Context) error { return nil }

// Stop is a placeholder for graceful shutdown of future background workers.
func (m *Module) Stop(_ context.Context) error { return nil }
