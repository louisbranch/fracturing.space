package payload

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

// --- Level up ---

// LevelUpApplyPayload captures the payload for sys.daggerheart.level_up.apply commands.
type LevelUpApplyPayload struct {
	CharacterID                  ids.CharacterID                           `json:"character_id"`
	LevelBefore                  int                                       `json:"level_before"`
	LevelAfter                   int                                       `json:"level_after"`
	Advancements                 []LevelUpAdvancementPayload               `json:"advancements"`
	Rewards                      []LevelUpRewardPayload                    `json:"rewards,omitempty"`
	MarkedTraits                 []string                                  `json:"marked_traits,omitempty"`
	SubclassTracksAfter          []daggerheartstate.CharacterSubclassTrack `json:"subclass_tracks_after,omitempty"`
	SubclassHpMaxDelta           int                                       `json:"subclass_hp_max_delta,omitempty"`
	SubclassStressMaxDelta       int                                       `json:"subclass_stress_max_delta,omitempty"`
	SubclassEvasionDelta         int                                       `json:"subclass_evasion_delta,omitempty"`
	SubclassMajorThresholdDelta  int                                       `json:"subclass_major_threshold_delta,omitempty"`
	SubclassSevereThresholdDelta int                                       `json:"subclass_severe_threshold_delta,omitempty"`
	Tier                         int                                       `json:"tier"`
	PreviousTier                 int                                       `json:"previous_tier"`
	IsTierEntry                  bool                                      `json:"is_tier_entry"`
	ClearMarks                   bool                                      `json:"clear_marks"`
	MarkedAfter                  []string                                  `json:"marked_after,omitempty"`
	ThresholdDelta               int                                       `json:"threshold_delta"`
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
	CharacterID                  ids.CharacterID                           `json:"character_id"`
	Level                        int                                       `json:"level_after"`
	Advancements                 []LevelUpAdvancementPayload               `json:"advancements"`
	Rewards                      []LevelUpRewardPayload                    `json:"rewards,omitempty"`
	SubclassTracksAfter          []daggerheartstate.CharacterSubclassTrack `json:"subclass_tracks_after,omitempty"`
	SubclassHpMaxDelta           int                                       `json:"subclass_hp_max_delta,omitempty"`
	SubclassStressMaxDelta       int                                       `json:"subclass_stress_max_delta,omitempty"`
	SubclassEvasionDelta         int                                       `json:"subclass_evasion_delta,omitempty"`
	SubclassMajorThresholdDelta  int                                       `json:"subclass_major_threshold_delta,omitempty"`
	SubclassSevereThresholdDelta int                                       `json:"subclass_severe_threshold_delta,omitempty"`
	Tier                         int                                       `json:"tier"`
	IsTierEntry                  bool                                      `json:"is_tier_entry"`
	ClearMarks                   bool                                      `json:"clear_marks"`
	Marked                       []string                                  `json:"marked_after,omitempty"`
	ThresholdDelta               int                                       `json:"threshold_delta"`
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

// --- Stat Modifiers ---

// StatModifierChangePayload captures the payload for sys.daggerheart.stat_modifier.change commands.
type StatModifierChangePayload struct {
	CharacterID     ids.CharacterID           `json:"character_id"`
	ModifiersBefore []rules.StatModifierState `json:"modifiers_before,omitempty"`
	ModifiersAfter  []rules.StatModifierState `json:"modifiers_after"`
	Added           []rules.StatModifierState `json:"added,omitempty"`
	Removed         []rules.StatModifierState `json:"removed,omitempty"`
	Source          string                    `json:"source,omitempty"`
}

// StatModifierChangedPayload captures the payload for sys.daggerheart.stat_modifier_changed events.
type StatModifierChangedPayload struct {
	CharacterID ids.CharacterID           `json:"character_id"`
	Modifiers   []rules.StatModifierState `json:"modifiers_after"`
	Added       []rules.StatModifierState `json:"added,omitempty"`
	Removed     []rules.StatModifierState `json:"removed,omitempty"`
	Source      string                    `json:"source,omitempty"`
}
