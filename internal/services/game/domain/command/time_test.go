package command

import (
	"testing"
	"time"
)

func TestRequireNowFunc_ReturnsProvidedFunction(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fn := RequireNowFunc(func() time.Time { return fixed })
	got := fn()
	if !got.Equal(fixed) {
		t.Fatalf("RequireNowFunc(fn)() = %v, want %v", got, fixed)
	}
}

func TestRequireNowFunc_PanicsOnNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil now function")
		}
	}()
	RequireNowFunc(nil)
}
