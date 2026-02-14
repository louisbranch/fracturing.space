package templates

import "testing"

func TestT_NilLocalizer(t *testing.T) {
	got := T(nil, "some.key")
	if got != "some.key" {
		t.Fatalf("T(nil, ...) = %q, want %q", got, "some.key")
	}
}

func TestT_NilLocalizerNonStringKey(t *testing.T) {
	got := T(nil, 42)
	if got != "" {
		t.Fatalf("T(nil, 42) = %q, want empty", got)
	}
}
