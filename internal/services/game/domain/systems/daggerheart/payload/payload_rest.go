package payload

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"

// --- Loadout ---

// LoadoutSwapPayload captures the payload for sys.daggerheart.loadout.swap commands.
type LoadoutSwapPayload struct {
	CharacterID  ids.CharacterID `json:"character_id"`
	CardID       string          `json:"card_id"`
	From         string          `json:"from"`
	To           string          `json:"to"`
	RecallCost   int             `json:"recall_cost,omitempty"`
	StressBefore *int            `json:"stress_before,omitempty"`
	StressAfter  *int            `json:"stress_after,omitempty"`
}

// LoadoutSwappedPayload captures the payload for sys.daggerheart.loadout_swapped events.
type LoadoutSwappedPayload struct {
	CharacterID ids.CharacterID `json:"character_id"`
	CardID      string          `json:"card_id"`
	From        string          `json:"from"`
	To          string          `json:"to"`
	RecallCost  int             `json:"recall_cost,omitempty"`
	Stress      *int            `json:"stress_after,omitempty"`
}

// --- Rest ---

// RestTakePayload captures the payload for sys.daggerheart.rest.take commands.
type RestTakePayload struct {
	RestType         string                       `json:"rest_type"`
	Interrupted      bool                         `json:"interrupted"`
	GMFearBefore     int                          `json:"gm_fear_before"`
	GMFearAfter      int                          `json:"gm_fear_after"`
	ShortRestsBefore int                          `json:"short_rests_before"`
	ShortRestsAfter  int                          `json:"short_rests_after"`
	RefreshRest      bool                         `json:"refresh_rest"`
	RefreshLongRest  bool                         `json:"refresh_long_rest"`
	Participants     []ids.CharacterID            `json:"participants,omitempty"`
	DowntimeMoves    []DowntimeMoveAppliedPayload `json:"downtime_moves,omitempty"`
	CountdownUpdates []CountdownUpdatePayload     `json:"countdown_updates,omitempty"`
}

// RestTakenPayload captures the payload for sys.daggerheart.rest_taken events.
type RestTakenPayload struct {
	RestType        string            `json:"rest_type"`
	Interrupted     bool              `json:"interrupted"`
	GMFear          int               `json:"gm_fear_after"`
	ShortRests      int               `json:"short_rests_after"`
	RefreshRest     bool              `json:"refresh_rest"`
	RefreshLongRest bool              `json:"refresh_long_rest"`
	Participants    []ids.CharacterID `json:"participants,omitempty"`
}

// --- Temporary armor ---

// CharacterTemporaryArmorApplyPayload captures the payload for sys.daggerheart.character_temporary_armor.apply commands.
type CharacterTemporaryArmorApplyPayload struct {
	CharacterID ids.CharacterID `json:"character_id"`
	Source      string          `json:"source"`
	Duration    string          `json:"duration"`
	Amount      int             `json:"amount"`
	SourceID    string          `json:"source_id,omitempty"`
}

// CharacterTemporaryArmorAppliedPayload captures the payload for sys.daggerheart.character_temporary_armor_applied events.
type CharacterTemporaryArmorAppliedPayload = CharacterTemporaryArmorApplyPayload
