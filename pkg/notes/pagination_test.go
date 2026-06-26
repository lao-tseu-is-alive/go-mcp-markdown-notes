package notes

import (
	"errors"
	"testing"
)

func TestParsePageToken(t *testing.T) {
	t.Parallel()
	cases := []struct {
		token string
		want  int
		err   bool
	}{
		{"", 0, false},
		{"  ", 0, false},
		{"0", 0, false},
		{"10", 10, false},
		{"-1", 0, true},
		{"abc", 0, true},
	}
	for _, tc := range cases {
		got, err := ParsePageToken(tc.token)
		if tc.err {
			if err == nil || !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("ParsePageToken(%q) err = %v, want ErrInvalidInput", tc.token, err)
			}
			continue
		}
		if err != nil {
			t.Fatalf("ParsePageToken(%q) unexpected error: %v", tc.token, err)
		}
		if got != tc.want {
			t.Fatalf("ParsePageToken(%q) = %d, want %d", tc.token, got, tc.want)
		}
	}
}

func TestNextSearchPageToken(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		offset int
		result SearchResult
		want   string
	}{
		{
			name:   "more pages remain",
			offset: 0,
			result: SearchResult{Notes: []*Note{{}, {}}, TotalSize: 5},
			want:   "2",
		},
		{
			name:   "last page partial",
			offset: 4,
			result: SearchResult{Notes: []*Note{{}}, TotalSize: 5},
			want:   "",
		},
		{
			name:   "exact final page",
			offset: 2,
			result: SearchResult{Notes: []*Note{{}, {}}, TotalSize: 4},
			want:   "",
		},
		{
			name:   "single result total",
			offset: 0,
			result: SearchResult{Notes: []*Note{{}}, TotalSize: 1},
			want:   "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := NextSearchPageToken(tc.offset, tc.result); got != tc.want {
				t.Fatalf("NextSearchPageToken() = %q, want %q", got, tc.want)
			}
		})
	}
}
