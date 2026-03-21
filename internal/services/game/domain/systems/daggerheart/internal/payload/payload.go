package payload

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/rules"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/snapstate"
)

// --- GM Fear ---

// GMFearSetPayload captures the payload for sys.daggerheart.gm_fear.set commands.
type GMFearSetPayload struct {
	After  *int   `json:"after,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// GMFearChangedPayload captures the payload for sys.daggerheart.gm_fear_changed events.
type GMFearChangedPayload struct {
	Value  int    `json:"after"`
	Reason string `json:"reason,omitempty"`
}

// --- GM Moves ---

// GMMoveTarget captures the typed Fear-spend target stored on GM move
// commands and audit events.
type GMMoveTarget struct {
	Type                rules.GMMoveTargetType  `json:"type"`
	Kind                rules.GMMoveKind        `json:"kind,omitempty"`
	Shape               rules.GMMoveShape       `json:"shape,omitempty"`
	Description         string                  `json:"description,omitempty"`
	AdversaryID         ids.AdversaryID         `json:"adversary_id,omitempty"`
	EnvironmentEntityID ids.EnvironmentEntityID `json:"environment_entity_id,omitempty"`
	EnvironmentID       string                  `json:"environment_id,omitempty"`
	FeatureID           string                  `json:"feature_id,omitempty"`
	ExperienceName      string                  `json:"experience_name,omitempty"`
}

// GMMoveApplyPayload captures the payload for sys.daggerheart.gm_move.apply
// commands.
type GMMoveApplyPayload struct {
	Target    GMMoveTarget `json:"target"`
	FearSpent int          `json:"fear_spent"`
}

// GMMoveAppliedPayload captures the payload for sys.daggerheart.gm_move_applied
// events.
type GMMoveAppliedPayload struct {
	Target    GMMoveTarget `json:"target"`
	FearSpent int          `json:"fear_spent"`
}

// --- Character state patch ---

// CharacterStatePatchPayload captures the payload for sys.daggerheart.character_state.patch commands.
// Source is an optional discriminator indicating what triggered the patch
// (e.g. "hope.spend", "stress.spend"), enabling journal queries to distinguish
// spend events from generic GM adjustments without introducing separate event types.
type CharacterStatePatchPayload struct {
	CharacterID                         ids.CharacterID                   `json:"character_id"`
	Source                              string                            `json:"source,omitempty"`
	HPBefore                            *int                              `json:"hp_before,omitempty"`
	HPAfter                             *int                              `json:"hp_after,omitempty"`
	HopeBefore                          *int                              `json:"hope_before,omitempty"`
	HopeAfter                           *int                              `json:"hope_after,omitempty"`
	HopeMaxBefore                       *int                              `json:"hope_max_before,omitempty"`
	HopeMaxAfter                        *int                              `json:"hope_max_after,omitempty"`
	StressBefore                        *int                              `json:"stress_before,omitempty"`
	StressAfter                         *int                              `json:"stress_after,omitempty"`
	ArmorBefore                         *int                              `json:"armor_before,omitempty"`
	ArmorAfter                          *int                              `json:"armor_after,omitempty"`
	LifeStateBefore                     *string                           `json:"life_state_before,omitempty"`
	LifeStateAfter                      *string                           `json:"life_state_after,omitempty"`
	ClassStateBefore                    *snapstate.CharacterClassState    `json:"class_state_before,omitempty"`
	ClassStateAfter                     *snapstate.CharacterClassState    `json:"class_state_after,omitempty"`
	SubclassStateBefore                 *snapstate.CharacterSubclassState `json:"subclass_state_before,omitempty"`
	SubclassStateAfter                  *snapstate.CharacterSubclassState `json:"subclass_state_after,omitempty"`
	ImpenetrableUsedThisShortRestBefore *bool                             `json:"impenetrable_used_this_short_rest_before,omitempty"`
	ImpenetrableUsedThisShortRestAfter  *bool                             `json:"impenetrable_used_this_short_rest_after,omitempty"`
}

// CharacterStatePatchedPayload captures the payload for sys.daggerheart.character_state_patched events.
type CharacterStatePatchedPayload struct {
	CharacterID                   ids.CharacterID                   `json:"character_id"`
	Source                        string                            `json:"source,omitempty"`
	HP                            *int                              `json:"hp_after,omitempty"`
	Hope                          *int                              `json:"hope_after,omitempty"`
	HopeMax                       *int                              `json:"hope_max_after,omitempty"`
	Stress                        *int                              `json:"stress_after,omitempty"`
	Armor                         *int                              `json:"armor_after,omitempty"`
	LifeState                     *string                           `json:"life_state_after,omitempty"`
	ClassState                    *snapstate.CharacterClassState    `json:"class_state_after,omitempty"`
	SubclassState                 *snapstate.CharacterSubclassState `json:"subclass_state_after,omitempty"`
	ImpenetrableUsedThisShortRest *bool                             `json:"impenetrable_used_this_short_rest_after,omitempty"`
}

// --- Class features ---

// ClassFeatureTargetPatchPayload captures one typed class feature activation that
// resolves to a durable character state patch.
type ClassFeatureTargetPatchPayload struct {
	CharacterID      ids.CharacterID                `json:"character_id"`
	HPBefore         *int                           `json:"hp_before,omitempty"`
	HPAfter          *int                           `json:"hp_after,omitempty"`
	HopeBefore       *int                           `json:"hope_before,omitempty"`
	HopeAfter        *int                           `json:"hope_after,omitempty"`
	ArmorBefore      *int                           `json:"armor_before,omitempty"`
	ArmorAfter       *int                           `json:"armor_after,omitempty"`
	ClassStateBefore *snapstate.CharacterClassState `json:"class_state_before,omitempty"`
	ClassStateAfter  *snapstate.CharacterClassState `json:"class_state_after,omitempty"`
}

type ClassFeatureApplyPayload struct {
	ActorCharacterID ids.CharacterID                  `json:"actor_character_id"`
	Feature          string                           `json:"feature"`
	Targets          []ClassFeatureTargetPatchPayload `json:"targets"`
}

// --- Subclass features ---

type SubclassFeatureTargetPatchPayload struct {
	CharacterID         ids.CharacterID                   `json:"character_id"`
	HPBefore            *int                              `json:"hp_before,omitempty"`
	HPAfter             *int                              `json:"hp_after,omitempty"`
	HopeBefore          *int                              `json:"hope_before,omitempty"`
	HopeAfter           *int                              `json:"hope_after,omitempty"`
	StressBefore        *int                              `json:"stress_before,omitempty"`
	StressAfter         *int                              `json:"stress_after,omitempty"`
	ArmorBefore         *int                              `json:"armor_before,omitempty"`
	ArmorAfter          *int                              `json:"armor_after,omitempty"`
	ClassStateBefore    *snapstate.CharacterClassState    `json:"class_state_before,omitempty"`
	ClassStateAfter     *snapstate.CharacterClassState    `json:"class_state_after,omitempty"`
	SubclassStateBefore *snapstate.CharacterSubclassState `json:"subclass_state_before,omitempty"`
	SubclassStateAfter  *snapstate.CharacterSubclassState `json:"subclass_state_after,omitempty"`
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
	ActorCharacterID ids.CharacterID                `json:"actor_character_id"`
	CharacterID      ids.CharacterID                `json:"character_id"`
	BeastformID      string                         `json:"beastform_id"`
	UseEvolution     bool                           `json:"use_evolution,omitempty"`
	EvolutionTrait   string                         `json:"evolution_trait,omitempty"`
	HopeBefore       *int                           `json:"hope_before,omitempty"`
	HopeAfter        *int                           `json:"hope_after,omitempty"`
	StressBefore     *int                           `json:"stress_before,omitempty"`
	StressAfter      *int                           `json:"stress_after,omitempty"`
	ClassStateBefore *snapstate.CharacterClassState `json:"class_state_before,omitempty"`
	ClassStateAfter  *snapstate.CharacterClassState `json:"class_state_after,omitempty"`
}

// BeastformDropPayload captures one beastform drop command and the resulting
// class-state mutation.
type BeastformDropPayload struct {
	ActorCharacterID ids.CharacterID                `json:"actor_character_id"`
	CharacterID      ids.CharacterID                `json:"character_id"`
	BeastformID      string                         `json:"beastform_id"`
	Source           string                         `json:"source,omitempty"`
	ClassStateBefore *snapstate.CharacterClassState `json:"class_state_before,omitempty"`
	ClassStateAfter  *snapstate.CharacterClassState `json:"class_state_after,omitempty"`
}

// BeastformTransformedPayload captures the event payload emitted when a
// character enters beastform.
type BeastformTransformedPayload struct {
	CharacterID     ids.CharacterID                          `json:"character_id"`
	BeastformID     string                                   `json:"beastform_id"`
	Hope            *int                                     `json:"hope_after,omitempty"`
	Stress          *int                                     `json:"stress_after,omitempty"`
	ActiveBeastform *snapstate.CharacterActiveBeastformState `json:"active_beastform,omitempty"`
	Source          string                                   `json:"source,omitempty"`
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
	ActorCharacterID     ids.CharacterID                    `json:"actor_character_id"`
	CharacterID          ids.CharacterID                    `json:"character_id"`
	ExperienceID         string                             `json:"experience_id"`
	CompanionStateBefore *snapstate.CharacterCompanionState `json:"companion_state_before,omitempty"`
	CompanionStateAfter  *snapstate.CharacterCompanionState `json:"companion_state_after,omitempty"`
}

// CompanionReturnPayload captures one companion return command and the
// resulting state mutation.
type CompanionReturnPayload struct {
	ActorCharacterID     ids.CharacterID                    `json:"actor_character_id"`
	CharacterID          ids.CharacterID                    `json:"character_id"`
	Resolution           string                             `json:"resolution,omitempty"`
	StressBefore         *int                               `json:"stress_before,omitempty"`
	StressAfter          *int                               `json:"stress_after,omitempty"`
	CompanionStateBefore *snapstate.CharacterCompanionState `json:"companion_state_before,omitempty"`
	CompanionStateAfter  *snapstate.CharacterCompanionState `json:"companion_state_after,omitempty"`
}

// CompanionExperienceBegunPayload captures the event payload emitted when a
// companion leaves on an experience.
type CompanionExperienceBegunPayload struct {
	CharacterID    ids.CharacterID                    `json:"character_id"`
	ExperienceID   string                             `json:"experience_id"`
	CompanionState *snapstate.CharacterCompanionState `json:"companion_state,omitempty"`
	Source         string                             `json:"source,omitempty"`
}

// CompanionReturnedPayload captures the event payload emitted when a companion
// returns from an active experience.
type CompanionReturnedPayload struct {
	CharacterID    ids.CharacterID                    `json:"character_id"`
	Resolution     string                             `json:"resolution,omitempty"`
	Stress         *int                               `json:"stress_after,omitempty"`
	CompanionState *snapstate.CharacterCompanionState `json:"companion_state,omitempty"`
	Source         string                             `json:"source,omitempty"`
}

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

// --- Roll ---

// RollRngInfo captures RNG metadata for roll events.
type RollRngInfo struct {
	SeedUsed   uint64 `json:"seed_used"`
	RngAlgo    string `json:"rng_algo"`
	SeedSource string `json:"seed_source"`
	RollMode   string `json:"roll_mode"`
}

// --- Countdowns ---

// CountdownCreatePayload captures the payload for sys.daggerheart.countdown.create commands.
type CountdownCreatePayload struct {
	CountdownID       ids.CountdownID `json:"countdown_id"`
	Name              string          `json:"name"`
	Kind              string          `json:"kind"`
	Current           int             `json:"current"`
	Max               int             `json:"max"`
	Direction         string          `json:"direction"`
	Looping           bool            `json:"looping"`
	Variant           string          `json:"variant,omitempty"`
	TriggerEventType  string          `json:"trigger_event_type,omitempty"`
	LinkedCountdownID ids.CountdownID `json:"linked_countdown_id,omitempty"`
}

// CountdownCreatedPayload captures the payload for sys.daggerheart.countdown_created events.
type CountdownCreatedPayload = CountdownCreatePayload

// CountdownUpdatePayload captures the payload for sys.daggerheart.countdown.update commands.
type CountdownUpdatePayload struct {
	CountdownID ids.CountdownID `json:"countdown_id"`
	Before      int             `json:"before"`
	After       int             `json:"after"`
	Delta       int             `json:"delta"`
	Looped      bool            `json:"looped"`
	Reason      string          `json:"reason,omitempty"`
}

// CountdownUpdatedPayload captures the payload for sys.daggerheart.countdown_updated events.
type CountdownUpdatedPayload struct {
	CountdownID ids.CountdownID `json:"countdown_id"`
	Value       int             `json:"after"`
	Delta       int             `json:"delta"`
	Looped      bool            `json:"looped"`
	Reason      string          `json:"reason,omitempty"`
}

// CountdownDeletePayload captures the payload for sys.daggerheart.countdown.delete commands.
type CountdownDeletePayload struct {
	CountdownID ids.CountdownID `json:"countdown_id"`
	Reason      string          `json:"reason,omitempty"`
}

// CountdownDeletedPayload captures the payload for sys.daggerheart.countdown_deleted events.
type CountdownDeletedPayload = CountdownDeletePayload

// --- Damage ---

// DamageApplyPayload captures the payload for sys.daggerheart.damage.apply commands.
type DamageApplyPayload struct {
	CharacterID        ids.CharacterID   `json:"character_id"`
	HpBefore           *int              `json:"hp_before,omitempty"`
	HpAfter            *int              `json:"hp_after,omitempty"`
	StressAfter        *int              `json:"stress_after,omitempty"`
	ArmorBefore        *int              `json:"armor_before,omitempty"`
	ArmorAfter         *int              `json:"armor_after,omitempty"`
	ArmorSpent         int               `json:"armor_spent,omitempty"`
	Severity           string            `json:"severity,omitempty"`
	Marks              int               `json:"marks,omitempty"`
	DamageType         string            `json:"damage_type,omitempty"`
	RollSeq            *uint64           `json:"roll_seq,omitempty"`
	ResistPhysical     bool              `json:"resist_physical,omitempty"`
	ResistMagic        bool              `json:"resist_magic,omitempty"`
	ImmunePhysical     bool              `json:"immune_physical,omitempty"`
	ImmuneMagic        bool              `json:"immune_magic,omitempty"`
	Direct             bool              `json:"direct,omitempty"`
	MassiveDamage      bool              `json:"massive_damage,omitempty"`
	Mitigated          bool              `json:"mitigated,omitempty"`
	Source             string            `json:"source,omitempty"`
	SourceCharacterIDs []ids.CharacterID `json:"source_character_ids,omitempty"`
}

// DamageAppliedPayload captures the payload for sys.daggerheart.damage_applied events.
type DamageAppliedPayload struct {
	CharacterID        ids.CharacterID   `json:"character_id"`
	Hp                 *int              `json:"hp_after,omitempty"`
	Stress             *int              `json:"stress_after,omitempty"`
	Armor              *int              `json:"armor_after,omitempty"`
	ArmorSpent         int               `json:"armor_spent,omitempty"`
	Severity           string            `json:"severity,omitempty"`
	Marks              int               `json:"marks,omitempty"`
	DamageType         string            `json:"damage_type,omitempty"`
	RollSeq            *uint64           `json:"roll_seq,omitempty"`
	ResistPhysical     bool              `json:"resist_physical,omitempty"`
	ResistMagic        bool              `json:"resist_magic,omitempty"`
	ImmunePhysical     bool              `json:"immune_physical,omitempty"`
	ImmuneMagic        bool              `json:"immune_magic,omitempty"`
	Direct             bool              `json:"direct,omitempty"`
	MassiveDamage      bool              `json:"massive_damage,omitempty"`
	Mitigated          bool              `json:"mitigated,omitempty"`
	Source             string            `json:"source,omitempty"`
	SourceCharacterIDs []ids.CharacterID `json:"source_character_ids,omitempty"`
}

// MultiTargetDamageApplyPayload captures the payload for
// sys.daggerheart.multi_target_damage.apply commands.
type MultiTargetDamageApplyPayload struct {
	Targets []DamageApplyPayload `json:"targets"`
}

// --- Adversary damage ---

// AdversaryDamageApplyPayload captures the payload for sys.daggerheart.adversary_damage.apply commands.
type AdversaryDamageApplyPayload struct {
	AdversaryID        ids.AdversaryID   `json:"adversary_id"`
	HpBefore           *int              `json:"hp_before,omitempty"`
	HpAfter            *int              `json:"hp_after,omitempty"`
	ArmorBefore        *int              `json:"armor_before,omitempty"`
	ArmorAfter         *int              `json:"armor_after,omitempty"`
	ArmorSpent         int               `json:"armor_spent,omitempty"`
	Severity           string            `json:"severity,omitempty"`
	Marks              int               `json:"marks,omitempty"`
	DamageType         string            `json:"damage_type,omitempty"`
	RollSeq            *uint64           `json:"roll_seq,omitempty"`
	ResistPhysical     bool              `json:"resist_physical,omitempty"`
	ResistMagic        bool              `json:"resist_magic,omitempty"`
	ImmunePhysical     bool              `json:"immune_physical,omitempty"`
	ImmuneMagic        bool              `json:"immune_magic,omitempty"`
	Direct             bool              `json:"direct,omitempty"`
	MassiveDamage      bool              `json:"massive_damage,omitempty"`
	Mitigated          bool              `json:"mitigated,omitempty"`
	Source             string            `json:"source,omitempty"`
	SourceCharacterIDs []ids.CharacterID `json:"source_character_ids,omitempty"`
}

// AdversaryDamageAppliedPayload captures the payload for sys.daggerheart.adversary_damage_applied events.
type AdversaryDamageAppliedPayload struct {
	AdversaryID        ids.AdversaryID   `json:"adversary_id"`
	Hp                 *int              `json:"hp_after,omitempty"`
	Armor              *int              `json:"armor_after,omitempty"`
	ArmorSpent         int               `json:"armor_spent,omitempty"`
	Severity           string            `json:"severity,omitempty"`
	Marks              int               `json:"marks,omitempty"`
	DamageType         string            `json:"damage_type,omitempty"`
	RollSeq            *uint64           `json:"roll_seq,omitempty"`
	ResistPhysical     bool              `json:"resist_physical,omitempty"`
	ResistMagic        bool              `json:"resist_magic,omitempty"`
	ImmunePhysical     bool              `json:"immune_physical,omitempty"`
	ImmuneMagic        bool              `json:"immune_magic,omitempty"`
	Direct             bool              `json:"direct,omitempty"`
	MassiveDamage      bool              `json:"massive_damage,omitempty"`
	Mitigated          bool              `json:"mitigated,omitempty"`
	Source             string            `json:"source,omitempty"`
	SourceCharacterIDs []ids.CharacterID `json:"source_character_ids,omitempty"`
}

// --- Downtime ---

// DowntimeMoveAppliedPayload captures the payload for sys.daggerheart.downtime_move_applied events.
type DowntimeMoveAppliedPayload struct {
	ActorCharacterID  ids.CharacterID `json:"actor_character_id"`
	TargetCharacterID ids.CharacterID `json:"target_character_id,omitempty"`
	Move              string          `json:"move"`
	RestType          string          `json:"rest_type,omitempty"`
	GroupID           string          `json:"group_id,omitempty"`
	CountdownID       ids.CountdownID `json:"countdown_id,omitempty"`
	HP                *int            `json:"hp_after,omitempty"`
	Hope              *int            `json:"hope_after,omitempty"`
	Stress            *int            `json:"stress_after,omitempty"`
	Armor             *int            `json:"armor_after,omitempty"`
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

// --- Adversary CRUD ---

// AdversaryCreatePayload captures the payload for sys.daggerheart.adversary.create commands.
type AdversaryCreatePayload struct {
	AdversaryID       ids.AdversaryID                   `json:"adversary_id"`
	AdversaryEntryID  string                            `json:"adversary_entry_id"`
	Name              string                            `json:"name"`
	Kind              string                            `json:"kind,omitempty"`
	SessionID         ids.SessionID                     `json:"session_id"`
	SceneID           ids.SceneID                       `json:"scene_id"`
	Notes             string                            `json:"notes,omitempty"`
	HP                int                               `json:"hp"`
	HPMax             int                               `json:"hp_max"`
	Stress            int                               `json:"stress"`
	StressMax         int                               `json:"stress_max"`
	Evasion           int                               `json:"evasion"`
	Major             int                               `json:"major_threshold"`
	Severe            int                               `json:"severe_threshold"`
	Armor             int                               `json:"armor"`
	FeatureStates     []rules.AdversaryFeatureState     `json:"feature_states,omitempty"`
	PendingExperience *rules.AdversaryPendingExperience `json:"pending_experience,omitempty"`
	SpotlightGateID   ids.GateID                        `json:"spotlight_gate_id,omitempty"`
	SpotlightCount    int                               `json:"spotlight_count,omitempty"`
}

// AdversaryCreatedPayload captures the payload for sys.daggerheart.adversary_created events.
type AdversaryCreatedPayload = AdversaryCreatePayload

// AdversaryUpdatePayload captures the payload for sys.daggerheart.adversary.update commands.
type AdversaryUpdatePayload struct {
	AdversaryID       ids.AdversaryID                   `json:"adversary_id"`
	AdversaryEntryID  string                            `json:"adversary_entry_id"`
	Name              string                            `json:"name"`
	Kind              string                            `json:"kind,omitempty"`
	SessionID         ids.SessionID                     `json:"session_id"`
	SceneID           ids.SceneID                       `json:"scene_id"`
	Notes             string                            `json:"notes,omitempty"`
	HP                int                               `json:"hp"`
	HPMax             int                               `json:"hp_max"`
	Stress            int                               `json:"stress"`
	StressMax         int                               `json:"stress_max"`
	Evasion           int                               `json:"evasion"`
	Major             int                               `json:"major_threshold"`
	Severe            int                               `json:"severe_threshold"`
	Armor             int                               `json:"armor"`
	FeatureStates     []rules.AdversaryFeatureState     `json:"feature_states,omitempty"`
	PendingExperience *rules.AdversaryPendingExperience `json:"pending_experience,omitempty"`
	SpotlightGateID   ids.GateID                        `json:"spotlight_gate_id,omitempty"`
	SpotlightCount    int                               `json:"spotlight_count,omitempty"`
}

// AdversaryFeatureApplyPayload captures one supported adversary feature state
// mutation and the resulting adversary projection update.
type AdversaryFeatureApplyPayload struct {
	ActorAdversaryID        ids.AdversaryID                   `json:"actor_adversary_id"`
	AdversaryID             ids.AdversaryID                   `json:"adversary_id"`
	FeatureID               string                            `json:"feature_id"`
	TargetCharacterID       ids.CharacterID                   `json:"target_character_id,omitempty"`
	TargetAdversaryID       ids.AdversaryID                   `json:"target_adversary_id,omitempty"`
	StressBefore            *int                              `json:"stress_before,omitempty"`
	StressAfter             *int                              `json:"stress_after,omitempty"`
	FeatureStatesBefore     []rules.AdversaryFeatureState     `json:"feature_states_before,omitempty"`
	FeatureStatesAfter      []rules.AdversaryFeatureState     `json:"feature_states_after,omitempty"`
	PendingExperienceBefore *rules.AdversaryPendingExperience `json:"pending_experience_before,omitempty"`
	PendingExperienceAfter  *rules.AdversaryPendingExperience `json:"pending_experience_after,omitempty"`
}

// AdversaryUpdatedPayload captures the payload for sys.daggerheart.adversary_updated events.
type AdversaryUpdatedPayload = AdversaryUpdatePayload

// AdversaryDeletePayload captures the payload for sys.daggerheart.adversary.delete commands.
type AdversaryDeletePayload struct {
	AdversaryID ids.AdversaryID `json:"adversary_id"`
	Reason      string          `json:"reason,omitempty"`
}

// AdversaryDeletedPayload captures the payload for sys.daggerheart.adversary_deleted events.
type AdversaryDeletedPayload = AdversaryDeletePayload

// --- Environment entities ---

// EnvironmentEntityCreatePayload captures the payload for
// sys.daggerheart.environment_entity.create commands.
type EnvironmentEntityCreatePayload struct {
	EnvironmentEntityID ids.EnvironmentEntityID `json:"environment_entity_id"`
	EnvironmentID       string                  `json:"environment_id"`
	Name                string                  `json:"name"`
	Type                string                  `json:"type"`
	Tier                int                     `json:"tier"`
	Difficulty          int                     `json:"difficulty"`
	SessionID           ids.SessionID           `json:"session_id"`
	SceneID             ids.SceneID             `json:"scene_id"`
	Notes               string                  `json:"notes,omitempty"`
}

// EnvironmentEntityCreatedPayload captures the payload for
// sys.daggerheart.environment_entity_created events.
type EnvironmentEntityCreatedPayload = EnvironmentEntityCreatePayload

// EnvironmentEntityUpdatePayload captures the payload for
// sys.daggerheart.environment_entity.update commands.
type EnvironmentEntityUpdatePayload struct {
	EnvironmentEntityID ids.EnvironmentEntityID `json:"environment_entity_id"`
	EnvironmentID       string                  `json:"environment_id"`
	Name                string                  `json:"name"`
	Type                string                  `json:"type"`
	Tier                int                     `json:"tier"`
	Difficulty          int                     `json:"difficulty"`
	SessionID           ids.SessionID           `json:"session_id"`
	SceneID             ids.SceneID             `json:"scene_id"`
	Notes               string                  `json:"notes,omitempty"`
}

// EnvironmentEntityUpdatedPayload captures the payload for
// sys.daggerheart.environment_entity_updated events.
type EnvironmentEntityUpdatedPayload = EnvironmentEntityUpdatePayload

// EnvironmentEntityDeletePayload captures the payload for
// sys.daggerheart.environment_entity.delete commands.
type EnvironmentEntityDeletePayload struct {
	EnvironmentEntityID ids.EnvironmentEntityID `json:"environment_entity_id"`
	Reason              string                  `json:"reason,omitempty"`
}

// EnvironmentEntityDeletedPayload captures the payload for
// sys.daggerheart.environment_entity_deleted events.
type EnvironmentEntityDeletedPayload = EnvironmentEntityDeletePayload

// --- Level up ---

// LevelUpApplyPayload captures the payload for sys.daggerheart.level_up.apply commands.
type LevelUpApplyPayload struct {
	CharacterID                  ids.CharacterID                    `json:"character_id"`
	LevelBefore                  int                                `json:"level_before"`
	LevelAfter                   int                                `json:"level_after"`
	Advancements                 []LevelUpAdvancementPayload        `json:"advancements"`
	Rewards                      []LevelUpRewardPayload             `json:"rewards,omitempty"`
	MarkedTraits                 []string                           `json:"marked_traits,omitempty"`
	SubclassTracksAfter          []snapstate.CharacterSubclassTrack `json:"subclass_tracks_after,omitempty"`
	SubclassHpMaxDelta           int                                `json:"subclass_hp_max_delta,omitempty"`
	SubclassStressMaxDelta       int                                `json:"subclass_stress_max_delta,omitempty"`
	SubclassEvasionDelta         int                                `json:"subclass_evasion_delta,omitempty"`
	SubclassMajorThresholdDelta  int                                `json:"subclass_major_threshold_delta,omitempty"`
	SubclassSevereThresholdDelta int                                `json:"subclass_severe_threshold_delta,omitempty"`
	Tier                         int                                `json:"tier"`
	PreviousTier                 int                                `json:"previous_tier"`
	IsTierEntry                  bool                               `json:"is_tier_entry"`
	ClearMarks                   bool                               `json:"clear_marks"`
	MarkedAfter                  []string                           `json:"marked_after,omitempty"`
	ThresholdDelta               int                                `json:"threshold_delta"`
}

// LevelUpAdvancementPayload represents a single advancement choice.
type LevelUpAdvancementPayload struct {
	Type            string                    `json:"type"`
	Trait           string                    `json:"trait,omitempty"`
	DomainCardID    string                    `json:"domain_card_id,omitempty"`
	DomainCardLevel int                       `json:"domain_card_level,omitempty"`
	Multiclass      *LevelUpMulticlassPayload `json:"multiclass,omitempty"`
}

// LevelUpRewardPayload represents one non-budget reward granted during level-up.
type LevelUpRewardPayload struct {
	Type                  string `json:"type"`
	DomainCardID          string `json:"domain_card_id,omitempty"`
	DomainCardLevel       int    `json:"domain_card_level,omitempty"`
	CompanionBonusChoices int    `json:"companion_bonus_choices,omitempty"`
}

// LevelUpMulticlassPayload captures multiclass advancement choices.
type LevelUpMulticlassPayload struct {
	SecondaryClassID    string `json:"secondary_class_id"`
	SecondarySubclassID string `json:"secondary_subclass_id"`
	SpellcastTrait      string `json:"spellcast_trait"`
	DomainID            string `json:"domain_id"`
}

// LevelUpAppliedPayload captures the payload for sys.daggerheart.level_up_applied events.
type LevelUpAppliedPayload struct {
	CharacterID                  ids.CharacterID                    `json:"character_id"`
	Level                        int                                `json:"level_after"`
	Advancements                 []LevelUpAdvancementPayload        `json:"advancements"`
	Rewards                      []LevelUpRewardPayload             `json:"rewards,omitempty"`
	SubclassTracksAfter          []snapstate.CharacterSubclassTrack `json:"subclass_tracks_after,omitempty"`
	SubclassHpMaxDelta           int                                `json:"subclass_hp_max_delta,omitempty"`
	SubclassStressMaxDelta       int                                `json:"subclass_stress_max_delta,omitempty"`
	SubclassEvasionDelta         int                                `json:"subclass_evasion_delta,omitempty"`
	SubclassMajorThresholdDelta  int                                `json:"subclass_major_threshold_delta,omitempty"`
	SubclassSevereThresholdDelta int                                `json:"subclass_severe_threshold_delta,omitempty"`
	Tier                         int                                `json:"tier"`
	IsTierEntry                  bool                               `json:"is_tier_entry"`
	ClearMarks                   bool                               `json:"clear_marks"`
	Marked                       []string                           `json:"marked_after,omitempty"`
	ThresholdDelta               int                                `json:"threshold_delta"`
}

// --- Gold ---

// GoldUpdatePayload captures the payload for sys.daggerheart.gold.update commands.
type GoldUpdatePayload struct {
	CharacterID    ids.CharacterID `json:"character_id"`
	HandfulsBefore int             `json:"handfuls_before"`
	HandfulsAfter  int             `json:"handfuls_after"`
	BagsBefore     int             `json:"bags_before"`
	BagsAfter      int             `json:"bags_after"`
	ChestsBefore   int             `json:"chests_before"`
	ChestsAfter    int             `json:"chests_after"`
	Reason         string          `json:"reason,omitempty"`
}

// GoldUpdatedPayload captures the payload for sys.daggerheart.gold_updated events.
type GoldUpdatedPayload struct {
	CharacterID ids.CharacterID `json:"character_id"`
	Handfuls    int             `json:"handfuls_after"`
	Bags        int             `json:"bags_after"`
	Chests      int             `json:"chests_after"`
	Reason      string          `json:"reason,omitempty"`
}

// --- Domain card ---

// DomainCardAcquirePayload captures the payload for sys.daggerheart.domain_card.acquire commands.
type DomainCardAcquirePayload struct {
	CharacterID ids.CharacterID `json:"character_id"`
	CardID      string          `json:"card_id"`
	CardLevel   int             `json:"card_level"`
	Destination string          `json:"destination"`
}

// DomainCardAcquiredPayload captures the payload for sys.daggerheart.domain_card_acquired events.
type DomainCardAcquiredPayload = DomainCardAcquirePayload

// --- Equipment ---

// EquipmentSwapPayload captures the payload for sys.daggerheart.equipment.swap commands.
type EquipmentSwapPayload struct {
	CharacterID             ids.CharacterID `json:"character_id"`
	ItemID                  string          `json:"item_id"`
	ItemType                string          `json:"item_type"`
	From                    string          `json:"from"`
	To                      string          `json:"to"`
	StressCost              int             `json:"stress_cost,omitempty"`
	EquippedArmorID         string          `json:"equipped_armor_id,omitempty"`
	EvasionAfter            *int            `json:"evasion_after,omitempty"`
	MajorThresholdAfter     *int            `json:"major_threshold_after,omitempty"`
	SevereThresholdAfter    *int            `json:"severe_threshold_after,omitempty"`
	ArmorScoreAfter         *int            `json:"armor_score_after,omitempty"`
	ArmorMaxAfter           *int            `json:"armor_max_after,omitempty"`
	SpellcastRollBonusAfter *int            `json:"spellcast_roll_bonus_after,omitempty"`
	AgilityAfter            *int            `json:"agility_after,omitempty"`
	StrengthAfter           *int            `json:"strength_after,omitempty"`
	FinesseAfter            *int            `json:"finesse_after,omitempty"`
	InstinctAfter           *int            `json:"instinct_after,omitempty"`
	PresenceAfter           *int            `json:"presence_after,omitempty"`
	KnowledgeAfter          *int            `json:"knowledge_after,omitempty"`
	ArmorAfter              *int            `json:"armor_after,omitempty"`
}

// EquipmentSwappedPayload captures the payload for sys.daggerheart.equipment_swapped events.
type EquipmentSwappedPayload = EquipmentSwapPayload

// --- Consumables ---

// ConsumableUsePayload captures the payload for sys.daggerheart.consumable.use commands.
type ConsumableUsePayload struct {
	CharacterID    ids.CharacterID `json:"character_id"`
	ConsumableID   string          `json:"consumable_id"`
	QuantityBefore int             `json:"quantity_before"`
	QuantityAfter  int             `json:"quantity_after"`
}

// ConsumableUsedPayload captures the payload for sys.daggerheart.consumable_used events.
type ConsumableUsedPayload struct {
	CharacterID  ids.CharacterID `json:"character_id"`
	ConsumableID string          `json:"consumable_id"`
	Quantity     int             `json:"quantity_after"`
}

// ConsumableAcquirePayload captures the payload for sys.daggerheart.consumable.acquire commands.
type ConsumableAcquirePayload struct {
	CharacterID    ids.CharacterID `json:"character_id"`
	ConsumableID   string          `json:"consumable_id"`
	QuantityBefore int             `json:"quantity_before"`
	QuantityAfter  int             `json:"quantity_after"`
}

// ConsumableAcquiredPayload captures the payload for sys.daggerheart.consumable_acquired events.
type ConsumableAcquiredPayload struct {
	CharacterID  ids.CharacterID `json:"character_id"`
	ConsumableID string          `json:"consumable_id"`
	Quantity     int             `json:"quantity_after"`
}
