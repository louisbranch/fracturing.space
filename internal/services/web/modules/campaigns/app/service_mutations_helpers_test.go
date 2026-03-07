package app

import (
	"net/http"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestBuildCampaignUpdatePatch(t *testing.T) {
	t.Parallel()

	current := CampaignWorkspace{
		Name:   "Voyage",
		Theme:  "stormy sea",
		Locale: "pt-BR",
	}

	t.Run("no changes", func(t *testing.T) {
		t.Parallel()
		name := " Voyage "
		theme := " stormy sea "
		locale := "portuguese (brazil)"
		patch, changed, err := buildCampaignUpdatePatch(UpdateCampaignInput{
			Name:        &name,
			ThemePrompt: &theme,
			Locale:      &locale,
		}, current)
		if err != nil {
			t.Fatalf("buildCampaignUpdatePatch() error = %v", err)
		}
		if changed {
			t.Fatalf("changed = true, want false")
		}
		if patch != (UpdateCampaignInput{}) {
			t.Fatalf("patch = %#v, want empty patch", patch)
		}
	})

	t.Run("changed fields", func(t *testing.T) {
		t.Parallel()
		theme := "calm waters"
		locale := "english (us)"
		patch, changed, err := buildCampaignUpdatePatch(UpdateCampaignInput{
			ThemePrompt: &theme,
			Locale:      &locale,
		}, current)
		if err != nil {
			t.Fatalf("buildCampaignUpdatePatch() error = %v", err)
		}
		if !changed {
			t.Fatalf("changed = false, want true")
		}
		if patch.ThemePrompt == nil || *patch.ThemePrompt != "calm waters" {
			t.Fatalf("ThemePrompt patch = %#v, want calm waters", patch.ThemePrompt)
		}
		if patch.Locale == nil || *patch.Locale != "en-US" {
			t.Fatalf("Locale patch = %#v, want en-US", patch.Locale)
		}
	})

	t.Run("invalid name", func(t *testing.T) {
		t.Parallel()
		name := "   "
		_, _, err := buildCampaignUpdatePatch(UpdateCampaignInput{Name: &name}, current)
		if err == nil {
			t.Fatalf("expected invalid name error")
		}
		if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
			t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
		}
	})

	t.Run("invalid locale", func(t *testing.T) {
		t.Parallel()
		locale := "xx-YY"
		_, _, err := buildCampaignUpdatePatch(UpdateCampaignInput{Locale: &locale}, current)
		if err == nil {
			t.Fatalf("expected invalid locale error")
		}
		if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
			t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
		}
	})
}

func TestNormalizeParticipantUpdateRequest(t *testing.T) {
	t.Parallel()

	t.Run("normalizes valid input", func(t *testing.T) {
		t.Parallel()
		request, err := normalizeParticipantUpdateRequest(" c-1 ", UpdateParticipantInput{
			ParticipantID:  " p-1 ",
			Name:           " Player One ",
			Role:           " gm ",
			Pronouns:       " they/them ",
			CampaignAccess: " owner ",
		})
		if err != nil {
			t.Fatalf("normalizeParticipantUpdateRequest() error = %v", err)
		}
		if request.CampaignID != "c-1" || request.ParticipantID != "p-1" {
			t.Fatalf("request ids = %#v", request)
		}
		if request.Name != "Player One" || request.Role != "gm" || request.Pronouns != "they/them" || request.RequestedAccess != "owner" {
			t.Fatalf("normalized request = %#v", request)
		}
	})

	t.Run("invalid role", func(t *testing.T) {
		t.Parallel()
		_, err := normalizeParticipantUpdateRequest("c-1", UpdateParticipantInput{
			ParticipantID: "p-1",
			Role:          "invalid",
		})
		if err == nil {
			t.Fatalf("expected invalid role error")
		}
		if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
			t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
		}
	})

	t.Run("invalid campaign access", func(t *testing.T) {
		t.Parallel()
		_, err := normalizeParticipantUpdateRequest("c-1", UpdateParticipantInput{
			ParticipantID:  "p-1",
			Role:           "player",
			CampaignAccess: "invalid",
		})
		if err == nil {
			t.Fatalf("expected invalid campaign access error")
		}
		if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
			t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
		}
	})
}

func TestParticipantUpdateHasChanges(t *testing.T) {
	t.Parallel()

	current := CampaignParticipant{
		ID:             "p-1",
		Name:           "Player One",
		Role:           "GM",
		Pronouns:       "they/them",
		CampaignAccess: "Owner",
	}
	request := participantUpdateRequest{
		CampaignID:      "c-1",
		ParticipantID:   "p-1",
		Name:            "Player One",
		Role:            "gm",
		Pronouns:        "they/them",
		RequestedAccess: normalizeRequestedParticipantAccess("owner", current),
	}
	if participantUpdateHasChanges(request, current) {
		t.Fatalf("participantUpdateHasChanges() = true, want false")
	}

	request.Name = "Player Prime"
	if !participantUpdateHasChanges(request, current) {
		t.Fatalf("participantUpdateHasChanges() = false, want true")
	}
}
