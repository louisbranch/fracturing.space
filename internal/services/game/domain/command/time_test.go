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

func TestNowFunc_DefaultsToTimeNow(t *testing.T) {
	before := time.Now()
	fn := NowFunc(nil)
	got := fn()
	after := time.Now()
	if got.Before(before) || got.After(after) {
		t.Fatalf("NowFunc(nil)() = %v, want between %v and %v", got, before, after)
	}
}
