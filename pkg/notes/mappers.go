package notes

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	notesv1 "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1"
)

func DomainNoteToProto(note *Note) *notesv1.Note {
	if note == nil {
		return nil
	}
	return &notesv1.Note{
		Id:           note.ID.String(),
		OwnerUserId:  strconv.FormatInt(note.OwnerUserID, 10),
		Title:        note.Title,
		BodyMarkdown: note.BodyMarkdown,
		Category:     note.Category,
		Tags:         append([]string(nil), note.Tags...),
		CreatedAt:    note.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:    note.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func DomainNotesToProto(notes []*Note) []*notesv1.Note {
	result := make([]*notesv1.Note, 0, len(notes))
	for _, note := range notes {
		result = append(result, DomainNoteToProto(note))
	}
	return result
}

func ProtoNoteToDomain(note *notesv1.Note) (*Note, error) {
	if note == nil {
		return nil, nil
	}
	id, err := uuid.Parse(note.Id)
	if err != nil {
		return nil, fmt.Errorf("parse note ID: %w", err)
	}
	ownerUserID, err := strconv.ParseInt(note.OwnerUserId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse owner user ID: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, note.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, note.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}
	return &Note{
		ID:           id,
		OwnerUserID:  ownerUserID,
		Title:        note.Title,
		BodyMarkdown: note.BodyMarkdown,
		Category:     note.Category,
		Tags:         append([]string(nil), note.Tags...),
		CreatedAt:    createdAt,
		UpdatedAt:    updatedAt,
	}, nil
}
