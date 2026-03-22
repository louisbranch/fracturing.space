package payload

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"

type CountdownStartingRollPayload struct {
	Min   int `json:"min"`
	Max   int `json:"max"`
	Value int `json:"value"`
}

// --- Scene countdowns ---

type SceneCountdownCreatePayload struct {
	SessionID         ids.SessionID                 `json:"session_id,omitempty"`
	SceneID           ids.SceneID                   `json:"scene_id,omitempty"`
	CountdownID       ids.CountdownID               `json:"countdown_id"`
	Name              string                        `json:"name"`
	Tone              string                        `json:"tone"`
	AdvancementPolicy string                        `json:"advancement_policy"`
	StartingValue     int                           `json:"starting_value"`
	RemainingValue    int                           `json:"remaining_value"`
	LoopBehavior      string                        `json:"loop_behavior"`
	Status            string                        `json:"status"`
	LinkedCountdownID ids.CountdownID               `json:"linked_countdown_id,omitempty"`
	StartingRoll      *CountdownStartingRollPayload `json:"starting_roll,omitempty"`

	Kind             string `json:"kind,omitempty"`
	Current          int    `json:"current,omitempty"`
	Max              int    `json:"max,omitempty"`
	Direction        string `json:"direction,omitempty"`
	Looping          bool   `json:"looping,omitempty"`
	Variant          string `json:"variant,omitempty"`
	TriggerEventType string `json:"trigger_event_type,omitempty"`
}

type SceneCountdownCreatedPayload = SceneCountdownCreatePayload

type SceneCountdownAdvancePayload struct {
	CountdownID     ids.CountdownID `json:"countdown_id"`
	BeforeRemaining int             `json:"before_remaining"`
	AfterRemaining  int             `json:"after_remaining"`
	AdvancedBy      int             `json:"advanced_by"`
	StatusBefore    string          `json:"status_before"`
	StatusAfter     string          `json:"status_after"`
	Triggered       bool            `json:"triggered"`
	Reason          string          `json:"reason,omitempty"`

	Before int  `json:"before,omitempty"`
	After  int  `json:"after,omitempty"`
	Delta  int  `json:"delta,omitempty"`
	Looped bool `json:"looped,omitempty"`
	Value  int  `json:"value,omitempty"`
}

type SceneCountdownAdvancedPayload = SceneCountdownAdvancePayload

type SceneCountdownTriggerResolvePayload struct {
	CountdownID          ids.CountdownID `json:"countdown_id"`
	StartingValueBefore  int             `json:"starting_value_before"`
	StartingValueAfter   int             `json:"starting_value_after"`
	RemainingValueBefore int             `json:"remaining_value_before"`
	RemainingValueAfter  int             `json:"remaining_value_after"`
	StatusBefore         string          `json:"status_before"`
	StatusAfter          string          `json:"status_after"`
	Reason               string          `json:"reason,omitempty"`
}

type SceneCountdownTriggerResolvedPayload = SceneCountdownTriggerResolvePayload

type SceneCountdownDeletePayload struct {
	CountdownID ids.CountdownID `json:"countdown_id"`
	Reason      string          `json:"reason,omitempty"`
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

// Legacy aliases retained temporarily while tests and read surfaces finish
// moving to the SRD-native advance/resolve vocabulary.
type CountdownCreatePayload = SceneCountdownCreatePayload
type CountdownCreatedPayload = SceneCountdownCreatedPayload
type CountdownUpdatePayload = CampaignCountdownAdvancePayload
type CountdownUpdatedPayload = CampaignCountdownAdvancedPayload
type CountdownDeletePayload = SceneCountdownDeletePayload
type CountdownDeletedPayload = SceneCountdownDeletedPayload
type SceneCountdownUpdatePayload = SceneCountdownAdvancePayload
type SceneCountdownUpdatedPayload = SceneCountdownAdvancedPayload
type CampaignCountdownUpdatePayload = CampaignCountdownAdvancePayload
type CampaignCountdownUpdatedPayload = CampaignCountdownAdvancedPayload
