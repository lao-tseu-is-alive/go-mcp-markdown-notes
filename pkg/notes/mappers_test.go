package notes

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	notesv1 "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1"
)

func TestDomainNotesToProto(t *testing.T) {
	notes := []*Note{
		{ID: uuid.New(), OwnerUserID: 1, Title: "a"},
		{ID: uuid.New(), OwnerUserID: 1, Title: "b"},
	}
	protos := DomainNotesToProto(notes)
	if len(protos) != 2 || protos[0].Title != "a" || protos[1].Title != "b" {
		t.Fatalf("DomainNotesToProto = %v", protos)
	}
	if got := DomainNotesToProto(nil); len(got) != 0 {
		t.Fatalf("nil input: len = %d, want 0", len(got))
	}
}

func TestDomainNoteToProtoNil(t *testing.T) {
	if got := DomainNoteToProto(nil); got != nil {
		t.Fatalf("DomainNoteToProto(nil) = %v, want nil", got)
	}
}

func TestProtoNoteToDomainErrors(t *testing.T) {
	if got, err := ProtoNoteToDomain(nil); got != nil || err != nil {
		t.Fatalf("nil note: got=%v err=%v", got, err)
	}
	if _, err := ProtoNoteToDomain(&notesv1.Note{Id: "bad-uuid"}); err == nil {
		t.Error("bad UUID should return an error")
	}
}

func TestNoteProtoRoundTrip(t *testing.T) {
	now := time.Date(2026, 6, 8, 10, 30, 0, 123, time.UTC)
	want := &Note{
		ID:           uuid.New(),
		OwnerUserID:  123,
		Title:        "title",
		BodyMarkdown: "body",
		Category:     "architecture",
		Status:       NoteStatusActive,
		Tags:         []string{"go", "mcp"},
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	got, err := ProtoNoteToDomain(DomainNoteToProto(want))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip = %#v, want %#v", got, want)
	}
}

// TestNoteProtoRoundTripAllStatuses exercises every NoteStatus value through the
// mapper. This acts as a basic drift detector when the proto enum or domain
// NoteStatus constants are changed.
func TestNoteProtoRoundTripAllStatuses(t *testing.T) {
	statuses := []struct {
		val  NoteStatus
		name string
	}{
		{NoteStatusDraft, "draft"},
		{NoteStatusActive, "active"},
		{NoteStatusFinal, "final"},
		{NoteStatusArchived, "archived"},
	}
	for _, tc := range statuses {
		t.Run(tc.name, func(t *testing.T) {
			want := &Note{
				ID: uuid.New(), OwnerUserID: 1, Title: "x", BodyMarkdown: "y",
				Status: tc.val, CreatedAt: time.Now(), UpdatedAt: time.Now(),
			}
			pb := DomainNoteToProto(want)
			got, err := ProtoNoteToDomain(pb)
			if err != nil {
				t.Fatal(err)
			}
			if got.Status != tc.val {
				t.Errorf("status roundtrip: got %v, want %v", got.Status, tc.val)
			}
		})
	}
}

// TestNoteDBTagsPresent is a cheap drift detector. It ensures that the fields
// referenced by our noteColumns (the columns we actually select and scan) have
// corresponding `db:"..."` tags. This catches cases where a new column is added
// to SQL but forgotten on the struct (which would break named scanning).
func TestNoteDBTagsPresent(t *testing.T) {
	// Columns we project in normal reads (from sql.go noteColumns).
	expectedDBCols := []string{
		"id", "owner_user_id", "title", "body_markdown", "category", "status", "tags",
		"created_at", "updated_at",
	}

	typ := reflect.TypeOf(Note{})
	seen := map[string]bool{}
	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i)
		tag := f.Tag.Get("db")
		if tag != "" {
			seen[tag] = true
		}
	}

	for _, col := range expectedDBCols {
		if !seen[col] {
			t.Errorf("missing db tag for column %q on Note struct (add field + `db:%q` tag)", col, col)
		}
	}

	// Also sanity-check that we don't have obviously wrong tags.
	for _, col := range expectedDBCols {
		if strings.TrimSpace(col) == "" {
			t.Error("empty column name in expected list")
		}
	}
}
