package gateway

import campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"

// SessionReadDeps keeps session query dependencies explicit.
type SessionReadDeps struct {
	Session  SessionReadClient
	Campaign CampaignReadClient
}

// SessionMutationDeps keeps session mutation dependencies explicit.
type SessionMutationDeps struct {
	Session SessionMutationClient
}

// InviteReadDeps keeps invite read/search dependencies explicit.
type InviteReadDeps struct {
	Invite      InviteReadClient
	Participant ParticipantReadClient
	Social      SocialClient
	Auth        AuthClient
}

// InviteMutationDeps keeps invite mutation dependencies explicit.
type InviteMutationDeps struct {
	Invite InviteMutationClient
	Auth   AuthClient
}

// ConfigurationMutationDeps keeps campaign settings mutation dependencies explicit.
type ConfigurationMutationDeps struct {
	Campaign CampaignMutationClient
}

// sessionReadGateway maps session reads independently of session mutations.
type sessionReadGateway struct {
	read SessionReadDeps
}

// sessionMutationGateway maps session lifecycle mutations only.
type sessionMutationGateway struct {
	mutation SessionMutationDeps
}

// inviteReadGateway maps invite reads and invite-search side effects from owned deps.
type inviteReadGateway struct {
	read InviteReadDeps
}

// inviteMutationGateway maps invite mutations without widening read authority.
type inviteMutationGateway struct {
	mutation InviteMutationDeps
}

// configurationMutationGateway maps campaign settings mutations only.
type configurationMutationGateway struct {
	mutation ConfigurationMutationDeps
}

// NewSessionReadGateway builds the session read adapter from explicit
// dependencies.
func NewSessionReadGateway(readDeps SessionReadDeps) campaignapp.CampaignSessionReadGateway {
	if readDeps.Session == nil || readDeps.Campaign == nil {
		return nil
	}
	return sessionReadGateway{read: readDeps}
}

// NewSessionMutationGateway builds the session mutation adapter from explicit
// dependencies.
func NewSessionMutationGateway(mutationDeps SessionMutationDeps) campaignapp.CampaignSessionMutationGateway {
	if mutationDeps.Session == nil {
		return nil
	}
	return sessionMutationGateway{mutation: mutationDeps}
}

// NewInviteReadGateway builds the invite read adapter from explicit
// dependencies.
func NewInviteReadGateway(readDeps InviteReadDeps) campaignapp.CampaignInviteReadGateway {
	if readDeps.Invite == nil || readDeps.Participant == nil || readDeps.Social == nil || readDeps.Auth == nil {
		return nil
	}
	return inviteReadGateway{read: readDeps}
}

// NewInviteMutationGateway builds the invite mutation adapter from explicit
// dependencies.
func NewInviteMutationGateway(mutationDeps InviteMutationDeps) campaignapp.CampaignInviteMutationGateway {
	if mutationDeps.Invite == nil || mutationDeps.Auth == nil {
		return nil
	}
	return inviteMutationGateway{mutation: mutationDeps}
}

// NewConfigurationMutationGateway builds the configuration mutation adapter
// from explicit dependencies.
func NewConfigurationMutationGateway(mutationDeps ConfigurationMutationDeps) campaignapp.CampaignConfigurationMutationGateway {
	if mutationDeps.Campaign == nil {
		return nil
	}
	return configurationMutationGateway{mutation: mutationDeps}
}
