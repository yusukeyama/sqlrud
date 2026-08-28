package sqlrud

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Client runs CRUD operations through sqlx.
type Client struct {
	db       *sqlx.DB
	ext      sqlx.ExtContext
	bindType int
	dialect  dialect
}

// New creates a Client that executes queries through db.
func New(db *sqlx.DB) *Client {
	if db == nil {
		panic("sqlrud: nil *sqlx.DB")
	}
	driverName := db.DriverName()
	// Use a private sqlx wrapper so sqlrud's field mapping does not mutate the caller's DB.
	mappedDB := sqlx.NewDb(db.DB, driverName)
	mappedDB.MapperFunc(toSnakeCase)
	return &Client{
		db:       mappedDB,
		ext:      mappedDB,
		bindType: sqlx.BindType(driverName),
		dialect:  dialectForDriver(driverName),
	}
}

// First loads one record by the primary key values set on destination.
func (client *Client) First(ctx context.Context, destination any) error {
	info, value, err := modelInfoForStructDestinationValue(destination)
	if err != nil {
		return err
	}

	filters, args, ok, err := primaryFilters(info, value)
	if err != nil {
		return err
	}
	if !ok {
		return ErrMissingPrimaryValue
	}

	query := fmt.Sprintf("SELECT %s FROM %s WHERE %s LIMIT ?", strings.Join(info.selectColumns(), ", "), info.table, strings.Join(filters, " AND "))
	args = append(args, 1)

	return sqlx.GetContext(ctx, client.ext, destination, client.rebind(query), args...)
}

// Find loads all matching records into destination.
func (client *Client) Find(ctx context.Context, destination any, options ...QueryOption) error {
	info, err := modelInfoForSliceDestination(destination)
	if err != nil {
		return err
	}

	queryOptions, err := collectOptions(options)
	if err != nil {
		return err
	}

	query, args, err := buildSelect(info, queryOptions, client.dialect)
	if err != nil {
		return err
	}

	return sqlx.SelectContext(ctx, client.ext, destination, client.rebind(query), args...)
}

// Query executes query exactly as provided and returns its rows.
// The caller must close the returned rows. Placeholders must use the database's native syntax.
func (client *Client) Query(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	return client.ext.QueryxContext(ctx, query, args...)
}

// Create inserts model.
func (client *Client) Create(ctx context.Context, model any) error {
	info, value, err := modelInfoForValue(model)
	if err != nil {
		return err
	}

	columns, args := createColumns(info, value)
	if len(columns) == 0 {
		return ErrNoColumns
	}

	placeholders := make([]string, len(columns))
	for index := range placeholders {
		placeholders[index] = "?"
	}

	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", info.table, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	_, err = client.ext.ExecContext(ctx, client.rebind(query), args...)
	return err
}

// Update updates model by primary key or by explicit Where conditions.
func (client *Client) Update(ctx context.Context, model any, options ...QueryOption) error {
	info, value, err := modelInfoForValue(model)
	if err != nil {
		return err
	}

	columns, args := updateColumns(info, value)
	if len(columns) == 0 {
		return ErrNoColumns
	}

	queryOptions, err := collectOptions(options)
	if err != nil {
		return err
	}
	where, whereArgs, err := mutationWhere(info, value, queryOptions)
	if err != nil {
		return err
	}

	sets := make([]string, 0, len(columns))
	for _, column := range columns {
		sets = append(sets, column+" = ?")
	}

	query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", info.table, strings.Join(sets, ", "), where)
	args = append(args, whereArgs...)
	_, err = client.ext.ExecContext(ctx, client.rebind(query), args...)
	return err
}

// CreateOrUpdate inserts model or atomically updates it on primary key conflict.
func (client *Client) CreateOrUpdate(ctx context.Context, model any) error {
	info, value, err := modelInfoForValue(model)
	if err != nil {
		return err
	}

	_, _, ok, err := primaryFilters(info, value)
	if err != nil {
		return err
	}
	if !ok {
		return client.Create(ctx, model)
	}

	insertColumns, insertArgs := createColumns(info, value)
	if len(insertColumns) == 0 {
		return ErrNoColumns
	}

	setColumns, updateArgs := updateColumns(info, value)
	query, args, err := client.buildUpsert(info, insertColumns, insertArgs, setColumns, updateArgs)
	if err != nil {
		return err
	}

	_, err = client.ext.ExecContext(ctx, client.rebind(query), args...)
	return err
}

// Delete deletes model by primary key or by explicit Where conditions.
func (client *Client) Delete(ctx context.Context, model any, options ...QueryOption) error {
	info, value, err := modelInfoForValue(model)
	if err != nil {
		return err
	}

	queryOptions, err := collectOptions(options)
	if err != nil {
		return err
	}
	where, args, err := mutationWhere(info, value, queryOptions)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE %s", info.table, where)
	_, err = client.ext.ExecContext(ctx, client.rebind(query), args...)
	return err
}

// Transaction runs fn in a transaction. Returning an error from fn rolls back the transaction.
func (client *Client) Transaction(ctx context.Context, fn func(*Client) error) error {
	return client.TransactionOptions(ctx, nil, fn)
}

// TransactionOptions runs fn in a transaction using options.
func (client *Client) TransactionOptions(ctx context.Context, options *sql.TxOptions, fn func(*Client) error) (err error) {
	if fn == nil {
		return fmt.Errorf("sqlrud: transaction function is nil")
	}

	tx, err := client.db.BeginTxx(ctx, options)
	if err != nil {
		return err
	}

	txClient := &Client{
		db:       client.db,
		ext:      tx,
		bindType: client.bindType,
		dialect:  client.dialect,
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			_ = tx.Rollback()
			panic(recovered)
		}
		if err != nil {
			rollbackErr := tx.Rollback()
			if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				err = errors.Join(err, rollbackErr)
			}
		}
	}()

	if err = fn(txClient); err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func (client *Client) rebind(query string) string {
	if client.bindType == sqlx.UNKNOWN {
		return query
	}
	return sqlx.Rebind(client.bindType, query)
}

type dialect string

const (
	dialectUnknown  dialect = ""
	dialectPostgres dialect = "postgres"
	dialectMySQL    dialect = "mysql"
)

func dialectForDriver(driverName string) dialect {
	switch strings.ToLower(driverName) {
	case "postgres", "pgx":
		return dialectPostgres
	case "mysql":
		return dialectMySQL
	default:
		return dialectUnknown
	}
}

func (client *Client) buildUpsert(info *modelInfo, insertColumns []string, insertArgs []any, setColumns []string, updateArgs []any) (string, []any, error) {
	switch client.dialect {
	case dialectPostgres:
		return buildPostgresUpsert(info, insertColumns, setColumns), appendArgs(insertArgs, updateArgs), nil
	case dialectMySQL:
		return buildMySQLUpsert(info, insertColumns, setColumns), appendArgs(insertArgs, updateArgs), nil
	default:
		return "", nil, ErrUnsupportedDialect
	}
}

func buildPostgresUpsert(info *modelInfo, insertColumns []string, setColumns []string) string {
	query := insertQuery(info, insertColumns)
	query += fmt.Sprintf(" ON CONFLICT (%s)", strings.Join(primaryColumns(info), ", "))

	if len(setColumns) == 0 {
		return query + " DO NOTHING"
	}

	return query + " DO UPDATE SET " + strings.Join(assignments(setColumns), ", ")
}

func buildMySQLUpsert(info *modelInfo, insertColumns []string, setColumns []string) string {
	query := insertQuery(info, insertColumns)
	if len(setColumns) == 0 {
		return query + fmt.Sprintf(" ON DUPLICATE KEY UPDATE %s = %s", info.primary[0].column, info.primary[0].column)
	}

	return query + " ON DUPLICATE KEY UPDATE " + strings.Join(assignments(setColumns), ", ")
}

func appendArgs(first []any, second []any) []any {
	args := make([]any, 0, len(first)+len(second))
	args = append(args, first...)
	args = append(args, second...)
	return args
}

func insertQuery(info *modelInfo, columns []string) string {
	placeholders := make([]string, len(columns))
	for index := range placeholders {
		placeholders[index] = "?"
	}
	return fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", info.table, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
}

func assignments(columns []string) []string {
	sets := make([]string, 0, len(columns))
	for _, column := range columns {
		sets = append(sets, column+" = ?")
	}
	return sets
}

func primaryColumns(info *modelInfo) []string {
	columns := make([]string, 0, len(info.primary))
	for _, field := range info.primary {
		columns = append(columns, field.column)
	}
	return columns
}

func createColumns(info *modelInfo, value reflect.Value) ([]string, []any) {
	columns := make([]string, 0, len(info.fields))
	args := make([]any, 0, len(info.fields))
	for _, field := range info.fields {
		if field.readOnly || field.updateOnly {
			continue
		}

		fieldValue := fieldValueForModel(value, field)
		if field.auto && isZero(fieldValue) {
			continue
		}
		if field.omitEmpty && isZero(fieldValue) {
			continue
		}

		columns = append(columns, field.column)
		args = append(args, fieldValue.Interface())
	}
	return columns, args
}

func updateColumns(info *modelInfo, value reflect.Value) ([]string, []any) {
	columns := make([]string, 0, len(info.fields))
	args := make([]any, 0, len(info.fields))
	for _, field := range info.fields {
		if field.primary || field.readOnly || field.createOnly {
			continue
		}

		fieldValue := fieldValueForModel(value, field)
		if field.omitEmpty && isZero(fieldValue) {
			continue
		}

		columns = append(columns, field.column)
		args = append(args, fieldValue.Interface())
	}
	return columns, args
}

func mutationWhere(info *modelInfo, value reflect.Value, options queryOptions) (string, []any, error) {
	if err := options.validateMutation(); err != nil {
		return "", nil, err
	}

	if len(options.filters) > 0 {
		return buildWhere(info, options.filters)
	}

	filters, args, ok, err := primaryFilters(info, value)
	if err != nil {
		return "", nil, err
	}
	if !ok {
		return "", nil, ErrMissingWhere
	}
	return strings.Join(filters, " AND "), args, nil
}

func primaryFilters(info *modelInfo, value reflect.Value) ([]string, []any, bool, error) {
	if len(info.primary) == 0 {
		return nil, nil, false, ErrMissingPrimaryKey
	}

	filters := make([]string, 0, len(info.primary))
	args := make([]any, 0, len(info.primary))
	for _, field := range info.primary {
		fieldValue := fieldValueForModel(value, field)
		if isZero(fieldValue) {
			return nil, nil, false, nil
		}
		filters = append(filters, field.column+" = ?")
		args = append(args, fieldValue.Interface())
	}

	return filters, args, true, nil
}
