package module_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1/notesv1connect"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/authadapter"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/notes/module"
)

// stubVerifier implements authadapter.TokenVerifier for tests.
type stubVerifier struct{}

func (v *stubVerifier) VerifyBearerToken(_ context.Context, _ string) (*authadapter.AuthenticatedUser, error) {
	return nil, authadapter.ErrInvalidToken
}

// testDeps returns a Deps with a lazy (non-connected) pool and a stub verifier.
// The pool never makes real DB calls during construction, so tests do not require a database.
func testDeps(t *testing.T) module.Deps {
	t.Helper()
	pool, err := pgxpool.New(context.Background(),
		"host=localhost port=5432 user=test password=test dbname=test sslmode=disable")
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return module.Deps{
		Pool:     pool,
		Verifier: &stubVerifier{},
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func TestNew_Success(t *testing.T) {
	mod, err := module.New(context.Background(), module.Config{}, testDeps(t))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if mod == nil {
		t.Fatal("New() returned nil module")
	}
}

func TestNew_NilPool(t *testing.T) {
	deps := testDeps(t)
	deps.Pool = nil
	_, err := module.New(context.Background(), module.Config{}, deps)
	if err == nil {
		t.Fatal("New() with nil pool should return an error")
	}
}

func TestNew_NilVerifier(t *testing.T) {
	deps := testDeps(t)
	deps.Verifier = nil
	_, err := module.New(context.Background(), module.Config{}, deps)
	if err == nil {
		t.Fatal("New() with nil verifier should return an error")
	}
}

func TestNew_NilLogger_FallsBackToDefault(t *testing.T) {
	deps := testDeps(t)
	deps.Logger = nil
	mod, err := module.New(context.Background(), module.Config{}, deps)
	if err != nil {
		t.Fatalf("New() with nil logger should succeed: %v", err)
	}
	if mod == nil {
		t.Fatal("New() returned nil module")
	}
}

func TestConnectHandlers_ReturnsOneHandler(t *testing.T) {
	mod, err := module.New(context.Background(), module.Config{}, testDeps(t))
	if err != nil {
		t.Fatal(err)
	}
	handlers := mod.ConnectHandlers()
	if len(handlers) != 1 {
		t.Fatalf("ConnectHandlers() len = %d, want 1", len(handlers))
	}
	want := "/" + notesv1connect.NotesServiceName + "/"
	if handlers[0].Pattern != want {
		t.Errorf("ConnectHandlers()[0].Pattern = %q, want %q", handlers[0].Pattern, want)
	}
	if handlers[0].Handler == nil {
		t.Error("ConnectHandlers()[0].Handler is nil")
	}
}

func TestRoutePatterns(t *testing.T) {
	mod, err := module.New(context.Background(), module.Config{}, testDeps(t))
	if err != nil {
		t.Fatal(err)
	}
	patterns := mod.RoutePatterns()
	if len(patterns) == 0 {
		t.Fatal("RoutePatterns() returned empty slice")
	}
	want := "/" + notesv1connect.NotesServiceName + "/"
	if patterns[0].Pattern != want {
		t.Errorf("RoutePatterns()[0].Pattern = %q, want %q", patterns[0].Pattern, want)
	}
}

func TestConnectPatterns(t *testing.T) {
	mod, err := module.New(context.Background(), module.Config{}, testDeps(t))
	if err != nil {
		t.Fatal(err)
	}
	patterns := mod.ConnectPatterns()
	if len(patterns) == 0 {
		t.Fatal("ConnectPatterns() returned empty slice")
	}
	want := "/" + notesv1connect.NotesServiceName + "/"
	if patterns[0] != want {
		t.Errorf("ConnectPatterns()[0] = %q, want %q", patterns[0], want)
	}
}

func TestRegisterRoutes_MountsHandlerOnMux(t *testing.T) {
	mod, err := module.New(context.Background(), module.Config{}, testDeps(t))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	if err := mod.RegisterRoutes(mux); err != nil {
		t.Fatalf("RegisterRoutes() error = %v", err)
	}

	// Probe a Connect endpoint; without a valid token we expect 401, not 404 or 500.
	path := notesv1connect.NotesServiceListRecentNotesProcedure
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")
	rw := httptest.NewRecorder()
	mux.ServeHTTP(rw, req)

	if rw.Code == http.StatusNotFound {
		t.Fatalf("RegisterRoutes did not mount the handler: got 404 for %s", path)
	}
}

func TestStartStop_AreNoOps(t *testing.T) {
	mod, err := module.New(context.Background(), module.Config{}, testDeps(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := mod.Start(context.Background()); err != nil {
		t.Errorf("Start() error = %v", err)
	}
	if err := mod.Stop(context.Background()); err != nil {
		t.Errorf("Stop() error = %v", err)
	}
}

// --- migrate.go unit tests (no database required) ---

func TestModuleParseDBMateUp_ParsesMigration(t *testing.T) {
	content, err := module.Migrations.ReadFile("db/migrations/000001_create_notes.sql")
	if err != nil {
		t.Fatal(err)
	}
	statements, err := module.ParseDBMateUp(string(content))
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 7 {
		t.Fatalf("statement count = %d, want 7", len(statements))
	}
	joined := strings.Join(statements, "\n")
	if strings.Contains(joined, "migrate:down") || !strings.Contains(joined, "CREATE FUNCTION set_notes_updated_at") {
		t.Fatalf("unexpected parsed migration:\n%s", joined)
	}
}

func TestModuleParseDBMateUp_RejectsMalformedInput(t *testing.T) {
	tests := []string{
		"SELECT 1;",
		"-- migrate:up\n-- migrate:statementbegin\nSELECT 1;",
		"-- migrate:up\n-- migrate:statementend",
	}
	for _, input := range tests {
		if _, err := module.ParseDBMateUp(input); err == nil {
			t.Fatalf("ParseDBMateUp(%q) succeeded, want error", input)
		}
	}
}
