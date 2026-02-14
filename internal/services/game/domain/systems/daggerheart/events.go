package daggerheart

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/core/dice"
)

const (
	EventTypeDamageApplied             event.Type = "action.damage_applied"
	EventTypeRestTaken                 event.Type = "action.rest_taken"
	EventTypeDowntimeMoveApplied       event.Type = "action.downtime_move_applied"
	EventTypeLoadoutSwapped            event.Type = "action.loadout_swapped"
	EventTypeCharacterStatePatched     event.Type = "action.character_state_patched"
	EventTypeConditionChanged          event.Type = "action.condition_changed"
	EventTypeGMFearChanged             event.Type = "action.gm_fear_changed"
	EventTypeGMMoveApplied             event.Type = "action.gm_move_applied"
	EventTypeHopeSpent                 event.Type = "action.hope_spent"
	EventTypeStressSpent               event.Type = "action.stress_spent"
	EventTypeDeathMoveResolved         event.Type = "action.death_move_resolved"
	EventTypeBlazeOfGloryResolved      event.Type = "action.blaze_of_glory_resolved"
	EventTypeAttackResolved            event.Type = "action.attack_resolved"
	EventTypeReactionResolved          event.Type = "action.reaction_resolved"
	EventTypeDamageRollResolved        event.Type = "action.damage_roll_resolved"
	EventTypeGroupActionResolved       event.Type = "action.group_action_resolved"
	EventTypeTagTeamResolved           event.Type = "action.tag_team_resolved"
	EventTypeCountdownCreated          event.Type = "action.countdown_created"
	EventTypeCountdownUpdated          event.Type = "action.countdown_updated"
	EventTypeCountdownDeleted          event.Type = "action.countdown_deleted"
	EventTypeAdversaryRollResolved     event.Type = "action.adversary_roll_resolved"
	EventTypeAdversaryActionResolved   event.Type = "action.adversary_action_resolved"
	EventTypeAdversaryAttackResolved   event.Type = "action.adversary_attack_resolved"
	EventTypeAdversaryCreated          event.Type = "action.adversary_created"
	EventTypeAdversaryConditionChanged event.Type = "action.adversary_condition_changed"
	EventTypeAdversaryDamageApplied    event.Type = "action.adversary_damage_applied"
	EventTypeAdversaryUpdated          event.Type = "action.adversary_updated"
	EventTypeAdversaryDeleted          event.Type = "action.adversary_deleted"
)

// DamageAppliedPayload captures the payload for action.damage_applied events.
type DamageAppliedPayload struct {
	CharacterID        string   `json:"character_id"`
	HpBefore           *int     `json:"hp_before,omitempty"`
	HpAfter            *int     `json:"hp_after,omitempty"`
	ArmorBefore        *int     `json:"armor_before,omitempty"`
	ArmorAfter         *int     `json:"armor_after,omitempty"`
	ArmorSpent         int      `json:"armor_spent,omitempty"`
	Severity           string   `json:"severity,omitempty"`
	Marks              int      `json:"marks,omitempty"`
	DamageType         string   `json:"damage_type,omitempty"`
	RollSeq            *uint64  `json:"roll_seq,omitempty"`
	ResistPhysical     bool     `json:"resist_physical,omitempty"`
	ResistMagic        bool     `json:"resist_magic,omitempty"`
	ImmunePhysical     bool     `json:"immune_physical,omitempty"`
	ImmuneMagic        bool     `json:"immune_magic,omitempty"`
	Direct             bool     `json:"direct,omitempty"`
	MassiveDamage      bool     `json:"massive_damage,omitempty"`
	Mitigated          bool     `json:"mitigated,omitempty"`
	Source             string   `json:"source,omitempty"`
	SourceCharacterIDs []string `json:"source_character_ids,omitempty"`
}

// RestTakenPayload captures the payload for action.rest_taken events.
type RestTakenPayload struct {
	RestType         string                    `json:"rest_type"`
	Interrupted      bool                      `json:"interrupted"`
	GMFearBefore     int                       `json:"gm_fear_before"`
	GMFearAfter      int                       `json:"gm_fear_after"`
	ShortRestsBefore int                       `json:"short_rests_before"`
	ShortRestsAfter  int                       `json:"short_rests_after"`
	RefreshRest      bool                      `json:"refresh_rest"`
	RefreshLongRest  bool                      `json:"refresh_long_rest"`
	CharacterStates  []RestCharacterStatePatch `json:"character_states,omitempty"`
}

// RestCharacterStatePatch describes per-character rest adjustments.
type RestCharacterStatePatch struct {
	CharacterID  string `json:"character_id"`
	HopeBefore   *int   `json:"hope_before,omitempty"`
	HopeAfter    *int   `json:"hope_after,omitempty"`
	StressBefore *int   `json:"stress_before,omitempty"`
	StressAfter  *int   `json:"stress_after,omitempty"`
	ArmorBefore  *int   `json:"armor_before,omitempty"`
	ArmorAfter   *int   `json:"armor_after,omitempty"`
}

// DowntimeMoveAppliedPayload captures the payload for action.downtime_move_applied events.
type DowntimeMoveAppliedPayload struct {
	CharacterID  string `json:"character_id"`
	Move         string `json:"move"`
	HopeBefore   *int   `json:"hope_before,omitempty"`
	HopeAfter    *int   `json:"hope_after,omitempty"`
	StressBefore *int   `json:"stress_before,omitempty"`
	StressAfter  *int   `json:"stress_after,omitempty"`
	ArmorBefore  *int   `json:"armor_before,omitempty"`
	ArmorAfter   *int   `json:"armor_after,omitempty"`
}

// LoadoutSwappedPayload captures the payload for action.loadout_swapped events.
type LoadoutSwappedPayload struct {
	CharacterID  string `json:"character_id"`
	CardID       string `json:"card_id"`
	From         string `json:"from"`
	To           string `json:"to"`
	RecallCost   int    `json:"recall_cost,omitempty"`
	StressBefore *int   `json:"stress_before,omitempty"`
	StressAfter  *int   `json:"stress_after,omitempty"`
}

// CharacterStatePatchedPayload captures the payload for action.character_state_patched events.
type CharacterStatePatchedPayload struct {
	CharacterID     string  `json:"character_id"`
	HpBefore        *int    `json:"hp_before,omitempty"`
	HpAfter         *int    `json:"hp_after,omitempty"`
	HopeBefore      *int    `json:"hope_before,omitempty"`
	HopeAfter       *int    `json:"hope_after,omitempty"`
	HopeMaxBefore   *int    `json:"hope_max_before,omitempty"`
	HopeMaxAfter    *int    `json:"hope_max_after,omitempty"`
	StressBefore    *int    `json:"stress_before,omitempty"`
	StressAfter     *int    `json:"stress_after,omitempty"`
	ArmorBefore     *int    `json:"armor_before,omitempty"`
	ArmorAfter      *int    `json:"armor_after,omitempty"`
	LifeStateBefore *string `json:"life_state_before,omitempty"`
	LifeStateAfter  *string `json:"life_state_after,omitempty"`
}

// ConditionChangedPayload captures the payload for action.condition_changed events.
type ConditionChangedPayload struct {
	CharacterID      string   `json:"character_id"`
	ConditionsBefore []string `json:"conditions_before,omitempty"`
	ConditionsAfter  []string `json:"conditions_after"`
	Added            []string `json:"added,omitempty"`
	Removed          []string `json:"removed,omitempty"`
	Source           string   `json:"source,omitempty"`
	RollSeq          *uint64  `json:"roll_seq,omitempty"`
}

// AdversaryConditionChangedPayload captures the payload for action.adversary_condition_changed events.
type AdversaryConditionChangedPayload struct {
	AdversaryID      string   `json:"adversary_id"`
	ConditionsBefore []string `json:"conditions_before,omitempty"`
	ConditionsAfter  []string `json:"conditions_after"`
	Added            []string `json:"added,omitempty"`
	Removed          []string `json:"removed,omitempty"`
	Source           string   `json:"source,omitempty"`
	RollSeq          *uint64  `json:"roll_seq,omitempty"`
}

// GMFearChangedPayload captures the payload for action.gm_fear_changed events.
type GMFearChangedPayload struct {
	Before int    `json:"before"`
	After  int    `json:"after"`
	Reason string `json:"reason,omitempty"`
}

// GMMoveAppliedPayload captures the payload for action.gm_move_applied events.
type GMMoveAppliedPayload struct {
	Move        string `json:"move"`
	Description string `json:"description,omitempty"`
	FearSpent   int    `json:"fear_spent,omitempty"`
	Severity    string `json:"severity,omitempty"`
	Source      string `json:"source,omitempty"`
}

// AdversaryDamageAppliedPayload captures the payload for action.adversary_damage_applied events.
type AdversaryDamageAppliedPayload struct {
	AdversaryID        string   `json:"adversary_id"`
	HpBefore           *int     `json:"hp_before,omitempty"`
	HpAfter            *int     `json:"hp_after,omitempty"`
	ArmorBefore        *int     `json:"armor_before,omitempty"`
	ArmorAfter         *int     `json:"armor_after,omitempty"`
	ArmorSpent         int      `json:"armor_spent,omitempty"`
	Severity           string   `json:"severity,omitempty"`
	Marks              int      `json:"marks,omitempty"`
	DamageType         string   `json:"damage_type,omitempty"`
	RollSeq            *uint64  `json:"roll_seq,omitempty"`
	ResistPhysical     bool     `json:"resist_physical,omitempty"`
	ResistMagic        bool     `json:"resist_magic,omitempty"`
	ImmunePhysical     bool     `json:"immune_physical,omitempty"`
	ImmuneMagic        bool     `json:"immune_magic,omitempty"`
	Direct             bool     `json:"direct,omitempty"`
	MassiveDamage      bool     `json:"massive_damage,omitempty"`
	Mitigated          bool     `json:"mitigated,omitempty"`
	Source             string   `json:"source,omitempty"`
	SourceCharacterIDs []string `json:"source_character_ids,omitempty"`
}

// HopeSpentPayload captures the payload for action.hope_spent events.
type HopeSpentPayload struct {
	CharacterID string  `json:"character_id"`
	Amount      int     `json:"amount"`
	Before      int     `json:"before"`
	After       int     `json:"after"`
	RollSeq     *uint64 `json:"roll_seq,omitempty"`
	Source      string  `json:"source,omitempty"`
}

// StressSpentPayload captures the payload for action.stress_spent events.
type StressSpentPayload struct {
	CharacterID string  `json:"character_id"`
	Amount      int     `json:"amount"`
	Before      int     `json:"before"`
	After       int     `json:"after"`
	RollSeq     *uint64 `json:"roll_seq,omitempty"`
	Source      string  `json:"source,omitempty"`
}

// DeathMoveResolvedPayload captures the payload for action.death_move_resolved events.
type DeathMoveResolvedPayload struct {
	CharacterID     string  `json:"character_id"`
	Move            string  `json:"move"`
	LifeStateBefore *string `json:"life_state_before,omitempty"`
	LifeStateAfter  string  `json:"life_state_after"`
	HpBefore        *int    `json:"hp_before,omitempty"`
	HpAfter         *int    `json:"hp_after,omitempty"`
	HopeBefore      *int    `json:"hope_before,omitempty"`
	HopeAfter       *int    `json:"hope_after,omitempty"`
	HopeMaxBefore   *int    `json:"hope_max_before,omitempty"`
	HopeMaxAfter    *int    `json:"hope_max_after,omitempty"`
	StressBefore    *int    `json:"stress_before,omitempty"`
	StressAfter     *int    `json:"stress_after,omitempty"`
	HopeDie         *int    `json:"hope_die,omitempty"`
	FearDie         *int    `json:"fear_die,omitempty"`
	ScarGained      bool    `json:"scar_gained,omitempty"`
	HPCleared       int     `json:"hp_cleared,omitempty"`
	StressCleared   int     `json:"stress_cleared,omitempty"`
}

// BlazeOfGloryResolvedPayload captures the payload for action.blaze_of_glory_resolved events.
type BlazeOfGloryResolvedPayload struct {
	CharacterID     string  `json:"character_id"`
	LifeStateBefore *string `json:"life_state_before,omitempty"`
	LifeStateAfter  string  `json:"life_state_after"`
}

// AttackResolvedPayload captures the payload for action.attack_resolved events.
type AttackResolvedPayload struct {
	CharacterID string   `json:"character_id"`
	RollSeq     uint64   `json:"roll_seq"`
	Targets     []string `json:"targets"`
	Outcome     string   `json:"outcome"`
	Success     bool     `json:"success"`
	Crit        bool     `json:"crit"`
	Flavor      string   `json:"flavor,omitempty"`
}

// ReactionResolvedPayload captures the payload for action.reaction_resolved events.
type ReactionResolvedPayload struct {
	CharacterID        string `json:"character_id"`
	RollSeq            uint64 `json:"roll_seq"`
	Outcome            string `json:"outcome"`
	Success            bool   `json:"success"`
	Crit               bool   `json:"crit"`
	CritNegatesEffects bool   `json:"crit_negates_effects"`
	EffectsNegated     bool   `json:"effects_negated"`
}

// GroupActionSupporterRoll captures the supporter roll details for group action resolution.
type GroupActionSupporterRoll struct {
	CharacterID string `json:"character_id"`
	RollSeq     uint64 `json:"roll_seq"`
	Success     bool   `json:"success"`
}

// GroupActionResolvedPayload captures the payload for action.group_action_resolved events.
type GroupActionResolvedPayload struct {
	LeaderCharacterID string                     `json:"leader_character_id"`
	LeaderRollSeq     uint64                     `json:"leader_roll_seq"`
	Supporters        []GroupActionSupporterRoll `json:"supporters"`
	SupportSuccesses  int                        `json:"support_successes"`
	SupportFailures   int                        `json:"support_failures"`
	SupportModifier   int                        `json:"support_modifier"`
}

// TagTeamResolvedPayload captures the payload for action.tag_team_resolved events.
type TagTeamResolvedPayload struct {
	FirstCharacterID    string `json:"first_character_id"`
	FirstRollSeq        uint64 `json:"first_roll_seq"`
	SecondCharacterID   string `json:"second_character_id"`
	SecondRollSeq       uint64 `json:"second_roll_seq"`
	SelectedCharacterID string `json:"selected_character_id"`
	SelectedRollSeq     uint64 `json:"selected_roll_seq"`
}

// CountdownCreatedPayload captures the payload for action.countdown_created events.
type CountdownCreatedPayload struct {
	CountdownID string `json:"countdown_id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Current     int    `json:"current"`
	Max         int    `json:"max"`
	Direction   string `json:"direction"`
	Looping     bool   `json:"looping"`
}

// CountdownUpdatedPayload captures the payload for action.countdown_updated events.
type CountdownUpdatedPayload struct {
	CountdownID string `json:"countdown_id"`
	Before      int    `json:"before"`
	After       int    `json:"after"`
	Delta       int    `json:"delta"`
	Looped      bool   `json:"looped"`
	Reason      string `json:"reason,omitempty"`
}

// CountdownDeletedPayload captures the payload for action.countdown_deleted events.
type CountdownDeletedPayload struct {
	CountdownID string `json:"countdown_id"`
	Reason      string `json:"reason,omitempty"`
}

// AdversaryRollResolvedPayload captures the payload for action.adversary_roll_resolved events.
type AdversaryRollResolvedPayload struct {
	AdversaryID  string `json:"adversary_id"`
	RollSeq      uint64 `json:"roll_seq"`
	Rolls        []int  `json:"rolls"`
	Roll         int    `json:"roll"`
	Modifier     int    `json:"modifier"`
	Total        int    `json:"total"`
	Advantage    int    `json:"advantage,omitempty"`
	Disadvantage int    `json:"disadvantage,omitempty"`
}

// AdversaryActionResolvedPayload captures the payload for action.adversary_action_resolved events.
type AdversaryActionResolvedPayload struct {
	AdversaryID string       `json:"adversary_id"`
	RollSeq     uint64       `json:"roll_seq"`
	Difficulty  int          `json:"difficulty"`
	Dramatic    bool         `json:"dramatic"`
	AutoSuccess bool         `json:"auto_success"`
	Roll        int          `json:"roll,omitempty"`
	Modifier    int          `json:"modifier,omitempty"`
	Total       int          `json:"total,omitempty"`
	Success     bool         `json:"success"`
	Rng         *RollRngInfo `json:"rng,omitempty"`
}

// AdversaryAttackResolvedPayload captures the payload for action.adversary_attack_resolved events.
type AdversaryAttackResolvedPayload struct {
	AdversaryID string   `json:"adversary_id"`
	RollSeq     uint64   `json:"roll_seq"`
	Targets     []string `json:"targets"`
	Roll        int      `json:"roll"`
	Modifier    int      `json:"modifier"`
	Total       int      `json:"total"`
	Difficulty  int      `json:"difficulty"`
	Success     bool     `json:"success"`
	Crit        bool     `json:"crit"`
}

// AdversaryCreatedPayload captures the payload for action.adversary_created events.
type AdversaryCreatedPayload struct {
	AdversaryID string `json:"adversary_id"`
	Name        string `json:"name"`
	Kind        string `json:"kind,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	Notes       string `json:"notes,omitempty"`
	HP          int    `json:"hp"`
	HPMax       int    `json:"hp_max"`
	Stress      int    `json:"stress"`
	StressMax   int    `json:"stress_max"`
	Evasion     int    `json:"evasion"`
	Major       int    `json:"major_threshold"`
	Severe      int    `json:"severe_threshold"`
	Armor       int    `json:"armor"`
}

// AdversaryUpdatedPayload captures the payload for action.adversary_updated events.
type AdversaryUpdatedPayload struct {
	AdversaryID string `json:"adversary_id"`
	Name        string `json:"name"`
	Kind        string `json:"kind,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	Notes       string `json:"notes,omitempty"`
	HP          int    `json:"hp"`
	HPMax       int    `json:"hp_max"`
	Stress      int    `json:"stress"`
	StressMax   int    `json:"stress_max"`
	Evasion     int    `json:"evasion"`
	Major       int    `json:"major_threshold"`
	Severe      int    `json:"severe_threshold"`
	Armor       int    `json:"armor"`
}

// AdversaryDeletedPayload captures the payload for action.adversary_deleted events.
type AdversaryDeletedPayload struct {
	AdversaryID string `json:"adversary_id"`
	Reason      string `json:"reason,omitempty"`
}

// RollRngInfo captures RNG metadata for roll events.
type RollRngInfo struct {
	SeedUsed   uint64 `json:"seed_used"`
	RngAlgo    string `json:"rng_algo"`
	SeedSource string `json:"seed_source"`
	RollMode   string `json:"roll_mode"`
}

// DamageRollResolvedPayload captures the payload for action.damage_roll_resolved events.
type DamageRollResolvedPayload struct {
	CharacterID   string      `json:"character_id"`
	RollSeq       uint64      `json:"roll_seq"`
	Rolls         []dice.Roll `json:"rolls"`
	BaseTotal     int         `json:"base_total"`
	Modifier      int         `json:"modifier"`
	CriticalBonus int         `json:"critical_bonus"`
	Total         int         `json:"total"`
	Critical      bool        `json:"critical"`
	Rng           RollRngInfo `json:"rng"`
}
