package sqlrud

import (
	"testing"
)

func TestMySQLDialect(t *testing.T) {
	d := MySQL
	if got := d.Quote("users"); got != "`users`" {
		t.Errorf("Quote() = %q, want %q", got, "`users`")
	}
	if got := d.Placeholder(1); got != "?" {
		t.Errorf("Placeholder() = %q, want %q", got, "?")
	}
}

func TestPostgresDialect(t *testing.T) {
	d := Postgres
	if got := d.Quote("users"); got != `"users"` {
		t.Errorf("Quote() = %q, want %q", got, `"users"`)
	}
	if got := d.Placeholder(1); got != "$1" {
		t.Errorf("Placeholder(1) = %q, want %q", got, "$1")
	}
	if got := d.Placeholder(3); got != "$3" {
		t.Errorf("Placeholder(3) = %q, want %q", got, "$3")
	}
}
