package userid

import "testing"

func TestNormalize(t *testing.T) {
	t.Parallel()

	if got := Normalize("  user-1\t"); got != "user-1" {
		t.Fatalf("Normalize() = %q, want %q", got, "user-1")
	}
	if got := Normalize("   "); got != "" {
		t.Fatalf("Normalize(blank) = %q, want empty", got)
	}
}

func TestRequire(t *testing.T) {
	t.Parallel()

	got, err := Require(" user-1 ")
	if err != nil {
		t.Fatalf("Require() error = %v", err)
	}
	if got != "user-1" {
		t.Fatalf("Require() = %q, want %q", got, "user-1")
	}
}

func TestRequireRejectsBlank(t *testing.T) {
	t.Parallel()

	if _, err := Require("   "); err == nil {
		t.Fatal("Require() error = nil, want error")
	}
}
