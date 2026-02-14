package templates

import (
	"testing"

	"golang.org/x/text/message"
)

type fakeLocalizer struct {
	value string
}

func (f fakeLocalizer) Sprintf(key message.Reference, args ...any) string {
	return f.value
}

func TestImpersonationLabel(t *testing.T) {
	if ImpersonationLabel(nil) != "" {
		t.Fatal("expected empty label for nil view")
	}
	if ImpersonationLabel(&ImpersonationView{UserID: "user"}) != "user" {
		t.Fatal("expected user id fallback label")
	}
	if ImpersonationLabel(&ImpersonationView{UserID: "user", DisplayName: "User"}) != "User" {
		t.Fatal("expected display name label")
	}
}

func TestTranslateFallback(t *testing.T) {
	if T(nil, "hello") != "hello" {
		t.Fatal("expected key fallback")
	}

	if T(nil, message.Reference(123)) != "" {
		t.Fatal("expected empty string for non-string key")
	}
}

func TestTranslateLocalizer(t *testing.T) {
	loc := fakeLocalizer{value: "translated"}
	if T(loc, "hello") != "translated" {
		t.Fatal("expected translated value")
	}
}
