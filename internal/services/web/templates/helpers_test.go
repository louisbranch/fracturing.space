package templates

import (
	"fmt"
	"net/http"
	"testing"

	"golang.org/x/text/message"
)

type testLocalizer map[string]string

func (l testLocalizer) Sprintf(key message.Reference, args ...any) string {
	keyString := fmt.Sprint(key)
	format, ok := l[keyString]
	if !ok {
		format = keyString
	}
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

func TestT(t *testing.T) {
	t.Parallel()

	if got := T(nil, "web.test.message"); got != "web.test.message" {
		t.Fatalf("T(nil, key) = %q, want %q", got, "web.test.message")
	}
	if got := T(nil, "welcome %s", "traveler"); got != "welcome traveler" {
		t.Fatalf("T(nil, format, args) = %q, want %q", got, "welcome traveler")
	}
	if got := T(nil, 42); got != "" {
		t.Fatalf("T(nil, non-string key) = %q, want empty string", got)
	}

	loc := testLocalizer{
		"game.participants.value_she_her": "She/Her",
		"web.message.greeting":            "Hello %s",
	}
	if got := T(loc, "game.participants.value_she_her"); got != "She/Her" {
		t.Fatalf("T(loc, key) = %q, want %q", got, "She/Her")
	}
	if got := T(loc, "web.message.greeting", "Ranger"); got != "Hello Ranger" {
		t.Fatalf("T(loc, format, args) = %q, want %q", got, "Hello Ranger")
	}
}

func TestAppErrorCopyHelpers(t *testing.T) {
	t.Parallel()

	loc := testLocalizer{
		appErrorPageTitleNotFoundKey:  "Not Found",
		appErrorPageTitleClientErrKey: "Request Error",
		appErrorPageTitleServerErrKey: "Server Error",
		appErrorHeadingNotFoundKey:    "Missing Route",
		appErrorHeadingClientErrKey:   "Bad Request",
		appErrorHeadingServerErrKey:   "Something Went Wrong",
		appErrorMessageNotFoundKey:    "No route matched your request.",
		appErrorMessageClientErrKey:   "Please check your request.",
		appErrorMessageServerErrKey:   "Please try again later.",
	}

	if got := AppErrorPageTitle(http.StatusNotFound, loc); got != "Not Found" {
		t.Fatalf("AppErrorPageTitle(404) = %q, want %q", got, "Not Found")
	}
	if got := AppErrorPageTitle(http.StatusTeapot, loc); got != "Request Error" {
		t.Fatalf("AppErrorPageTitle(4xx) = %q, want %q", got, "Request Error")
	}
	if got := AppErrorPageTitle(http.StatusInternalServerError, loc); got != "Server Error" {
		t.Fatalf("AppErrorPageTitle(500) = %q, want %q", got, "Server Error")
	}
	if got := appErrorHeading(http.StatusNotFound, loc); got != "Missing Route" {
		t.Fatalf("appErrorHeading(404) = %q, want %q", got, "Missing Route")
	}
	if got := appErrorHeading(http.StatusForbidden, loc); got != "Bad Request" {
		t.Fatalf("appErrorHeading(403) = %q, want %q", got, "Bad Request")
	}
	if got := appErrorHeading(http.StatusInternalServerError, loc); got != "Something Went Wrong" {
		t.Fatalf("appErrorHeading(500) = %q, want %q", got, "Something Went Wrong")
	}
	if got := appErrorMessage(http.StatusNotFound, loc); got != "No route matched your request." {
		t.Fatalf("appErrorMessage(404) = %q, want %q", got, "No route matched your request.")
	}
	if got := appErrorMessage(http.StatusForbidden, loc); got != "Please check your request." {
		t.Fatalf("appErrorMessage(403) = %q, want %q", got, "Please check your request.")
	}
	if got := appErrorMessage(http.StatusInternalServerError, loc); got != "Please try again later." {
		t.Fatalf("appErrorMessage(500) = %q, want %q", got, "Please try again later.")
	}
	if got := appErrorDisplayMessage(http.StatusConflict, "User already has a pending invite in this campaign", loc); got != "User already has a pending invite in this campaign" {
		t.Fatalf("appErrorDisplayMessage(explicit) = %q", got)
	}
	if got := appErrorDisplayMessage(http.StatusConflict, "   ", loc); got != "Please check your request." {
		t.Fatalf("appErrorDisplayMessage(fallback) = %q, want %q", got, "Please check your request.")
	}
	if got := normalizeAppErrorStatus(http.StatusNotFound); got != http.StatusNotFound {
		t.Fatalf("normalizeAppErrorStatus(404) = %d, want %d", got, http.StatusNotFound)
	}
	if got := normalizeAppErrorStatus(http.StatusConflict); got != http.StatusBadRequest {
		t.Fatalf("normalizeAppErrorStatus(409) = %d, want %d", got, http.StatusBadRequest)
	}
	if got := normalizeAppErrorStatus(http.StatusInternalServerError); got != http.StatusInternalServerError {
		t.Fatalf("normalizeAppErrorStatus(500) = %d, want %d", got, http.StatusInternalServerError)
	}
}

func TestParticipantPronounsLabel(t *testing.T) {
	t.Parallel()

	loc := testLocalizer{
		"game.participants.value_she_her":   "She/Her",
		"game.participants.value_he_him":    "He/Him",
		"game.participants.value_they_them": "They/Them",
		"game.participants.value_it_its":    "It/Its",
	}

	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "empty", value: "", want: ""},
		{name: "unspecified", value: "Unspecified", want: "Unspecified"},
		{name: "she/her", value: "she/her", want: "She/Her"},
		{name: "he/him", value: "he/him", want: "He/Him"},
		{name: "they/them", value: "they/them", want: "They/Them"},
		{name: "it/its", value: "it/its", want: "It/Its"},
		{name: "fallback", value: " custom ", want: "custom"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := participantPronounsLabel(loc, tc.value); got != tc.want {
				t.Fatalf("participantPronounsLabel(%q) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}
