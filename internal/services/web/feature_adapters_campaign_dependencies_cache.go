package web

import (
	"context"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
)

func buildCampaignFeatureCacheDependencies(_ *handler, campaignCache *campaignfeature.CampaignCache, d *campaignfeature.AppCampaignDependencies) {
	d.CachedUserCampaigns = func(ctx context.Context, userID string) ([]*statev1.Campaign, bool) {
		return campaignCache.CachedUserCampaigns(ctx, userID)
	}
	d.SetUserCampaignsCache = func(ctx context.Context, userID string, campaigns []*statev1.Campaign) {
		campaignCache.SetUserCampaignsCache(ctx, userID, campaigns)
	}
	d.ExpireUserCampaignsCache = func(ctx context.Context, userID string) {
		campaignCache.ExpireUserCampaignsCache(ctx, userID)
	}
	d.SetCampaignCache = func(ctx context.Context, campaign *statev1.Campaign) {
		campaignCache.SetCampaignCache(ctx, campaign)
	}
	d.CachedCampaignSessions = func(ctx context.Context, campaignID string) ([]*statev1.Session, bool) {
		return campaignCache.CachedCampaignSessions(ctx, campaignID)
	}
	d.SetCampaignSessionsCache = func(ctx context.Context, campaignID string, sessions []*statev1.Session) {
		campaignCache.SetCampaignSessionsCache(ctx, campaignID, sessions)
	}
	d.CachedCampaignParticipants = func(ctx context.Context, campaignID string) ([]*statev1.Participant, bool) {
		return campaignCache.CachedCampaignParticipants(ctx, campaignID)
	}
	d.SetCampaignParticipantsCache = func(ctx context.Context, campaignID string, participants []*statev1.Participant) {
		campaignCache.SetCampaignParticipantsCache(ctx, campaignID, participants)
	}
	d.CachedCampaignCharacters = func(ctx context.Context, campaignID string) ([]*statev1.Character, bool) {
		return campaignCache.CachedCampaignCharacters(ctx, campaignID)
	}
	d.SetCampaignCharactersCache = func(ctx context.Context, campaignID string, characters []*statev1.Character) {
		campaignCache.SetCampaignCharactersCache(ctx, campaignID, characters)
	}
	d.CachedCampaignInvites = func(ctx context.Context, campaignID string, userID string) ([]*statev1.Invite, bool) {
		return campaignCache.CachedCampaignInvites(ctx, campaignID, userID)
	}
	d.SetCampaignInvitesCache = func(ctx context.Context, campaignID string, userID string, invites []*statev1.Invite) {
		campaignCache.SetCampaignInvitesCache(ctx, campaignID, userID, invites)
	}
}
