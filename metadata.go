package sqlrud

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"
)

// TableNamer overrides the table name for a model.
type TableNamer interface {
	TableName() string
}

type fieldInfo struct {
	name       string
	column     string
	index      []int
	primary    bool
	auto       bool
	readOnly   bool
	createOnly bool
	updateOnly bool
	omitEmpty  bool
}

type modelInfo struct {
	typ     reflect.Type
	table   string
	fields  []fieldInfo
	primary []fieldInfo
	byName  map[string]fieldInfo
}

var modelCache sync.Map

func modelInfoForValue(model any) (*modelInfo, reflect.Value, error) {
	value := reflect.ValueOf(model)
	if !value.IsValid() {
		return nil, reflect.Value{}, ErrInvalidModel
	}

	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, reflect.Value{}, ErrInvalidModel
		}
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return nil, reflect.Value{}, ErrInvalidModel
	}

	info, err := modelInfoForType(value.Type())
	if err != nil {
		return nil, reflect.Value{}, err
	}
	return info, value, nil
}

func modelInfoForStructDestination(destination any) (*modelInfo, error) {
	typ, err := destinationStructType(destination)
	if err != nil {
		return nil, err
	}
	return modelInfoForType(typ)
}

func modelInfoForStructDestinationValue(destination any) (*modelInfo, reflect.Value, error) {
	value := reflect.ValueOf(destination)
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() {
		return nil, reflect.Value{}, ErrInvalidDestination
	}

	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return nil, reflect.Value{}, ErrInvalidDestination
	}

	info, err := modelInfoForType(value.Type())
	if err != nil {
		return nil, reflect.Value{}, err
	}
	return info, value, nil
}

func modelInfoForSliceDestination(destination any) (*modelInfo, error) {
	value := reflect.ValueOf(destination)
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() {
		return nil, ErrInvalidDestination
	}

	value = value.Elem()
	if value.Kind() != reflect.Slice {
		return nil, ErrInvalidDestination
	}

	typ := value.Type().Elem()
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, ErrInvalidDestination
	}

	return modelInfoForType(typ)
}

func destinationStructType(destination any) (reflect.Type, error) {
	value := reflect.ValueOf(destination)
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() {
		return nil, ErrInvalidDestination
	}

	value = value.Elem()
	if value.Kind() != reflect.Struct {
		return nil, ErrInvalidDestination
	}
	return value.Type(), nil
}

func modelInfoForType(typ reflect.Type) (*modelInfo, error) {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct {
		return nil, ErrInvalidModel
	}

	if cached, ok := modelCache.Load(typ); ok {
		return cached.(*modelInfo), nil
	}

	info, err := buildModelInfo(typ)
	if err != nil {
		return nil, err
	}
	modelCache.Store(typ, info)
	return info, nil
}

func buildModelInfo(typ reflect.Type) (*modelInfo, error) {
	table := tableNameForType(typ)
	if !validIdentifierPath(table) {
		return nil, fmt.Errorf("%w: table %q", ErrInvalidIdentifier, table)
	}

	info := &modelInfo{
		typ:    typ,
		table:  table,
		byName: make(map[string]fieldInfo),
	}

	for index := 0; index < typ.NumField(); index++ {
		field := typ.Field(index)
		if !field.IsExported() {
			continue
		}
		if field.Anonymous && field.Tag.Get("db") == "" {
			continue
		}

		dbTag := parseDBTag(field.Tag.Get("db"))
		compatOptions := parseTagOptions(field.Tag.Get("sqlrud"))
		if dbTag.ignored || compatOptions.has("-") {
			continue
		}

		column := columnNameForField(field, dbTag)
		if !validIdentifier(column) {
			return nil, fmt.Errorf("%w: column %q", ErrInvalidIdentifier, column)
		}

		options := dbTag.options.merge(compatOptions)
		fieldInfo := fieldInfo{
			name:       field.Name,
			column:     column,
			index:      field.Index,
			primary:    options.has("pk", "primary", "primary_key"),
			auto:       options.has("auto", "auto_increment"),
			readOnly:   options.has("readonly", "read_only"),
			createOnly: options.has("createonly", "create_only"),
			updateOnly: options.has("updateonly", "update_only"),
			omitEmpty:  options.has("omitempty", "omit_empty"),
		}

		info.fields = append(info.fields, fieldInfo)
		if fieldInfo.primary {
			info.primary = append(info.primary, fieldInfo)
		}
	}

	if len(info.primary) == 0 {
		for index := range info.fields {
			if isDefaultPrimary(info.fields[index]) {
				info.fields[index].primary = true
				info.fields[index].auto = true
				info.primary = append(info.primary, info.fields[index])
				break
			}
		}
	}

	for _, field := range info.fields {
		info.byName[field.name] = field
		info.byName[strings.ToLower(field.name)] = field
		info.byName[field.column] = field
		info.byName[strings.ToLower(field.column)] = field
	}

	return info, nil
}

func tableNameForType(typ reflect.Type) string {
	value := reflect.New(typ).Interface()
	if namer, ok := value.(TableNamer); ok {
		return namer.TableName()
	}
	return toSnakeCase(typ.Name()) + "s"
}

func columnNameForField(field reflect.StructField, tag dbTag) string {
	if tag.name != "" {
		return tag.name
	}
	return toSnakeCase(field.Name)
}

func isDefaultPrimary(field fieldInfo) bool {
	return strings.EqualFold(field.name, "id") || field.column == "id"
}

func (info *modelInfo) selectColumns() []string {
	columns := make([]string, 0, len(info.fields))
	for _, field := range info.fields {
		columns = append(columns, field.column)
	}
	return columns
}

func (info *modelInfo) resolveColumn(name string) (string, error) {
	field, ok := info.byName[name]
	if !ok {
		field, ok = info.byName[strings.ToLower(name)]
	}
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrUnknownField, name)
	}
	return field.column, nil
}

type dbTag struct {
	name    string
	options tagOptions
	ignored bool
}

type tagOptions []string

func parseDBTag(tag string) dbTag {
	if tag == "" {
		return dbTag{}
	}

	parts := strings.Split(tag, ",")
	name := strings.TrimSpace(parts[0])
	if name == "-" {
		return dbTag{ignored: true}
	}

	return dbTag{
		name:    name,
		options: parseTagOptions(strings.Join(parts[1:], ",")),
	}
}

func parseTagOptions(tag string) tagOptions {
	if tag == "" {
		return nil
	}

	options := make(tagOptions, 0)
	for _, option := range strings.Split(tag, ",") {
		option = strings.ToLower(strings.TrimSpace(option))
		if option != "" {
			options = append(options, option)
		}
	}
	return options
}

func (options tagOptions) merge(other tagOptions) tagOptions {
	if len(options) == 0 {
		return other
	}
	if len(other) == 0 {
		return options
	}

	merged := make(tagOptions, 0, len(options)+len(other))
	merged = append(merged, options...)
	merged = append(merged, other...)
	return merged
}

func (options tagOptions) has(names ...string) bool {
	for _, option := range options {
		for _, name := range names {
			if option == name {
				return true
			}
		}
	}
	return false
}

func validIdentifier(identifier string) bool {
	if identifier == "" {
		return false
	}
	for index, char := range identifier {
		if index == 0 {
			if char != '_' && !unicode.IsLetter(char) {
				return false
			}
			continue
		}
		if char != '_' && !unicode.IsLetter(char) && !unicode.IsDigit(char) {
			return false
		}
	}
	return true
}

func validIdentifierPath(identifier string) bool {
	parts := strings.Split(identifier, ".")
	for _, part := range parts {
		if !validIdentifier(part) {
			return false
		}
	}
	return true
}

func toSnakeCase(value string) string {
	if value == "" {
		return ""
	}

	runes := []rune(value)
	var builder strings.Builder
	for index, char := range runes {
		if unicode.IsUpper(char) {
			if index > 0 {
				previous := runes[index-1]
				nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
				if unicode.IsLower(previous) || unicode.IsDigit(previous) || nextIsLower {
					builder.WriteByte('_')
				}
			}
			builder.WriteRune(unicode.ToLower(char))
			continue
		}
		builder.WriteRune(char)
	}
	return builder.String()
}

func isZero(value reflect.Value) bool {
	return value.IsZero()
}
