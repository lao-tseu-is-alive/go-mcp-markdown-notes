package notes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository with pgx.
type PostgresRepository struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func NewPostgresRepository(pool *pgxpool.Pool, log *slog.Logger) (*PostgresRepository, error) {
	if pool == nil {
		return nil, fmt.Errorf("%w: PostgreSQL pool is required", ErrInvalidInput)
	}
	if log == nil {
		log = slog.Default()
	}
	return &PostgresRepository{pool: pool, log: log}, nil
}

func (r *PostgresRepository) CreateNote(ctx context.Context, ownerUserID int64, input CreateNoteInput) (*Note, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create note transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	note := &Note{Tags: append([]string(nil), input.Tags...)}
	err = tx.QueryRow(ctx, createNoteSQL, ownerUserID, input.Title, input.BodyMarkdown, input.Category).Scan(
		&note.ID, &note.OwnerUserID, &note.Title, &note.BodyMarkdown, &note.Category, &note.CreatedAt, &note.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}
	if err := insertTags(ctx, tx, ownerUserID, note.ID, input.Tags); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create note: %w", err)
	}
	return note, nil
}

func (r *PostgresRepository) GetNote(ctx context.Context, ownerUserID int64, noteID uuid.UUID) (*Note, error) {
	return scanNote(r.pool.QueryRow(ctx, getNoteSQL, ownerUserID, noteID))
}

func (r *PostgresRepository) ListRecentNotes(ctx context.Context, ownerUserID int64, limit int) ([]*Note, error) {
	rows, err := r.pool.Query(ctx, listRecentNotesSQL, ownerUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent notes: %w", err)
	}
	return collectNotes(rows)
}

func (r *PostgresRepository) SearchNotes(ctx context.Context, ownerUserID int64, filter SearchFilter) ([]*Note, error) {
	rows, err := r.pool.Query(ctx, searchNotesSQL, ownerUserID, filter.Query, filter.Category, filter.Tags, filter.Limit)
	if err != nil {
		return nil, fmt.Errorf("search notes: %w", err)
	}
	return collectNotes(rows)
}

func (r *PostgresRepository) AddTags(ctx context.Context, ownerUserID int64, noteID uuid.UUID, tags []string) (*Note, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin add tags transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id uuid.UUID
	if err := tx.QueryRow(ctx, touchNoteSQL, ownerUserID, noteID).Scan(&id); err != nil {
		return nil, mapNotFound(err)
	}
	var resultingTagCount int
	if err := tx.QueryRow(ctx, countTagsAfterAddSQL, ownerUserID, noteID, tags).Scan(&resultingTagCount); err != nil {
		return nil, fmt.Errorf("count resulting note tags: %w", err)
	}
	if resultingTagCount > MaxTags {
		return nil, fmt.Errorf("%w: no more than %d tags are allowed", ErrInvalidInput, MaxTags)
	}
	if err := insertTags(ctx, tx, ownerUserID, noteID, tags); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit add tags: %w", err)
	}
	return r.GetNote(ctx, ownerUserID, noteID)
}

func (r *PostgresRepository) UpdateNote(ctx context.Context, ownerUserID int64, noteID uuid.UUID, input UpdateNoteInput) (*Note, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin update note transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id uuid.UUID
	if err := tx.QueryRow(ctx, updateNoteSQL, ownerUserID, noteID, input.Title, input.BodyMarkdown, input.Category).Scan(&id); err != nil {
		return nil, mapNotFound(err)
	}
	if _, err := tx.Exec(ctx, deleteTagsSQL, ownerUserID, noteID); err != nil {
		return nil, fmt.Errorf("replace note tags: %w", err)
	}
	if err := insertTags(ctx, tx, ownerUserID, noteID, input.Tags); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update note: %w", err)
	}
	return r.GetNote(ctx, ownerUserID, noteID)
}

func (r *PostgresRepository) DeleteNote(ctx context.Context, ownerUserID int64, noteID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, deleteNoteSQL, ownerUserID, noteID)
	if err != nil {
		return fmt.Errorf("delete note: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNoteNotFound
	}
	return nil
}

type execer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func insertTags(ctx context.Context, q execer, ownerUserID int64, noteID uuid.UUID, tags []string) error {
	for _, tag := range tags {
		if _, err := q.Exec(ctx, insertTagSQL, noteID, ownerUserID, tag); err != nil {
			return fmt.Errorf("insert note tag: %w", err)
		}
	}
	return nil
}

func scanNote(row pgx.Row) (*Note, error) {
	note := &Note{}
	err := row.Scan(&note.ID, &note.OwnerUserID, &note.Title, &note.BodyMarkdown, &note.Category, &note.Tags, &note.CreatedAt, &note.UpdatedAt)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return note, nil
}

func collectNotes(rows pgx.Rows) ([]*Note, error) {
	defer rows.Close()
	notes := make([]*Note, 0)
	for rows.Next() {
		note, err := scanNote(rows)
		if err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read notes: %w", err)
	}
	return notes, nil
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNoteNotFound
	}
	return err
}
