package notes

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type recordingRepository struct {
	ownerUserID int64
	createInput CreateNoteInput
	addedTags   []string
	search      SearchFilter
	limit       int
	deletedID   uuid.UUID
	note        *Note
	err         error
}

func (r *recordingRepository) CreateNote(_ context.Context, ownerUserID int64, input CreateNoteInput) (*Note, error) {
	r.ownerUserID, r.createInput = ownerUserID, input
	return r.note, r.err
}
func (r *recordingRepository) GetNote(_ context.Context, ownerUserID int64, _ uuid.UUID) (*Note, error) {
	r.ownerUserID = ownerUserID
	return r.note, r.err
}
func (r *recordingRepository) ListRecentNotes(_ context.Context, ownerUserID int64, limit int) ([]*Note, error) {
	r.ownerUserID, r.limit = ownerUserID, limit
	return []*Note{r.note}, r.err
}
func (r *recordingRepository) SearchNotes(_ context.Context, ownerUserID int64, filter SearchFilter) ([]*Note, error) {
	r.ownerUserID, r.search = ownerUserID, filter
	return []*Note{r.note}, r.err
}
func (r *recordingRepository) AddTags(_ context.Context, ownerUserID int64, _ uuid.UUID, tags []string) (*Note, error) {
	r.ownerUserID, r.addedTags = ownerUserID, tags
	return r.note, r.err
}
func (r *recordingRepository) UpdateNote(_ context.Context, ownerUserID int64, _ uuid.UUID, _ UpdateNoteInput) (*Note, error) {
	r.ownerUserID = ownerUserID
	return r.note, r.err
}
func (r *recordingRepository) DeleteNote(_ context.Context, ownerUserID int64, noteID uuid.UUID) error {
	r.ownerUserID, r.deletedID = ownerUserID, noteID
	return r.err
}

func newTestService(t *testing.T, repository Repository) *Service {
	t.Helper()
	service, err := NewService(repository, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func TestServiceCreateNoteNormalizesInputAndScopesOwner(t *testing.T) {
	repository := &recordingRepository{note: &Note{ID: uuid.New(), OwnerUserID: 42}}
	service := newTestService(t, repository)

	_, err := service.CreateNote(context.Background(), 42, CreateNoteInput{
		Title: "  Architecture  ", Tags: []string{" Go ", "go", "MCP"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if repository.ownerUserID != 42 {
		t.Fatalf("owner ID = %d, want 42", repository.ownerUserID)
	}
	if repository.createInput.Title != "Architecture" {
		t.Fatalf("title = %q", repository.createInput.Title)
	}
	if got := strings.Join(repository.createInput.Tags, ","); got != "go,mcp" {
		t.Fatalf("tags = %q, want go,mcp", got)
	}
}

func TestServiceRejectsInvalidNoteInput(t *testing.T) {
	service := newTestService(t, &recordingRepository{})
	tooManyTags := make([]string, MaxTags+1)
	for i := range tooManyTags {
		tooManyTags[i] = string(rune('a' + i))
	}
	tests := []struct {
		name  string
		input CreateNoteInput
	}{
		{name: "empty title", input: CreateNoteInput{Title: "  "}},
		{name: "long title", input: CreateNoteInput{Title: strings.Repeat("x", MaxTitleLength+1)}},
		{name: "long body", input: CreateNoteInput{Title: "ok", BodyMarkdown: strings.Repeat("x", MaxBodyLength+1)}},
		{name: "too many tags", input: CreateNoteInput{Title: "ok", Tags: tooManyTags}},
		{name: "long tag", input: CreateNoteInput{Title: "ok", Tags: []string{strings.Repeat("x", MaxTagLength+1)}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.CreateNote(context.Background(), 1, test.input)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("error = %v, want ErrInvalidInput", err)
			}
		})
	}
}

func TestServiceRequiresAuthenticatedOwner(t *testing.T) {
	service := newTestService(t, &recordingRepository{})
	_, err := service.ListRecentNotes(context.Background(), 0, 10)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("error = %v, want ErrUnauthenticated", err)
	}
}

func TestServiceEnforcesAndDefaultsLimits(t *testing.T) {
	repository := &recordingRepository{note: &Note{ID: uuid.New()}}
	service := newTestService(t, repository)

	if _, err := service.ListRecentNotes(context.Background(), 7, 0); err != nil {
		t.Fatal(err)
	}
	if repository.limit != DefaultLimit {
		t.Fatalf("limit = %d, want %d", repository.limit, DefaultLimit)
	}
	if _, err := service.SearchNotes(context.Background(), 7, SearchFilter{Limit: MaxLimit + 1}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}

func TestServiceSearchNormalizesFilters(t *testing.T) {
	repository := &recordingRepository{note: &Note{ID: uuid.New()}}
	service := newTestService(t, repository)

	_, err := service.SearchNotes(context.Background(), 9, SearchFilter{
		Query: "  connect ", Category: " architecture ", Tags: []string{" Go ", "go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if repository.search.Query != "connect" || repository.search.Category != "architecture" {
		t.Fatalf("filter = %#v", repository.search)
	}
	if repository.search.Limit != DefaultLimit || len(repository.search.Tags) != 1 || repository.search.Tags[0] != "go" {
		t.Fatalf("filter = %#v", repository.search)
	}
}

func TestServiceDeleteNote(t *testing.T) {
	repository := &recordingRepository{}
	service := newTestService(t, repository)
	noteID := uuid.New()

	if err := service.DeleteNote(context.Background(), 42, noteID); err != nil {
		t.Fatal(err)
	}
	if repository.ownerUserID != 42 || repository.deletedID != noteID {
		t.Fatalf("delete was not scoped: owner=%d noteID=%v", repository.ownerUserID, repository.deletedID)
	}

	if err := service.DeleteNote(context.Background(), 0, noteID); !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("error = %v, want ErrUnauthenticated", err)
	}
	if err := service.DeleteNote(context.Background(), 42, uuid.Nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}

	// A repository miss (wrong owner or unknown id) surfaces as not-found.
	repository.err = ErrNoteNotFound
	if err := service.DeleteNote(context.Background(), 42, noteID); !errors.Is(err, ErrNoteNotFound) {
		t.Fatalf("error = %v, want ErrNoteNotFound", err)
	}
}

func TestServiceAddTagsEnforcesResultingTagLimit(t *testing.T) {
	existingTags := make([]string, MaxTags)
	for i := range existingTags {
		existingTags[i] = string(rune('a' + i))
	}
	repository := &recordingRepository{note: &Note{ID: uuid.New(), OwnerUserID: 3, Tags: existingTags}}
	service := newTestService(t, repository)

	_, err := service.AddTags(context.Background(), 3, repository.note.ID, []string{"new-tag"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("error = %v, want ErrInvalidInput", err)
	}
}
