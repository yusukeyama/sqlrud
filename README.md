# sqlrud

A small CRUD helper for [sqlx](https://github.com/jmoiron/sqlx) that generates
basic `INSERT`, `UPDATE`, and `DELETE` statements from struct `db` tags.

> **This is not a full-featured ORM.**  
> Complex `SELECT` queries are written by hand; sqlrud only handles the
> repetitive mutation side of CRUD.

## Installation

```bash
go get github.com/yusukeyama/sqlrud
```

Go 1.21 or later is required.

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    _ "github.com/go-sql-driver/mysql"
    "github.com/jmoiron/sqlx"
    "github.com/yusukeyama/sqlrud"
)

// User maps to the `users` table.
type User struct {
    ID    int    `db:"id,auto_increment"`
    Name  string `db:"name"`
    Email string `db:"email"`
}

func main() {
    db, err := sqlx.Connect("mysql", "user:pass@tcp(127.0.0.1:3306)/mydb")
    if err != nil {
        log.Fatal(err)
    }

    crud := sqlrud.New()
    ctx := context.Background()

    // Insert
    user := &User{Name: "Alice", Email: "alice@example.com"}
    result, err := crud.Insert(ctx, db, "users", user)
    if err != nil {
        log.Fatal(err)
    }
    id, _ := result.LastInsertId()
    user.ID = int(id)
    fmt.Println("inserted:", user.ID)

    // Update (WHERE uses the auto_increment field by default)
    user.Name = "Alice Smith"
    if _, err = crud.Update(ctx, db, "users", user); err != nil {
        log.Fatal(err)
    }

    // Query (you write the SELECT; sqlrud scans the result)
    var found User
    err = sqlrud.FirstInto(ctx, db, &found, "SELECT * FROM users WHERE id = ?", user.ID)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("found: %+v\n", found)

    // Delete
    if _, err = crud.Delete(ctx, db, "users", user); err != nil {
        log.Fatal(err)
    }
}
```

## Struct tags

| Tag                      | Meaning                                         |
|--------------------------|-------------------------------------------------|
| `db:"column_name"`       | Maps the field to the given column.             |
| `db:"id,auto_increment"` | Marks the field as an auto-increment PK.        |
| `db:"-"`                 | Excludes the field from all CRUD operations.    |

Auto-increment fields are automatically excluded from `INSERT` statements and
used as the default `WHERE` column in `UPDATE` / `DELETE`.

## API

### CRUD methods

```go
crud := sqlrud.New()

// Insert — auto_increment fields are excluded automatically.
result, err := crud.Insert(ctx, db, "users", &user)

// Update — WHERE defaults to the auto_increment field.
result, err := crud.Update(ctx, db, "users", &user,
    sqlrud.WhereColumns("id"),
    sqlrud.ExcludeColumns("created_at", "updated_at"),
)

// Delete — WHERE defaults to the auto_increment field.
result, err := crud.Delete(ctx, db, "users", &user,
    sqlrud.WhereColumns("id"),
)
```

### Query helpers

```go
// FirstInto — wraps GetContext; returns ErrNotFound for no rows.
var u User
err := sqlrud.FirstInto(ctx, db, &u, "SELECT * FROM users WHERE id = ?", 1)

// FindInto — wraps SelectContext; returns an empty slice (not nil) for no rows.
var users []User
err := sqlrud.FindInto(ctx, db, &users, "SELECT * FROM users WHERE active = ?", true)

// Count — executes a COUNT query.
n, err := sqlrud.Count(ctx, db, "SELECT COUNT(*) FROM users")
```

### Transaction

```go
err := sqlrud.Transaction(ctx, db, func(ctx context.Context, tx *sqlx.Tx) error {
    _, err := crud.Insert(ctx, tx, "users", &user)
    return err
})
```

### Error handling

```go
if errors.Is(err, sqlrud.ErrNotFound)       { /* query returned no rows    */ }
if errors.Is(err, sqlrud.ErrDuplicate)      { /* unique constraint violated */ }
if errors.Is(err, sqlrud.ErrNoRowsAffected) { /* update/delete hit nothing  */ }
if errors.Is(err, sqlrud.ErrInvalidArgument){ /* bad table/column name      */ }
```

### Options

```go
crud := sqlrud.New(
    sqlrud.WithMySQLDialect(),                          // default
    sqlrud.WithDefaultInsertExcludeColumns("created_at", "updated_at"),
    sqlrud.WithDefaultUpdateExcludeColumns("id", "created_at", "updated_at"),
    sqlrud.WithAutoIncrementTag("auto_increment"),      // default
)
```

## Dialect support

Currently **MySQL** is officially supported.  
Other databases may work for basic `INSERT` / `UPDATE` / `DELETE` operations
but are not fully tested.  Implement the `Dialect` interface and pass it with
`WithDialect(d)` to add support for another database.

## Security

**Do not pass untrusted strings as table or column names.**  
sqlrud validates all identifier arguments against the pattern
`^[A-Za-z_][A-Za-z0-9_]*$` and returns `ErrInvalidArgument` for any name
that does not match. Application-level tenant IDs or other external values
used to construct table names should be validated independently before use.

## License

MIT — see [LICENSE](LICENSE).
