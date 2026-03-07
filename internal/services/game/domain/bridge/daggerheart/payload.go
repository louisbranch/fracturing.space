package daggerheart

// GMFearSetPayload captures the payload for sys.daggerheart.gm_fear.set commands.
type GMFearSetPayload struct {
	After  *int   `json:"after,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// GMFearChangedPayload captures the payload for sys.daggerheart.gm_fear_changed events.
type GMFearChangedPayload struct {
	Before int    `json:"before"`
	After  int    `json:"after"`
	Reason string `json:"reason,omitempty"`
}

// CharacterStatePatchPayload captures the payload for sys.daggerheart.character_state.patch commands.
// Source is an optional discriminator indicating what triggered the patch
// (e.g. "hope.spend", "stress.spend"), enabling journal queries to distinguish
// spend events from generic GM adjustments without introducing separate event types.
type CharacterStatePatchPayload struct {
	CharacterID     string  `json:"character_id"`
	Source          string  `json:"source,omitempty"`
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

// CharacterStatePatchedPayload captures the payload for sys.daggerheart.character_state_patched events.
type CharacterStatePatchedPayload = CharacterStatePatchPayload

// ConditionChangePayload captures the payload for sys.daggerheart.condition.change commands.
type ConditionChangePayload struct {
	CharacterID      string   `json:"character_id"`
	ConditionsBefore []string `json:"conditions_before,omitempty"`
	ConditionsAfter  []string `json:"conditions_after"`
	Added            []string `json:"added,omitempty"`
	Removed          []string `json:"removed,omitempty"`
	Source           string   `json:"source,omitempty"`
	RollSeq          *uint64  `json:"roll_seq,omitempty"`
}

// ConditionChangedPayload captures the payload for sys.daggerheart.condition_changed events.
type ConditionChangedPayload = ConditionChangePayload

// HopeSpendPayload captures the payload for sys.daggerheart.hope.spend commands.
type HopeSpendPayload struct {
	CharacterID string  `json:"character_id"`
	Amount      int     `json:"amount"`
	Before      int     `json:"before"`
	After       int     `json:"after"`
	RollSeq     *uint64 `json:"roll_seq,omitempty"`
	Source      string  `json:"source,omitempty"`
}

// StressSpendPayload captures the payload for sys.daggerheart.stress.spend commands.
type StressSpendPayload struct {
	CharacterID string  `json:"character_id"`
	Amount      int     `json:"amount"`
	Before      int     `json:"before"`
	After       int     `json:"after"`
	RollSeq     *uint64 `json:"roll_seq,omitempty"`
	Source      string  `json:"source,omitempty"`
}

// LoadoutSwapPayload captures the payload for sys.daggerheart.loadout.swap commands.
type LoadoutSwapPayload struct {
	CharacterID  string `json:"character_id"`
	CardID       string `json:"card_id"`
	From         string `json:"from"`
	To           string `json:"to"`
	RecallCost   int    `json:"recall_cost,omitempty"`
	StressBefore *int   `json:"stress_before,omitempty"`
	StressAfter  *int   `json:"stress_after,omitempty"`
}

// LoadoutSwappedPayload captures the payload for sys.daggerheart.loadout_swapped events.
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

// RestTakePayload captures the payload for sys.daggerheart.rest.take commands.
type RestTakePayload struct {
	RestType          string                    `json:"rest_type"`
	Interrupted       bool                      `json:"interrupted"`
	GMFearBefore      int                       `json:"gm_fear_before"`
	GMFearAfter       int                       `json:"gm_fear_after"`
	ShortRestsBefore  int                       `json:"short_rests_before"`
	ShortRestsAfter   int                       `json:"short_rests_after"`
	RefreshRest       bool                      `json:"refresh_rest"`
	RefreshLongRest   bool                      `json:"refresh_long_rest"`
	LongTermCountdown *CountdownUpdatePayload   `json:"long_term_countdown,omitempty"`
	CharacterStates   []RestCharacterStatePatch `json:"character_states,omitempty"`
}

// RestTakenPayload captures the payload for sys.daggerheart.rest_taken events.
type RestTakenPayload = RestTakePayload

// CharacterTemporaryArmorApplyPayload captures the payload for sys.daggerheart.character_temporary_armor.apply commands.
type CharacterTemporaryArmorApplyPayload struct {
	CharacterID string `json:"character_id"`
	Source      string `json:"source"`
	Duration    string `json:"duration"`
	Amount      int    `json:"amount"`
	SourceID    string `json:"source_id,omitempty"`
}

// CharacterTemporaryArmorAppliedPayload captures the payload for sys.daggerheart.character_temporary_armor_applied events.
type CharacterTemporaryArmorAppliedPayload = CharacterTemporaryArmorApplyPayload

// RollRngInfo captures RNG metadata for roll events.
type RollRngInfo struct {
	SeedUsed   uint64 `json:"seed_used"`
	RngAlgo    string `json:"rng_algo"`
	SeedSource string `json:"seed_source"`
	RollMode   string `json:"roll_mode"`
}

// CountdownCreatePayload captures the payload for sys.daggerheart.countdown.create commands.
type CountdownCreatePayload struct {
	CountdownID       string `json:"countdown_id"`
	Name              string `json:"name"`
	Kind              string `json:"kind"`
	Current           int    `json:"current"`
	Max               int    `json:"max"`
	Direction         string `json:"direction"`
	Looping           bool   `json:"looping"`
	Variant           string `json:"variant,omitempty"`             // "standard", "dynamic", "linked"
	TriggerEventType  string `json:"trigger_event_type,omitempty"`  // For dynamic countdowns: event type that triggers advancement.
	LinkedCountdownID string `json:"linked_countdown_id,omitempty"` // For linked countdowns: the countdown ID this one is linked to.
}

// CountdownCreatedPayload captures the payload for sys.daggerheart.countdown_created events.
type CountdownCreatedPayload = CountdownCreatePayload

// CountdownUpdatePayload captures the payload for sys.daggerheart.countdown.update commands.
type CountdownUpdatePayload struct {
	CountdownID string `json:"countdown_id"`
	Before      int    `json:"before"`
	After       int    `json:"after"`
	Delta       int    `json:"delta"`
	Looped      bool   `json:"looped"`
	Reason      string `json:"reason,omitempty"`
}

// CountdownUpdatedPayload captures the payload for sys.daggerheart.countdown_updated events.
type CountdownUpdatedPayload = CountdownUpdatePayload

// CountdownDeletePayload captures the payload for sys.daggerheart.countdown.delete commands.
type CountdownDeletePayload struct {
	CountdownID string `json:"countdown_id"`
	Reason      string `json:"reason,omitempty"`
}

// CountdownDeletedPayload captures the payload for sys.daggerheart.countdown_deleted events.
type CountdownDeletedPayload = CountdownDeletePayload

// DamageApplyPayload captures the payload for sys.daggerheart.damage.apply commands.
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

// DamageAppliedPayload captures the payload for sys.daggerheart.damage_applied events.
type DamageAppliedPayload = DamageApplyPayload

// MultiTargetDamageApplyPayload captures the payload for
// sys.daggerheart.multi_target_damage.apply commands. Each target entry
// produces one damage_applied event, all batch-appended atomically.
type MultiTargetDamageApplyPayload struct {
	Targets []DamageApplyPayload `json:"targets"`
}

// AdversaryDamageApplyPayload captures the payload for sys.daggerheart.adversary_damage.apply commands.
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

// AdversaryDamageAppliedPayload captures the payload for sys.daggerheart.adversary_damage_applied events.
type AdversaryDamageAppliedPayload = AdversaryDamageApplyPayload

// DowntimeMoveApplyPayload captures the payload for sys.daggerheart.downtime_move.apply commands.
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

// DowntimeMoveAppliedPayload captures the payload for sys.daggerheart.downtime_move_applied events.
type DowntimeMoveAppliedPayload = DowntimeMoveApplyPayload

// AdversaryConditionChangePayload captures the payload for sys.daggerheart.adversary_condition.change commands.
type AdversaryConditionChangePayload struct {
	AdversaryID      string   `json:"adversary_id"`
	ConditionsBefore []string `json:"conditions_before,omitempty"`
	ConditionsAfter  []string `json:"conditions_after"`
	Added            []string `json:"added,omitempty"`
	Removed          []string `json:"removed,omitempty"`
	Source           string   `json:"source,omitempty"`
	RollSeq          *uint64  `json:"roll_seq,omitempty"`
}

// AdversaryConditionChangedPayload captures the payload for sys.daggerheart.adversary_condition_changed events.
type AdversaryConditionChangedPayload = AdversaryConditionChangePayload

// AdversaryCreatePayload captures the payload for sys.daggerheart.adversary.create commands.
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

// AdversaryCreatedPayload captures the payload for sys.daggerheart.adversary_created events.
type AdversaryCreatedPayload = AdversaryCreatePayload

// AdversaryUpdatePayload captures the payload for sys.daggerheart.adversary.update commands.
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

// AdversaryUpdatedPayload captures the payload for sys.daggerheart.adversary_updated events.
type AdversaryUpdatedPayload = AdversaryUpdatePayload

// AdversaryDeletePayload captures the payload for sys.daggerheart.adversary.delete commands.
type AdversaryDeletePayload struct {
	AdversaryID string `json:"adversary_id"`
	Reason      string `json:"reason,omitempty"`
}

// AdversaryDeletedPayload captures the payload for sys.daggerheart.adversary_deleted events.
type AdversaryDeletedPayload = AdversaryDeletePayload

// LevelUpApplyPayload captures the payload for sys.daggerheart.level_up.apply commands.
type LevelUpApplyPayload struct {
	CharacterID  string                      `json:"character_id"`
	LevelBefore  int                         `json:"level_before"`
	LevelAfter   int                         `json:"level_after"`
	Advancements []LevelUpAdvancementPayload `json:"advancements"`

	// NewDomainCardID is the domain card acquired at SRD Step 4.
	NewDomainCardID    string `json:"new_domain_card_id,omitempty"`
	NewDomainCardLevel int    `json:"new_domain_card_level,omitempty"`

	// MarkedTraits lists traits already marked from prior level-ups in this tier.
	MarkedTraits []string `json:"marked_traits,omitempty"`

	// Derived fields populated by the decider and included in the event.
	Tier           int      `json:"tier"`
	PreviousTier   int      `json:"previous_tier"`
	IsTierEntry    bool     `json:"is_tier_entry"`
	ClearMarks     bool     `json:"clear_marks"`
	MarkedAfter    []string `json:"marked_after,omitempty"`
	ThresholdDelta int      `json:"threshold_delta"`
}

// LevelUpAdvancementPayload represents a single advancement choice.
type LevelUpAdvancementPayload struct {
	Type            string                    `json:"type"`
	Trait           string                    `json:"trait,omitempty"`
	DomainCardID    string                    `json:"domain_card_id,omitempty"`
	DomainCardLevel int                       `json:"domain_card_level,omitempty"`
	SubclassCardID  string                    `json:"subclass_card_id,omitempty"`
	Multiclass      *LevelUpMulticlassPayload `json:"multiclass,omitempty"`
}

// LevelUpMulticlassPayload captures multiclass advancement choices.
type LevelUpMulticlassPayload struct {
	SecondaryClassID    string `json:"secondary_class_id"`
	SecondarySubclassID string `json:"secondary_subclass_id"`
	FoundationCardID    string `json:"foundation_card_id"`
	SpellcastTrait      string `json:"spellcast_trait"`
	DomainID            string `json:"domain_id"`
}

// LevelUpAppliedPayload captures the payload for sys.daggerheart.level_up_applied events.
type LevelUpAppliedPayload = LevelUpApplyPayload

// GoldUpdatePayload captures the payload for sys.daggerheart.gold.update commands.
type GoldUpdatePayload struct {
	CharacterID    string `json:"character_id"`
	HandfulsBefore int    `json:"handfuls_before"`
	HandfulsAfter  int    `json:"handfuls_after"`
	BagsBefore     int    `json:"bags_before"`
	BagsAfter      int    `json:"bags_after"`
	ChestsBefore   int    `json:"chests_before"`
	ChestsAfter    int    `json:"chests_after"`
	Reason         string `json:"reason,omitempty"`
}

// GoldUpdatedPayload captures the payload for sys.daggerheart.gold_updated events.
type GoldUpdatedPayload = GoldUpdatePayload

// DomainCardAcquirePayload captures the payload for sys.daggerheart.domain_card.acquire commands.
type DomainCardAcquirePayload struct {
	CharacterID string `json:"character_id"`
	CardID      string `json:"card_id"`
	CardLevel   int    `json:"card_level"`
	Destination string `json:"destination"` // "vault" or "loadout"
}

// DomainCardAcquiredPayload captures the payload for sys.daggerheart.domain_card_acquired events.
type DomainCardAcquiredPayload = DomainCardAcquirePayload

// EquipmentSwapPayload captures the payload for sys.daggerheart.equipment.swap commands.
type EquipmentSwapPayload struct {
	CharacterID string `json:"character_id"`
	ItemID      string `json:"item_id"`
	ItemType    string `json:"item_type"` // "weapon" or "armor"
	From        string `json:"from"`      // "active", "inventory", "none"
	To          string `json:"to"`        // "active", "inventory", "none"
	StressCost  int    `json:"stress_cost,omitempty"`
}

// EquipmentSwappedPayload captures the payload for sys.daggerheart.equipment_swapped events.
type EquipmentSwappedPayload = EquipmentSwapPayload

// ConsumableUsePayload captures the payload for sys.daggerheart.consumable.use commands.
type ConsumableUsePayload struct {
	CharacterID    string `json:"character_id"`
	ConsumableID   string `json:"consumable_id"`
	QuantityBefore int    `json:"quantity_before"`
	QuantityAfter  int    `json:"quantity_after"`
}

// ConsumableUsedPayload captures the payload for sys.daggerheart.consumable_used events.
type ConsumableUsedPayload = ConsumableUsePayload

// ConsumableAcquirePayload captures the payload for sys.daggerheart.consumable.acquire commands.
type ConsumableAcquirePayload struct {
	CharacterID    string `json:"character_id"`
	ConsumableID   string `json:"consumable_id"`
	QuantityBefore int    `json:"quantity_before"`
	QuantityAfter  int    `json:"quantity_after"`
}

// ConsumableAcquiredPayload captures the payload for sys.daggerheart.consumable_acquired events.
type ConsumableAcquiredPayload = ConsumableAcquirePayload
