package i18n

import (
	"errors"
	"fmt"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web2/platform/errors"
	"golang.org/x/text/message"
)

func TestLocalizeErrorUsesStructuredErrorKey(t *testing.T) {
	t.Parallel()

	err := apperrors.EK(apperrors.KindInvalidInput, "web.settings.user_profile.error_username_required", "username must be set")
	loc := mapLocalizer{"web.settings.user_profile.error_username_required": "translated username required"}

	if got := LocalizeError(loc, err); got != "translated username required" {
		t.Fatalf("LocalizeError() = %q, want %q", got, "translated username required")
	}
}

func TestLocalizeErrorReturnsRawMessageWithoutStructuredKey(t *testing.T) {
	t.Parallel()

	loc := mapLocalizer{"error.web.message.failed_to_parse_profile_form": "translated profile parse"}

	if got := LocalizeError(loc, errors.New("failed to parse profile form")); got != "failed to parse profile form" {
		t.Fatalf("LocalizeError() = %q, want %q", got, "failed to parse profile form")
	}
}

func TestLocalizeErrorDoesNotTreatDotMessageAsLocalizationKey(t *testing.T) {
	t.Parallel()

	loc := mapLocalizer{"error.web.message.profile_not_found": "translated profile not found"}

	if got := LocalizeError(loc, errors.New("error.web.message.profile_not_found")); got != "error.web.message.profile_not_found" {
		t.Fatalf("LocalizeError() = %q, want %q", got, "error.web.message.profile_not_found")
	}
}

type mapLocalizer map[string]string

func (m mapLocalizer) Sprintf(key message.Reference, _ ...any) string {
	resolvedKey := fmt.Sprint(key)
	if translated, ok := m[resolvedKey]; ok {
		return translated
	}
	return resolvedKey
}
