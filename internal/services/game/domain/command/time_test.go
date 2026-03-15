package command

import (
	"testing"
	"time"
)

func TestNowFunc_ReturnsProvidedFunction(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	fn := NowFunc(func() time.Time { return fixed })
	got := fn()
	if !got.Equal(fixed) {
		t.Fatalf("NowFunc(fn)() = %v, want %v", got, fixed)
	}
}

func TestNowFunc_PanicsOnNil(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil now function")
		}
	}()
	NowFunc(nil)
}
