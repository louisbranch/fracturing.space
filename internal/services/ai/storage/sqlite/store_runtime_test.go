package sqlite

import (
	"testing"
)

func TestOpenRequiresPath(t *testing.T) {
	if _, err := Open(""); err == nil {
		t.Fatal("expected error for empty path")
	}
}
