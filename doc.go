// Package sqlrud provides small CRUD helpers for github.com/jmoiron/sqlx.
//
// It is not a full-featured ORM. It generates basic INSERT, UPDATE,
// DELETE and SELECT queries from struct tags and delegates query execution
// to sqlx-compatible interfaces.
//
// # Basic usage
//
//	type User struct {
//		ID   int    `db:"id"`
//		Name string `db:"name"`
//	}
//
//	db, _ := sqlx.Open("mysql", dsn)
//	r := sqlrud.New(db, sqlrud.MySQL)
//
//	// Insert
//	user := User{Name: "Alice"}
//	err := r.Insert(ctx, "users", &user)
//
//	// Find all
//	var users []User
//	err = r.Find(ctx, "users", &users, "")
//
//	// Update
//	err = r.Update(ctx, "users", &user, "id = ?", user.ID)
//
//	// Delete
//	err = r.Delete(ctx, "users", "id = ?", user.ID)
package sqlrud
