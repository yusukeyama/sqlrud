package sqlrud

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// TxFunc is a function that runs inside a database transaction.
// Return a non-nil error to trigger a rollback; return nil to commit.
type TxFunc func(ctx context.Context, tx *sqlx.Tx) error

// Transaction begins a transaction on db, calls fn, and commits on success or
// rolls back on error. Panics inside fn are recovered, the transaction is
// rolled back, and the panic is re-raised.
func Transaction(ctx context.Context, db *sqlx.DB, fn TxFunc) (err error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlrud: begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = fn(ctx, tx); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("sqlrud: commit transaction: %w", err)
	}

	return nil
}
