package sqlrud

import (
	"testing"
)

func TestExtractFields(t *testing.T) {
	type User struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
		skip string // unexported
	}

	u := User{ID: 1, Name: "Alice"}
	fields, err := extractFields(&u)
	if err != nil {
		t.Fatalf("extractFields() error = %v", err)
	}
	if len(fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(fields))
	}
	if fields[0].column != "id" || fields[0].value != 1 {
		t.Errorf("unexpected field[0]: %+v", fields[0])
	}
	if fields[1].column != "name" || fields[1].value != "Alice" {
		t.Errorf("unexpected field[1]: %+v", fields[1])
	}
}

func TestExtractFields_DashTag(t *testing.T) {
	type Partial struct {
		Name  string `db:"name"`
		Extra string `db:"-"`
	}
	p := Partial{Name: "Bob", Extra: "ignored"}
	fields, err := extractFields(&p)
	if err != nil {
		t.Fatalf("extractFields() error = %v", err)
	}
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	if fields[0].column != "name" {
		t.Errorf("unexpected column: %s", fields[0].column)
	}
}

func TestExtractFields_NonStruct(t *testing.T) {
	_, err := extractFields("not a struct")
	if err == nil {
		t.Error("expected error for non-struct input")
	}
}
