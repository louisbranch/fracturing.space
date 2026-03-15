package app

import (
	"context"

	"golang.org/x/text/language"
)

// CampaignSessionReadGateway loads session reads for the web service.
type CampaignSessionReadGateway interface {
	CampaignSessions(context.Context, string) ([]CampaignSession, error)
	CampaignSessionReadiness(context.Context, string, language.Tag) (CampaignSessionReadiness, error)
}

// CampaignInviteReadGateway loads invite reads for the web service.
type CampaignInviteReadGateway interface {
	CampaignInvites(context.Context, string) ([]CampaignInvite, error)
	SearchInviteUsers(context.Context, SearchInviteUsersInput) ([]InviteUserSearchResult, error)
}

// CampaignSessionMutationGateway applies session lifecycle mutations for the web service.
type CampaignSessionMutationGateway interface {
	StartSession(context.Context, string, StartSessionInput) error
	EndSession(context.Context, string, EndSessionInput) error
}

// CampaignInviteMutationGateway applies invite mutations for the web service.
type CampaignInviteMutationGateway interface {
	CreateInvite(context.Context, string, CreateInviteInput) error
	RevokeInvite(context.Context, string, RevokeInviteInput) error
}
