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
	if ImpersonationLabel(&ImpersonationView{UserID: "user", Name: "User"}) != "User" {
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

func TestAppendQueryParam(t *testing.T) {
	if got := AppendQueryParam("/campaigns", "page_token", "abc"); got != "/campaigns?page_token=abc" {
		t.Fatalf("expected query param appended, got %q", got)
	}
	if got := AppendQueryParam("/campaigns?event_type=foo", "page_token", "a b"); got != "/campaigns?event_type=foo&page_token=a+b" {
		t.Fatalf("expected encoded param appended, got %q", got)
	}
}

func TestEventFilterBaseURL(t *testing.T) {
	filters := EventFilterOptions{EventType: "session.started", StartDate: "2024-02-01"}
	if got := EventFilterBaseURL("/campaigns/camp-1/events", filters); got != "/campaigns/camp-1/events?event_type=session.started&start_date=2024-02-01" {
		t.Fatalf("expected filters encoded, got %q", got)
	}
	if got := EventFilterBaseURL("/campaigns/camp-1/events", EventFilterOptions{}); got != "/campaigns/camp-1/events" {
		t.Fatalf("expected base url, got %q", got)
	}
}

func TestComposeAdminPageTitleStripsBrandSuffix(t *testing.T) {
	got := ComposeAdminPageTitle("Campaigns - " + AppName())
	want := "Campaigns - Admin | " + AppName()
	if got != want {
		t.Fatalf("composeAdminPageTitle = %q, want %q", got, want)
	}
}

func TestComposeAdminPageTitleAppendsAdminSuffix(t *testing.T) {
	got := ComposeAdminPageTitle("Campaigns")
	want := "Campaigns - Admin | " + AppName()
	if got != want {
		t.Fatalf("composeAdminPageTitle = %q, want %q", got, want)
	}
}
