package projection

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/character"

func registerCharacterProjectionHandlers(r *CoreRouter) {
	HandleProjection(r, character.EventTypeCreated, requirements(needsStores(storeCharacter, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyCharacterCreated)
	HandleProjection(r, character.EventTypeUpdated, requirements(needsStores(storeCharacter, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyCharacterUpdated)
	HandleProjection(r, character.EventTypeDeleted, requirements(needsStores(storeCharacter, storeCampaign), needsEnvelope(fieldCampaignID, fieldEntityID)), Applier.applyCharacterDeleted)
}
