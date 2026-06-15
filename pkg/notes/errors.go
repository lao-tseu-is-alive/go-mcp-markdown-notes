package notes

import "errors"

var (
	// ErrInvalidInput is returned when caller-supplied data fails validation.
	ErrInvalidInput = errors.New("invalid input")
	// ErrNoteNotFound is returned when a note does not exist or belongs to another owner.
	ErrNoteNotFound = errors.New("note not found")
	// ErrUnauthenticated is returned when a request carries no recognised identity.
	ErrUnauthenticated = errors.New("authenticated user is required")
)
