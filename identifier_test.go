package sqlrud

import (
	"testing"
)

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name    string
		dialect Dialect
		id      string
		want    string
		wantErr bool
	}{
		{"mysql valid", MySQL, "users", "`users`", false},
		{"mysql underscore", MySQL, "user_roles", "`user_roles`", false},
		{"postgres valid", Postgres, "users", `"users"`, false},
		{"invalid space", MySQL, "user roles", "", true},
		{"invalid hyphen", MySQL, "user-roles", "", true},
		{"invalid semicolon", MySQL, "users;DROP TABLE", "", true},
		{"empty", MySQL, "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := quoteIdentifier(tt.dialect, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("quoteIdentifier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("quoteIdentifier() = %q, want %q", got, tt.want)
			}
		})
	}
}
