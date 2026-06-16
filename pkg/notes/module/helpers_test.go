package module

import "testing"

func TestModuleMigrationVersion(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"db/migrations/000001_create_notes.sql", "000001"},
		{"db/migrations/000002_add_index.sql", "000002"},
		{"db/migrations/nounderscore.sql", "nounderscore"},
	}
	for _, tc := range cases {
		if got := moduleMigrationVersion(tc.path); got != tc.want {
			t.Errorf("moduleMigrationVersion(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}
