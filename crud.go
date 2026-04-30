package sqlrud

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Rudder is the main entry-point for sqlrud operations.
type Rudder struct {
	db      sqlx.ExtContext
	dialect Dialect
}

// New creates a new Rudder backed by db using the given dialect.
// db can be *sqlx.DB or *sqlx.Tx.
func New(db sqlx.ExtContext, dialect Dialect) *Rudder {
	return &Rudder{db: db, dialect: dialect}
}

// Insert inserts the struct pointed to by v into table.
func (r *Rudder) Insert(ctx context.Context, table string, v any, opts ...InsertOption) error {
	cfg := &insertConfig{}
	for _, o := range opts {
		o(cfg)
	}

	fields, err := extractFields(v)
	if err != nil {
		return err
	}

	quotedTable, err := quoteIdentifier(r.dialect, table)
	if err != nil {
		return err
	}

	cols := make([]string, 0, len(fields))
	placeholders := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields))

	for i, f := range fields {
		qc, err := quoteIdentifier(r.dialect, f.column)
		if err != nil {
			return err
		}
		cols = append(cols, qc)
		placeholders = append(placeholders, r.dialect.Placeholder(i+1))
		args = append(args, f.value)
	}

	keyword := "INSERT"
	if cfg.ignore {
		keyword = "INSERT IGNORE"
	}

	query := fmt.Sprintf(
		"%s INTO %s (%s) VALUES (%s)",
		keyword,
		quotedTable,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err = r.db.ExecContext(ctx, query, args...)
	return err
}

// Update updates table using the fields in v filtered by where.
func (r *Rudder) Update(ctx context.Context, table string, v any, where string, whereArgs ...any) error {
	fields, err := extractFields(v)
	if err != nil {
		return err
	}

	quotedTable, err := quoteIdentifier(r.dialect, table)
	if err != nil {
		return err
	}

	setClauses := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields)+len(whereArgs))

	for i, f := range fields {
		qc, err := quoteIdentifier(r.dialect, f.column)
		if err != nil {
			return err
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = %s", qc, r.dialect.Placeholder(i+1)))
		args = append(args, f.value)
	}
	args = append(args, whereArgs...)

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		quotedTable,
		strings.Join(setClauses, ", "),
		where,
	)

	res, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNoRowsAffected
	}
	return nil
}

// Delete deletes rows from table matching where.
func (r *Rudder) Delete(ctx context.Context, table string, where string, whereArgs ...any) error {
	quotedTable, err := quoteIdentifier(r.dialect, table)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s", quotedTable, where)
	res, err := r.db.ExecContext(ctx, query, whereArgs...)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNoRowsAffected
	}
	return nil
}

// Count returns the number of rows in table matching where.
func (r *Rudder) Count(ctx context.Context, table string, where string, whereArgs ...any) (int64, error) {
	quotedTable, err := quoteIdentifier(r.dialect, table)
	if err != nil {
		return 0, err
	}

	var q string
	if where != "" {
		q = fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", quotedTable, where)
	} else {
		q = fmt.Sprintf("SELECT COUNT(*) FROM %s", quotedTable)
	}

	rows, err := r.db.QueryContext(ctx, q, whereArgs...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, rows.Err()
	}
	var count int64
	if err := rows.Scan(&count); err != nil {
		return 0, err
	}
	return count, rows.Err()
}
