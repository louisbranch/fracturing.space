package daggerheart

import "github.com/louisbranch/fracturing.space/internal/services/game/core/dice"

// GMFearSetPayload captures the payload for sys.daggerheart.action.gm_fear.set commands.
type GMFearSetPayload struct {
	After  *int   `json:"after"`
	Reason string `json:"reason,omitempty"`
}

// GMFearChangedPayload captures the payload for sys.daggerheart.action.gm_fear_changed events.
type GMFearChangedPayload struct {
	Before int    `json:"before"`
	After  int    `json:"after"`
	Reason string `json:"reason,omitempty"`
}

// CharacterStatePatchPayload captures the payload for sys.daggerheart.action.character_state.patch commands.
type CharacterStatePatchPayload struct {
	CharacterID     string  `json:"character_id"`
	HPBefore        *int    `json:"hp_before,omitempty"`
	HPAfter         *int    `json:"hp_after,omitempty"`
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

// CharacterStatePatchedPayload captures the payload for sys.daggerheart.action.character_state_patched events.
type CharacterStatePatchedPayload = CharacterStatePatchPayload

// ConditionChangePayload captures the payload for sys.daggerheart.action.condition.change commands.
type ConditionChangePayload struct {
	CharacterID      string   `json:"character_id"`
	ConditionsBefore []string `json:"conditions_before,omitempty"`
	ConditionsAfter  []string `json:"conditions_after"`
	Added            []string `json:"added,omitempty"`
	Removed          []string `json:"removed,omitempty"`
	Source           string   `json:"source,omitempty"`
	RollSeq          *uint64  `json:"roll_seq,omitempty"`
}

// ConditionChangedPayload captures the payload for sys.daggerheart.action.condition_changed events.
type ConditionChangedPayload = ConditionChangePayload

// HopeSpendPayload captures the payload for sys.daggerheart.action.hope.spend commands.
type HopeSpendPayload struct {
	CharacterID string  `json:"character_id"`
	Amount      int     `json:"amount"`
	Before      int     `json:"before"`
	After       int     `json:"after"`
	RollSeq     *uint64 `json:"roll_seq,omitempty"`
	Source      string  `json:"source,omitempty"`
}

// HopeSpentPayload captures the payload for sys.daggerheart.action.hope_spent events.
type HopeSpentPayload = HopeSpendPayload

// StressSpendPayload captures the payload for sys.daggerheart.action.stress.spend commands.
type StressSpendPayload struct {
	CharacterID string  `json:"character_id"`
	Amount      int     `json:"amount"`
	Before      int     `json:"before"`
	After       int     `json:"after"`
	RollSeq     *uint64 `json:"roll_seq,omitempty"`
	Source      string  `json:"source,omitempty"`
}

// StressSpentPayload captures the payload for sys.daggerheart.action.stress_spent events.
type StressSpentPayload = StressSpendPayload

// LoadoutSwapPayload captures the payload for sys.daggerheart.action.loadout.swap commands.
type LoadoutSwapPayload struct {
	CharacterID  string `json:"character_id"`
	CardID       string `json:"card_id"`
	From         string `json:"from"`
	To           string `json:"to"`
	RecallCost   int    `json:"recall_cost,omitempty"`
	StressBefore *int   `json:"stress_before,omitempty"`
	StressAfter  *int   `json:"stress_after,omitempty"`
}

// LoadoutSwappedPayload captures the payload for sys.daggerheart.action.loadout_swapped events.
type LoadoutSwappedPayload = LoadoutSwapPayload

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

// RestTakePayload captures the payload for sys.daggerheart.action.rest.take commands.
type RestTakePayload struct {
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

// RestTakenPayload captures the payload for sys.daggerheart.action.rest_taken events.
type RestTakenPayload = RestTakePayload

// AttackResolvePayload captures the payload for sys.daggerheart.action.attack.resolve commands.
type AttackResolvePayload struct {
	CharacterID string   `json:"character_id"`
	RollSeq     uint64   `json:"roll_seq"`
	Targets     []string `json:"targets"`
	Outcome     string   `json:"outcome"`
	Success     bool     `json:"success"`
	Crit        bool     `json:"crit"`
	Flavor      string   `json:"flavor,omitempty"`
}

// AttackResolvedPayload captures the payload for sys.daggerheart.action.attack_resolved events.
type AttackResolvedPayload = AttackResolvePayload

// ReactionResolvePayload captures the payload for sys.daggerheart.action.reaction.resolve commands.
type ReactionResolvePayload struct {
	CharacterID        string `json:"character_id"`
	RollSeq            uint64 `json:"roll_seq"`
	Outcome            string `json:"outcome"`
	Success            bool   `json:"success"`
	Crit               bool   `json:"crit"`
	CritNegatesEffects bool   `json:"crit_negates_effects"`
	EffectsNegated     bool   `json:"effects_negated"`
}

// ReactionResolvedPayload captures the payload for sys.daggerheart.action.reaction_resolved events.
type ReactionResolvedPayload = ReactionResolvePayload

// AdversaryRollResolvePayload captures the payload for sys.daggerheart.action.adversary_roll.resolve commands.
type AdversaryRollResolvePayload struct {
	AdversaryID  string `json:"adversary_id"`
	RollSeq      uint64 `json:"roll_seq"`
	Rolls        []int  `json:"rolls"`
	Roll         int    `json:"roll"`
	Modifier     int    `json:"modifier"`
	Total        int    `json:"total"`
	Advantage    int    `json:"advantage,omitempty"`
	Disadvantage int    `json:"disadvantage,omitempty"`
}

// AdversaryRollResolvedPayload captures the payload for sys.daggerheart.action.adversary_roll_resolved events.
type AdversaryRollResolvedPayload = AdversaryRollResolvePayload

// AdversaryAttackResolvePayload captures the payload for sys.daggerheart.action.adversary_attack.resolve commands.
type AdversaryAttackResolvePayload struct {
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

// AdversaryAttackResolvedPayload captures the payload for sys.daggerheart.action.adversary_attack_resolved events.
type AdversaryAttackResolvedPayload = AdversaryAttackResolvePayload

// AdversaryActionResolvePayload captures the payload for sys.daggerheart.action.adversary_action.resolve commands.
type AdversaryActionResolvePayload struct {
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

// AdversaryActionResolvedPayload captures the payload for sys.daggerheart.action.adversary_action_resolved events.
type AdversaryActionResolvedPayload = AdversaryActionResolvePayload

// RollRngInfo captures RNG metadata for roll events.
type RollRngInfo struct {
	SeedUsed   uint64 `json:"seed_used"`
	RngAlgo    string `json:"rng_algo"`
	SeedSource string `json:"seed_source"`
	RollMode   string `json:"roll_mode"`
}

// DamageRollResolvePayload captures the payload for sys.daggerheart.action.damage_roll.resolve commands.
type DamageRollResolvePayload struct {
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

// DamageRollResolvedPayload captures the payload for sys.daggerheart.action.damage_roll_resolved events.
type DamageRollResolvedPayload = DamageRollResolvePayload

// GroupActionSupporterRoll captures the supporter roll details for group action resolution.
type GroupActionSupporterRoll struct {
	CharacterID string `json:"character_id"`
	RollSeq     uint64 `json:"roll_seq"`
	Success     bool   `json:"success"`
}

// GroupActionResolvePayload captures the payload for sys.daggerheart.action.group_action.resolve commands.
type GroupActionResolvePayload struct {
	LeaderCharacterID string                     `json:"leader_character_id"`
	LeaderRollSeq     uint64                     `json:"leader_roll_seq"`
	Supporters        []GroupActionSupporterRoll `json:"supporters,omitempty"`
	SupportSuccesses  int                        `json:"support_successes"`
	SupportFailures   int                        `json:"support_failures"`
	SupportModifier   int                        `json:"support_modifier"`
}

// GroupActionResolvedPayload captures the payload for sys.daggerheart.action.group_action_resolved events.
type GroupActionResolvedPayload = GroupActionResolvePayload

// TagTeamResolvePayload captures the payload for sys.daggerheart.action.tag_team.resolve commands.
type TagTeamResolvePayload struct {
	FirstCharacterID    string `json:"first_character_id"`
	FirstRollSeq        uint64 `json:"first_roll_seq"`
	SecondCharacterID   string `json:"second_character_id"`
	SecondRollSeq       uint64 `json:"second_roll_seq"`
	SelectedCharacterID string `json:"selected_character_id"`
	SelectedRollSeq     uint64 `json:"selected_roll_seq"`
}

// TagTeamResolvedPayload captures the payload for sys.daggerheart.action.tag_team_resolved events.
type TagTeamResolvedPayload = TagTeamResolvePayload

// CountdownCreatePayload captures the payload for sys.daggerheart.action.countdown.create commands.
type CountdownCreatePayload struct {
	CountdownID string `json:"countdown_id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	Current     int    `json:"current"`
	Max         int    `json:"max"`
	Direction   string `json:"direction"`
	Looping     bool   `json:"looping"`
}

// CountdownCreatedPayload captures the payload for sys.daggerheart.action.countdown_created events.
type CountdownCreatedPayload = CountdownCreatePayload

// CountdownUpdatePayload captures the payload for sys.daggerheart.action.countdown.update commands.
type CountdownUpdatePayload struct {
	CountdownID string `json:"countdown_id"`
	Before      int    `json:"before"`
	After       int    `json:"after"`
	Delta       int    `json:"delta"`
	Looped      bool   `json:"looped"`
	Reason      string `json:"reason,omitempty"`
}

// CountdownUpdatedPayload captures the payload for sys.daggerheart.action.countdown_updated events.
type CountdownUpdatedPayload = CountdownUpdatePayload

// CountdownDeletePayload captures the payload for sys.daggerheart.action.countdown.delete commands.
type CountdownDeletePayload struct {
	CountdownID string `json:"countdown_id"`
	Reason      string `json:"reason,omitempty"`
}

// CountdownDeletedPayload captures the payload for sys.daggerheart.action.countdown_deleted events.
type CountdownDeletedPayload = CountdownDeletePayload

// DamageApplyPayload captures the payload for sys.daggerheart.action.damage.apply commands.
type DamageApplyPayload struct {
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

// DamageAppliedPayload captures the payload for sys.daggerheart.action.damage_applied events.
type DamageAppliedPayload = DamageApplyPayload

// AdversaryDamageApplyPayload captures the payload for sys.daggerheart.action.adversary_damage.apply commands.
type AdversaryDamageApplyPayload struct {
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

// AdversaryDamageAppliedPayload captures the payload for sys.daggerheart.action.adversary_damage_applied events.
type AdversaryDamageAppliedPayload = AdversaryDamageApplyPayload

// DowntimeMoveApplyPayload captures the payload for sys.daggerheart.action.downtime_move.apply commands.
type DowntimeMoveApplyPayload struct {
	CharacterID  string `json:"character_id"`
	Move         string `json:"move"`
	HopeBefore   *int   `json:"hope_before,omitempty"`
	HopeAfter    *int   `json:"hope_after,omitempty"`
	StressBefore *int   `json:"stress_before,omitempty"`
	StressAfter  *int   `json:"stress_after,omitempty"`
	ArmorBefore  *int   `json:"armor_before,omitempty"`
	ArmorAfter   *int   `json:"armor_after,omitempty"`
}

// DowntimeMoveAppliedPayload captures the payload for sys.daggerheart.action.downtime_move_applied events.
type DowntimeMoveAppliedPayload = DowntimeMoveApplyPayload

// DeathMoveResolvePayload captures the payload for sys.daggerheart.action.death_move.resolve commands.
type DeathMoveResolvePayload struct {
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

// DeathMoveResolvedPayload captures the payload for sys.daggerheart.action.death_move_resolved events.
type DeathMoveResolvedPayload = DeathMoveResolvePayload

// BlazeOfGloryResolvePayload captures the payload for sys.daggerheart.action.blaze_of_glory.resolve commands.
type BlazeOfGloryResolvePayload struct {
	CharacterID     string  `json:"character_id"`
	LifeStateBefore *string `json:"life_state_before,omitempty"`
	LifeStateAfter  string  `json:"life_state_after"`
}

// BlazeOfGloryResolvedPayload captures the payload for sys.daggerheart.action.blaze_of_glory_resolved events.
type BlazeOfGloryResolvedPayload = BlazeOfGloryResolvePayload

// GMMoveApplyPayload captures the payload for sys.daggerheart.action.gm_move.apply commands.
type GMMoveApplyPayload struct {
	Move        string `json:"move"`
	Description string `json:"description,omitempty"`
	FearSpent   int    `json:"fear_spent,omitempty"`
	Severity    string `json:"severity,omitempty"`
	Source      string `json:"source,omitempty"`
}

// GMMoveAppliedPayload captures the payload for sys.daggerheart.action.gm_move_applied events.
type GMMoveAppliedPayload = GMMoveApplyPayload

// AdversaryConditionChangePayload captures the payload for sys.daggerheart.action.adversary_condition.change commands.
type AdversaryConditionChangePayload struct {
	AdversaryID      string   `json:"adversary_id"`
	ConditionsBefore []string `json:"conditions_before,omitempty"`
	ConditionsAfter  []string `json:"conditions_after"`
	Added            []string `json:"added,omitempty"`
	Removed          []string `json:"removed,omitempty"`
	Source           string   `json:"source,omitempty"`
	RollSeq          *uint64  `json:"roll_seq,omitempty"`
}

// AdversaryConditionChangedPayload captures the payload for sys.daggerheart.action.adversary_condition_changed events.
type AdversaryConditionChangedPayload = AdversaryConditionChangePayload

// AdversaryCreatePayload captures the payload for sys.daggerheart.action.adversary.create commands.
type AdversaryCreatePayload struct {
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

// AdversaryCreatedPayload captures the payload for sys.daggerheart.action.adversary_created events.
type AdversaryCreatedPayload = AdversaryCreatePayload

// AdversaryUpdatePayload captures the payload for sys.daggerheart.action.adversary.update commands.
type AdversaryUpdatePayload struct {
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

// AdversaryUpdatedPayload captures the payload for sys.daggerheart.action.adversary_updated events.
type AdversaryUpdatedPayload = AdversaryUpdatePayload

// AdversaryDeletePayload captures the payload for sys.daggerheart.action.adversary.delete commands.
type AdversaryDeletePayload struct {
	AdversaryID string `json:"adversary_id"`
	Reason      string `json:"reason,omitempty"`
}

// AdversaryDeletedPayload captures the payload for sys.daggerheart.action.adversary_deleted events.
type AdversaryDeletedPayload = AdversaryDeletePayload
