package daggerheartcontent

import "testing"

func TestOpenEmptyPath(t *testing.T) {
	_, err := openStore("")
	if err == nil {
		t.Fatal("expected error for empty path")
	}

	_, err = openStore("   ")
	if err == nil {
		t.Fatal("expected error for whitespace path")
	}
}
