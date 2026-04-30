package ident

import (
	"fmt"
	"regexp"
)

// identPattern matches valid SQL identifiers: start with a letter or
// underscore, followed by letters, digits, or underscores.
var identPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ErrInvalidIdent is returned by Validate for identifiers that do not match
// the allowed pattern.
var ErrInvalidIdent = fmt.Errorf("invalid identifier")

// Validate returns a non-nil error if s is not a safe SQL identifier.
// Callers must validate every table name and column name that originates from
// external input before interpolating it into a SQL string.
func Validate(s string) error {
	if !identPattern.MatchString(s) {
		return fmt.Errorf("%w: %q", ErrInvalidIdent, s)
	}
	return nil
}
