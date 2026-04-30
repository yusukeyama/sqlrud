package sqlrud

import (
	"fmt"
	"reflect"
	"strings"
)

// fieldInfo holds parsed information for a single struct field.
type fieldInfo struct {
	Column        string
	AutoIncrement bool
	Skip          bool
}

// parseFields returns field info for each exported field of v in struct
// definition order.
func parseFields(v any, autoIncrTag string) ([]fieldInfo, error) {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		rv = rv.Elem()
	}

	if rv.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: expected struct, got %s", ErrInvalidArgument, rv.Kind())
	}

	rt := rv.Type()
	fields := make([]fieldInfo, 0, rt.NumField())

	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}

		tag := f.Tag.Get("db")
		if tag == "" {
			continue
		}

		parts := strings.Split(tag, ",")
		col := strings.TrimSpace(parts[0])

		if col == "-" {
			fields = append(fields, fieldInfo{Skip: true})
			continue
		}

		fi := fieldInfo{Column: col}
		for _, opt := range parts[1:] {
			if strings.TrimSpace(opt) == autoIncrTag {
				fi.AutoIncrement = true
			}
		}

		fields = append(fields, fi)
	}

	return fields, nil
}
