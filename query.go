package sqlrud

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// First queries table for the first row matching where and scans the result
// into dest (pointer to a struct). Returns ErrNotFound when no row matches.
func (r *Rudder) First(ctx context.Context, table string, dest any, where string, whereArgs ...any) error {
	quotedTable, err := quoteIdentifier(r.dialect, table)
	if err != nil {
		return err
	}

	var q string
	if where != "" {
		q = fmt.Sprintf("SELECT * FROM %s WHERE %s LIMIT 1", quotedTable, where)
	} else {
		q = fmt.Sprintf("SELECT * FROM %s LIMIT 1", quotedTable)
	}

	if err := sqlx.GetContext(ctx, r.db, dest, q, whereArgs...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// Find queries table for all rows matching where and scans them into dest
// (pointer to a slice of structs).
func (r *Rudder) Find(ctx context.Context, table string, dest any, where *string, whereArgs ...any) error {
	quotedTable, err := quoteIdentifier(r.dialect, table)
	if err != nil {
		return err
	}

	var q string
	if where != nil && *where != "" {
		q = fmt.Sprintf("SELECT * FROM %s WHERE %s", quotedTable, *where)
	} else {
		q = fmt.Sprintf("SELECT * FROM %s", quotedTable)
	}

	return sqlx.SelectContext(ctx, r.db, dest, q, whereArgs...)
}
