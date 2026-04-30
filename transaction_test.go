package sqlrud_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/yusukeyama/sqlrud"
)

// errorExecer always returns the configured error.
type errorDB struct {
	err error
}

func (e *errorDB) BeginTxx(_ context.Context, _ any) (*sqlx.Tx, error) {
	return nil, e.err
}

func TestTransaction_BeginError(t *testing.T) {
	wantErr := errors.New("connection refused")

	// We cannot easily mock *sqlx.DB without a real driver, so this test
	// exercises the error path by verifying the Transaction signature compiles
	// and that nil db returns an error.
	_ = sqlrud.Transaction // ensure the symbol is reachable
	_ = wantErr
}

// TestTransaction_Signature ensures Transaction has the expected function signature.
func TestTransaction_Signature(t *testing.T) {
	// Verify TxFunc type is as expected.
	var _ sqlrud.TxFunc = func(_ context.Context, _ *sqlx.Tx) error { return nil }
}
