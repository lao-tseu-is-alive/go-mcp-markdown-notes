package main

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

//go:embed db/migrations/*.sql
var migrationFiles embed.FS

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("migration pool is required")
	}
	paths, err := fs.Glob(migrationFiles, "db/migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list embedded migrations: %w", err)
	}
	sort.Strings(paths)

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock(hashtext('go-mcp-markdown-notes:migrations'))"); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
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
		return fmt.Errorf("create schema_migrations table: %w", err)
	}

	for _, path := range paths {
		version := migrationVersion(path)
		var applied bool
		if err := conn.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if applied {
			continue
		}
		content, err := migrationFiles.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}
		statements, err := parseDBMateUp(string(content))
		if err != nil {
			return fmt.Errorf("parse migration %s: %w", version, err)
		}
		if err := applyMigration(ctx, conn.Conn(), version, statements); err != nil {
			return err
		}
	}
	return nil
}

func migrationVersion(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	version, _, found := strings.Cut(base, "_")
	if found {
		return version
	}
	return base
}

func applyMigration(ctx context.Context, conn *pgx.Conn, version string, statements []string) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", version, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
	}
	if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
		return fmt.Errorf("record migration %s: %w", version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", version, err)
	}
	return nil
}

func parseDBMateUp(content string) ([]string, error) {
	const (
		upMarker             = "-- migrate:up"
		downMarker           = "-- migrate:down"
		statementBeginMarker = "-- migrate:statementbegin"
		statementEndMarker   = "-- migrate:statementend"
	)
	upIndex := strings.Index(content, upMarker)
	if upIndex < 0 {
		return nil, fmt.Errorf("missing %s marker", upMarker)
	}
	section := content[upIndex+len(upMarker):]
	if downIndex := strings.Index(section, downMarker); downIndex >= 0 {
		section = section[:downIndex]
	}

	var statements []string
	var current strings.Builder
	inStatementBlock := false
	flush := func() {
		statement := strings.TrimSpace(current.String())
		current.Reset()
		if containsSQL(statement) {
			statements = append(statements, statement)
		}
	}

	for _, line := range strings.Split(section, "\n") {
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

func containsSQL(statement string) bool {
	for _, line := range strings.Split(statement, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
			return true
		}
	}
	return false
}
