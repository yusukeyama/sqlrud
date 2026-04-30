package sqlrud

import (
	"errors"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// MySQLDialect implements Dialect for MySQL.
type MySQLDialect struct{}

// QuoteIdent wraps identifier in MySQL backtick quotes.
func (MySQLDialect) QuoteIdent(identifier string) string {
	// Escape any existing backticks by doubling them.
	return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
}

// IsDuplicateError reports whether err is a MySQL duplicate-key error (1062).
func (MySQLDialect) IsDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return me.Number == 1062
	}
	return false
}
