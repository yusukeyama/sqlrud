package sqlrud

import "errors"

// ErrNotFound is returned when a query that expects exactly one row finds none.
var ErrNotFound = errors.New("sqlrud: not found")

// ErrNoRowsAffected is returned when an UPDATE or DELETE affects no rows.
var ErrNoRowsAffected = errors.New("sqlrud: no rows affected")
