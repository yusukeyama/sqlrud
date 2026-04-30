package sqlrud

import "fmt"

type mysqlDialect struct{}

func (mysqlDialect) Quote(id string) string {
	return "`" + id + "`"
}

func (mysqlDialect) Placeholder(_ int) string {
	return "?"
}

// postgresDialect is included as an example of a non-MySQL dialect.
type postgresDialect struct{}

func (postgresDialect) Quote(id string) string {
	return `"` + id + `"`
}

func (postgresDialect) Placeholder(n int) string {
	return fmt.Sprintf("$%d", n)
}

// Postgres is the built-in Dialect for PostgreSQL.
var Postgres Dialect = postgresDialect{}
