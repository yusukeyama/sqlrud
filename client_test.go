package sqlrud

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
)

type testUser struct {
	ID    int64  `db:"id,auto_increment,primary_key"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

func (testUser) TableName() string {
	return "users"
}

func TestFirst(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id", "name", "email"}).AddRow(int64(1), "Yusuke", "y@example.com")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, email FROM users WHERE email = ? LIMIT ?")).
		WithArgs("y@example.com", 1).
		WillReturnRows(rows)

	var user testUser
	if err := client.First(context.Background(), &user, Where("Email", Eq("y@example.com"))); err != nil {
		t.Fatalf("First returned error: %v", err)
	}

	if user.ID != 1 || user.Name != "Yusuke" || user.Email != "y@example.com" {
		t.Fatalf("unexpected user: %+v", user)
	}
	assertExpectations(t, mock)
}

func TestFind(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id", "name", "email"}).
		AddRow(int64(2), "Yusuke", "y@example.com").
		AddRow(int64(3), "Yuki", "yuki@example.com")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, email FROM users WHERE name LIKE ? ORDER BY id DESC LIMIT ?")).
		WithArgs("Yu%", 10).
		WillReturnRows(rows)

	var users []testUser
	if err := client.Find(context.Background(), &users, Where("Name", Like("Yu%")), OrderBy("ID", Desc), Limit(10)); err != nil {
		t.Fatalf("Find returned error: %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	assertExpectations(t, mock)
}

func TestCreate(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (name, email) VALUES (?, ?)")).
		WithArgs("Yusuke", "y@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))

	user := testUser{Name: "Yusuke", Email: "y@example.com"}
	if err := client.Create(context.Background(), &user); err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestUpdate(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET name = ?, email = ? WHERE id = ?")).
		WithArgs("Yusuke", "new@example.com", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	user := testUser{ID: 1, Name: "Yusuke", Email: "new@example.com"}
	if err := client.Update(context.Background(), &user); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestCreateOrUpdateUpdatesExistingRecord(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM users WHERE id = ? LIMIT 1")).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))
	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET name = ?, email = ? WHERE id = ?")).
		WithArgs("Yusuke", "new@example.com", int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	user := testUser{ID: 1, Name: "Yusuke", Email: "new@example.com"}
	if err := client.CreateOrUpdate(context.Background(), &user); err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestCreateOrUpdateCreatesMissingRecord(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	mock.ExpectQuery(regexp.QuoteMeta("SELECT 1 FROM users WHERE id = ? LIMIT 1")).
		WithArgs(int64(1)).
		WillReturnError(sql.ErrNoRows)
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (id, name, email) VALUES (?, ?, ?)")).
		WithArgs(int64(1), "Yusuke", "y@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))

	user := testUser{ID: 1, Name: "Yusuke", Email: "y@example.com"}
	if err := client.CreateOrUpdate(context.Background(), &user); err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestDelete(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM users WHERE id = ?")).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	user := testUser{ID: 1}
	if err := client.Delete(context.Background(), &user); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestTransactionRollsBackOnError(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	expectedErr := errors.New("stop")
	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (name, email) VALUES (?, ?)")).
		WithArgs("Yusuke", "y@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectRollback()

	err := client.Transaction(context.Background(), func(tx *Client) error {
		if err := tx.Create(context.Background(), &testUser{Name: "Yusuke", Email: "y@example.com"}); err != nil {
			return err
		}
		return expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected rollback error %v, got %v", expectedErr, err)
	}
	assertExpectations(t, mock)
}

func TestTransactionRollsBackOnPanic(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (name, email) VALUES (?, ?)")).
		WithArgs("Yusuke", "y@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectRollback()

	defer func() {
		recovered := recover()
		if recovered != "stop" {
			t.Fatalf("expected panic %q, got %v", "stop", recovered)
		}
		assertExpectations(t, mock)
	}()

	_ = client.Transaction(context.Background(), func(tx *Client) error {
		if err := tx.Create(context.Background(), &testUser{Name: "Yusuke", Email: "y@example.com"}); err != nil {
			return err
		}
		panic("stop")
	})
}

func newMockClient(t *testing.T) (*Client, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	sqlxDB := sqlx.NewDb(db, "sqlmock")
	cleanup := func() {
		_ = sqlxDB.Close()
	}
	return New(sqlxDB), mock, cleanup
}

func assertExpectations(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
