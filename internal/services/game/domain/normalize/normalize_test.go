package normalize

import "testing"

func TestString(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"already trimmed", "hello", "hello"},
		{"leading space", "  hello", "hello"},
		{"trailing space", "hello  ", "hello"},
		{"both sides", "  hello  ", "hello"},
		{"tabs and spaces", "\t hello \t", "hello"},
		{"whitespace only", "   ", ""},
		{"inner spaces preserved", "  a b  ", "a b"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := String(tc.input); got != tc.want {
				t.Fatalf("String(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// testID is a typed string used to verify the generic ID and RequireID functions.
type testID string

func TestID(t *testing.T) {
	cases := []struct {
		name  string
		input testID
		want  testID
	}{
		{"empty", testID(""), testID("")},
		{"trimmed", testID("abc"), testID("abc")},
		{"leading", testID("  abc"), testID("abc")},
		{"trailing", testID("abc  "), testID("abc")},
		{"both", testID("  abc  "), testID("abc")},
		{"whitespace only", testID("   "), testID("")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ID(tc.input); got != tc.want {
				t.Fatalf("ID(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestRequireID(t *testing.T) {
	t.Run("empty returns false", func(t *testing.T) {
		got, ok := RequireID(testID(""))
		if ok {
			t.Fatal("expected ok=false for empty input")
		}
		if got != "" {
			t.Fatalf("expected zero value, got %q", got)
		}
	})

	t.Run("whitespace only returns false", func(t *testing.T) {
		got, ok := RequireID(testID("   "))
		if ok {
			t.Fatal("expected ok=false for whitespace-only input")
		}
		if got != "" {
			t.Fatalf("expected zero value, got %q", got)
		}
	})

	t.Run("tab only returns false", func(t *testing.T) {
		got, ok := RequireID(testID("\t"))
		if ok {
			t.Fatal("expected ok=false for tab-only input")
		}
		if got != "" {
			t.Fatalf("expected zero value, got %q", got)
		}
	})

	t.Run("valid returns trimmed and true", func(t *testing.T) {
		got, ok := RequireID(testID("  abc  "))
		if !ok {
			t.Fatal("expected ok=true for valid input")
		}
		if got != "abc" {
			t.Fatalf("RequireID(%q) = %q, want %q", "  abc  ", got, "abc")
		}
	})

	t.Run("already trimmed returns unchanged", func(t *testing.T) {
		got, ok := RequireID(testID("xyz"))
		if !ok {
			t.Fatal("expected ok=true")
		}
		if got != "xyz" {
			t.Fatalf("RequireID(%q) = %q, want %q", "xyz", got, "xyz")
		}
	})
}
