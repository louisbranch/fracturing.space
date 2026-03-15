package app

import (
	"context"

	"golang.org/x/text/language"
)

// CampaignSessionReadService exposes session list/readiness reads.
type CampaignSessionReadService interface {
	CampaignSessions(context.Context, string) ([]CampaignSession, error)
	CampaignSessionReadiness(context.Context, string, language.Tag) (CampaignSessionReadiness, error)
}

// CampaignSessionMutationService exposes session lifecycle mutations.
type CampaignSessionMutationService interface {
	StartSession(context.Context, string, StartSessionInput) error
	EndSession(context.Context, string, EndSessionInput) error
}

// CampaignInviteReadService exposes invite-focused reads and search.
type CampaignInviteReadService interface {
	CampaignInvites(context.Context, string) ([]CampaignInvite, error)
	SearchInviteUsers(context.Context, string, SearchInviteUsersInput) ([]InviteUserSearchResult, error)
}

// CampaignInviteMutationService exposes invite create/revoke mutations.
type CampaignInviteMutationService interface {
	CreateInvite(context.Context, string, CreateInviteInput) error
	RevokeInvite(context.Context, string, RevokeInviteInput) error
}
