package notes

import (
	"context"

	"github.com/google/uuid"
)

// Repository persists notes. Every operation requires an owner ID so ownership
// cannot be accidentally omitted by callers.
type Repository interface {
	CreateNote(ctx context.Context, ownerUserID int64, input CreateNoteInput) (*Note, error)
	GetNote(ctx context.Context, ownerUserID int64, noteID uuid.UUID) (*Note, error)
	ListRecentNotes(ctx context.Context, ownerUserID int64, limit int) ([]*Note, error)
	SearchNotes(ctx context.Context, ownerUserID int64, filter SearchFilter) ([]*Note, error)
	AddTags(ctx context.Context, ownerUserID int64, noteID uuid.UUID, tags []string) (*Note, error)
	UpdateNote(ctx context.Context, ownerUserID int64, noteID uuid.UUID, input UpdateNoteInput) (*Note, error)
	DeleteNote(ctx context.Context, ownerUserID int64, noteID uuid.UUID) error
}
