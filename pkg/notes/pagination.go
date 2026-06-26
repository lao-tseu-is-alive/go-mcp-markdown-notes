package notes

import (
	"fmt"
	"strconv"
	"strings"
)

// ParsePageToken decodes a SearchNotes page_token into a zero-based offset.
// An empty token means the first page.
func ParsePageToken(token string) (int, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return 0, nil
	}
	offset, err := strconv.Atoi(token)
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("%w: invalid page token", ErrInvalidInput)
	}
	return offset, nil
}

// FormatPageToken encodes a zero-based offset for use as next_page_token.
func FormatPageToken(offset int) string {
	return strconv.Itoa(offset)
}

// NextSearchPageToken returns the token for the page after result, or "" when
// there are no more rows.
func NextSearchPageToken(offset int, result SearchResult) string {
	if len(result.Notes) == 0 {
		return ""
	}
	nextOffset := offset + len(result.Notes)
	if int32(nextOffset) >= result.TotalSize {
		return ""
	}
	return FormatPageToken(nextOffset)
}
