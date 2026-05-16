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
}

// New creates a Client that executes queries through db.
func New(db *sqlx.DB) *Client {
	if db == nil {
		panic("sqlrud: nil *sqlx.DB")
	}
	return &Client{
		db:       db,
		ext:      db,
		bindType: sqlx.BindType(db.DriverName()),
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

	query, args, err := buildSelect(info, queryOptions)
	if err != nil {
		return err
	}

	return sqlx.SelectContext(ctx, client.ext, destination, client.rebind(query), args...)
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

// CreateOrUpdate inserts model when its primary key is empty or no row exists, otherwise it updates the existing row.
func (client *Client) CreateOrUpdate(ctx context.Context, model any) error {
	info, value, err := modelInfoForValue(model)
	if err != nil {
		return err
	}

	filters, args, ok, err := primaryFilters(info, value)
	if err != nil {
		return err
	}
	if !ok {
		return client.Create(ctx, model)
	}

	query := fmt.Sprintf("SELECT 1 FROM %s WHERE %s LIMIT 1", info.table, strings.Join(filters, " AND "))
	var exists int
	err = sqlx.GetContext(ctx, client.ext, &exists, client.rebind(query), args...)
	if err == nil {
		return client.Update(ctx, model)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return client.Create(ctx, model)
	}
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

func createColumns(info *modelInfo, value reflect.Value) ([]string, []any) {
	columns := make([]string, 0, len(info.fields))
	args := make([]any, 0, len(info.fields))
	for _, field := range info.fields {
		if field.readOnly || field.updateOnly {
			continue
		}

		fieldValue := value.FieldByIndex(field.index)
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

		fieldValue := value.FieldByIndex(field.index)
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
		fieldValue := value.FieldByIndex(field.index)
		if isZero(fieldValue) {
			return nil, nil, false, nil
		}
		filters = append(filters, field.column+" = ?")
		args = append(args, fieldValue.Interface())
	}

	return filters, args, true, nil
}
