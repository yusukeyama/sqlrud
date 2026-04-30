package sqlrud

import (
	"errors"
	"testing"
)

func TestErrors(t *testing.T) {
	if !errors.Is(ErrNotFound, ErrNotFound) {
		t.Error("ErrNotFound should be itself")
	}
	if !errors.Is(ErrNoRowsAffected, ErrNoRowsAffected) {
		t.Error("ErrNoRowsAffected should be itself")
	}
	if errors.Is(ErrNotFound, ErrNoRowsAffected) {
		t.Error("ErrNotFound should not equal ErrNoRowsAffected")
	}
}
