package payload

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

// --- Conditions ---

// ConditionChangePayload captures the payload for sys.daggerheart.condition.change commands.
type ConditionChangePayload struct {
	CharacterID      ids.CharacterID        `json:"character_id"`
	ConditionsBefore []rules.ConditionState `json:"conditions_before,omitempty"`
	ConditionsAfter  []rules.ConditionState `json:"conditions_after"`
	Added            []rules.ConditionState `json:"added,omitempty"`
	Removed          []rules.ConditionState `json:"removed,omitempty"`
	Source           string                 `json:"source,omitempty"`
	RollSeq          *uint64                `json:"roll_seq,omitempty"`
}

// ConditionChangedPayload captures the payload for sys.daggerheart.condition_changed events.
type ConditionChangedPayload struct {
	CharacterID ids.CharacterID        `json:"character_id"`
	Conditions  []rules.ConditionState `json:"conditions_after"`
	Added       []rules.ConditionState `json:"added,omitempty"`
	Removed     []rules.ConditionState `json:"removed,omitempty"`
	Source      string                 `json:"source,omitempty"`
	RollSeq     *uint64                `json:"roll_seq,omitempty"`
}

// --- Adversary conditions ---

// AdversaryConditionChangePayload captures the payload for sys.daggerheart.adversary_condition.change commands.
type AdversaryConditionChangePayload struct {
	AdversaryID      ids.AdversaryID        `json:"adversary_id"`
	ConditionsBefore []rules.ConditionState `json:"conditions_before,omitempty"`
	ConditionsAfter  []rules.ConditionState `json:"conditions_after"`
	Added            []rules.ConditionState `json:"added,omitempty"`
	Removed          []rules.ConditionState `json:"removed,omitempty"`
	Source           string                 `json:"source,omitempty"`
	RollSeq          *uint64                `json:"roll_seq,omitempty"`
}

// AdversaryConditionChangedPayload captures the payload for sys.daggerheart.adversary_condition_changed events.
type AdversaryConditionChangedPayload struct {
	AdversaryID ids.AdversaryID        `json:"adversary_id"`
	Conditions  []rules.ConditionState `json:"conditions_after"`
	Added       []rules.ConditionState `json:"added,omitempty"`
	Removed     []rules.ConditionState `json:"removed,omitempty"`
	Source      string                 `json:"source,omitempty"`
	RollSeq     *uint64                `json:"roll_seq,omitempty"`
}
