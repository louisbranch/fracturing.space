package app

import "golang.org/x/text/language"

// UpdateCharacterInput stores character update form values.
type UpdateCharacterInput struct {
	Name     string
	Pronouns string
}

// CreateCampaignInput stores create-campaign form values.
type CreateCampaignInput struct {
	Name        string
	Locale      language.Tag
	System      GameSystem
	GMMode      GmMode
	ThemePrompt string
}

// UpdateCampaignInput stores campaign update form values.
type UpdateCampaignInput struct {
	Name        *string
	ThemePrompt *string
	Locale      *string
}

// CreateCampaignResult stores create-campaign response values.
type CreateCampaignResult struct {
	CampaignID string
}

// StartSessionInput stores start-session form values.
type StartSessionInput struct {
	Name string
}

// EndSessionInput stores end-session form values.
type EndSessionInput struct {
	SessionID string
}

// CreateInviteInput stores create-invite form values.
type CreateInviteInput struct {
	ParticipantID   string
	RecipientUserID string
}

// RevokeInviteInput stores revoke-invite form values.
type RevokeInviteInput struct {
	InviteID string
}

// CreateCharacterInput stores create-character form values.
type CreateCharacterInput struct {
	Name string
	Kind CharacterKind
}

// UpdateParticipantInput stores participant update form values.
type UpdateParticipantInput struct {
	ParticipantID  string
	Name           string
	Role           string
	Pronouns       string
	CampaignAccess string
}

// UpdateCampaignAIBindingInput stores AI-binding form values.
type UpdateCampaignAIBindingInput struct {
	ParticipantID string
	AIAgentID     string
}

// CreateCharacterResult stores create-character response values.
type CreateCharacterResult struct {
	CharacterID string
}
