package payload

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/dhids"
)

type CountdownStartingRollPayload struct {
	Min   int `json:"min"`
	Max   int `json:"max"`
	Value int `json:"value"`
}

// --- Scene countdowns ---

type SceneCountdownCreatePayload struct {
	SessionID         ids.SessionID                 `json:"session_id,omitempty"`
	SceneID           ids.SceneID                   `json:"scene_id,omitempty"`
	CountdownID       dhids.CountdownID             `json:"countdown_id"`
	Name              string                        `json:"name"`
	Tone              string                        `json:"tone"`
	AdvancementPolicy string                        `json:"advancement_policy"`
	StartingValue     int                           `json:"starting_value"`
	RemainingValue    int                           `json:"remaining_value"`
	LoopBehavior      string                        `json:"loop_behavior"`
	Status            string                        `json:"status"`
	LinkedCountdownID dhids.CountdownID             `json:"linked_countdown_id,omitempty"`
	StartingRoll      *CountdownStartingRollPayload `json:"starting_roll,omitempty"`
}

type SceneCountdownCreatedPayload = SceneCountdownCreatePayload

type SceneCountdownAdvancePayload struct {
	CountdownID     dhids.CountdownID `json:"countdown_id"`
	BeforeRemaining int               `json:"before_remaining"`
	AfterRemaining  int               `json:"after_remaining"`
	AdvancedBy      int               `json:"advanced_by"`
	StatusBefore    string            `json:"status_before"`
	StatusAfter     string            `json:"status_after"`
	Triggered       bool              `json:"triggered"`
	Reason          string            `json:"reason,omitempty"`
}

type SceneCountdownAdvancedPayload = SceneCountdownAdvancePayload

type SceneCountdownTriggerResolvePayload struct {
	CountdownID          dhids.CountdownID `json:"countdown_id"`
	StartingValueBefore  int               `json:"starting_value_before"`
	StartingValueAfter   int               `json:"starting_value_after"`
	RemainingValueBefore int               `json:"remaining_value_before"`
	RemainingValueAfter  int               `json:"remaining_value_after"`
	StatusBefore         string            `json:"status_before"`
	StatusAfter          string            `json:"status_after"`
	Reason               string            `json:"reason,omitempty"`
}

type SceneCountdownTriggerResolvedPayload = SceneCountdownTriggerResolvePayload

type SceneCountdownDeletePayload struct {
	CountdownID dhids.CountdownID `json:"countdown_id"`
	Reason      string            `json:"reason,omitempty"`
}

type SceneCountdownDeletedPayload = SceneCountdownDeletePayload

// --- Campaign countdowns ---

type CampaignCountdownCreatePayload = SceneCountdownCreatePayload
type CampaignCountdownCreatedPayload = CampaignCountdownCreatePayload
type CampaignCountdownAdvancePayload = SceneCountdownAdvancePayload
type CampaignCountdownAdvancedPayload = CampaignCountdownAdvancePayload
type CampaignCountdownTriggerResolvePayload = SceneCountdownTriggerResolvePayload
type CampaignCountdownTriggerResolvedPayload = CampaignCountdownTriggerResolvePayload
type CampaignCountdownDeletePayload = SceneCountdownDeletePayload
type CampaignCountdownDeletedPayload = CampaignCountdownDeletePayload
