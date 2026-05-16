package sqlrud

import (
	"fmt"
	"reflect"
	"strings"
)

// Direction controls ORDER BY direction.
type Direction string

const (
	// Asc sorts results in ascending order.
	Asc Direction = "ASC"
	// Desc sorts results in descending order.
	Desc Direction = "DESC"
)

// Predicate represents a safe field comparison used by Where.
type Predicate struct {
	operator string
	values   []any
	noValue  bool
}

// Eq builds an equality predicate.
func Eq(value any) Predicate {
	return Predicate{operator: "=", values: []any{value}}
}

// NotEq builds an inequality predicate.
func NotEq(value any) Predicate {
	return Predicate{operator: "<>", values: []any{value}}
}

// Gt builds a greater-than predicate.
func Gt(value any) Predicate {
	return Predicate{operator: ">", values: []any{value}}
}

// Gte builds a greater-than-or-equal predicate.
func Gte(value any) Predicate {
	return Predicate{operator: ">=", values: []any{value}}
}

// Lt builds a less-than predicate.
func Lt(value any) Predicate {
	return Predicate{operator: "<", values: []any{value}}
}

// Lte builds a less-than-or-equal predicate.
func Lte(value any) Predicate {
	return Predicate{operator: "<=", values: []any{value}}
}

// Like builds a LIKE predicate.
func Like(value any) Predicate {
	return Predicate{operator: "LIKE", values: []any{value}}
}

// In builds an IN predicate.
func In(values ...any) Predicate {
	return Predicate{operator: "IN", values: expandValues(values)}
}

// NotIn builds a NOT IN predicate.
func NotIn(values ...any) Predicate {
	return Predicate{operator: "NOT IN", values: expandValues(values)}
}

// IsNull builds an IS NULL predicate.
func IsNull() Predicate {
	return Predicate{operator: "IS NULL", noValue: true}
}

// IsNotNull builds an IS NOT NULL predicate.
func IsNotNull() Predicate {
	return Predicate{operator: "IS NOT NULL", noValue: true}
}

// QueryOption configures CRUD queries.
type QueryOption interface {
	applyQuery(*queryOptions) error
}

type queryOptionFunc func(*queryOptions) error

func (function queryOptionFunc) applyQuery(options *queryOptions) error {
	return function(options)
}

type filter struct {
	field     string
	predicate Predicate
}

type order struct {
	field     string
	direction Direction
}

type queryOptions struct {
	filters []filter
	orders  []order
	limit   *int
	offset  *int
}

// Where adds an AND condition. field can be either a struct field name or db column name.
func Where(field string, predicate Predicate) QueryOption {
	return queryOptionFunc(func(options *queryOptions) error {
		options.filters = append(options.filters, filter{field: field, predicate: predicate})
		return nil
	})
}

// OrderBy adds an ORDER BY clause.
func OrderBy(field string, direction Direction) QueryOption {
	return queryOptionFunc(func(options *queryOptions) error {
		if direction != Asc && direction != Desc {
			return fmt.Errorf("%w: order direction %q", ErrInvalidIdentifier, direction)
		}
		options.orders = append(options.orders, order{field: field, direction: direction})
		return nil
	})
}

// Limit adds a LIMIT clause.
func Limit(limit int) QueryOption {
	return queryOptionFunc(func(options *queryOptions) error {
		if limit < 0 {
			return fmt.Errorf("sqlrud: limit must be greater than or equal to zero")
		}
		options.limit = &limit
		return nil
	})
}

// Offset adds an OFFSET clause.
func Offset(offset int) QueryOption {
	return queryOptionFunc(func(options *queryOptions) error {
		if offset < 0 {
			return fmt.Errorf("sqlrud: offset must be greater than or equal to zero")
		}
		options.offset = &offset
		return nil
	})
}

func collectOptions(queryOptionsList []QueryOption) (queryOptions, error) {
	var options queryOptions
	for _, queryOption := range queryOptionsList {
		if queryOption == nil {
			continue
		}
		if err := queryOption.applyQuery(&options); err != nil {
			return queryOptions{}, err
		}
	}
	return options, nil
}

func (options queryOptions) validateMutation() error {
	if len(options.orders) > 0 {
		return fmt.Errorf("%w: OrderBy is only supported by Find", ErrUnsupportedOption)
	}
	if options.limit != nil {
		return fmt.Errorf("%w: Limit is only supported by Find", ErrUnsupportedOption)
	}
	if options.offset != nil {
		return fmt.Errorf("%w: Offset is only supported by Find", ErrUnsupportedOption)
	}
	return nil
}

func buildSelect(info *modelInfo, options queryOptions) (string, []any, error) {
	query := fmt.Sprintf("SELECT %s FROM %s", strings.Join(info.selectColumns(), ", "), info.table)
	where, args, err := buildWhere(info, options.filters)
	if err != nil {
		return "", nil, err
	}
	if where != "" {
		query += " WHERE " + where
	}

	if len(options.orders) > 0 {
		orders := make([]string, 0, len(options.orders))
		for _, order := range options.orders {
			column, err := info.resolveColumn(order.field)
			if err != nil {
				return "", nil, err
			}
			orders = append(orders, fmt.Sprintf("%s %s", column, order.direction))
		}
		query += " ORDER BY " + strings.Join(orders, ", ")
	}

	if options.limit != nil {
		query += " LIMIT ?"
		args = append(args, *options.limit)
	}
	if options.offset != nil {
		query += " OFFSET ?"
		args = append(args, *options.offset)
	}

	return query, args, nil
}

func buildWhere(info *modelInfo, filters []filter) (string, []any, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}

	parts := make([]string, 0, len(filters))
	args := make([]any, 0, len(filters))
	for _, filter := range filters {
		column, err := info.resolveColumn(filter.field)
		if err != nil {
			return "", nil, err
		}
		clause, values, err := filter.predicate.build(column)
		if err != nil {
			return "", nil, err
		}
		parts = append(parts, clause)
		args = append(args, values...)
	}

	return strings.Join(parts, " AND "), args, nil
}

func (predicate Predicate) build(column string) (string, []any, error) {
	if predicate.operator == "" {
		return "", nil, fmt.Errorf("sqlrud: predicate operator is empty")
	}

	if predicate.noValue {
		return fmt.Sprintf("%s %s", column, predicate.operator), nil, nil
	}

	if predicate.operator == "IN" || predicate.operator == "NOT IN" {
		if len(predicate.values) == 0 {
			return "", nil, ErrEmptyIn
		}
		placeholders := make([]string, len(predicate.values))
		for index := range placeholders {
			placeholders[index] = "?"
		}
		return fmt.Sprintf("%s %s (%s)", column, predicate.operator, strings.Join(placeholders, ", ")), predicate.values, nil
	}

	if len(predicate.values) != 1 {
		return "", nil, fmt.Errorf("sqlrud: predicate %s requires one value", predicate.operator)
	}

	return fmt.Sprintf("%s %s ?", column, predicate.operator), predicate.values, nil
}

func expandValues(values []any) []any {
	if len(values) != 1 || values[0] == nil {
		return values
	}

	value := reflect.ValueOf(values[0])
	if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
		return values
	}
	if value.Type().Elem().Kind() == reflect.Uint8 {
		return values
	}

	expanded := make([]any, 0, value.Len())
	for index := 0; index < value.Len(); index++ {
		expanded = append(expanded, value.Index(index).Interface())
	}
	return expanded
}
