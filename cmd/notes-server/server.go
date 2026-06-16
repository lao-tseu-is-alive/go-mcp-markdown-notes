package main

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common-libs/pkg/goHttpEcho"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/authadapter"
	notesmodule "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/notes/module"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/version"
)

//go:embed notesFront/dist/*
var frontendFiles embed.FS

// application holds the shared resources for the notes server: the database pool, composed HTTP handler, and logger.
type application struct {
	pool    *pgxpool.Pool
	handler http.Handler
	log     *slog.Logger
}

// newApplication opens the database pool, runs schema migrations, wires all service layers, and returns a ready-to-serve application.
// The pool is closed on any error path; the caller is responsible for calling application.close on success.
func newApplication(ctx context.Context, config serverConfig, log *slog.Logger) (*application, error) {
	poolConfig, err := pgxpool.ParseConfig(config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	poolConfig.MaxConns = config.MaxConnections
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("open database pool: %w", err)
	}
	cleanup := true
	defer func() {
		if cleanup {
			pool.Close()
		}
	}()
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}
	if err := notesmodule.Migrate(ctx, pool); err != nil {
		return nil, err
	}

	verifier, err := buildTokenVerifier(config, log)
	if err != nil {
		return nil, err
	}
	mod, err := notesmodule.New(ctx, notesmodule.Config{
		RequestTimeout: config.RequestTimeout,
	}, notesmodule.Deps{
		Pool:     pool,
		Verifier: verifier,
		Logger:   log,
	})
	if err != nil {
		return nil, fmt.Errorf("notes module: %w", err)
	}

	mux := http.NewServeMux()
	if err := mod.RegisterRoutes(mux); err != nil {
		return nil, fmt.Errorf("register notes routes: %w", err)
	}
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /readiness", readinessHandler(pool))
	mux.HandleFunc("GET /goAppInfo", appInfoHandler)
	mux.HandleFunc("GET /config", frontendConfigHandler(config))

	// Serve embedded frontend (SPA fallback to index.html)
	frontendFS, err := fs.Sub(frontendFiles, "notesFront/dist")
	if err != nil {
		return nil, fmt.Errorf("sub-filesystem for frontend: %w", err)
	}
	fileServer := http.FileServer(http.FS(frontendFS))
	mux.Handle("/", spaHandler(fileServer, frontendFS))

	cleanup = false
	return &application{
		pool:    pool,
		handler: recoverMiddleware(log, requestLogMiddleware(log, mux)),
		log:     log,
	}, nil
}

// spaHandler serves static assets from the embedded FS and falls back to index.html for any path that does not match
// an embedded file, enabling client-side routing in the SPA.
func spaHandler(fileServer http.Handler, frontendFS fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Strip leading slash
		name := path[1:]

		// Check if file exists in embedded FS
		_, err := fs.Stat(frontendFS, name)
		if err == nil {
			// File exists, serve it
			fileServer.ServeHTTP(w, r)
			return
		}

		// File does not exist, serve index.html (fallback)
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	}
}

// buildTokenVerifier selects the appropriate TokenVerifier for the configured auth mode:
// DevTokenVerifier for "dev", or a CompositeVerifier (JWT + PAT introspection) for "jwt".
func buildTokenVerifier(config serverConfig, log *slog.Logger) (authadapter.TokenVerifier, error) {
	scopes := []string{"notes:read", "notes:write", "notes:mcp"}
	if config.AuthMode == "dev" {
		return authadapter.NewDevTokenVerifier(config.DevToken, authadapter.AuthenticatedUser{
			AppUserID:   config.DevUserID,
			Email:       config.DevUserEmail,
			DisplayName: config.DevDisplayName,
			Scopes:      scopes,
		})
	}
	checker, err := goHttpEcho.GetNewJwtCheckerFromConfig(version.AppName, 60, log)
	if err != nil {
		return nil, fmt.Errorf("configure JWT verifier: %w", err)
	}
	jwtVerifier, err := authadapter.NewJWTVerifier(checker, scopes)
	if err != nil {
		return nil, err
	}
	// Personal access tokens (pat_...) are verified by introspection against
	// the auth service; JWTs keep being parsed locally with the shared secret.
	patVerifier, err := authadapter.NewPatVerifier(config.AuthServerURL)
	if err != nil {
		return nil, fmt.Errorf("configure PAT verifier: %w", err)
	}
	return authadapter.NewCompositeVerifier(jwtVerifier, patVerifier)
}

// close releases the database connection pool.
func (a *application) close() {
	a.pool.Close()
}

// serve starts the HTTP server on the provided listener and blocks until ctx is cancelled or the server fails.
// On cancellation it initiates a graceful shutdown within shutdownPeriod before forcing a close.
func (a *application) serve(ctx context.Context, listener net.Listener, shutdownPeriod time.Duration) error {
	server := &http.Server{
		Handler:           a.handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownPeriod)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close()
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func healthHandler(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{"status": "ok"})
}

// readinessHandler returns 503 when the database pool cannot be reached within 2 seconds, used by Kubernetes readiness probes.
func readinessHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
		defer cancel()
		if err := pool.Ping(ctx); err != nil {
			writeJSON(writer, http.StatusServiceUnavailable, map[string]string{"status": "not ready"})
			return
		}
		writeJSON(writer, http.StatusOK, map[string]string{"status": "ready"})
	}
}

// frontendConfigHandler tells the SPA how to authenticate: in jwt mode it
// silently mints tokens from the auth service; in dev mode it shows the
// dev-token form.
func frontendConfigHandler(config serverConfig) http.HandlerFunc {
	return func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(writer, http.StatusOK, map[string]string{
			"authMode":    config.AuthMode,
			"authBaseUrl": config.AuthServerURL,
		})
	}
}

func appInfoHandler(writer http.ResponseWriter, _ *http.Request) {
	writeJSON(writer, http.StatusOK, map[string]string{
		"app":        version.AppName,
		"version":    version.Version,
		"revision":   version.Revision,
		"build":      version.BuildStamp,
		"repository": version.Repository,
	})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

// requestLogMiddleware logs the HTTP method, path, and elapsed time for every request.
func requestLogMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		next.ServeHTTP(writer, request)
		log.Info("HTTP request", "method", request.Method, "path", request.URL.Path, "duration", time.Since(started))
	})
}

// recoverMiddleware catches panics in downstream handlers, logs the stack trace, and returns a 500 JSON response.
func recoverMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Error("HTTP panic", "panic", recovered, "stack", string(debug.Stack()))
				writeJSON(writer, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			}
		}()
		next.ServeHTTP(writer, request)
	})
}
