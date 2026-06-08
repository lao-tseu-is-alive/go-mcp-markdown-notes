package notes

import "errors"

var (
	ErrInvalidInput    = errors.New("invalid input")
	ErrNoteNotFound    = errors.New("note not found")
	ErrUnauthenticated = errors.New("authenticated user is required")
)
