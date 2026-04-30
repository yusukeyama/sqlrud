# sqlrud

[![Go Reference](https://pkg.go.dev/badge/github.com/yusukeyama/sqlrud.svg)](https://pkg.go.dev/github.com/yusukeyama/sqlrud)
[![Go](https://github.com/yusukeyama/sqlrud/actions/workflows/go.yml/badge.svg)](https://github.com/yusukeyama/sqlrud/actions/workflows/go.yml)

Small CRUD helper for [jmoiron/sqlx](https://github.com/jmoiron/sqlx).

sqlrud is **not an ORM**. It generates basic `INSERT`, `UPDATE`, `DELETE`, `COUNT`, and `SELECT` queries from struct `db` tags and delegates execution to sqlx.

## Requirements

Go 1.21 or later.

## Installation

```bash
go get github.com/yusukeyama/sqlrud
```

## Quick start

```go
import "github.com/yusukeyama/sqlrud"

type User struct {
    ID   int    `db:"id"`
    Name string `db:"name"`
}

db, _ := sqlx.Open("mysql", dsn)
r := sqlrud.New(db, sqlrud.MySQL)

ctx := context.Background()

// Insert
user := User{Name: "Alice"}
err := r.Insert(ctx, "users", &user)

// Find all
var users []User
err = r.Find(ctx, "users", &users, nil)

// First matching row
var found User
err = r.First(ctx, "users", &found, "name = ?", "Alice")

// Count
count, err := r.Count(ctx, "users", "name = ?", "Alice")

// Update
err = r.Update(ctx, "users", &user, "id = ?", user.ID)

// Delete
err = r.Delete(ctx, "users", "id = ?", user.ID)
```

## Dialects

| Constant          | Database        |
|-------------------|-----------------|
| `sqlrud.MySQL`    | MySQL / MariaDB |
| `sqlrud.Postgres` | PostgreSQL      |

Implement the `Dialect` interface to add support for other databases.

## Transactions

```go
err := sqlrud.WithTransaction(ctx, db, sqlrud.MySQL, func(r *sqlrud.Rudder) error {
    if err := r.Insert(ctx, "orders", &order1); err != nil {
        return err
    }
    return r.Insert(ctx, "orders", &order2)
})
```

## Error handling

| Error                      | Meaning                                  |
|----------------------------|------------------------------------------|
| `sqlrud.ErrNotFound`       | `First()` found no matching row          |
| `sqlrud.ErrNoRowsAffected` | `Update()` / `Delete()` affected no rows |

## Struct tags

sqlrud reads the standard `db` struct tag used by sqlx:

```go
type Article struct {
    ID      int    `db:"id"`
    Title   string `db:"title"`
    Ignored string `db:"-"` // skipped
}
```

## License

[MIT](LICENSE)
