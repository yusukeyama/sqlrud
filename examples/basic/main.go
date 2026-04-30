//go:build ignore
// +build ignore

// This example demonstrates basic CRUD operations with sqlrud.
package main

import (
	"context"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/yusukeyama/sqlrud"
)

type User struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

func main() {
	db, err := sqlx.Open("mysql", "user:pass@tcp(localhost:3306)/testdb?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()
	r := sqlrud.New(db, sqlrud.MySQL)

	// Insert
	user := User{Name: "Alice"}
	if err := r.Insert(ctx, "users", &user); err != nil {
		log.Fatal(err)
	}
	fmt.Println("inserted:", user)

	// Find all
	var users []User
	if err := r.Find(ctx, "users", &users, nil); err != nil {
		log.Fatal(err)
	}
	fmt.Println("users:", users)

	// First by condition
	var found User
	if err := r.First(ctx, "users", &found, "name = ?", "Alice"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("found:", found)

	// Count
	count, err := r.Count(ctx, "users", "", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("count:", count)

	// Delete
	if err := r.Delete(ctx, "users", "name = ?", "Alice"); err != nil {
		log.Fatal(err)
	}
	fmt.Println("deleted")
}
