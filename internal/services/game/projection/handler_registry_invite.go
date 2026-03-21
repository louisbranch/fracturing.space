package projection

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"

func registerInviteProjectionHandlers(r *CoreRouter) {
	// InviteID comes from payload with EntityID fallback for created/updated.
	HandleProjection(r, invite.EventTypeCreated, requirements(needsStores(storeInvite, storeCampaign), needsEnvelope(fieldCampaignID)), Applier.applyInviteCreated)
	HandleProjection(r, invite.EventTypeClaimed, requirements(needsStores(storeInvite, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyInviteClaimed)
	HandleProjection(r, invite.EventTypeDeclined, requirements(needsStores(storeInvite, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyInviteDeclined)
	HandleProjection(r, invite.EventTypeRevoked, requirements(needsStores(storeInvite, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyInviteRevoked)
	HandleProjection(r, invite.EventTypeUpdated, requirements(needsStores(storeInvite)), Applier.applyInviteUpdated)
}
