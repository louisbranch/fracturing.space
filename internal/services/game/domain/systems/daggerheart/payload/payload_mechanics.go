package payload

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// --- Class features ---

// ClassFeatureTargetPatchPayload captures one typed class feature activation that
// resolves to a durable character state patch.
type ClassFeatureTargetPatchPayload struct {
	CharacterID      ids.CharacterID                       `json:"character_id"`
	HPBefore         *int                                  `json:"hp_before,omitempty"`
	HPAfter          *int                                  `json:"hp_after,omitempty"`
	HopeBefore       *int                                  `json:"hope_before,omitempty"`
	HopeAfter        *int                                  `json:"hope_after,omitempty"`
	ArmorBefore      *int                                  `json:"armor_before,omitempty"`
	ArmorAfter       *int                                  `json:"armor_after,omitempty"`
	ClassStateBefore *daggerheartstate.CharacterClassState `json:"class_state_before,omitempty"`
	ClassStateAfter  *daggerheartstate.CharacterClassState `json:"class_state_after,omitempty"`
}

type ClassFeatureApplyPayload struct {
	ActorCharacterID ids.CharacterID                  `json:"actor_character_id"`
	Feature          string                           `json:"feature"`
	Targets          []ClassFeatureTargetPatchPayload `json:"targets"`
}

// --- Subclass features ---

type SubclassFeatureTargetPatchPayload struct {
	CharacterID         ids.CharacterID                          `json:"character_id"`
	HPBefore            *int                                     `json:"hp_before,omitempty"`
	HPAfter             *int                                     `json:"hp_after,omitempty"`
	HopeBefore          *int                                     `json:"hope_before,omitempty"`
	HopeAfter           *int                                     `json:"hope_after,omitempty"`
	StressBefore        *int                                     `json:"stress_before,omitempty"`
	StressAfter         *int                                     `json:"stress_after,omitempty"`
	ArmorBefore         *int                                     `json:"armor_before,omitempty"`
	ArmorAfter          *int                                     `json:"armor_after,omitempty"`
	ClassStateBefore    *daggerheartstate.CharacterClassState    `json:"class_state_before,omitempty"`
	ClassStateAfter     *daggerheartstate.CharacterClassState    `json:"class_state_after,omitempty"`
	SubclassStateBefore *daggerheartstate.CharacterSubclassState `json:"subclass_state_before,omitempty"`
	SubclassStateAfter  *daggerheartstate.CharacterSubclassState `json:"subclass_state_after,omitempty"`
}

type SubclassFeatureApplyPayload struct {
	ActorCharacterID          ids.CharacterID                     `json:"actor_character_id"`
	Feature                   string                              `json:"feature"`
	Targets                   []SubclassFeatureTargetPatchPayload `json:"targets,omitempty"`
	CharacterConditionTargets []ConditionChangePayload            `json:"character_condition_targets,omitempty"`
	AdversaryConditionTargets []AdversaryConditionChangePayload   `json:"adversary_condition_targets,omitempty"`
}

// --- Beastform ---

// BeastformTransformPayload captures one beastform transform command and the
// resulting state mutation.
type BeastformTransformPayload struct {
	ActorCharacterID ids.CharacterID                       `json:"actor_character_id"`
	CharacterID      ids.CharacterID                       `json:"character_id"`
	BeastformID      string                                `json:"beastform_id"`
	UseEvolution     bool                                  `json:"use_evolution,omitempty"`
	EvolutionTrait   string                                `json:"evolution_trait,omitempty"`
	HopeBefore       *int                                  `json:"hope_before,omitempty"`
	HopeAfter        *int                                  `json:"hope_after,omitempty"`
	StressBefore     *int                                  `json:"stress_before,omitempty"`
	StressAfter      *int                                  `json:"stress_after,omitempty"`
	ClassStateBefore *daggerheartstate.CharacterClassState `json:"class_state_before,omitempty"`
	ClassStateAfter  *daggerheartstate.CharacterClassState `json:"class_state_after,omitempty"`
}

// BeastformDropPayload captures one beastform drop command and the resulting
// class-state mutation.
type BeastformDropPayload struct {
	ActorCharacterID ids.CharacterID                       `json:"actor_character_id"`
	CharacterID      ids.CharacterID                       `json:"character_id"`
	BeastformID      string                                `json:"beastform_id"`
	Source           string                                `json:"source,omitempty"`
	ClassStateBefore *daggerheartstate.CharacterClassState `json:"class_state_before,omitempty"`
	ClassStateAfter  *daggerheartstate.CharacterClassState `json:"class_state_after,omitempty"`
}

// BeastformTransformedPayload captures the event payload emitted when a
// character enters beastform.
type BeastformTransformedPayload struct {
	CharacterID     ids.CharacterID                                 `json:"character_id"`
	BeastformID     string                                          `json:"beastform_id"`
	Hope            *int                                            `json:"hope_after,omitempty"`
	Stress          *int                                            `json:"stress_after,omitempty"`
	ActiveBeastform *daggerheartstate.CharacterActiveBeastformState `json:"active_beastform,omitempty"`
	Source          string                                          `json:"source,omitempty"`
}

// BeastformDroppedPayload captures the event payload emitted when a character
// leaves beastform.
type BeastformDroppedPayload struct {
	CharacterID ids.CharacterID `json:"character_id"`
	BeastformID string          `json:"beastform_id"`
	Source      string          `json:"source,omitempty"`
}

// --- Companion ---

// CompanionExperienceBeginPayload captures one companion dispatch command and
// the resulting runtime-state mutation.
type CompanionExperienceBeginPayload struct {
	ActorCharacterID     ids.CharacterID                           `json:"actor_character_id"`
	CharacterID          ids.CharacterID                           `json:"character_id"`
	ExperienceID         string                                    `json:"experience_id"`
	CompanionStateBefore *daggerheartstate.CharacterCompanionState `json:"companion_state_before,omitempty"`
	CompanionStateAfter  *daggerheartstate.CharacterCompanionState `json:"companion_state_after,omitempty"`
}

// CompanionReturnPayload captures one companion return command and the
// resulting state mutation.
type CompanionReturnPayload struct {
	ActorCharacterID     ids.CharacterID                           `json:"actor_character_id"`
	CharacterID          ids.CharacterID                           `json:"character_id"`
	Resolution           string                                    `json:"resolution,omitempty"`
	StressBefore         *int                                      `json:"stress_before,omitempty"`
	StressAfter          *int                                      `json:"stress_after,omitempty"`
	CompanionStateBefore *daggerheartstate.CharacterCompanionState `json:"companion_state_before,omitempty"`
	CompanionStateAfter  *daggerheartstate.CharacterCompanionState `json:"companion_state_after,omitempty"`
}

// CompanionExperienceBegunPayload captures the event payload emitted when a
// companion leaves on an experience.
type CompanionExperienceBegunPayload struct {
	CharacterID    ids.CharacterID                           `json:"character_id"`
	ExperienceID   string                                    `json:"experience_id"`
	CompanionState *daggerheartstate.CharacterCompanionState `json:"companion_state,omitempty"`
	Source         string                                    `json:"source,omitempty"`
}

// CompanionReturnedPayload captures the event payload emitted when a companion
// returns from an active experience.
type CompanionReturnedPayload struct {
	CharacterID    ids.CharacterID                           `json:"character_id"`
	Resolution     string                                    `json:"resolution,omitempty"`
	Stress         *int                                      `json:"stress_after,omitempty"`
	CompanionState *daggerheartstate.CharacterCompanionState `json:"companion_state,omitempty"`
	Source         string                                    `json:"source,omitempty"`
}
