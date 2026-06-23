package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/authadapter"
)

func TestSpaHandler(t *testing.T) {
	fsys := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>index</html>")},
		"app.js":     &fstest.MapFile{Data: []byte("// js")},
	}
	fileServer := http.FileServer(http.FS(fsys))
	handler := spaHandler(fileServer, fsys)

	t.Run("root serves index.html", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rw := httptest.NewRecorder()
		handler.ServeHTTP(rw, req)
		if rw.Code != http.StatusOK {
			t.Fatalf("/ status = %d", rw.Code)
		}
		if !strings.Contains(rw.Body.String(), "index") {
			t.Fatalf("body = %q", rw.Body.String())
		}
	})

	t.Run("existing file served directly", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
		rw := httptest.NewRecorder()
		handler.ServeHTTP(rw, req)
		if rw.Code != http.StatusOK {
			t.Fatalf("/app.js status = %d", rw.Code)
		}
		if !strings.Contains(rw.Body.String(), "// js") {
			t.Fatalf("body = %q", rw.Body.String())
		}
	})

	t.Run("unknown path falls back to index.html", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/some/deep/route", nil)
		rw := httptest.NewRecorder()
		handler.ServeHTTP(rw, req)
		if rw.Code != http.StatusOK {
			t.Fatalf("/some/deep/route status = %d", rw.Code)
		}
		if !strings.Contains(rw.Body.String(), "index") {
			t.Fatalf("expected index.html fallback, got: %s", rw.Body.String())
		}
	})
}

func TestReadinessHandlerNotReady(t *testing.T) {
	pool, err := pgxpool.New(context.Background(),
		"host=127.0.0.1 port=1 user=x password=x dbname=x sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	req := httptest.NewRequest(http.MethodGet, "/readiness", nil)
	rw := httptest.NewRecorder()
	readinessHandler(pool).ServeHTTP(rw, req)
	if rw.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503 when DB is unreachable", rw.Code)
	}
}

func TestFrontendConfigHandler(t *testing.T) {
	cfg := serverConfig{AuthMode: "dev", AuthServerURL: "http://auth.example.com"}
	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	rw := httptest.NewRecorder()
	frontendConfigHandler(cfg).ServeHTTP(rw, req)

	if rw.Code != http.StatusOK {
		t.Fatalf("status = %d", rw.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rw.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["authMode"] != "dev" || body["authBaseUrl"] != "http://auth.example.com" {
		t.Fatalf("body = %v", body)
	}
}

func TestRequestLogMiddleware(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rw := httptest.NewRecorder()
	requestLogMiddleware(log, inner).ServeHTTP(rw, req)
	if !called {
		t.Error("requestLogMiddleware did not call the next handler")
	}
}

func TestRecoverMiddleware(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	panicking := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("test panic")
	})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rw := httptest.NewRecorder()
	recoverMiddleware(log, panicking).ServeHTTP(rw, req)

	if rw.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500 after panic recovery", rw.Code)
	}
	var body map[string]string
	if err := json.NewDecoder(rw.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] == "" {
		t.Error("response should include an error field")
	}
}

func TestParseLogLevelAllValues(t *testing.T) {
	cases := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
	}
	for _, tc := range cases {
		got, err := parseLogLevel(tc.input)
		if err != nil || got != tc.want {
			t.Errorf("parseLogLevel(%q) = %v, %v; want %v", tc.input, got, err, tc.want)
		}
	}
	if _, err := parseLogLevel("invalid"); err == nil {
		t.Error("invalid level should return an error")
	}
}

func TestBuildTokenVerifierDevMode(t *testing.T) {
	cfg := serverConfig{
		AuthMode:     "dev",
		DevToken:     "test-dev-token",
		DevUserID:    42,
		DevUserEmail: "dev@localhost",
	}
	verifier, err := buildTokenVerifier(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	user, err := verifier.VerifyBearerToken(context.Background(), "test-dev-token")
	if err != nil || user == nil || user.AppUserID != 42 {
		t.Fatalf("VerifyBearerToken: user=%v err=%v", user, err)
	}
	if _, err := verifier.VerifyBearerToken(context.Background(), "wrong"); err == nil {
		t.Error("wrong token should be rejected")
	}
	for _, scope := range []string{"notes:read", "notes:write", "notes:mcp"} {
		if !user.HasScope(scope) {
			t.Errorf("dev user missing scope %q", scope)
		}
	}
}

// Ensure authadapter import is used by package-level helper that may be used
// by future server tests requiring a stub verifier.
var _ authadapter.TokenVerifier = (*stubTokenVerifier)(nil)

type stubTokenVerifier struct {
	user *authadapter.AuthenticatedUser
	err  error
}

func (s *stubTokenVerifier) VerifyBearerToken(_ context.Context, _ string) (*authadapter.AuthenticatedUser, error) {
	return s.user, s.err
}

func TestPublicHandlers(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		status  int
	}{
		{name: "health", handler: healthHandler(nil), status: http.StatusOK},
		{name: "app info", handler: appInfoHandler, status: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			test.handler(response, httptest.NewRequest(http.MethodGet, "/", nil))
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body["status"] == "" && body["app"] == "" {
				t.Fatalf("unexpected response: %v", body)
			}
		})
	}
}
