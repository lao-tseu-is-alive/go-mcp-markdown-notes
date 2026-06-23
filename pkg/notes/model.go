package notes

import (
	"time"

	"github.com/google/uuid"
)

// NoteStatus represents the lifecycle state of a note (mirrors NoteStatus proto enum).
type NoteStatus int32

const (
	NoteStatusUnspecified NoteStatus = 0
	NoteStatusDraft       NoteStatus = 1
	NoteStatusActive      NoteStatus = 2
	NoteStatusFinal       NoteStatus = 3
	NoteStatusArchived    NoteStatus = 4
)

// Note is the transport-independent representation of a Markdown note.
//
// The `db` struct tags are used by pgx named scanning (RowToStructByNameLax) in
// the repository. Keep them in sync with the column names/aliases produced by
// the queries in sql.go (especially noteColumns and searchNoteColumns).
//
// When adding a persistent field:
//  1. Add a migration (see pkg/notes/module/db/migrations)
//  2. Add the column (with alias if needed) to noteColumns / searchNoteColumns
//  3. Add the Go field here with `json` and `db` tags
//  4. Update mappers.go (DomainNoteToProto / ProtoNoteToDomain)
//  5. Handle the field in service.go (normalization, Create/UpdateInput) if user-visible
//  6. Update any callers (ConnectServer, MCP inputs, notes-client, frontend)
//
// The proto definition (proto/notes/v1/notes.proto) is the contract for the wire API,
// but the DB model and internal Note can (and sometimes must) diverge slightly.
type Note struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	OwnerUserID  int64      `json:"owner_user_id" db:"owner_user_id"`
	Title        string     `json:"title" db:"title"`
	BodyMarkdown string     `json:"body_markdown" db:"body_markdown"`
	Category     string     `json:"category" db:"category"`
	Status       NoteStatus `json:"status" db:"status"`
	Tags         []string   `json:"tags" db:"tags"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateNoteInput contains client-controlled fields for a new note.
type CreateNoteInput struct {
	Title        string
	BodyMarkdown string
	Category     string
	Tags         []string
	Status       NoteStatus
}

// UpdateNoteInput contains the complete client-controlled state of a note.
type UpdateNoteInput struct {
	Title        string
	BodyMarkdown string
	Category     string
	Tags         []string
	Status       NoteStatus
}

// SearchFilter controls note search for one owner.
type SearchFilter struct {
	Query    string
	Tags     []string
	Category string
	Limit    int
	Offset   int
}

// SearchResult holds the matching notes and the total count before pagination.
type SearchResult struct {
	Notes     []*Note
	TotalSize int32
}
