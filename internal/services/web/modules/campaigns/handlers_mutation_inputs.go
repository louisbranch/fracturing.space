package campaigns

import (
	"net/url"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// parseStartSessionInput maps form values into session start input.
func parseStartSessionInput(form url.Values) StartSessionInput {
	return StartSessionInput{Name: strings.TrimSpace(form.Get("name"))}
}

// parseEndSessionInput maps form values into session end input.
func parseEndSessionInput(form url.Values) EndSessionInput {
	return EndSessionInput{SessionID: strings.TrimSpace(form.Get("session_id"))}
}

// parseCreateCharacterInput maps and validates character-create form values.
func parseCreateCharacterInput(form url.Values) (CreateCharacterInput, error) {
	kindValue := strings.TrimSpace(form.Get("kind"))
	if kindValue == "" {
		kindValue = "pc"
	}
	kind, ok := parseAppCharacterKind(kindValue)
	if !ok {
		return CreateCharacterInput{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_kind_value_is_invalid", "character kind value is invalid")
	}
	return CreateCharacterInput{
		Name:     strings.TrimSpace(form.Get("name")),
		Pronouns: strings.TrimSpace(form.Get("pronouns")),
		Kind:     kind,
	}, nil
}

// parseUpdateCharacterInput maps form values into character-update input.
func parseUpdateCharacterInput(form url.Values) UpdateCharacterInput {
	return UpdateCharacterInput{
		Name:     strings.TrimSpace(form.Get("name")),
		Pronouns: strings.TrimSpace(form.Get("pronouns")),
	}
}

// parseSetCharacterControllerInput maps form values into controller-set input.
func parseSetCharacterControllerInput(form url.Values) string {
	return strings.TrimSpace(form.Get("participant_id"))
}

// parseCreateInviteInput maps form values into invite-create input.
func parseCreateInviteInput(form url.Values) CreateInviteInput {
	return CreateInviteInput{
		ParticipantID:     strings.TrimSpace(form.Get("participant_id")),
		RecipientUsername: strings.TrimSpace(form.Get("username")),
	}
}

// parseCreateParticipantInput maps form values into participant-create input.
func parseCreateParticipantInput(form url.Values) CreateParticipantInput {
	return CreateParticipantInput{
		Name:           strings.TrimSpace(form.Get("name")),
		Role:           strings.TrimSpace(form.Get("role")),
		CampaignAccess: strings.TrimSpace(form.Get("campaign_access")),
	}
}

// parseRevokeInviteInput maps form values into invite-revoke input.
func parseRevokeInviteInput(form url.Values) RevokeInviteInput {
	return RevokeInviteInput{InviteID: strings.TrimSpace(form.Get("invite_id"))}
}

// parseUpdateParticipantInput maps form values into participant-update input.
func parseUpdateParticipantInput(participantID string, form url.Values) UpdateParticipantInput {
	return UpdateParticipantInput{
		ParticipantID:  strings.TrimSpace(participantID),
		Name:           strings.TrimSpace(form.Get("name")),
		Role:           strings.TrimSpace(form.Get("role")),
		Pronouns:       strings.TrimSpace(form.Get("pronouns")),
		CampaignAccess: strings.TrimSpace(form.Get("campaign_access")),
	}
}

// parseUpdateCampaignAIBindingInput maps form values into AI-binding input.
func parseUpdateCampaignAIBindingInput(form url.Values) UpdateCampaignAIBindingInput {
	return UpdateCampaignAIBindingInput{
		ParticipantID: strings.TrimSpace(form.Get("participant_id")),
		AIAgentID:     strings.TrimSpace(form.Get("ai_agent_id")),
	}
}

// parseUpdateCampaignInput maps form values into campaign-update patch input.
func parseUpdateCampaignInput(form url.Values) UpdateCampaignInput {
	name := strings.TrimSpace(form.Get("name"))
	themePrompt := strings.TrimSpace(form.Get("theme_prompt"))
	locale := strings.TrimSpace(form.Get("locale"))
	return UpdateCampaignInput{
		Name:        &name,
		ThemePrompt: &themePrompt,
		Locale:      &locale,
	}
}
