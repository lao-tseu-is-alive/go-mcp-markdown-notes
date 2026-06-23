package notes

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresRepository implements Repository with pgx.
//
// Row scanning for Note uses pgx named scanning (RowToStructByNameLax /
// RowToAddrOfStructByNameLax + CollectRows). See scanNote, collectNotes,
// collectSearchResults and the `db` tags on the Note struct.
type PostgresRepository struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

// NewPostgresRepository builds a PostgresRepository from an existing connection pool. A nil logger falls back to slog.Default.
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

	var id uuid.UUID
	if err := tx.QueryRow(ctx, createNoteSQL, ownerUserID, input.Title, input.BodyMarkdown, input.Category, input.Status).Scan(&id); err != nil {
		return nil, fmt.Errorf("create note: %w", err)
	}
	if err := insertTags(ctx, tx, ownerUserID, id, input.Tags); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create note: %w", err)
	}
	// Re-read via the normal path so the returned note always includes the full
	// denormalized tags and any future columns. This also keeps CreateNote in sync
	// with GetNote without duplicating scan logic.
	return r.GetNote(ctx, ownerUserID, id)
}

func (r *PostgresRepository) GetNote(ctx context.Context, ownerUserID int64, noteID uuid.UUID) (*Note, error) {
	rows, err := r.pool.Query(ctx, getNoteSQL, ownerUserID, noteID)
	if err != nil {
		return nil, fmt.Errorf("get note query: %w", err)
	}
	// Use CollectOneRow + named scanning so GetNote benefits from the same
	// maintainability as list/search (db tags drive field mapping).
	note, err := pgx.CollectOneRow(rows, pgx.RowToAddrOfStructByNameLax[Note])
	if err != nil {
		return nil, mapNotFound(err)
	}
	return note, nil
}

func (r *PostgresRepository) ListRecentNotes(ctx context.Context, ownerUserID int64, limit int) ([]*Note, error) {
	rows, err := r.pool.Query(ctx, listRecentNotesSQL, ownerUserID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent notes: %w", err)
	}
	return collectNotes(rows)
}

func (r *PostgresRepository) SearchNotes(ctx context.Context, ownerUserID int64, filter SearchFilter) (SearchResult, error) {
	rows, err := r.pool.Query(ctx, searchNotesSQL, ownerUserID, filter.Query, filter.Category, filter.Tags, filter.Limit)
	if err != nil {
		return SearchResult{}, fmt.Errorf("search notes: %w", err)
	}
	return collectSearchResults(rows)
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
	if err := tx.QueryRow(ctx, updateNoteSQL, ownerUserID, noteID, input.Title, input.BodyMarkdown, input.Category, input.Status).Scan(&id); err != nil {
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

// execer is the subset of pgx.Tx used by insertTags so it can accept both a pool connection and a transaction.
type execer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

// insertTags writes each tag as a row in note_tags within the caller's transaction, ignoring duplicate (note_id, tag) pairs.
func insertTags(ctx context.Context, q execer, ownerUserID int64, noteID uuid.UUID, tags []string) error {
	for _, tag := range tags {
		if _, err := q.Exec(ctx, insertTagSQL, noteID, ownerUserID, tag); err != nil {
			return fmt.Errorf("insert note tag: %w", err)
		}
	}
	return nil
}

// collectNotes uses pgx named scanning to drain rows into Note structs.
// The cursor is closed even on error.
func collectNotes(rows pgx.Rows) ([]*Note, error) {
	defer rows.Close()
	notes, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByNameLax[Note])
	if err != nil {
		return nil, fmt.Errorf("read notes: %w", err)
	}
	return notes, nil
}

// searchRow augments the note fields with the window aggregate from search queries.
// We duplicate the fields (instead of embedding) for reliable named scanning.
type searchRow struct {
	ID           uuid.UUID  `db:"id"`
	OwnerUserID  int64      `db:"owner_user_id"`
	Title        string     `db:"title"`
	BodyMarkdown string     `db:"body_markdown"`
	Category     string     `db:"category"`
	Status       NoteStatus `db:"status"`
	Tags         []string   `db:"tags"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	TotalSize    int32      `db:"total_count"`
}

// collectSearchResults uses named scanning for the Note part + the total count.
// All rows carry the same total_size (from COUNT(*) OVER()).
func collectSearchResults(rows pgx.Rows) (SearchResult, error) {
	defer rows.Close()
	rowsData, err := pgx.CollectRows(rows, pgx.RowToStructByNameLax[searchRow])
	if err != nil {
		return SearchResult{}, fmt.Errorf("read notes: %w", err)
	}
	result := SearchResult{Notes: make([]*Note, len(rowsData))}
	for i := range rowsData {
		r := rowsData[i]
		result.Notes[i] = &Note{
			ID:           r.ID,
			OwnerUserID:  r.OwnerUserID,
			Title:        r.Title,
			BodyMarkdown: r.BodyMarkdown,
			Category:     r.Category,
			Status:       r.Status,
			Tags:         r.Tags,
			CreatedAt:    r.CreatedAt,
			UpdatedAt:    r.UpdatedAt,
		}
		result.TotalSize = r.TotalSize
	}
	return result, nil
}

// mapNotFound translates pgx.ErrNoRows to the domain ErrNoteNotFound so callers don't depend on the pgx package.
func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNoteNotFound
	}
	return err
}
