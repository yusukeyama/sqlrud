package sqlrud

import "errors"

var (
	ErrInvalidModel       = errors.New("sqlrud: model must be a struct or a pointer to struct")
	ErrInvalidDestination = errors.New("sqlrud: destination must be a pointer to struct or slice")
	ErrInvalidIdentifier  = errors.New("sqlrud: invalid identifier")
	ErrMissingPrimaryKey  = errors.New("sqlrud: model has no primary key")
	ErrMissingWhere       = errors.New("sqlrud: update or delete requires conditions or a primary key value")
	ErrNoColumns          = errors.New("sqlrud: no writable columns")
	ErrUnknownField       = errors.New("sqlrud: unknown field")
	ErrEmptyIn            = errors.New("sqlrud: IN predicate requires at least one value")
)
