package notes

import (
	"time"

	"github.com/google/uuid"
)

// Note is the transport-independent representation of a Markdown note.
type Note struct {
	ID           uuid.UUID `json:"id" db:"id"`
	OwnerUserID  int64     `json:"owner_user_id" db:"owner_user_id"`
	Title        string    `json:"title" db:"title"`
	BodyMarkdown string    `json:"body_markdown" db:"body_markdown"`
	Category     string    `json:"category" db:"category"`
	Tags         []string  `json:"tags" db:"tags"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// CreateNoteInput contains client-controlled fields for a new note.
type CreateNoteInput struct {
	Title        string
	BodyMarkdown string
	Category     string
	Tags         []string
}

// UpdateNoteInput contains the complete client-controlled state of a note.
type UpdateNoteInput struct {
	Title        string
	BodyMarkdown string
	Category     string
	Tags         []string
}

// SearchFilter controls note search for one owner.
type SearchFilter struct {
	Query    string
	Tags     []string
	Category string
	Limit    int
}
