package integration

import "fmt"

const AIGMTurnRequestedOutboxEventType = "game.ai_gm_turn.requested.v1"

// AIGMTurnRequestedOutboxPayload is the durable worker-facing request emitted
// when interaction-owned state reaches a GM-owned AI decision point.
type AIGMTurnRequestedOutboxPayload struct {
	CampaignID      string `json:"campaign_id"`
	SessionID       string `json:"session_id"`
	SourceEventType string `json:"source_event_type,omitempty"`
	SourceSceneID   string `json:"source_scene_id,omitempty"`
	SourcePhaseID   string `json:"source_phase_id,omitempty"`
}

// AIGMTurnRequestedDedupeKey returns the stable dedupe key for one AI GM turn
// request sourced from a persisted journal event. Duplicate enqueue attempts
// for the same campaign event must collapse onto the same outbox item.
func AIGMTurnRequestedDedupeKey(campaignID string, sourceEventSeq uint64) string {
	return fmt.Sprintf("ai-gm-turn:%s:%d", campaignID, sourceEventSeq)
}
