package notes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	// MaxTitleLength is the maximum number of Unicode code points allowed in a note title.
	MaxTitleLength = 200
	// MaxBodyLength is the maximum number of Unicode code points allowed in a note body.
	MaxBodyLength = 200_000
	// MaxCategoryLength is the maximum number of Unicode code points allowed in a note category.
	// This matches the validation in proto/notes/v1/notes.proto.
	MaxCategoryLength = 100
	// MaxTags is the maximum number of tags a single note may carry.
	MaxTags = 20
	// MaxTagLength is the maximum number of Unicode code points allowed in one tag.
	MaxTagLength = 50
	// DefaultLimit is the number of notes returned when the caller omits a limit.
	DefaultLimit = 10
	// MaxLimit is the upper bound on the number of notes that may be requested in a single call.
	MaxLimit = 100
)

// Service contains transport-independent note business logic.
type Service struct {
	repository Repository
	log        *slog.Logger
}

// NewService constructs a Service backed by the given repository. A nil logger falls back to slog.Default.
func NewService(repository Repository, log *slog.Logger) (*Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: repository is required", ErrInvalidInput)
	}
	if log == nil {
		log = slog.Default()
	}
	return &Service{repository: repository, log: log}, nil
}

// CreateNote validates and normalises the input, persists a new note, and logs the creation.
func (s *Service) CreateNote(ctx context.Context, ownerUserID int64, input CreateNoteInput) (*Note, error) {
	if err := validateOwner(ownerUserID); err != nil {
		return nil, err
	}
	normalized, err := normalizeCreateInput(input)
	if err != nil {
		return nil, err
	}
	note, err := s.repository.CreateNote(ctx, ownerUserID, normalized)
	if err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}
	if note == nil {
		return nil, errors.New("create note: repository returned a nil note")
	}
	s.log.Info("created note", "note_id", note.ID, "owner_user_id", ownerUserID)
	return note, nil
}

// GetNote retrieves a single note by ID, enforcing owner isolation.
func (s *Service) GetNote(ctx context.Context, ownerUserID int64, noteID uuid.UUID) (*Note, error) {
	if err := validateOwnerAndNoteID(ownerUserID, noteID); err != nil {
		return nil, err
	}
	note, err := s.repository.GetNote(ctx, ownerUserID, noteID)
	if err != nil {
		return nil, fmt.Errorf("get note: %w", err)
	}
	if note == nil {
		return nil, errors.New("get note: repository returned a nil note")
	}
	return note, nil
}

// ListRecentNotes returns the most recently updated notes for the owner, capped at MaxLimit.
func (s *Service) ListRecentNotes(ctx context.Context, ownerUserID int64, limit int, offset int) (SearchResult, error) {
	if err := validateOwner(ownerUserID); err != nil {
		return SearchResult{}, err
	}
	limit, err := normalizeLimit(limit)
	if err != nil {
		return SearchResult{}, err
	}
	if offset < 0 {
		offset = 0
	}
	result, err := s.repository.ListRecentNotes(ctx, ownerUserID, limit, offset)
	if err != nil {
		return SearchResult{}, fmt.Errorf("list recent notes: %w", err)
	}
	return result, nil
}

// SearchNotes filters notes by free-text query, category, and tags for the given owner.
func (s *Service) SearchNotes(ctx context.Context, ownerUserID int64, filter SearchFilter) (SearchResult, error) {
	if err := validateOwner(ownerUserID); err != nil {
		return SearchResult{}, err
	}
	limit, err := normalizeLimit(filter.Limit)
	if err != nil {
		return SearchResult{}, err
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	tags, err := normalizeTags(filter.Tags)
	if err != nil {
		return SearchResult{}, err
	}
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Category = strings.TrimSpace(filter.Category)
	filter.Tags = tags
	filter.Limit = limit

	result, err := s.repository.SearchNotes(ctx, ownerUserID, filter)
	if err != nil {
		return SearchResult{}, fmt.Errorf("search notes: %w", err)
	}
	return result, nil
}

// AddTags merges the supplied tags with the note's existing tags, deduplicates, and enforces MaxTags.
func (s *Service) AddTags(ctx context.Context, ownerUserID int64, noteID uuid.UUID, tags []string) (*Note, error) {
	if err := validateOwnerAndNoteID(ownerUserID, noteID); err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, fmt.Errorf("%w: at least one tag is required", ErrInvalidInput)
	}
	existing, err := s.repository.GetNote(ctx, ownerUserID, noteID)
	if err != nil {
		return nil, fmt.Errorf("get note before adding tags: %w", err)
	}
	if existing == nil {
		return nil, errors.New("get note before adding tags: repository returned a nil note")
	}
	normalized, err := normalizeTags(append(append([]string(nil), existing.Tags...), tags...))
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		return nil, fmt.Errorf("%w: at least one tag is required", ErrInvalidInput)
	}
	note, err := s.repository.AddTags(ctx, ownerUserID, noteID, normalized)
	if err != nil {
		return nil, fmt.Errorf("add tags: %w", err)
	}
	if note == nil {
		return nil, errors.New("add tags: repository returned a nil note")
	}
	s.log.Info("added note tags", "note_id", noteID, "owner_user_id", ownerUserID, "tag_count", len(normalized))
	return note, nil
}

// UpdateNote replaces all editable fields of a note, including the full tag set.
func (s *Service) UpdateNote(ctx context.Context, ownerUserID int64, noteID uuid.UUID, input UpdateNoteInput) (*Note, error) {
	if err := validateOwnerAndNoteID(ownerUserID, noteID); err != nil {
		return nil, err
	}
	normalized, err := normalizeUpdateInput(input)
	if err != nil {
		return nil, err
	}
	note, err := s.repository.UpdateNote(ctx, ownerUserID, noteID, normalized)
	if err != nil {
		return nil, fmt.Errorf("update note: %w", err)
	}
	if note == nil {
		return nil, errors.New("update note: repository returned a nil note")
	}
	s.log.Info("updated note", "note_id", noteID, "owner_user_id", ownerUserID)
	return note, nil
}

// DeleteNote removes the note and logs the deletion. Returns ErrNoteNotFound if the note does not belong to the owner.
func (s *Service) DeleteNote(ctx context.Context, ownerUserID int64, noteID uuid.UUID) error {
	if err := validateOwnerAndNoteID(ownerUserID, noteID); err != nil {
		return err
	}
	if err := s.repository.DeleteNote(ctx, ownerUserID, noteID); err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	s.log.Info("deleted note", "note_id", noteID, "owner_user_id", ownerUserID)
	return nil
}

func normalizeCreateInput(input CreateNoteInput) (CreateNoteInput, error) {
	title, body, category, tags, err := normalizeNoteFields(input.Title, input.BodyMarkdown, input.Category, input.Tags)
	if err != nil {
		return CreateNoteInput{}, err
	}
	status := input.Status
	if status == NoteStatusUnspecified {
		status = NoteStatusActive
	}
	return CreateNoteInput{Title: title, BodyMarkdown: body, Category: category, Tags: tags, Status: status}, nil
}

func normalizeUpdateInput(input UpdateNoteInput) (UpdateNoteInput, error) {
	title, body, category, tags, err := normalizeNoteFields(input.Title, input.BodyMarkdown, input.Category, input.Tags)
	if err != nil {
		return UpdateNoteInput{}, err
	}
	status := input.Status
	if status == NoteStatusUnspecified {
		status = NoteStatusActive
	}
	return UpdateNoteInput{Title: title, BodyMarkdown: body, Category: category, Tags: tags, Status: status}, nil
}

func normalizeNoteFields(title, body, category string, tags []string) (string, string, string, []string, error) {
	title = strings.TrimSpace(title)
	category = strings.TrimSpace(category)
	if title == "" {
		return "", "", "", nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if utf8.RuneCountInString(title) > MaxTitleLength {
		return "", "", "", nil, fmt.Errorf("%w: title exceeds %d characters", ErrInvalidInput, MaxTitleLength)
	}
	if utf8.RuneCountInString(body) > MaxBodyLength {
		return "", "", "", nil, fmt.Errorf("%w: body_markdown exceeds %d characters", ErrInvalidInput, MaxBodyLength)
	}
	if utf8.RuneCountInString(category) > MaxCategoryLength {
		return "", "", "", nil, fmt.Errorf("%w: category exceeds %d characters", ErrInvalidInput, MaxCategoryLength)
	}
	normalizedTags, err := normalizeTags(tags)
	if err != nil {
		return "", "", "", nil, err
	}
	return title, body, category, normalizedTags, nil
}

// normalizeTags lowercases and trims each tag, drops blank and duplicate values, and enforces MaxTags and MaxTagLength.
func normalizeTags(tags []string) ([]string, error) {
	result := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, raw := range tags {
		tag := strings.ToLower(strings.TrimSpace(raw))
		if tag == "" {
			continue
		}
		if utf8.RuneCountInString(tag) > MaxTagLength {
			return nil, fmt.Errorf("%w: tag exceeds %d characters", ErrInvalidInput, MaxTagLength)
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		result = append(result, tag)
		if len(result) > MaxTags {
			return nil, fmt.Errorf("%w: no more than %d tags are allowed", ErrInvalidInput, MaxTags)
		}
	}
	return result, nil
}

func normalizeLimit(limit int) (int, error) {
	if limit == 0 {
		return DefaultLimit, nil
	}
	if limit < 0 || limit > MaxLimit {
		return 0, fmt.Errorf("%w: limit must be between 1 and %d", ErrInvalidInput, MaxLimit)
	}
	return limit, nil
}

// validateOwner returns ErrUnauthenticated for non-positive IDs, which indicates a missing or anonymous caller.
func validateOwner(ownerUserID int64) error {
	if ownerUserID <= 0 {
		return ErrUnauthenticated
	}
	return nil
}

func validateOwnerAndNoteID(ownerUserID int64, noteID uuid.UUID) error {
	if err := validateOwner(ownerUserID); err != nil {
		return err
	}
	if noteID == uuid.Nil {
		return fmt.Errorf("%w: note ID is required", ErrInvalidInput)
	}
	return nil
}
