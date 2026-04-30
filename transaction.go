package sqlrud

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// WithTransaction executes fn inside a transaction. If fn returns an error
// the transaction is rolled back; otherwise it is committed.
func WithTransaction(ctx context.Context, db *sqlx.DB, dialect Dialect, fn func(r *Rudder) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("sqlrud: begin transaction: %w", err)
	}

	r := New(tx, dialect)
	if err := fn(r); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("sqlrud: rollback: %w (original: %v)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("sqlrud: commit: %w", err)
	}
	return nil
}
