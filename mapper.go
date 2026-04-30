package sqlrud

import (
	"fmt"
	"reflect"
)

// field represents a struct field mapped to a DB column via the "db" tag.
type field struct {
	column string
	value  any
}

// extractFields returns the db-tagged fields and their values from a struct
// pointer. It skips fields whose db tag is "-" or that have no db tag.
func extractFields(v any) ([]field, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("sqlrud: expected struct, got %T", v)
	}

	rt := rv.Type()
	fields := make([]field, 0, rt.NumField())
	for i := range rt.NumField() {
		tf := rt.Field(i)
		tag := tf.Tag.Get("db")
		if tag == "" || tag == "-" {
			continue
		}
		fields = append(fields, field{column: tag, value: rv.Field(i).Interface()})
	}
	return fields, nil
}
