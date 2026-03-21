package payload

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// --- Character state patch ---

// CharacterStatePatchPayload captures the payload for sys.daggerheart.character_state.patch commands.
// Source is an optional discriminator indicating what triggered the patch
// (e.g. "hope.spend", "stress.spend"), enabling journal queries to distinguish
// spend events from generic GM adjustments without introducing separate event types.
type CharacterStatePatchPayload struct {
	CharacterID                         ids.CharacterID                          `json:"character_id"`
	Source                              string                                   `json:"source,omitempty"`
	MutationSource                      *daggerheartstate.MutationSource         `json:"mutation_source,omitempty"`
	HPBefore                            *int                                     `json:"hp_before,omitempty"`
	HPAfter                             *int                                     `json:"hp_after,omitempty"`
	HopeBefore                          *int                                     `json:"hope_before,omitempty"`
	HopeAfter                           *int                                     `json:"hope_after,omitempty"`
	HopeMaxBefore                       *int                                     `json:"hope_max_before,omitempty"`
	HopeMaxAfter                        *int                                     `json:"hope_max_after,omitempty"`
	StressBefore                        *int                                     `json:"stress_before,omitempty"`
	StressAfter                         *int                                     `json:"stress_after,omitempty"`
	ArmorBefore                         *int                                     `json:"armor_before,omitempty"`
	ArmorAfter                          *int                                     `json:"armor_after,omitempty"`
	LifeStateBefore                     *string                                  `json:"life_state_before,omitempty"`
	LifeStateAfter                      *string                                  `json:"life_state_after,omitempty"`
	ClassStateBefore                    *daggerheartstate.CharacterClassState    `json:"class_state_before,omitempty"`
	ClassStateAfter                     *daggerheartstate.CharacterClassState    `json:"class_state_after,omitempty"`
	SubclassStateBefore                 *daggerheartstate.CharacterSubclassState `json:"subclass_state_before,omitempty"`
	SubclassStateAfter                  *daggerheartstate.CharacterSubclassState `json:"subclass_state_after,omitempty"`
	ImpenetrableUsedThisShortRestBefore *bool                                    `json:"impenetrable_used_this_short_rest_before,omitempty"`
	ImpenetrableUsedThisShortRestAfter  *bool                                    `json:"impenetrable_used_this_short_rest_after,omitempty"`
}

// CharacterStatePatchedPayload captures the payload for sys.daggerheart.character_state_patched events.
type CharacterStatePatchedPayload struct {
	CharacterID                   ids.CharacterID                          `json:"character_id"`
	Source                        string                                   `json:"source,omitempty"`
	HP                            *int                                     `json:"hp_after,omitempty"`
	Hope                          *int                                     `json:"hope_after,omitempty"`
	HopeMax                       *int                                     `json:"hope_max_after,omitempty"`
	Stress                        *int                                     `json:"stress_after,omitempty"`
	Armor                         *int                                     `json:"armor_after,omitempty"`
	LifeState                     *string                                  `json:"life_state_after,omitempty"`
	ClassState                    *daggerheartstate.CharacterClassState    `json:"class_state_after,omitempty"`
	SubclassState                 *daggerheartstate.CharacterSubclassState `json:"subclass_state_after,omitempty"`
	ImpenetrableUsedThisShortRest *bool                                    `json:"impenetrable_used_this_short_rest_after,omitempty"`
}

// --- Hope/Stress spend ---

// HopeSpendPayload captures the payload for sys.daggerheart.hope.spend commands.
type HopeSpendPayload struct {
	CharacterID ids.CharacterID `json:"character_id"`
	Amount      int             `json:"amount"`
	Before      int             `json:"before"`
	After       int             `json:"after"`
	RollSeq     *uint64         `json:"roll_seq,omitempty"`
	Source      string          `json:"source,omitempty"`
}

// StressSpendPayload captures the payload for sys.daggerheart.stress.spend commands.
type StressSpendPayload struct {
	CharacterID ids.CharacterID `json:"character_id"`
	Amount      int             `json:"amount"`
	Before      int             `json:"before"`
	After       int             `json:"after"`
	RollSeq     *uint64         `json:"roll_seq,omitempty"`
	Source      string          `json:"source,omitempty"`
}
