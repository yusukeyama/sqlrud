//go:build ignore
// +build ignore

// This example demonstrates using WithTransaction.
package main

import (
	"context"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/yusukeyama/sqlrud"
)

type Order struct {
	ID     int `db:"id"`
	Amount int `db:"amount"`
}

func main() {
	db, err := sqlx.Open("mysql", "user:pass@tcp(localhost:3306)/testdb?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	err = sqlrud.WithTransaction(ctx, db, sqlrud.MySQL, func(r *sqlrud.Rudder) error {
		o1 := Order{Amount: 100}
		if err := r.Insert(ctx, "orders", &o1); err != nil {
			return err
		}
		o2 := Order{Amount: 200}
		if err := r.Insert(ctx, "orders", &o2); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("transaction committed")
}
