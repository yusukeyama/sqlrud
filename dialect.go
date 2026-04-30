package sqlrud

// Dialect abstracts database-specific SQL syntax.
type Dialect interface {
	// Quote wraps an identifier (table or column name) in the appropriate
	// quoting characters for the database.
	Quote(identifier string) string

	// Placeholder returns the parameter placeholder for the nth argument
	// (1-indexed). MySQL uses "?" while PostgreSQL uses "$1", "$2", etc.
	Placeholder(n int) string
}

// MySQL is the built-in Dialect for MySQL / MariaDB.
var MySQL Dialect = mysqlDialect{}
