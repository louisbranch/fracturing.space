package app

import "context"

// RequireManageCampaign enforces owner/manager campaign access.
func (s authorizationService) RequireManageCampaign(ctx context.Context, campaignID string) error {
	return s.auth.requireManageCampaign(ctx, campaignID)
}

// RequireManageParticipants enforces owner/manager participant governance access.
func (s authorizationService) RequireManageParticipants(ctx context.Context, campaignID string) error {
	return s.auth.requireManageParticipants(ctx, campaignID)
}

// RequireManageInvites enforces owner/manager invite governance access.
func (s authorizationService) RequireManageInvites(ctx context.Context, campaignID string) error {
	return s.auth.requireManageInvites(ctx, campaignID)
}

// RequireMutateCharacters enforces character-mutation access.
func (s authorizationService) RequireMutateCharacters(ctx context.Context, campaignID string) error {
	return s.auth.requireMutateCharacters(ctx, campaignID)
}
