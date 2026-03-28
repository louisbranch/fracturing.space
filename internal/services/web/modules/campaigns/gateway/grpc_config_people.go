package gateway

import campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"

// ParticipantReadDeps keeps participant query dependencies explicit.
type ParticipantReadDeps struct {
	Participant ParticipantReadClient
}

// ParticipantMutationDeps keeps participant mutation dependencies explicit.
type ParticipantMutationDeps struct {
	Participant ParticipantMutationClient
}

// CharacterReadDeps keeps character query dependencies explicit.
type CharacterReadDeps struct {
	Character          CharacterReadClient
	Participant        ParticipantReadClient
	DaggerheartContent DaggerheartContentClient
}

// CharacterMutationDeps keeps character mutation dependencies explicit.
type CharacterMutationDeps struct {
	Character CharacterMutationClient
}

// CharacterOwnershipMutationDeps keeps character-owner mutation dependencies
// explicit.
type CharacterOwnershipMutationDeps struct {
	Character CharacterMutationClient
}

// AutomationReadDeps keeps automation query dependencies explicit.
type AutomationReadDeps struct {
	Agent AgentClient
}

// AutomationMutationDeps keeps automation mutation dependencies explicit.
type AutomationMutationDeps struct {
	Campaign CampaignMutationClient
}

// participantReadGateway maps participant reads and view formatting inputs.
type participantReadGateway struct {
	read         ParticipantReadDeps
	assetBaseURL string
}

// participantMutationGateway maps participant mutations without carrying unrelated clients.
type participantMutationGateway struct {
	mutation ParticipantMutationDeps
}

// characterReadGateway maps character list/entity reads and derived view data.
type characterReadGateway struct {
	read         CharacterReadDeps
	assetBaseURL string
}

// characterMutationGateway maps character mutations only.
type characterMutationGateway struct {
	mutation CharacterMutationDeps
}

// characterOwnershipMutationGateway maps character-owner mutations only.
type characterOwnershipMutationGateway struct {
	mutation CharacterOwnershipMutationDeps
}

// automationReadGateway maps campaign automation editor reads.
type automationReadGateway struct {
	read AutomationReadDeps
}

// automationMutationGateway maps campaign automation edits.
type automationMutationGateway struct {
	mutation AutomationMutationDeps
}

// NewParticipantReadGateway builds the participant read adapter from explicit
// dependencies.
func NewParticipantReadGateway(readDeps ParticipantReadDeps, assetBaseURL string) campaignapp.CampaignParticipantReadGateway {
	if readDeps.Participant == nil {
		return nil
	}
	return participantReadGateway{
		read:         readDeps,
		assetBaseURL: assetBaseURL,
	}
}

// NewParticipantMutationGateway builds the participant mutation adapter from
// explicit dependencies.
func NewParticipantMutationGateway(mutationDeps ParticipantMutationDeps) campaignapp.CampaignParticipantMutationGateway {
	if mutationDeps.Participant == nil {
		return nil
	}
	return participantMutationGateway{mutation: mutationDeps}
}

// NewCharacterReadGateway builds the character read adapter from explicit
// dependencies.
func NewCharacterReadGateway(readDeps CharacterReadDeps, assetBaseURL string) campaignapp.CampaignCharacterReadGateway {
	if readDeps.Character == nil || readDeps.Participant == nil || readDeps.DaggerheartContent == nil {
		return nil
	}
	return characterReadGateway{
		read:         readDeps,
		assetBaseURL: assetBaseURL,
	}
}

// NewCharacterMutationGateway builds the character mutation adapter from
// explicit dependencies.
func NewCharacterMutationGateway(mutationDeps CharacterMutationDeps) campaignapp.CampaignCharacterMutationGateway {
	if mutationDeps.Character == nil {
		return nil
	}
	return characterMutationGateway{mutation: mutationDeps}
}

// NewCharacterOwnershipMutationGateway builds the character-owner mutation
// adapter from explicit dependencies.
func NewCharacterOwnershipMutationGateway(mutationDeps CharacterOwnershipMutationDeps) campaignapp.CampaignCharacterOwnershipMutationGateway {
	if mutationDeps.Character == nil {
		return nil
	}
	return characterOwnershipMutationGateway{mutation: mutationDeps}
}

// NewAutomationReadGateway builds the automation read adapter from explicit
// dependencies.
func NewAutomationReadGateway(readDeps AutomationReadDeps) campaignapp.CampaignAutomationReadGateway {
	if readDeps.Agent == nil {
		return nil
	}
	return automationReadGateway{read: readDeps}
}

// NewAutomationMutationGateway builds the automation mutation adapter from
// explicit dependencies.
func NewAutomationMutationGateway(mutationDeps AutomationMutationDeps) campaignapp.CampaignAutomationMutationGateway {
	if mutationDeps.Campaign == nil {
		return nil
	}
	return automationMutationGateway{mutation: mutationDeps}
}
