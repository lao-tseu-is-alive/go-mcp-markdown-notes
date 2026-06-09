package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"time"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lao-tseu-is-alive/go-cloud-k8s-common-libs/pkg/goHttpEcho"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1/notesv1connect"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/authadapter"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/notes"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/version"
)

type application struct {
	pool    *pgxpool.Pool
	handler http.Handler
	log     *slog.Logger
}

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
	if err := runMigrations(ctx, pool); err != nil {
		return nil, err
	}

	repository, err := notes.NewPostgresRepository(pool, log)
	if err != nil {
		return nil, err
	}
	service, err := notes.NewService(repository, log)
	if err != nil {
		return nil, err
	}
	connectServer, err := notes.NewConnectServer(service, log)
	if err != nil {
		return nil, err
	}
	verifier, err := buildTokenVerifier(config, log)
	if err != nil {
		return nil, err
	}
	interceptor := connect.WithInterceptors(authadapter.NewInterceptor(verifier, log))
	path, notesHandler := notesv1connect.NewNotesServiceHandler(connectServer, interceptor)

	mux := http.NewServeMux()
	mux.Handle(path, http.MaxBytesHandler(notesHandler, 1<<20))
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /readiness", readinessHandler(pool))
	mux.HandleFunc("GET /goAppInfo", appInfoHandler)

	cleanup = false
	return &application{
		pool:    pool,
		handler: recoverMiddleware(log, requestLogMiddleware(log, mux)),
		log:     log,
	}, nil
}

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
	return authadapter.NewJWTVerifier(checker, scopes)
}

func (a *application) close() {
	a.pool.Close()
}

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

func requestLogMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		next.ServeHTTP(writer, request)
		log.Info("HTTP request", "method", request.Method, "path", request.URL.Path, "duration", time.Since(started))
	})
}

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
