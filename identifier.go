package sqlrud

import (
	"fmt"
	"regexp"
)

var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// quoteIdentifier validates and quotes a SQL identifier using the given dialect.
// It returns an error if the identifier contains characters that could cause
// SQL injection.
func quoteIdentifier(d Dialect, id string) (string, error) {
	if !validIdentifier.MatchString(id) {
		return "", fmt.Errorf("sqlrud: invalid identifier %q", id)
	}
	return d.Quote(id), nil
}
