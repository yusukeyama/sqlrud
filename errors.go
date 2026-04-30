package sqlrud

import "errors"

// Sentinel errors returned by sqlrud operations.
// Use errors.Is to inspect returned errors.
var (
	// ErrNotFound is returned when a query produces no rows.
	ErrNotFound = errors.New("sqlrud: not found")

	// ErrDuplicate is returned when a unique constraint violation occurs.
	ErrDuplicate = errors.New("sqlrud: duplicate")

	// ErrNoRowsAffected is returned when an UPDATE or DELETE affects zero rows.
	ErrNoRowsAffected = errors.New("sqlrud: no rows affected")

	// ErrInvalidArgument is returned when an argument (e.g. a table or column
	// name) fails validation.
	ErrInvalidArgument = errors.New("sqlrud: invalid argument")
)
