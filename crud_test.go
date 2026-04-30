package sqlrud_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/yusukeyama/sqlrud"
)

// mockExecer records the last query and arg passed to NamedExecContext.
type mockExecer struct {
	query string
	arg   any
	result sql.Result
	err    error
}

func (m *mockExecer) NamedExecContext(_ context.Context, query string, arg any) (sql.Result, error) {
	m.query = query
	m.arg = arg
	return m.result, m.err
}

type mockResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (r mockResult) LastInsertId() (int64, error) { return r.lastInsertID, nil }
func (r mockResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

// sampleModel is a minimal struct used across tests.
type sampleModel struct {
	ID    int    `db:"id,auto_increment"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

func TestInsert_Basic(t *testing.T) {
	c := sqlrud.New()
	m := &mockExecer{result: mockResult{lastInsertID: 1, rowsAffected: 1}}

	model := &sampleModel{Name: "alice", Email: "alice@example.com"}
	_, err := c.Insert(context.Background(), m, "users", model)
	if err != nil {
		t.Fatalf("Insert returned unexpected error: %v", err)
	}

	want := "INSERT INTO `users` (`name`, `email`) VALUES (:name, :email)"
	if m.query != want {
		t.Errorf("Insert SQL\n got: %s\nwant: %s", m.query, want)
	}
}

func TestInsert_InvalidTable(t *testing.T) {
	c := sqlrud.New()
	m := &mockExecer{}

	_, err := c.Insert(context.Background(), m, "bad-table!", &sampleModel{})
	if err == nil {
		t.Fatal("expected error for invalid table name, got nil")
	}
}

func TestUpdate_Basic(t *testing.T) {
	c := sqlrud.New()
	m := &mockExecer{result: mockResult{rowsAffected: 1}}

	model := &sampleModel{ID: 1, Name: "bob", Email: "bob@example.com"}
	_, err := c.Update(context.Background(), m, "users", model)
	if err != nil {
		t.Fatalf("Update returned unexpected error: %v", err)
	}

	want := "UPDATE `users` SET `name` = :name, `email` = :email WHERE `id` = :id"
	if m.query != want {
		t.Errorf("Update SQL\n got: %s\nwant: %s", m.query, want)
	}
}

func TestDelete_Basic(t *testing.T) {
	c := sqlrud.New()
	m := &mockExecer{result: mockResult{rowsAffected: 1}}

	model := &sampleModel{ID: 42}
	_, err := c.Delete(context.Background(), m, "users", model)
	if err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	want := "DELETE FROM `users` WHERE `id` = :id"
	if m.query != want {
		t.Errorf("Delete SQL\n got: %s\nwant: %s", m.query, want)
	}
}

func TestExcludeColumns(t *testing.T) {
	c := sqlrud.New()
	m := &mockExecer{result: mockResult{lastInsertID: 1}}

	type withTimestamps struct {
		ID        int    `db:"id,auto_increment"`
		Name      string `db:"name"`
		CreatedAt string `db:"created_at"`
	}

	model := &withTimestamps{Name: "carol"}
	_, err := c.Insert(context.Background(), m, "items", model,
		sqlrud.ExcludeColumns("created_at"),
	)
	if err != nil {
		t.Fatalf("Insert with ExcludeColumns returned unexpected error: %v", err)
	}

	want := "INSERT INTO `items` (`name`) VALUES (:name)"
	if m.query != want {
		t.Errorf("Insert SQL\n got: %s\nwant: %s", m.query, want)
	}
}

func TestWhereColumns_Update(t *testing.T) {
	c := sqlrud.New()
	m := &mockExecer{result: mockResult{rowsAffected: 1}}

	type byEmail struct {
		ID    int    `db:"id"`
		Name  string `db:"name"`
		Email string `db:"email"`
	}

	model := &byEmail{Name: "dave", Email: "dave@example.com"}
	_, err := c.Update(context.Background(), m, "users", model,
		sqlrud.WhereColumns("email"),
	)
	if err != nil {
		t.Fatalf("Update returned unexpected error: %v", err)
	}

	want := "UPDATE `users` SET `id` = :id, `name` = :name WHERE `email` = :email"
	if m.query != want {
		t.Errorf("Update SQL\n got: %s\nwant: %s", m.query, want)
	}
}

func TestNew_DefaultsToMySQL(t *testing.T) {
	c := sqlrud.New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
}
