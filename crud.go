package sqlrud

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/yusukeyama/sqlrud/internal/ident"
)

// Queryer is the minimal interface required for SELECT operations.
// Both *sqlx.DB and *sqlx.Tx satisfy this interface.
type Queryer interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
}

// Execer is the minimal interface required for INSERT / UPDATE / DELETE.
// Both *sqlx.DB and *sqlx.Tx satisfy this interface.
type Execer interface {
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
}

// DB combines Queryer and Execer. *sqlx.DB and *sqlx.Tx both satisfy this.
type DB interface {
	Queryer
	Execer
}

// Config holds library-wide defaults.
type Config struct {
	// DefaultInsertExcludeColumns are columns excluded from INSERT by default.
	DefaultInsertExcludeColumns []string

	// DefaultUpdateExcludeColumns are columns excluded from UPDATE by default.
	DefaultUpdateExcludeColumns []string

	// AutoIncrementTag is the struct tag option name that marks a field as an
	// auto-increment primary key (default: "auto_increment").
	AutoIncrementTag string
}

// CRUD provides INSERT, UPDATE, DELETE, and SELECT helpers that generate SQL
// from struct db tags.
type CRUD struct {
	dialect Dialect
	config  Config
}

// New creates a CRUD instance with the supplied options.
// MySQLDialect is used by default when no Dialect option is provided.
func New(opts ...Option) *CRUD {
	c := &CRUD{
		config: Config{AutoIncrementTag: "auto_increment"},
	}
	for _, o := range opts {
		o(c)
	}
	if c.dialect == nil {
		c.dialect = MySQLDialect{}
	}
	return c
}

// Insert builds and executes an INSERT statement from the exported db-tagged
// fields of model. Fields tagged auto_increment are excluded automatically.
func (c *CRUD) Insert(ctx context.Context, q Execer, table string, model any, opts ...QueryOption) (sql.Result, error) {
	if err := validateTable(table); err != nil {
		return nil, err
	}

	o := applyQueryOptions(opts)
	excluded := mergeExcluded(c.config.DefaultInsertExcludeColumns, o.excludeColumns)

	fields, err := parseFields(model, c.config.AutoIncrementTag)
	if err != nil {
		return nil, err
	}

	cols := make([]string, 0, len(fields))
	for _, f := range fields {
		if f.Skip || f.AutoIncrement || isExcluded(f.Column, excluded) {
			continue
		}
		cols = append(cols, f.Column)
	}

	if len(cols) == 0 {
		return nil, fmt.Errorf("%w: no columns to insert", ErrInvalidArgument)
	}

	quotedCols := make([]string, len(cols))
	placeholders := make([]string, len(cols))
	for i, col := range cols {
		quotedCols[i] = c.dialect.QuoteIdent(col)
		placeholders[i] = ":" + col
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		c.dialect.QuoteIdent(table),
		strings.Join(quotedCols, ", "),
		strings.Join(placeholders, ", "),
	)

	return q.NamedExecContext(ctx, query, model)
}

// Update builds and executes an UPDATE statement. WhereColumns option specifies
// which fields form the WHERE clause (default: auto_increment field).
func (c *CRUD) Update(ctx context.Context, q Execer, table string, model any, opts ...QueryOption) (sql.Result, error) {
	if err := validateTable(table); err != nil {
		return nil, err
	}

	o := applyQueryOptions(opts)
	excluded := mergeExcluded(c.config.DefaultUpdateExcludeColumns, o.excludeColumns)

	fields, err := parseFields(model, c.config.AutoIncrementTag)
	if err != nil {
		return nil, err
	}

	whereCols := o.whereColumns
	if len(whereCols) == 0 {
		for _, f := range fields {
			if f.AutoIncrement {
				whereCols = append(whereCols, f.Column)
			}
		}
	}
	whereSet := make(map[string]struct{}, len(whereCols))
	for _, w := range whereCols {
		whereSet[w] = struct{}{}
	}

	setCols := make([]string, 0, len(fields))
	for _, f := range fields {
		if f.Skip {
			continue
		}
		if _, isWhere := whereSet[f.Column]; isWhere {
			continue
		}
		if isExcluded(f.Column, excluded) {
			continue
		}
		setCols = append(setCols, f.Column)
	}

	if len(setCols) == 0 {
		return nil, fmt.Errorf("%w: no columns to update", ErrInvalidArgument)
	}
	if len(whereCols) == 0 {
		return nil, fmt.Errorf("%w: no WHERE columns for update", ErrInvalidArgument)
	}

	setParts := make([]string, len(setCols))
	for i, col := range setCols {
		setParts[i] = fmt.Sprintf("%s = :%s", c.dialect.QuoteIdent(col), col)
	}
	whereParts := make([]string, len(whereCols))
	for i, col := range whereCols {
		whereParts[i] = fmt.Sprintf("%s = :%s", c.dialect.QuoteIdent(col), col)
	}

	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s",
		c.dialect.QuoteIdent(table),
		strings.Join(setParts, ", "),
		strings.Join(whereParts, " AND "),
	)

	return q.NamedExecContext(ctx, query, model)
}

// Delete builds and executes a DELETE statement. WhereColumns option specifies
// which fields form the WHERE clause (default: auto_increment field).
func (c *CRUD) Delete(ctx context.Context, q Execer, table string, model any, opts ...QueryOption) (sql.Result, error) {
	if err := validateTable(table); err != nil {
		return nil, err
	}

	o := applyQueryOptions(opts)
	fields, err := parseFields(model, c.config.AutoIncrementTag)
	if err != nil {
		return nil, err
	}

	whereCols := o.whereColumns
	if len(whereCols) == 0 {
		for _, f := range fields {
			if f.AutoIncrement {
				whereCols = append(whereCols, f.Column)
			}
		}
	}

	if len(whereCols) == 0 {
		return nil, fmt.Errorf("%w: no WHERE columns for delete", ErrInvalidArgument)
	}

	whereParts := make([]string, len(whereCols))
	for i, col := range whereCols {
		whereParts[i] = fmt.Sprintf("%s = :%s", c.dialect.QuoteIdent(col), col)
	}

	query := fmt.Sprintf(
		"DELETE FROM %s WHERE %s",
		c.dialect.QuoteIdent(table),
		strings.Join(whereParts, " AND "),
	)

	return q.NamedExecContext(ctx, query, model)
}

// FirstInto executes query with args, scanning the first result into dest.
// Returns ErrNotFound when the query returns no rows.
func FirstInto[T any](ctx context.Context, q Queryer, dest *T, query string, args ...any) error {
	if err := q.GetContext(ctx, dest, query, args...); err != nil {
		if isNotFound(err) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// FindInto executes query with args, scanning all results into dest.
// dest is never nil on success; an empty slice is returned when no rows match.
func FindInto[T any](ctx context.Context, q Queryer, dest *[]T, query string, args ...any) error {
	if err := q.SelectContext(ctx, dest, query, args...); err != nil {
		return err
	}
	if *dest == nil {
		*dest = []T{}
	}
	return nil
}

// Count executes a COUNT query and returns the result.
func Count(ctx context.Context, q Queryer, query string, args ...any) (int64, error) {
	var n int64
	if err := q.GetContext(ctx, &n, query, args...); err != nil {
		return 0, err
	}
	return n, nil
}

// internals

func mergeExcluded(defaults, perCall []string) map[string]struct{} {
	m := make(map[string]struct{}, len(defaults)+len(perCall))
	for _, c := range defaults {
		m[c] = struct{}{}
	}
	for _, c := range perCall {
		m[c] = struct{}{}
	}
	return m
}

func isExcluded(col string, excluded map[string]struct{}) bool {
	_, ok := excluded[col]
	return ok
}

func isNotFound(err error) bool {
	return err == sql.ErrNoRows
}

func validateTable(table string) error {
	if err := ident.Validate(table); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidArgument, err)
	}
	return nil
}

// ensure *sqlx.DB satisfies Queryer, Execer, and DB at compile time.
var _ Queryer = (*sqlx.DB)(nil)
var _ Execer = (*sqlx.DB)(nil)
var _ DB = (*sqlx.DB)(nil)
var _ Queryer = (*sqlx.Tx)(nil)
var _ Execer = (*sqlx.Tx)(nil)
var _ DB = (*sqlx.Tx)(nil)
