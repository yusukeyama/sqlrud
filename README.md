# sqlrud

Small CRUD helper for `database/sql`-style applications.

`sqlrud` keeps query strings out of everyday CRUD code while using `database/sql` and [sqlx](https://github.com/jmoiron/sqlx) underneath for execution, scanning, and placeholder binding.

## Requirements

- Go 1.22 or later

## Install

```sh
go get github.com/yusukeyama/sqlrud
```

## Model

Use `db` tags for column names and CRUD metadata. Mark primary keys with `primary_key`; add `auto_increment` when the database generates the value.

```go
type User struct {
	ID    int64  `db:"id,auto_increment,primary_key"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

func (User) TableName() string {
	return "users"
}
```

If `TableName` is omitted, the default table name is the snake_case struct name with `s` appended, such as `User` -> `users`.

## Usage

PostgreSQL:

```go
import _ "github.com/lib/pq"

db := sqlx.MustConnect("postgres", dsn)
client := sqlrud.New(db)
```

MySQL:

```go
import _ "github.com/go-sql-driver/mysql"

db := sqlx.MustConnect("mysql", "user:pass@tcp(localhost:3306)/app?parseTime=true")
client := sqlrud.New(db)
```

Then use the client in the same way for either database:

```go
var user User
user.ID = 1
err := client.First(ctx, &user)

var users []User
err = client.Find(
	ctx,
	&users,
	sqlrud.Where("Name", sqlrud.Like("Yu%")),
	sqlrud.OrderBy("ID", sqlrud.Desc),
	sqlrud.Limit(20),
)

err = client.Create(ctx, &User{Name: "Yusuke", Email: "y@example.com"})

user.Name = "New Name"
err = client.Update(ctx, &user)

err = client.CreateOrUpdate(ctx, &user)

err = client.Delete(ctx, &user)
```

`First` loads a record by primary key. `Find` accepts field names (`Email`) or column names (`email`) in conditions. Supported predicates are `Eq`, `NotEq`, `Gt`, `Gte`, `Lt`, `Lte`, `Like`, `In`, `NotIn`, `IsNull`, and `IsNotNull`.

## Transactions

Pass an anonymous function to `Transaction`. Returning an error rolls back; returning `nil` commits.

```go
err := client.Transaction(ctx, func(tx *sqlrud.Client) error {
	if err := tx.Create(ctx, &user); err != nil {
		return err
	}

	return tx.Update(ctx, &profile)
})
```

## Tags

| `db` option | Meaning |
| --- | --- |
| `primary_key` | Primary key used by `Update`, `Delete`, and `CreateOrUpdate` |
| `auto_increment` | Omit this field on `Create` when it has the zero value |
| `readonly` | Never write this field on create or update |
| `createonly` | Write this field on create, not update |
| `updateonly` | Write this field on update, not create |
| `omitempty` | Omit this field when it has the zero value |

`primary`, `pk`, and `auto` are also accepted as aliases.

## Notes

- `Update` and `Delete` require either a non-zero primary key value or explicit `Where` conditions.
- `CreateOrUpdate` checks existence by primary key. If the primary key is zero, it performs `Create`.
- `CreateOrUpdate` is implemented as a read followed by create or update, not as a database-native upsert.
- SQL identifiers come from struct metadata and are validated before query execution.
