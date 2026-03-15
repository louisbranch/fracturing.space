package integration

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

func AIGMTurnRequestedDedupeKey(eventID string) string {
	return "ai-gm-turn:" + eventID
}
