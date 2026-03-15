package contenttransport

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
)

func TestCompareInts(t *testing.T) {
	tests := []struct {
		left, right int64
		want        int
	}{
		{1, 2, -1},
		{2, 2, 0},
		{3, 2, 1},
		{-5, 5, -1},
		{0, 0, 0},
	}
	for _, tc := range tests {
		if got := compareInts(tc.left, tc.right); got != tc.want {
			t.Errorf("compareInts(%d, %d) = %d, want %d", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestCompareUints(t *testing.T) {
	tests := []struct {
		left, right uint64
		want        int
	}{
		{1, 2, -1},
		{2, 2, 0},
		{3, 2, 1},
		{0, 0, 0},
		{0, 100, -1},
	}
	for _, tc := range tests {
		if got := compareUints(tc.left, tc.right); got != tc.want {
			t.Errorf("compareUints(%d, %d) = %d, want %d", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestCompareCursorValue(t *testing.T) {
	t.Run("string comparison", func(t *testing.T) {
		left := pagination.StringValue("name", "alpha")
		right := pagination.StringValue("name", "beta")
		got, err := compareCursorValue(left, right)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != -1 {
			t.Errorf("expected -1, got %d", got)
		}
	})

	t.Run("int comparison", func(t *testing.T) {
		left := pagination.IntValue("seq", 10)
		right := pagination.IntValue("seq", 5)
		got, err := compareCursorValue(left, right)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 1 {
			t.Errorf("expected 1, got %d", got)
		}
	})

	t.Run("uint comparison", func(t *testing.T) {
		left := pagination.UintValue("id", 7)
		right := pagination.UintValue("id", 7)
		got, err := compareCursorValue(left, right)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 0 {
			t.Errorf("expected 0, got %d", got)
		}
	})

	t.Run("kind mismatch", func(t *testing.T) {
		left := pagination.StringValue("name", "a")
		right := pagination.IntValue("name", 1)
		_, err := compareCursorValue(left, right)
		if err == nil {
			t.Fatal("expected error for kind mismatch")
		}
	})

	t.Run("unsupported kind", func(t *testing.T) {
		left := pagination.CursorValue{Name: "x", Kind: "unknown"}
		right := pagination.CursorValue{Name: "x", Kind: "unknown"}
		_, err := compareCursorValue(left, right)
		if err == nil {
			t.Fatal("expected error for unsupported kind")
		}
	})
}

func TestCompareCursorValues(t *testing.T) {
	t.Run("equal multi-key", func(t *testing.T) {
		left := []pagination.CursorValue{
			pagination.StringValue("name", "alpha"),
			pagination.StringValue("id", "1"),
		}
		right := []pagination.CursorValue{
			pagination.StringValue("name", "alpha"),
			pagination.StringValue("id", "1"),
		}
		got, err := compareCursorValues(left, right)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 0 {
			t.Errorf("expected 0, got %d", got)
		}
	})

	t.Run("first key differs", func(t *testing.T) {
		left := []pagination.CursorValue{
			pagination.StringValue("name", "alpha"),
			pagination.StringValue("id", "2"),
		}
		right := []pagination.CursorValue{
			pagination.StringValue("name", "beta"),
			pagination.StringValue("id", "1"),
		}
		got, err := compareCursorValues(left, right)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != -1 {
			t.Errorf("expected -1, got %d", got)
		}
	})

	t.Run("length mismatch", func(t *testing.T) {
		left := []pagination.CursorValue{pagination.StringValue("a", "1")}
		right := []pagination.CursorValue{
			pagination.StringValue("a", "1"),
			pagination.StringValue("b", "2"),
		}
		_, err := compareCursorValues(left, right)
		if err == nil {
			t.Fatal("expected error for length mismatch")
		}
	})
}

func TestCursorKeysFromToken(t *testing.T) {
	t.Run("string key", func(t *testing.T) {
		cursor := pagination.Cursor{
			Values: []pagination.CursorValue{pagination.StringValue("name", "test")},
		}
		specs := []contentKeySpec{{Name: "name", Kind: pagination.CursorValueString}}
		keys, err := cursorKeysFromToken(cursor, specs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 || keys[0].StringValue != "test" {
			t.Errorf("unexpected keys: %+v", keys)
		}
	})

	t.Run("int key", func(t *testing.T) {
		cursor := pagination.Cursor{
			Values: []pagination.CursorValue{pagination.IntValue("seq", 42)},
		}
		specs := []contentKeySpec{{Name: "seq", Kind: pagination.CursorValueInt}}
		keys, err := cursorKeysFromToken(cursor, specs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 || keys[0].IntValue != 42 {
			t.Errorf("unexpected keys: %+v", keys)
		}
	})

	t.Run("uint key", func(t *testing.T) {
		cursor := pagination.Cursor{
			Values: []pagination.CursorValue{pagination.UintValue("id", 99)},
		}
		specs := []contentKeySpec{{Name: "id", Kind: pagination.CursorValueUint}}
		keys, err := cursorKeysFromToken(cursor, specs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 || keys[0].UintValue != 99 {
			t.Errorf("unexpected keys: %+v", keys)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		cursor := pagination.Cursor{}
		specs := []contentKeySpec{{Name: "missing", Kind: pagination.CursorValueString}}
		_, err := cursorKeysFromToken(cursor, specs)
		if err == nil {
			t.Fatal("expected error for missing key")
		}
	})

	t.Run("unsupported kind", func(t *testing.T) {
		cursor := pagination.Cursor{}
		specs := []contentKeySpec{{Name: "x", Kind: "bogus"}}
		_, err := cursorKeysFromToken(cursor, specs)
		if err == nil {
			t.Fatal("expected error for unsupported kind")
		}
	})
}
