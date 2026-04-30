package sqlrud

// Dialect abstracts database-specific behaviour such as identifier quoting and
// duplicate-key error detection.
type Dialect interface {
	// QuoteIdent wraps a table or column identifier in the appropriate quote
	// characters for the target database (e.g. backticks for MySQL).
	QuoteIdent(identifier string) string

	// IsDuplicateError reports whether err represents a unique-constraint
	// violation for the target database.
	IsDuplicateError(err error) bool
}
