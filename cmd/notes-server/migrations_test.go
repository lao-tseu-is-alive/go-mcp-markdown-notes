package main

import (
	"strings"
	"testing"
)

func TestMigrationVersionMatchesDBMate(t *testing.T) {
	if got := migrationVersion("db/migrations/000001_create_notes.sql"); got != "000001" {
		t.Fatalf("migrationVersion() = %q, want 000001", got)
	}
}

func TestParseDBMateUp(t *testing.T) {
	content, err := migrationFiles.ReadFile("db/migrations/000001_create_notes.sql")
	if err != nil {
		t.Fatal(err)
	}
	statements, err := parseDBMateUp(string(content))
	if err != nil {
		t.Fatal(err)
	}
	if len(statements) != 7 {
		t.Fatalf("statement count = %d, want 7", len(statements))
	}
	joined := strings.Join(statements, "\n")
	if strings.Contains(joined, "migrate:down") || !strings.Contains(joined, "CREATE FUNCTION set_notes_updated_at") {
		t.Fatalf("unexpected parsed migration:\n%s", joined)
	}
}

func TestParseDBMateUpRejectsMalformedInput(t *testing.T) {
	tests := []string{
		"SELECT 1;",
		"-- migrate:up\n-- migrate:statementbegin\nSELECT 1;",
		"-- migrate:up\n-- migrate:statementend",
	}
	for _, input := range tests {
		if _, err := parseDBMateUp(input); err == nil {
			t.Fatalf("parseDBMateUp(%q) succeeded, want error", input)
		}
	}
}
