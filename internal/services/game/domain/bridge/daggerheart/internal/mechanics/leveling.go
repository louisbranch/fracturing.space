package mechanics

import "fmt"

// ── Tier system ─────────────────────────────────────────────────────────

const (
	LevelMin = 1
	LevelMax = 10
)

// TierForLevel returns the tier (1-4) for a given character level (1-10).
// T1=level 1, T2=levels 2-4, T3=levels 5-7, T4=levels 8-10.
func TierForLevel(level int) int {
	switch {
	case level <= 1:
		return 1
	case level <= 4:
		return 2
	case level <= 7:
		return 3
	default:
		return 4
	}
}

// IsTierEntry returns true when the given level is the first level of a new
// tier (levels 2, 5, 8). Tier entry triggers automatic tier achievements.
func IsTierEntry(level int) bool {
	switch level {
	case 2, 5, 8:
		return true
	default:
		return false
	}
}

// ── Advancement options ─────────────────────────────────────────────────

// AdvancementType enumerates the advancement choices a player may select
// when leveling up.
type AdvancementType string

const (
	AdvTraitIncrease       AdvancementType = "trait_increase"
	AdvAddHPSlots          AdvancementType = "add_hp_slots"
	AdvAddStressSlots      AdvancementType = "add_stress_slots"
	AdvIncreaseExperience  AdvancementType = "increase_experience"
	AdvDomainCard          AdvancementType = "domain_card"
	AdvIncreaseEvasion     AdvancementType = "increase_evasion"
	AdvUpgradedSubclass    AdvancementType = "upgraded_subclass"
	AdvIncreaseProficiency AdvancementType = "increase_proficiency"
	AdvMulticlass          AdvancementType = "multiclass"
)

// AdvancementSlotCost returns how many of the 2 per-level advancement slots
// this option consumes. Proficiency and Multiclass cost 2; everything else
// costs 1.
func AdvancementSlotCost(adv AdvancementType) int {
	switch adv {
	case AdvIncreaseProficiency, AdvMulticlass:
		return 2
	default:
		return 1
	}
}

// ── Trait names ──────────────────────────────────────────────────────────

// TraitName enumerates the six Daggerheart ability traits.
type TraitName string

const (
	TraitAgility   TraitName = "agility"
	TraitStrength  TraitName = "strength"
	TraitFinesse   TraitName = "finesse"
	TraitInstinct  TraitName = "instinct"
	TraitPresence  TraitName = "presence"
	TraitKnowledge TraitName = "knowledge"
)

// AllTraitNames returns the canonical trait list.
func AllTraitNames() []TraitName {
	return []TraitName{
		TraitAgility, TraitStrength, TraitFinesse,
		TraitInstinct, TraitPresence, TraitKnowledge,
	}
}

// IsValidTrait checks whether a string is a known trait name.
func IsValidTrait(name string) bool {
	switch TraitName(name) {
	case TraitAgility, TraitStrength, TraitFinesse,
		TraitInstinct, TraitPresence, TraitKnowledge:
		return true
	default:
		return false
	}
}

// ── Level-up request / result ───────────────────────────────────────────

// Advancement represents a single player-chosen advancement during level-up.
type Advancement struct {
	Type AdvancementType `json:"type"`

	// TraitIncrease fields (only when Type == AdvTraitIncrease).
	Trait string `json:"trait,omitempty"`

	// DomainCard fields (only when Type == AdvDomainCard).
	DomainCardID    string `json:"domain_card_id,omitempty"`
	DomainCardLevel int    `json:"domain_card_level,omitempty"`

	// UpgradedSubclass fields (only when Type == AdvUpgradedSubclass).
	SubclassCardID string `json:"subclass_card_id,omitempty"`

	// Multiclass fields (only when Type == AdvMulticlass).
	Multiclass *MulticlassChoice `json:"multiclass,omitempty"`
}

// MulticlassChoice captures the choices made when selecting multiclass.
type MulticlassChoice struct {
	SecondaryClassID    string `json:"secondary_class_id"`
	SecondarySubclassID string `json:"secondary_subclass_id"`
	FoundationCardID    string `json:"foundation_card_id"`
	SpellcastTrait      string `json:"spellcast_trait"`
	DomainID            string `json:"domain_id"`
}

// LevelUpRequest captures all player choices for leveling up.
type LevelUpRequest struct {
	CharacterID  string        `json:"character_id"`
	LevelBefore  int           `json:"level_before"`
	LevelAfter   int           `json:"level_after"`
	Advancements []Advancement `json:"advancements"`

	// DomainCardID is the new domain card acquired at/below the new level.
	// SRD Step 4: "Choose a new domain card from your class's domains."
	NewDomainCardID    string `json:"new_domain_card_id,omitempty"`
	NewDomainCardLevel int    `json:"new_domain_card_level,omitempty"`

	// MarkedTraits tracks which traits were already marked (increased in
	// current tier). Used for validation — can't increase a marked trait.
	MarkedTraits []string `json:"marked_traits,omitempty"`
}

// LevelUpResult captures the derived consequences of a validated level-up.
type LevelUpResult struct {
	Tier           int
	PreviousTier   int
	IsTierEntry    bool
	SlotsConsumed  int
	MarkedTraits   []string // Updated marked traits after this level-up.
	ClearMarks     bool     // Whether marked traits should be cleared (tier entry at 5, 8).
	ThresholdDelta int      // Baseline threshold increase per level; replay/projection derive severe scaling from this.
}

// ── Validation ──────────────────────────────────────────────────────────

const advancementBudget = 2

// ValidateLevelUp checks a LevelUpRequest against SRD rules and returns the
// derived result. Errors describe the first rule violation found.
func ValidateLevelUp(req LevelUpRequest) (LevelUpResult, error) {
	// Level range checks.
	if req.LevelBefore < LevelMin || req.LevelBefore >= LevelMax {
		return LevelUpResult{}, fmt.Errorf("level_before must be in range %d..%d", LevelMin, LevelMax-1)
	}
	if req.LevelAfter != req.LevelBefore+1 {
		return LevelUpResult{}, fmt.Errorf("level_after must be level_before + 1 (got %d, want %d)", req.LevelAfter, req.LevelBefore+1)
	}

	tier := TierForLevel(req.LevelAfter)
	prevTier := TierForLevel(req.LevelBefore)
	tierEntry := IsTierEntry(req.LevelAfter)

	// Validate advancement budget.
	totalSlots := 0
	for _, adv := range req.Advancements {
		totalSlots += AdvancementSlotCost(adv.Type)
	}
	if totalSlots != advancementBudget {
		return LevelUpResult{}, fmt.Errorf("advancements must consume exactly %d slots (got %d)", advancementBudget, totalSlots)
	}

	// Build marked-trait set for duplicate checking.
	markedSet := make(map[string]bool, len(req.MarkedTraits))
	for _, t := range req.MarkedTraits {
		markedSet[t] = true
	}

	// Clear marks at tier entry for levels 5 and 8 (entering T3 and T4).
	// Marks are cleared BEFORE processing advancements, so the player can
	// re-increase previously marked traits.
	clearMarks := tierEntry && (req.LevelAfter == 5 || req.LevelAfter == 8)
	if clearMarks {
		markedSet = make(map[string]bool)
	}

	// Validate each advancement.
	for i, adv := range req.Advancements {
		if err := validateAdvancement(adv, i, req, markedSet); err != nil {
			return LevelUpResult{}, err
		}
		// Track newly marked traits.
		if adv.Type == AdvTraitIncrease && adv.Trait != "" {
			markedSet[adv.Trait] = true
		}
	}

	// Build updated marked trait list.
	updatedMarks := make([]string, 0, len(markedSet))
	for t := range markedSet {
		updatedMarks = append(updatedMarks, t)
	}

	return LevelUpResult{
		Tier:           tier,
		PreviousTier:   prevTier,
		IsTierEntry:    tierEntry,
		SlotsConsumed:  totalSlots,
		MarkedTraits:   updatedMarks,
		ClearMarks:     clearMarks,
		ThresholdDelta: 1,
	}, nil
}

func validateAdvancement(adv Advancement, index int, req LevelUpRequest, markedSet map[string]bool) error {
	prefix := fmt.Sprintf("advancements[%d]", index)
	switch adv.Type {
	case AdvTraitIncrease:
		if adv.Trait == "" {
			return fmt.Errorf("%s: trait is required for trait_increase", prefix)
		}
		if !IsValidTrait(adv.Trait) {
			return fmt.Errorf("%s: unknown trait %q", prefix, adv.Trait)
		}
		if markedSet[adv.Trait] {
			return fmt.Errorf("%s: trait %q is already marked and cannot be increased again this tier", prefix, adv.Trait)
		}

	case AdvAddHPSlots, AdvAddStressSlots, AdvIncreaseExperience,
		AdvIncreaseEvasion, AdvUpgradedSubclass:
		// No additional validation needed for these types.

	case AdvDomainCard:
		if adv.DomainCardID == "" {
			return fmt.Errorf("%s: domain_card_id is required for domain_card advancement", prefix)
		}
		if adv.DomainCardLevel > req.LevelAfter {
			return fmt.Errorf("%s: domain card level %d exceeds character level %d", prefix, adv.DomainCardLevel, req.LevelAfter)
		}

	case AdvIncreaseProficiency:
		// Costs 2 slots — budget check handles that.

	case AdvMulticlass:
		if req.LevelAfter < 5 {
			return fmt.Errorf("%s: multiclass requires level 5 or higher (level_after=%d)", prefix, req.LevelAfter)
		}
		if adv.Multiclass == nil {
			return fmt.Errorf("%s: multiclass choices are required", prefix)
		}
		mc := adv.Multiclass
		if mc.SecondaryClassID == "" {
			return fmt.Errorf("%s: secondary_class_id is required for multiclass", prefix)
		}
		if mc.SecondarySubclassID == "" {
			return fmt.Errorf("%s: secondary_subclass_id is required for multiclass", prefix)
		}
		if mc.FoundationCardID == "" {
			return fmt.Errorf("%s: foundation_card_id is required for multiclass", prefix)
		}
		if mc.DomainID == "" {
			return fmt.Errorf("%s: domain_id is required for multiclass", prefix)
		}

	default:
		return fmt.Errorf("%s: unknown advancement type %q", prefix, adv.Type)
	}
	return nil
}
