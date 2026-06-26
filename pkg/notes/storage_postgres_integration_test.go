package notes_test

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/notes"
	notesmodule "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/pkg/notes/module"
)

var integrationOwnerSeq atomic.Int64

func integrationDatabaseURL(t *testing.T) string {
	t.Helper()
	if value := strings.TrimSpace(os.Getenv("DATABASE_URL")); value != "" {
		return value
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		return ""
	}
	host := integrationEnvDefault("DB_HOST", "127.0.0.1")
	port := integrationEnvDefault("DB_PORT", "5432")
	name := integrationEnvDefault("DB_NAME", "go_mcp_notes")
	user := integrationEnvDefault("DB_USER", "go_mcp_notes")
	sslMode := integrationEnvDefault("DB_SSL_MODE", "prefer")
	result := &url.URL{
		Scheme:   "postgres",
		Host:     net.JoinHostPort(host, port),
		Path:     name,
		RawQuery: url.Values{"sslmode": []string{sslMode}}.Encode(),
		User:     url.UserPassword(user, password),
	}
	return result.String()
}

func integrationEnvDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func openIntegrationPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := integrationDatabaseURL(t)
	if dsn == "" {
		t.Skip("postgres integration tests require DATABASE_URL or DB_PASSWORD")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("postgres not available: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("postgres not available: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := notesmodule.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return pool
}

func newIntegrationRepository(t *testing.T) (*notes.PostgresRepository, int64) {
	t.Helper()
	pool := openIntegrationPool(t)
	repo, err := notes.NewPostgresRepository(pool, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Use a run-unique owner ID so repeated `make test` against a persistent local
	// database does not pick up rows left by earlier runs.
	ownerID := time.Now().UnixNano() + integrationOwnerSeq.Add(1)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = pool.Exec(ctx, "DELETE FROM notes WHERE owner_user_id = $1", ownerID)
	})
	return repo, ownerID
}

func createIntegrationNote(t *testing.T, repo *notes.PostgresRepository, ownerID int64, input notes.CreateNoteInput) *notes.Note {
	t.Helper()
	if input.Status == notes.NoteStatusUnspecified {
		input.Status = notes.NoteStatusActive
	}
	note, err := repo.CreateNote(context.Background(), ownerID, input)
	if err != nil {
		t.Fatalf("CreateNote: %v", err)
	}
	return note
}

func TestPostgresRepository_CreateAndGetNote(t *testing.T) {
	repo, ownerID := newIntegrationRepository(t)
	created := createIntegrationNote(t, repo, ownerID, notes.CreateNoteInput{
		Title:        "Integration title",
		BodyMarkdown: "Body with **markdown**",
		Category:     "testing",
		Tags:         []string{"go", "postgres"},
		Status:       notes.NoteStatusDraft,
	})

	got, err := repo.GetNote(context.Background(), ownerID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != created.Title || got.BodyMarkdown != created.BodyMarkdown {
		t.Fatalf("note content mismatch: got %+v", got)
	}
	if got.Category != "testing" || got.Status != notes.NoteStatusDraft {
		t.Fatalf("category/status = %q/%d, want testing/%d", got.Category, got.Status, notes.NoteStatusDraft)
	}
	if strings.Join(got.Tags, ",") != "go,postgres" {
		t.Fatalf("tags = %v, want [go postgres]", got.Tags)
	}
}

func TestPostgresRepository_OwnerIsolation(t *testing.T) {
	repo, ownerA := newIntegrationRepository(t)
	_, ownerB := newIntegrationRepository(t)
	created := createIntegrationNote(t, repo, ownerA, notes.CreateNoteInput{Title: "private"})

	_, err := repo.GetNote(context.Background(), ownerB, created.ID)
	if !errors.Is(err, notes.ErrNoteNotFound) {
		t.Fatalf("GetNote() err = %v, want ErrNoteNotFound", err)
	}
}

func TestPostgresRepository_SearchNotesFilters(t *testing.T) {
	repo, ownerID := newIntegrationRepository(t)
	runID := uuid.NewString()
	workCategory := "work-" + runID
	personalCategory := "personal-" + runID
	queryMarker := "needle-" + runID
	tagPair := []string{"go-" + runID, "mcp-" + runID}

	createIntegrationNote(t, repo, ownerID, notes.CreateNoteInput{
		Title: "Alpha note", Category: workCategory, Tags: tagPair,
	})
	createIntegrationNote(t, repo, ownerID, notes.CreateNoteInput{
		Title: "Beta note", Category: personalCategory, Tags: []string{"go-" + runID},
	})
	createIntegrationNote(t, repo, ownerID, notes.CreateNoteInput{
		Title: "Gamma", BodyMarkdown: "contains " + queryMarker, Category: workCategory, Tags: []string{"rust-" + runID},
	})

	ctx := context.Background()

	byQuery, err := repo.SearchNotes(ctx, ownerID, notes.SearchFilter{Query: queryMarker, Limit: 10})
	if err != nil || len(byQuery.Notes) != 1 || byQuery.Notes[0].Title != "Gamma" {
		t.Fatalf("query search = %+v err=%v", byQuery, err)
	}

	byCategory, err := repo.SearchNotes(ctx, ownerID, notes.SearchFilter{Category: workCategory, Limit: 10})
	if err != nil || len(byCategory.Notes) != 2 {
		t.Fatalf("category search count = %d, want 2", len(byCategory.Notes))
	}

	byTags, err := repo.SearchNotes(ctx, ownerID, notes.SearchFilter{Tags: tagPair, Limit: 10})
	if err != nil || len(byTags.Notes) != 1 || byTags.Notes[0].Title != "Alpha note" {
		t.Fatalf("tag search = %+v err=%v", byTags, err)
	}
}

func TestPostgresRepository_SearchNotesPagination(t *testing.T) {
	repo, ownerID := newIntegrationRepository(t)
	category := "pagination-" + uuid.NewString()
	for i := range 5 {
		createIntegrationNote(t, repo, ownerID, notes.CreateNoteInput{
			Title:    "Paged note",
			Category: category,
			Tags:     []string{"page-" + uuid.NewString()},
		})
		if i < 4 {
			time.Sleep(2 * time.Millisecond)
		}
	}

	ctx := context.Background()
	page1, err := repo.SearchNotes(ctx, ownerID, notes.SearchFilter{Category: category, Limit: 2, Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(page1.Notes) != 2 || page1.TotalSize != 5 {
		t.Fatalf("page1 = len %d total %d, want len 2 total 5", len(page1.Notes), page1.TotalSize)
	}

	page2, err := repo.SearchNotes(ctx, ownerID, notes.SearchFilter{Category: category, Limit: 2, Offset: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Notes) != 2 {
		t.Fatalf("page2 len = %d, want 2", len(page2.Notes))
	}
	if page1.Notes[0].ID == page2.Notes[0].ID {
		t.Fatal("page2 overlapped page1")
	}

	page3, err := repo.SearchNotes(ctx, ownerID, notes.SearchFilter{Category: category, Limit: 2, Offset: 4})
	if err != nil {
		t.Fatal(err)
	}
	if len(page3.Notes) != 1 {
		t.Fatalf("page3 len = %d, want 1", len(page3.Notes))
	}
}

func TestPostgresRepository_ListRecentNotes(t *testing.T) {
	repo, ownerID := newIntegrationRepository(t)
	older := createIntegrationNote(t, repo, ownerID, notes.CreateNoteInput{Title: "older"})
	newer := createIntegrationNote(t, repo, ownerID, notes.CreateNoteInput{Title: "newer"})

	_, err := repo.UpdateNote(context.Background(), ownerID, older.ID, notes.UpdateNoteInput{
		Title: "older touched", Tags: []string{"touch"}, Status: notes.NoteStatusActive,
	})
	if err != nil {
		t.Fatal(err)
	}

	recent, err := repo.ListRecentNotes(context.Background(), ownerID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(recent.Notes) < 2 || recent.Notes[0].ID != older.ID {
		t.Fatalf("recent order = %v, want %s first", noteIDs(recent.Notes), older.ID)
	}
	_ = newer
}

func TestPostgresRepository_AddTags(t *testing.T) {
	repo, ownerID := newIntegrationRepository(t)
	created := createIntegrationNote(t, repo, ownerID, notes.CreateNoteInput{
		Title: "tagged", Tags: []string{"alpha"},
	})

	updated, err := repo.AddTags(context.Background(), ownerID, created.ID, []string{"beta", "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(updated.Tags, ",") != "alpha,beta" {
		t.Fatalf("tags after add = %v", updated.Tags)
	}
}

func TestPostgresRepository_AddTagsRejectsOverflow(t *testing.T) {
	repo, ownerID := newIntegrationRepository(t)
	tags := make([]string, notes.MaxTags)
	for i := range tags {
		tags[i] = "tag" + string(rune('a'+i))
	}
	created := createIntegrationNote(t, repo, ownerID, notes.CreateNoteInput{Title: "full", Tags: tags})

	_, err := repo.AddTags(context.Background(), ownerID, created.ID, []string{"overflow"})
	if !errors.Is(err, notes.ErrInvalidInput) {
		t.Fatalf("AddTags() err = %v, want ErrInvalidInput", err)
	}
}

func TestPostgresRepository_UpdateAndDeleteNote(t *testing.T) {
	repo, ownerID := newIntegrationRepository(t)
	created := createIntegrationNote(t, repo, ownerID, notes.CreateNoteInput{
		Title: "before", Tags: []string{"old"}, Status: notes.NoteStatusActive,
	})

	updated, err := repo.UpdateNote(context.Background(), ownerID, created.ID, notes.UpdateNoteInput{
		Title: "after", BodyMarkdown: "new body", Category: "updated",
		Tags: []string{"new"}, Status: notes.NoteStatusArchived,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Title != "after" || updated.Status != notes.NoteStatusArchived || strings.Join(updated.Tags, ",") != "new" {
		t.Fatalf("updated note = %+v", updated)
	}

	if err := repo.DeleteNote(context.Background(), ownerID, created.ID); err != nil {
		t.Fatal(err)
	}
	_, err = repo.GetNote(context.Background(), ownerID, created.ID)
	if !errors.Is(err, notes.ErrNoteNotFound) {
		t.Fatalf("GetNote after delete err = %v, want ErrNoteNotFound", err)
	}
}

func TestPostgresRepository_GetNoteNotFound(t *testing.T) {
	repo, ownerID := newIntegrationRepository(t)
	_, err := repo.GetNote(context.Background(), ownerID, uuid.New())
	if !errors.Is(err, notes.ErrNoteNotFound) {
		t.Fatalf("GetNote() err = %v, want ErrNoteNotFound", err)
	}
}

func noteIDs(notesList []*notes.Note) []uuid.UUID {
	ids := make([]uuid.UUID, len(notesList))
	for i, note := range notesList {
		ids[i] = note.ID
	}
	return ids
}
