// Package module provides an importable Module for the notes domain.
package module

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations holds the embedded SQL migration files for the notes module.
// Bundle callers can use this to inspect migration files directly.
//
//go:embed db/migrations/*.sql
var Migrations embed.FS

// Migrate applies all pending SQL migrations for the notes module.
// It acquires a PostgreSQL advisory lock so only one instance migrates at a time.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("notes module: migration pool is required")
	}
	paths, err := fs.Glob(Migrations, "db/migrations/*.sql")
	if err != nil {
		return fmt.Errorf("notes module: list embedded migrations: %w", err)
	}
	sort.Strings(paths)

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("notes module: acquire migration connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(hashtext('go-mcp-markdown-notes:migrations'))"); err != nil {
		return fmt.Errorf("notes module: acquire migration lock: %w", err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = conn.Exec(unlockCtx, "SELECT pg_advisory_unlock(hashtext('go-mcp-markdown-notes:migrations'))")
	}()

	if _, err := conn.Exec(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`); err != nil {
		return fmt.Errorf("notes module: create schema_migrations table: %w", err)
	}

	for _, path := range paths {
		version := moduleMigrationVersion(path)
		var applied bool
		if err := conn.QueryRow(ctx,
			"SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("notes module: check migration %s: %w", version, err)
		}
		if applied {
			continue
		}
		content, err := Migrations.ReadFile(path)
		if err != nil {
			return fmt.Errorf("notes module: read migration %s: %w", version, err)
		}
		statements, err := ParseDBMateUp(string(content))
		if err != nil {
			return fmt.Errorf("notes module: parse migration %s: %w", version, err)
		}
		if err := moduleApplyMigration(ctx, conn.Conn(), version, statements); err != nil {
			return err
		}
	}
	return nil
}

// moduleMigrationVersion extracts the version prefix (before the first underscore) from a migration file path.
func moduleMigrationVersion(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	version, _, found := strings.Cut(base, "_")
	if found {
		return version
	}
	return base
}

// moduleApplyMigration executes the statements in a transaction and records the version in schema_migrations.
func moduleApplyMigration(ctx context.Context, conn *pgx.Conn, version string, statements []string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("notes module: begin migration %s: %w", version, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("notes module: apply migration %s: %w", version, err)
		}
	}
	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
		return fmt.Errorf("notes module: record migration %s: %w", version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("notes module: commit migration %s: %w", version, err)
	}
	return nil
}

// ParseDBMateUp extracts SQL statements from the "-- migrate:up" section of a DBMate migration file.
// Multi-statement blocks delimited by "-- migrate:statementbegin / end" are emitted as a single string.
// Exported so test code can verify the parser against embedded SQL files.
func ParseDBMateUp(content string) ([]string, error) {
	const (
		upMarker             = "-- migrate:up"
		downMarker           = "-- migrate:down"
		statementBeginMarker = "-- migrate:statementbegin"
		statementEndMarker   = "-- migrate:statementend"
	)
	_, rest, found := strings.Cut(content, upMarker)
	if !found {
		return nil, fmt.Errorf("missing %s marker", upMarker)
	}
	section := rest
	if before, _, cut := strings.Cut(section, downMarker); cut {
		section = before
	}

	var statements []string
	var current strings.Builder
	inStatementBlock := false
	flush := func() {
		statement := strings.TrimSpace(current.String())
		current.Reset()
		if moduleContainsSQL(statement) {
			statements = append(statements, statement)
		}
	}

	for line := range strings.SplitSeq(section, "\n") {
		trimmed := strings.TrimSpace(line)
		switch trimmed {
		case statementBeginMarker:
			if inStatementBlock {
				return nil, fmt.Errorf("nested statement block")
			}
			flush()
			inStatementBlock = true
			continue
		case statementEndMarker:
			if !inStatementBlock {
				return nil, fmt.Errorf("statement end without begin")
			}
			flush()
			inStatementBlock = false
			continue
		}
		current.WriteString(line)
		current.WriteByte('\n')
		if !inStatementBlock && strings.HasSuffix(trimmed, ";") {
			flush()
		}
	}
	if inStatementBlock {
		return nil, fmt.Errorf("unterminated statement block")
	}
	flush()
	if len(statements) == 0 {
		return nil, fmt.Errorf("migration has no up statements")
	}
	return statements, nil
}

// moduleContainsSQL reports whether the statement contains at least one non-blank, non-comment line.
func moduleContainsSQL(statement string) bool {
	for line := range strings.SplitSeq(statement, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
			return true
		}
	}
	return false
}
