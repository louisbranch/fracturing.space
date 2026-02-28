package campaigns

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestDaggerheartStepInputFromFormUsesLocalizationKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		step    int32
		body    string
		wantKey string
	}{
		{
			name:    "class and subclass required",
			step:    1,
			body:    "class_id=warrior",
			wantKey: "error.web.message.character_creation_class_and_subclass_are_required",
		},
		{
			name:    "ancestry and community required",
			step:    2,
			body:    "ancestry_id=elf",
			wantKey: "error.web.message.character_creation_ancestry_and_community_are_required",
		},
		{
			name:    "unknown step",
			step:    42,
			body:    "",
			wantKey: "error.web.message.character_creation_step_is_not_available",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			_, err := daggerheartStepInputFromForm(req, tt.step)
			if err == nil {
				t.Fatalf("expected error")
			}
			if got := apperrors.LocalizationKey(err); got != tt.wantKey {
				t.Fatalf("LocalizationKey(err) = %q, want %q", got, tt.wantKey)
			}
		})
	}
}

func TestParseRequiredInt32UsesLocalizationKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantKey string
	}{
		{
			name:    "missing value",
			raw:     "   ",
			wantKey: "error.web.message.character_creation_numeric_field_is_required",
		},
		{
			name:    "invalid integer",
			raw:     "abc",
			wantKey: "error.web.message.character_creation_numeric_field_must_be_valid_integer",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseRequiredInt32(tt.raw, "agility")
			if err == nil {
				t.Fatalf("expected error")
			}
			if got := apperrors.LocalizationKey(err); got != tt.wantKey {
				t.Fatalf("LocalizationKey(err) = %q, want %q", got, tt.wantKey)
			}
		})
	}
}

func TestParseOptionalInt32UsesLocalizationKey(t *testing.T) {
	t.Parallel()

	_, err := parseOptionalInt32("bad")
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.character_creation_modifier_must_be_valid_integer" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.character_creation_modifier_must_be_valid_integer")
	}
}
