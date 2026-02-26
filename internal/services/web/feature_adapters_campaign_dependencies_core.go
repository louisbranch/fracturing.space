package web

import (
	"context"

	v1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignfeature "github.com/louisbranch/fracturing.space/internal/services/web/feature/campaign"
)

func buildCampaignFeatureCoreDependencies(h *handler, d *campaignfeature.AppCampaignDependencies) {
	d.EnsureCampaignClients = func(ctx context.Context) error {
		return h.ensureCampaignClients(ctx)
	}
	d.CampaignClientReady = func() bool {
		return h != nil && h.campaignClient != nil && h.campaignAccess != nil
	}
	d.SessionClientReady = func() bool {
		return h != nil && h.sessionClient != nil
	}
	d.ParticipantClientReady = func() bool {
		return h != nil && h.participantClient != nil
	}
	d.CharacterClientReady = func() bool {
		return h != nil && h.characterClient != nil
	}
	d.InviteClientReady = func() bool {
		return h != nil && h.inviteClient != nil
	}
	d.CanManageCampaignAccess = func(access v1.CampaignAccess) bool {
		return campaignfeature.CanManageCampaignAccess(access)
	}
}
