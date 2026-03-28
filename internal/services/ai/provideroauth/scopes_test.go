package provideroauth

import "testing"

func TestNormalizeScopes(t *testing.T) {
	got := NormalizeScopes([]string{" responses.read ", "", "responses.read", "responses.write", "responses.write"})
	want := []string{"responses.read", "responses.write"}
	if len(got) != len(want) {
		t.Fatalf("len(NormalizeScopes(...)) = %d, want %d (%#v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("NormalizeScopes(...)[%d] = %q, want %q (full=%#v)", i, got[i], want[i], got)
		}
	}
}

func TestNormalizeScopesReturnsNilWhenEmptyAfterTrim(t *testing.T) {
	if got := NormalizeScopes([]string{"", " ", "\t"}); got != nil {
		t.Fatalf("NormalizeScopes(...) = %#v, want nil", got)
	}
}
