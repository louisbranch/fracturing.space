package projection

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"

func registerParticipantProjectionHandlers(r *CoreRouter) {
	HandleProjection(r, participant.EventTypeJoined, requirements(needsStores(storeParticipant, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyParticipantJoined)
	HandleProjection(r, participant.EventTypeUpdated, requirements(needsStores(storeParticipant, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyParticipantUpdated)
	HandleProjectionRaw(r, participant.EventTypeLeft, requirements(needsStores(storeParticipant, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyParticipantLeft)
	HandleProjection(r, participant.EventTypeBound, requirements(needsStores(storeParticipant, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyParticipantBound)
	HandleProjection(r, participant.EventTypeUnbound, requirements(needsStores(storeParticipant, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyParticipantUnbound)
	HandleProjection(r, participant.EventTypeSeatReassigned, requirements(needsStores(storeParticipant, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applySeatReassigned)
}
