package projection

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/session"

func registerSessionProjectionHandlers(r *CoreRouter) {
	// SessionID comes from payload with EntityID fallback, so EntityID is not a
	// hard envelope requirement for started/ended.
	HandleProjection(r, session.EventTypeStarted, requirements(needsStores(storeSession), needsEnvelope(fieldCampaignID)), Applier.applySessionStarted)
	HandleProjection(r, session.EventTypeEnded, requirements(needsStores(storeSession), needsEnvelope(fieldCampaignID)), Applier.applySessionEnded)

	// Gate handlers derive GateID from payload with EntityID fallback.
	HandleProjection(r, session.EventTypeGateOpened, requirements(needsStores(storeSessionGate), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionGateOpened)
	HandleProjection(r, session.EventTypeGateResponseRecorded, requirements(needsStores(storeSessionGate), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionGateResponseRecorded)
	HandleProjection(r, session.EventTypeGateResolved, requirements(needsStores(storeSessionGate), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionGateResolved)
	HandleProjection(r, session.EventTypeGateAbandoned, requirements(needsStores(storeSessionGate), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionGateAbandoned)

	HandleProjection(r, session.EventTypeSpotlightSet, requirements(needsStores(storeSessionSpotlight), needsEnvelope(fieldSessionID)), Applier.applySessionSpotlightSet)
	HandleProjectionRaw(r, session.EventTypeSpotlightCleared, requirements(needsStores(storeSessionSpotlight), needsEnvelope(fieldSessionID)), Applier.applySessionSpotlightCleared)

	HandleProjection(r, session.EventTypeActiveSceneSet, requirements(needsStores(storeSessionInteraction), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionActiveSceneSet)
	HandleProjection(r, session.EventTypeGMAuthoritySet, requirements(needsStores(storeSessionInteraction), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionGMAuthoritySet)
	HandleProjection(r, session.EventTypeOOCPaused, requirements(needsStores(storeSessionInteraction), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionOOCPaused)
	HandleProjection(r, session.EventTypeOOCPosted, requirements(needsStores(storeSessionInteraction), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionOOCPosted)
	HandleProjection(r, session.EventTypeOOCReadyMarked, requirements(needsStores(storeSessionInteraction), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionOOCReadyMarked)
	HandleProjection(r, session.EventTypeOOCReadyCleared, requirements(needsStores(storeSessionInteraction), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionOOCReadyCleared)
	HandleProjection(r, session.EventTypeOOCResumed, requirements(needsStores(storeSessionInteraction), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionOOCResumed)
	HandleProjection(r, session.EventTypeOOCInterruptionResolved, requirements(needsStores(storeSessionInteraction), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionOOCInterruptionResolved)
	HandleProjection(r, session.EventTypeAITurnQueued, requirements(needsStores(storeSessionInteraction), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionAITurnQueued)
	HandleProjection(r, session.EventTypeAITurnRunning, requirements(needsStores(storeSessionInteraction), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionAITurnRunning)
	HandleProjection(r, session.EventTypeAITurnFailed, requirements(needsStores(storeSessionInteraction), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionAITurnFailed)
	HandleProjection(r, session.EventTypeAITurnCleared, requirements(needsStores(storeSessionInteraction), needsEnvelope(fieldCampaignID, fieldSessionID)), Applier.applySessionAITurnCleared)
}
