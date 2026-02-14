package branding

import "testing"

func TestAppName(t *testing.T) {
	if AppName == "" {
		t.Fatal("expected AppName to be non-empty")
	}
	if AppName != "Fracturing.Space" {
		t.Fatalf("AppName = %q, want %q", AppName, "Fracturing.Space")
	}
}
