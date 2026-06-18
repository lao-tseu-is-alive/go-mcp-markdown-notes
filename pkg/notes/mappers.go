package notes

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	notesv1 "github.com/lao-tseu-is-alive/go-mcp-markdown-notes/gen/notes/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DomainNoteToProto converts a domain Note to its protobuf representation, copying the tags slice to avoid aliasing.
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
		CreatedAt:    timestamppb.New(note.CreatedAt),
		UpdatedAt:    timestamppb.New(note.UpdatedAt),
		Status:       notesv1.NoteStatus(note.Status),
	}
}

// DomainNotesToProto maps a slice of domain Notes to their protobuf equivalents.
func DomainNotesToProto(notes []*Note) []*notesv1.Note {
	result := make([]*notesv1.Note, 0, len(notes))
	for _, note := range notes {
		result = append(result, DomainNoteToProto(note))
	}
	return result
}

// ProtoNoteToDomain converts a protobuf Note to the domain model, validating IDs and timestamps.
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
	createdAt, err := protoTimestampToTime("created_at", note.CreatedAt)
	if err != nil {
		return nil, err
	}
	updatedAt, err := protoTimestampToTime("updated_at", note.UpdatedAt)
	if err != nil {
		return nil, err
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
		Status:       NoteStatus(note.Status),
	}, nil
}

func protoTimestampToTime(name string, ts *timestamppb.Timestamp) (time.Time, error) {
	if ts == nil {
		return time.Time{}, fmt.Errorf("%s is required", name)
	}
	if err := ts.CheckValid(); err != nil {
		return time.Time{}, fmt.Errorf("validate %s: %w", name, err)
	}
	return ts.AsTime(), nil
}
