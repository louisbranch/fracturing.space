package projection

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"

func registerSceneProjectionHandlers(r *CoreRouter) {
	HandleProjection(r, scene.EventTypeCreated, requirements(needsStores(storeScene, storeSceneCharacter), needsEnvelope(fieldCampaignID)), Applier.applySceneCreated)
	HandleProjection(r, scene.EventTypeUpdated, requirements(needsStores(storeScene), needsEnvelope(fieldCampaignID)), Applier.applySceneUpdated)
	HandleProjection(r, scene.EventTypeEnded, requirements(needsStores(storeScene, storeSceneSpotlight), needsEnvelope(fieldCampaignID)), Applier.applySceneEnded)
	HandleProjection(r, scene.EventTypeCharacterAdded, requirements(needsStores(storeSceneCharacter), needsEnvelope(fieldCampaignID)), Applier.applySceneCharacterAdded)
	HandleProjection(r, scene.EventTypeCharacterRemoved, requirements(needsStores(storeSceneCharacter), needsEnvelope(fieldCampaignID)), Applier.applySceneCharacterRemoved)
	HandleProjection(r, scene.EventTypeGateOpened, requirements(needsStores(storeSceneGate), needsEnvelope(fieldCampaignID)), Applier.applySceneGateOpened)
	HandleProjection(r, scene.EventTypeGateResolved, requirements(needsStores(storeSceneGate), needsEnvelope(fieldCampaignID)), Applier.applySceneGateResolved)
	HandleProjection(r, scene.EventTypeGateAbandoned, requirements(needsStores(storeSceneGate), needsEnvelope(fieldCampaignID)), Applier.applySceneGateAbandoned)
	HandleProjection(r, scene.EventTypeSpotlightSet, requirements(needsStores(storeSceneSpotlight), needsEnvelope(fieldCampaignID)), Applier.applySceneSpotlightSet)
	HandleProjection(r, scene.EventTypeSpotlightCleared, requirements(needsStores(storeSceneSpotlight), needsEnvelope(fieldCampaignID)), Applier.applySceneSpotlightCleared)
	HandleProjection(r, scene.EventTypePlayerPhaseStarted, requirements(needsStores(storeSceneInteraction), needsEnvelope(fieldCampaignID)), Applier.applyScenePlayerPhaseStarted)
	HandleProjection(r, scene.EventTypePlayerPhasePosted, requirements(needsStores(storeSceneInteraction), needsEnvelope(fieldCampaignID)), Applier.applyScenePlayerPhasePosted)
	HandleProjection(r, scene.EventTypePlayerPhaseYielded, requirements(needsStores(storeSceneInteraction), needsEnvelope(fieldCampaignID)), Applier.applyScenePlayerPhaseYielded)
	HandleProjection(r, scene.EventTypePlayerPhaseReviewStarted, requirements(needsStores(storeSceneInteraction), needsEnvelope(fieldCampaignID)), Applier.applyScenePlayerPhaseReviewStarted)
	HandleProjection(r, scene.EventTypePlayerPhaseUnyielded, requirements(needsStores(storeSceneInteraction), needsEnvelope(fieldCampaignID)), Applier.applyScenePlayerPhaseUnyielded)
	HandleProjection(r, scene.EventTypePlayerPhaseRevisionsRequested, requirements(needsStores(storeSceneInteraction), needsEnvelope(fieldCampaignID)), Applier.applyScenePlayerPhaseRevisionsRequested)
	HandleProjection(r, scene.EventTypePlayerPhaseAccepted, requirements(needsStores(storeSceneInteraction), needsEnvelope(fieldCampaignID)), Applier.applyScenePlayerPhaseAccepted)
	HandleProjection(r, scene.EventTypePlayerPhaseEnded, requirements(needsStores(storeSceneInteraction), needsEnvelope(fieldCampaignID)), Applier.applyScenePlayerPhaseEnded)
	HandleProjection(r, scene.EventTypeGMOutputCommitted, requirements(needsStores(storeSceneInteraction), needsEnvelope(fieldCampaignID)), Applier.applySceneGMOutputCommitted)
}
