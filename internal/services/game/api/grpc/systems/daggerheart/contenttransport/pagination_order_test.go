package contenttransport

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
)

func TestValidateKeySpec(t *testing.T) {
	t.Run("matching specs", func(t *testing.T) {
		base := []pagination.CursorValue{
			pagination.StringValue("name", "a"),
			pagination.StringValue("id", "1"),
		}
		candidate := []pagination.CursorValue{
			pagination.StringValue("name", "b"),
			pagination.StringValue("id", "2"),
		}
		if err := validateKeySpec(base, candidate); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("length mismatch", func(t *testing.T) {
		base := []pagination.CursorValue{pagination.StringValue("name", "a")}
		candidate := []pagination.CursorValue{
			pagination.StringValue("name", "a"),
			pagination.StringValue("id", "1"),
		}
		if err := validateKeySpec(base, candidate); err == nil {
			t.Fatal("expected error for length mismatch")
		}
	})

	t.Run("name mismatch", func(t *testing.T) {
		base := []pagination.CursorValue{pagination.StringValue("name", "a")}
		candidate := []pagination.CursorValue{pagination.StringValue("id", "a")}
		if err := validateKeySpec(base, candidate); err == nil {
			t.Fatal("expected error for name mismatch")
		}
	})

	t.Run("kind mismatch", func(t *testing.T) {
		base := []pagination.CursorValue{pagination.StringValue("name", "a")}
		candidate := []pagination.CursorValue{pagination.IntValue("name", 1)}
		if err := validateKeySpec(base, candidate); err == nil {
			t.Fatal("expected error for kind mismatch")
		}
	})
}
