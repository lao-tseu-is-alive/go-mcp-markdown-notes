package main

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
)

//go:embed db/migrations/*.sql
var migrationFiles embed.FS

// migrationVersion extracts the version prefix (everything before the first underscore) from a migration file's base name.
func migrationVersion(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	version, _, found := strings.Cut(base, "_")
	if found {
		return version
	}
	return base
}

// parseDBMateUp extracts the SQL statements from the "-- migrate:up" section of a DBMate migration file.
// Multi-statement blocks delimited by "-- migrate:statementbegin / end" are emitted as a single string.
func parseDBMateUp(content string) ([]string, error) {
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
		if containsSQL(statement) {
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

// containsSQL reports whether the statement contains at least one non-blank, non-comment line.
func containsSQL(statement string) bool {
	for line := range strings.SplitSeq(statement, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "--") {
			return true
		}
	}
	return false
}
