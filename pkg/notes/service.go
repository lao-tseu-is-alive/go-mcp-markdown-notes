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
	MaxTitleLength = 200
	MaxBodyLength  = 200_000
	MaxTags        = 20
	MaxTagLength   = 50
	DefaultLimit   = 10
	MaxLimit       = 50
)

// Service contains transport-independent note business logic.
type Service struct {
	repository Repository
	log        *slog.Logger
}

func NewService(repository Repository, log *slog.Logger) (*Service, error) {
	if repository == nil {
		return nil, fmt.Errorf("%w: repository is required", ErrInvalidInput)
	}
	if log == nil {
		log = slog.Default()
	}
	return &Service{repository: repository, log: log}, nil
}

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

func (s *Service) ListRecentNotes(ctx context.Context, ownerUserID int64, limit int) ([]*Note, error) {
	if err := validateOwner(ownerUserID); err != nil {
		return nil, err
	}
	limit, err := normalizeLimit(limit)
	if err != nil {
		return nil, err
	}
	notes, err := s.repository.ListRecentNotes(ctx, ownerUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent notes: %w", err)
	}
	return notes, nil
}

func (s *Service) SearchNotes(ctx context.Context, ownerUserID int64, filter SearchFilter) ([]*Note, error) {
	if err := validateOwner(ownerUserID); err != nil {
		return nil, err
	}
	limit, err := normalizeLimit(filter.Limit)
	if err != nil {
		return nil, err
	}
	tags, err := normalizeTags(filter.Tags)
	if err != nil {
		return nil, err
	}
	filter.Query = strings.TrimSpace(filter.Query)
	filter.Category = strings.TrimSpace(filter.Category)
	filter.Tags = tags
	filter.Limit = limit

	notes, err := s.repository.SearchNotes(ctx, ownerUserID, filter)
	if err != nil {
		return nil, fmt.Errorf("search notes: %w", err)
	}
	return notes, nil
}

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
	return CreateNoteInput{Title: title, BodyMarkdown: body, Category: category, Tags: tags}, err
}

func normalizeUpdateInput(input UpdateNoteInput) (UpdateNoteInput, error) {
	title, body, category, tags, err := normalizeNoteFields(input.Title, input.BodyMarkdown, input.Category, input.Tags)
	return UpdateNoteInput{Title: title, BodyMarkdown: body, Category: category, Tags: tags}, err
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
	normalizedTags, err := normalizeTags(tags)
	if err != nil {
		return "", "", "", nil, err
	}
	return title, body, category, normalizedTags, nil
}

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
