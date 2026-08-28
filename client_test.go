package sqlrud

import (
	"context"
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

type duplicateColumnUser struct {
	ID           int64  `db:"id,primary_key"`
	Email        string `db:"email"`
	PrimaryEmail string `db:"email"`
}

type ambiguousLookupUser struct {
	ID       int64  `db:"id,primary_key"`
	Name     string `db:"full_name"`
	FullName string `db:"name"`
}

type upsertTaggedUser struct {
	ID        int64  `db:"id,primary_key"`
	Name      string `db:"name"`
	CreatedAt string `db:"created_at,createonly"`
	UpdatedAt string `db:"updated_at,updateonly"`
	Ignored   string `db:"ignored,readonly"`
	Nickname  string `db:"nickname,omitempty"`
}

func (upsertTaggedUser) TableName() string {
	return "users"
}

type membership struct {
	OrgID  int64  `db:"org_id,primary_key"`
	UserID int64  `db:"user_id,primary_key"`
	Role   string `db:"role"`
}

func (membership) TableName() string {
	return "memberships"
}

type primaryOnlyUser struct {
	ID int64 `db:"id,primary_key"`
}

func (primaryOnlyUser) TableName() string {
	return "users"
}

type untaggedUser struct {
	ID        int64
	CreatedAt string
}

func (untaggedUser) TableName() string {
	return "users"
}

type EmbeddedFields struct {
	ID        int64  `db:"id,auto_increment,primary_key"`
	CreatedAt string `db:"created_at,omitempty"`
}

type embeddedUser struct {
	*EmbeddedFields
	Name string `db:"name"`
}

func (embeddedUser) TableName() string {
	return "users"
}

type privateEmbeddedFields struct {
	ID int64 `db:"id,primary_key"`
}

type privateEmbeddedUser struct {
	privateEmbeddedFields
	Name string `db:"name"`
}

type joinedUserPost struct {
	UserID    int64  `db:"user_id"`
	PostTitle string `db:"post_title"`
}

func TestFirst(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id", "name", "email"}).AddRow(int64(1), "Yusuke", "y@example.com")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, email FROM users WHERE id = ? LIMIT ?")).
		WithArgs(int64(1), 1).
		WillReturnRows(rows)

	user := testUser{ID: 1}
	if err := client.First(context.Background(), &user); err != nil {
		t.Fatalf("First returned error: %v", err)
	}

	if user.ID != 1 || user.Name != "Yusuke" || user.Email != "y@example.com" {
		t.Fatalf("unexpected user: %+v", user)
	}
	assertExpectations(t, mock)
}

func TestFirstRequiresPrimaryKeyValue(t *testing.T) {
	client, _, cleanup := newMockClient(t)
	defer cleanup()

	var user testUser
	if err := client.First(context.Background(), &user); !errors.Is(err, ErrMissingPrimaryValue) {
		t.Fatalf("expected ErrMissingPrimaryValue, got %v", err)
	}
}

func TestFirstScansUntaggedFieldsWithSnakeCaseNames(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id", "created_at"}).AddRow(int64(1), "2026-07-13")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, created_at FROM users WHERE id = ? LIMIT ?")).
		WithArgs(int64(1), 1).
		WillReturnRows(rows)

	user := untaggedUser{ID: 1}
	if err := client.First(context.Background(), &user); err != nil {
		t.Fatalf("First returned error: %v", err)
	}
	if user.CreatedAt != "2026-07-13" {
		t.Fatalf("expected CreatedAt to be scanned, got %+v", user)
	}
	assertExpectations(t, mock)
}

func TestFirstUsesFieldsFromEmbeddedStruct(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id", "created_at", "name"}).AddRow(int64(1), "2026-07-13", "Yusuke")
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, created_at, name FROM users WHERE id = ? LIMIT ?")).
		WithArgs(int64(1), 1).
		WillReturnRows(rows)

	user := embeddedUser{EmbeddedFields: &EmbeddedFields{ID: 1}}
	if err := client.First(context.Background(), &user); err != nil {
		t.Fatalf("First returned error: %v", err)
	}
	if user.ID != 1 || user.CreatedAt != "2026-07-13" || user.Name != "Yusuke" {
		t.Fatalf("unexpected user: %+v", user)
	}
	assertExpectations(t, mock)
}

func TestModelRejectsUnexportedEmbeddedStruct(t *testing.T) {
	client, _, cleanup := newMockClient(t)
	defer cleanup()

	user := privateEmbeddedUser{privateEmbeddedFields: privateEmbeddedFields{ID: 1}}
	err := client.First(context.Background(), &user)
	if !errors.Is(err, ErrInvalidModel) {
		t.Fatalf("expected ErrInvalidModel, got %v", err)
	}
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

func TestFindRejectsMySQLOffsetWithoutLimit(t *testing.T) {
	client, _, cleanup := newMockClientWithDriver(t, "mysql")
	defer cleanup()

	var users []testUser
	err := client.Find(context.Background(), &users, Offset(10))
	if !errors.Is(err, ErrUnsupportedOption) {
		t.Fatalf("expected ErrUnsupportedOption, got %v", err)
	}
}

func TestFindAllowsMySQLLimitWithOffset(t *testing.T) {
	client, mock, cleanup := newMockClientWithDriver(t, "mysql")
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id", "name", "email"})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, email FROM users LIMIT ? OFFSET ?")).
		WithArgs(20, 10).
		WillReturnRows(rows)

	var users []testUser
	if err := client.Find(context.Background(), &users, Limit(20), Offset(10)); err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestFindAllowsPostgresOffsetWithoutLimit(t *testing.T) {
	client, mock, cleanup := newMockClientWithDriver(t, "postgres")
	defer cleanup()

	rows := sqlmock.NewRows([]string{"id", "name", "email"})
	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, name, email FROM users OFFSET $1")).
		WithArgs(10).
		WillReturnRows(rows)

	var users []testUser
	if err := client.Find(context.Background(), &users, Offset(10)); err != nil {
		t.Fatalf("Find returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestQueryExecutesSQLExactlyAsProvided(t *testing.T) {
	client, mock, cleanup := newMockClientWithDriver(t, "postgres")
	defer cleanup()

	query := "SELECT u.id AS user_id, p.title AS post_title FROM users u JOIN posts p ON p.user_id = u.id WHERE u.id = $1 AND p.metadata ? $2"
	rows := sqlmock.NewRows([]string{"user_id", "post_title"}).AddRow(int64(1), "First post")
	mock.ExpectQuery(regexp.QuoteMeta(query)).
		WithArgs(int64(1), "published").
		WillReturnRows(rows)

	queryRows, err := client.Query(context.Background(), query, int64(1), "published")
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	defer queryRows.Close()

	if !queryRows.Next() {
		t.Fatal("expected one row")
	}
	var result joinedUserPost
	if err := queryRows.StructScan(&result); err != nil {
		t.Fatalf("StructScan returned error: %v", err)
	}
	if result.UserID != 1 || result.PostTitle != "First post" {
		t.Fatalf("unexpected query result: %+v", result)
	}
	if queryRows.Next() {
		t.Fatal("expected only one row")
	}
	if err := queryRows.Err(); err != nil {
		t.Fatalf("rows returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestQueryUsesTransaction(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectQuery(regexp.QuoteMeta("SELECT name FROM users WHERE id = ?")).
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("Yusuke"))
	mock.ExpectCommit()

	err := client.Transaction(context.Background(), func(tx *Client) error {
		rows, err := tx.Query(context.Background(), "SELECT name FROM users WHERE id = ?", int64(1))
		if err != nil {
			return err
		}
		defer rows.Close()

		if !rows.Next() {
			return errors.New("expected one row")
		}
		var name string
		if err := rows.Scan(&name); err != nil {
			return err
		}
		if name != "Yusuke" {
			return errors.New("unexpected name")
		}
		return rows.Err()
	})
	if err != nil {
		t.Fatalf("Transaction returned error: %v", err)
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

func TestCreateHandlesNilEmbeddedStructPointer(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (name) VALUES (?)")).
		WithArgs("Yusuke").
		WillReturnResult(sqlmock.NewResult(1, 1))

	user := embeddedUser{Name: "Yusuke"}
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

func TestUpdateRejectsUnsupportedMutationOption(t *testing.T) {
	client, _, cleanup := newMockClient(t)
	defer cleanup()

	user := testUser{ID: 1, Name: "Yusuke", Email: "new@example.com"}
	err := client.Update(context.Background(), &user, Limit(1))
	if !errors.Is(err, ErrUnsupportedOption) {
		t.Fatalf("expected ErrUnsupportedOption, got %v", err)
	}
}

func TestUpdateAllowsWhereOption(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET name = ?, email = ? WHERE email = ?")).
		WithArgs("Yusuke", "new@example.com", "old@example.com").
		WillReturnResult(sqlmock.NewResult(0, 1))

	user := testUser{Name: "Yusuke", Email: "new@example.com"}
	if err := client.Update(context.Background(), &user, Where("Email", Eq("old@example.com"))); err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestCreateOrUpdateUsesPostgresAtomicUpsert(t *testing.T) {
	client, mock, cleanup := newMockClientWithDriver(t, "postgres")
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (id, name, email) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET name = $4, email = $5")).
		WithArgs(int64(1), "Yusuke", "new@example.com", "Yusuke", "new@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))

	user := testUser{ID: 1, Name: "Yusuke", Email: "new@example.com"}
	if err := client.CreateOrUpdate(context.Background(), &user); err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestCreateOrUpdateUsesMySQLAtomicUpsert(t *testing.T) {
	client, mock, cleanup := newMockClientWithDriver(t, "mysql")
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (id, name, email) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE name = ?, email = ?")).
		WithArgs(int64(1), "Yusuke", "new@example.com", "Yusuke", "new@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))

	user := testUser{ID: 1, Name: "Yusuke", Email: "new@example.com"}
	if err := client.CreateOrUpdate(context.Background(), &user); err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestCreateOrUpdateCreatesWhenPrimaryKeyIsZero(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (name, email) VALUES (?, ?)")).
		WithArgs("Yusuke", "y@example.com").
		WillReturnResult(sqlmock.NewResult(1, 1))

	user := testUser{Name: "Yusuke", Email: "y@example.com"}
	if err := client.CreateOrUpdate(context.Background(), &user); err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestCreateOrUpdateRejectsUnsupportedDialect(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	user := testUser{ID: 1, Name: "Yusuke", Email: "y@example.com"}
	err := client.CreateOrUpdate(context.Background(), &user)
	if !errors.Is(err, ErrUnsupportedDialect) {
		t.Fatalf("expected ErrUnsupportedDialect, got %v", err)
	}
	assertExpectations(t, mock)
}

func TestCreateOrUpdateHonorsColumnWriteOptions(t *testing.T) {
	client, mock, cleanup := newMockClientWithDriver(t, "postgres")
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (id, name, created_at) VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET name = $4, updated_at = $5")).
		WithArgs(int64(7), "Yusuke", "created", "Yusuke", "updated").
		WillReturnResult(sqlmock.NewResult(1, 1))

	user := upsertTaggedUser{
		ID:        7,
		Name:      "Yusuke",
		CreatedAt: "created",
		UpdatedAt: "updated",
		Ignored:   "ignored",
	}
	if err := client.CreateOrUpdate(context.Background(), &user); err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestCreateOrUpdatePostgresCompositePrimaryKey(t *testing.T) {
	client, mock, cleanup := newMockClientWithDriver(t, "postgres")
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO memberships (org_id, user_id, role) VALUES ($1, $2, $3) ON CONFLICT (org_id, user_id) DO UPDATE SET role = $4")).
		WithArgs(int64(10), int64(20), "owner", "owner").
		WillReturnResult(sqlmock.NewResult(1, 1))

	model := membership{OrgID: 10, UserID: 20, Role: "owner"}
	if err := client.CreateOrUpdate(context.Background(), &model); err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestCreateOrUpdatePostgresDoesNothingWhenNoUpdateColumns(t *testing.T) {
	client, mock, cleanup := newMockClientWithDriver(t, "postgres")
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (id) VALUES ($1) ON CONFLICT (id) DO NOTHING")).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(1, 0))

	user := primaryOnlyUser{ID: 1}
	if err := client.CreateOrUpdate(context.Background(), &user); err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}
	assertExpectations(t, mock)
}

func TestCreateOrUpdateMySQLNoOpWhenNoUpdateColumns(t *testing.T) {
	client, mock, cleanup := newMockClientWithDriver(t, "mysql")
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (id) VALUES (?) ON DUPLICATE KEY UPDATE id = id")).
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(1, 0))

	user := primaryOnlyUser{ID: 1}
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

func TestDeleteRejectsUnsupportedMutationOption(t *testing.T) {
	client, _, cleanup := newMockClient(t)
	defer cleanup()

	user := testUser{ID: 1}
	err := client.Delete(context.Background(), &user, OrderBy("ID", Desc))
	if !errors.Is(err, ErrUnsupportedOption) {
		t.Fatalf("expected ErrUnsupportedOption, got %v", err)
	}
}

func TestDeleteAllowsWhereOption(t *testing.T) {
	client, mock, cleanup := newMockClient(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("DELETE FROM users WHERE email = ?")).
		WithArgs("y@example.com").
		WillReturnResult(sqlmock.NewResult(0, 1))

	var user testUser
	if err := client.Delete(context.Background(), &user, Where("Email", Eq("y@example.com"))); err != nil {
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

func TestModelRejectsDuplicateColumn(t *testing.T) {
	client, _, cleanup := newMockClient(t)
	defer cleanup()

	err := client.Create(context.Background(), &duplicateColumnUser{ID: 1})
	if !errors.Is(err, ErrDuplicateColumn) {
		t.Fatalf("expected ErrDuplicateColumn, got %v", err)
	}
}

func TestModelRejectsAmbiguousFieldLookup(t *testing.T) {
	client, _, cleanup := newMockClient(t)
	defer cleanup()

	err := client.Create(context.Background(), &ambiguousLookupUser{ID: 1})
	if !errors.Is(err, ErrAmbiguousField) {
		t.Fatalf("expected ErrAmbiguousField, got %v", err)
	}
}

func newMockClient(t *testing.T) (*Client, sqlmock.Sqlmock, func()) {
	return newMockClientWithDriver(t, "sqlmock")
}

func newMockClientWithDriver(t *testing.T, driverName string) (*Client, sqlmock.Sqlmock, func()) {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}

	sqlxDB := sqlx.NewDb(db, driverName)
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
