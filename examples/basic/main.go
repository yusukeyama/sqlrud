// Package main demonstrates basic usage of github.com/yusukeyama/sqlrud.
//
// This example is intentionally not runnable without a database; it shows the
// intended API shape and is compiled as part of CI via `go build ./...`.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/yusukeyama/sqlrud"
)

// User is an example domain model. The `db` struct tag maps fields to SQL
// column names. The `auto_increment` option marks the primary key.
type User struct {
	ID    int    `db:"id,auto_increment"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

func main() {
	dsn := "user:password@tcp(127.0.0.1:3306)/mydb?parseTime=true"
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer db.Close()

	crud := sqlrud.New(sqlrud.WithMySQLDialect())
	ctx := context.Background()

	// --- Insert ---
	user := &User{Name: "Alice", Email: "alice@example.com"}
	result, err := crud.Insert(ctx, db, "users", user)
	if err != nil {
		if errors.Is(err, sqlrud.ErrDuplicate) {
			log.Println("duplicate entry")
		} else {
			log.Fatalf("insert: %v", err)
		}
	}
	id, _ := result.LastInsertId()
	user.ID = int(id)
	fmt.Printf("inserted user id=%d\n", user.ID)

	// --- Update ---
	user.Name = "Alice Smith"
	_, err = crud.Update(ctx, db, "users", user)
	if err != nil {
		log.Fatalf("update: %v", err)
	}

	// --- Query ---
	var found User
	err = sqlrud.FirstInto(ctx, db, &found, "SELECT * FROM users WHERE id = ?", user.ID)
	if err != nil {
		if errors.Is(err, sqlrud.ErrNotFound) {
			log.Println("user not found")
		} else {
			log.Fatalf("query: %v", err)
		}
	}
	fmt.Printf("found: %+v\n", found)

	// --- Transaction ---
	err = sqlrud.Transaction(ctx, db, func(ctx context.Context, tx *sqlx.Tx) error {
		another := &User{Name: "Bob", Email: "bob@example.com"}
		_, err := crud.Insert(ctx, tx, "users", another)
		return err
	})
	if err != nil {
		log.Fatalf("transaction: %v", err)
	}

	// --- Delete ---
	_, err = crud.Delete(ctx, db, "users", user)
	if err != nil {
		log.Fatalf("delete: %v", err)
	}
	fmt.Println("done")
}
