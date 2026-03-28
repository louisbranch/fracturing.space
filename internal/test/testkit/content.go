package testkit

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	sqlitedaggerheartcontent "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/daggerheartcontent"
)

// ContentSeedProfile selects the minimal Daggerheart catalog variant a suite needs.
type ContentSeedProfile string

const (
	// ContentSeedProfileIntegration matches the integration readiness catalog.
	ContentSeedProfileIntegration ContentSeedProfile = "integration"
	// ContentSeedProfileScenario matches the scenario readiness catalog.
	ContentSeedProfileScenario ContentSeedProfile = "scenario"
)

type contentSeedTemplate struct {
	once sync.Once
	path string
	err  error
}

var (
	integrationContentTemplate contentSeedTemplate
	scenarioContentTemplate    contentSeedTemplate
)

// SeedDaggerheartContent copies a cached content seed into the configured content DB.
func SeedDaggerheartContent(t *testing.T, profile ContentSeedProfile) {
	t.Helper()

	contentPath := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_GAME_CONTENT_DB_PATH"))
	if contentPath == "" {
		t.Fatal("content DB path env is required")
	}

	templatePath := ensureContentSeedTemplate(t, profile)
	if err := copyFile(templatePath, contentPath); err != nil {
		t.Fatalf("copy content seed template: %v", err)
	}
}

// ensureContentSeedTemplate prepares a reusable seed database for one profile.
func ensureContentSeedTemplate(t *testing.T, profile ContentSeedProfile) string {
	t.Helper()

	template := contentSeedTemplateFor(t, profile)
	template.once.Do(func() {
		tmpDir, err := os.MkdirTemp("", "testkit-content-seed-*")
		if err != nil {
			template.err = fmt.Errorf("create content seed temp dir: %w", err)
			return
		}

		template.path = filepath.Join(tmpDir, "game-content-template.db")
		store, err := sqlitedaggerheartcontent.Open(template.path)
		if err != nil {
			template.err = fmt.Errorf("open content seed template store: %w", err)
			return
		}

		if err := writeDaggerheartSeedData(store, time.Now().UTC(), profile); err != nil {
			_ = store.Close()
			template.err = err
			return
		}
		if err := store.Close(); err != nil {
			template.err = fmt.Errorf("close content seed template store: %w", err)
			return
		}
	})

	if template.err != nil {
		t.Fatalf("initialize %s content seed template: %v", profile, template.err)
	}
	return template.path
}

// contentSeedTemplateFor routes callers to the profile-specific template cache.
func contentSeedTemplateFor(t *testing.T, profile ContentSeedProfile) *contentSeedTemplate {
	t.Helper()

	switch profile {
	case ContentSeedProfileIntegration:
		return &integrationContentTemplate
	case ContentSeedProfileScenario:
		return &scenarioContentTemplate
	default:
		t.Fatalf("unsupported content seed profile %q", profile)
		return nil
	}
}

// writeDaggerheartSeedData stores the minimal catalog rows needed by readiness helpers.
func writeDaggerheartSeedData(store contentstore.DaggerheartContentWriteStore, now time.Time, profile ContentSeedProfile) error {
	if store == nil {
		return fmt.Errorf("content store is required")
	}

	ctx := context.Background()
	if err := writeCommonDaggerheartSeedData(ctx, store, now, profile); err != nil {
		return err
	}

	switch profile {
	case ContentSeedProfileIntegration:
		return writeIntegrationSeedData(ctx, store, now)
	case ContentSeedProfileScenario:
		return writeScenarioSeedData(ctx, store, now)
	default:
		return fmt.Errorf("unsupported content seed profile %q", profile)
	}
}

// writeCommonDaggerheartSeedData stores rows shared across integration and scenario readiness.
func writeCommonDaggerheartSeedData(ctx context.Context, store contentstore.DaggerheartContentWriteStore, now time.Time, profile ContentSeedProfile) error {
	if err := store.PutDaggerheartClass(ctx, contentstore.DaggerheartClass{
		ID:              "class.guardian",
		Name:            "Guardian",
		StartingEvasion: 9,
		StartingHP:      7,
		Features: []contentstore.DaggerheartFeature{{
			ID:               "feature.guardian-unstoppable",
			Name:             "Unstoppable",
			Description:      "Scenario seed Guardian feature.",
			Level:            1,
			AutomationStatus: contentstore.DaggerheartFeatureAutomationStatusSupported,
			ClassRule: &contentstore.DaggerheartClassFeatureRule{
				Kind:     contentstore.DaggerheartClassFeatureRuleKindUnstoppable,
				DieSides: 4,
			},
		}},
		HopeFeature: contentstore.DaggerheartHopeFeature{
			Name:             "Frontline Tank",
			Description:      "Spend Hope to recover armor.",
			HopeCost:         3,
			AutomationStatus: contentstore.DaggerheartFeatureAutomationStatusSupported,
			HopeFeatureRule: &contentstore.DaggerheartHopeFeatureRule{
				Kind:     contentstore.DaggerheartHopeFeatureRuleKindFrontlineTank,
				Bonus:    2,
				HopeCost: 3,
			},
		},
		DomainIDs: []string{"domain.valor", "domain.blade"},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed class: %w", err)
	}

	if err := store.PutDaggerheartSubclass(ctx, contentstore.DaggerheartSubclass{
		ID:      "subclass.stalwart",
		Name:    "Stalwart",
		ClassID: "class.guardian",
		FoundationFeatures: []contentstore.DaggerheartFeature{{
			ID:               "feature.stalwart-unwavering",
			Name:             "Unwavering",
			Description:      "Scenario seed foundation feature granting one HP slot.",
			Level:            1,
			AutomationStatus: contentstore.DaggerheartFeatureAutomationStatusSupported,
			SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
				Kind:  contentstore.DaggerheartSubclassFeatureRuleKindHPSlotBonus,
				Bonus: 1,
			},
		}},
		SpecializationFeatures: []contentstore.DaggerheartFeature{{
			ID:               "feature.stalwart-unrelenting",
			Name:             "Unrelenting",
			Description:      "Scenario seed specialization feature raising both thresholds.",
			Level:            1,
			AutomationStatus: contentstore.DaggerheartFeatureAutomationStatusSupported,
			SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
				Kind:           contentstore.DaggerheartSubclassFeatureRuleKindThresholdBonus,
				Bonus:          1,
				ThresholdScope: contentstore.DaggerheartSubclassThresholdScopeAll,
			},
		}},
		MasteryFeatures: []contentstore.DaggerheartFeature{{
			ID:               "feature.stalwart-undaunted",
			Name:             "Undaunted",
			Description:      "Scenario seed mastery feature granting one Stress slot.",
			Level:            1,
			AutomationStatus: contentstore.DaggerheartFeatureAutomationStatusSupported,
			SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
				Kind:  contentstore.DaggerheartSubclassFeatureRuleKindStressSlotBonus,
				Bonus: 1,
			},
		}},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed subclass: %w", err)
	}

	if err := store.PutDaggerheartHeritage(ctx, contentstore.DaggerheartHeritage{
		ID:   "heritage.human",
		Name: "Human",
		Kind: "ancestry",
		Features: []contentstore.DaggerheartFeature{
			{
				ID:          "feature.human-high-stamina",
				Name:        "High Stamina",
				Description: "Gain one extra Stress slot during character creation.",
				Level:       1,
			},
			{
				ID:          "feature.human-adaptable",
				Name:        "Adaptability",
				Description: "When a roll tied to one of your experiences fails, you can mark Stress to reroll it.",
				Level:       1,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed ancestry heritage: %w", err)
	}

	if err := store.PutDaggerheartHeritage(ctx, contentstore.DaggerheartHeritage{
		ID:        "heritage.highborne",
		Name:      "Highborne",
		Kind:      "community",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed community heritage: %w", err)
	}

	if err := store.PutDaggerheartDomain(ctx, contentstore.DaggerheartDomain{
		ID:          "domain.valor",
		Name:        "Valor",
		Description: profileSeedText(profile, "domain"),
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		return fmt.Errorf("seed domain: %w", err)
	}

	if err := store.PutDaggerheartDomainCard(ctx, contentstore.DaggerheartDomainCard{
		ID:          "domain_card.valor-bare-bones",
		Name:        "Bare Bones",
		DomainID:    "domain.valor",
		Level:       1,
		Type:        "ability",
		UsageLimit:  "None",
		FeatureText: profileSeedText(profile, "card"),
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		return fmt.Errorf("seed domain card: %w", err)
	}

	if err := store.PutDaggerheartWeapon(ctx, contentstore.DaggerheartWeapon{
		ID:         "weapon.longsword",
		Name:       "Longsword",
		Category:   "primary",
		Tier:       1,
		Trait:      "Agility",
		Range:      "melee",
		DamageDice: []contentstore.DaggerheartDamageDie{{Sides: 10, Count: 1}},
		DamageType: "physical",
		Burden:     2,
		Feature:    profileSeedText(profile, "weapon"),
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		return fmt.Errorf("seed weapon: %w", err)
	}

	if err := store.PutDaggerheartArmor(ctx, contentstore.DaggerheartArmor{
		ID:                  "armor.gambeson-armor",
		Name:                "Gambeson Armor",
		Tier:                1,
		BaseMajorThreshold:  6,
		BaseSevereThreshold: 12,
		ArmorScore:          1,
		Feature:             profileSeedText(profile, "armor"),
		CreatedAt:           now,
		UpdatedAt:           now,
	}); err != nil {
		return fmt.Errorf("seed armor: %w", err)
	}

	if err := store.PutDaggerheartItem(ctx, contentstore.DaggerheartItem{
		ID:        "item.minor-health-potion",
		Name:      "Minor Health Potion",
		Rarity:    "common",
		Kind:      "consumable",
		StackMax:  99,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed health potion: %w", err)
	}

	return nil
}

// writeIntegrationSeedData extends the shared seed with integration-specific IDs.
func writeIntegrationSeedData(ctx context.Context, store contentstore.DaggerheartContentWriteStore, now time.Time) error {
	if err := store.PutDaggerheartAdversaryEntry(ctx, contentstore.DaggerheartAdversaryEntry{
		ID:              "adversary.integration-foe",
		Name:            "Integration Foe",
		Tier:            1,
		Role:            "standard",
		Description:     "Integration seed adversary for catalog-backed runtime tests.",
		Motives:         "Pressure, reposition, strike.",
		Difficulty:      11,
		MajorThreshold:  6,
		SevereThreshold: 12,
		HP:              6,
		Stress:          2,
		Armor:           0,
		AttackModifier:  1,
		StandardAttack: contentstore.DaggerheartAdversaryAttack{
			Name:        "Slash",
			Range:       "melee",
			DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 8, Count: 1}},
			DamageBonus: 1,
			DamageType:  "physical",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed integration adversary: %w", err)
	}

	if err := store.PutDaggerheartDomainCard(ctx, contentstore.DaggerheartDomainCard{
		ID:          "domain_card.valor-shield-wall",
		Name:        "Shield Wall",
		DomainID:    "domain.valor",
		Level:       1,
		Type:        "ability",
		UsageLimit:  "None",
		FeatureText: "Integration seed card 2",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		return fmt.Errorf("seed domain card 2: %w", err)
	}

	if err := store.PutDaggerheartItem(ctx, contentstore.DaggerheartItem{
		ID:        "item.minor-stamina-potion",
		Name:      "Minor Stamina Potion",
		Rarity:    "common",
		Kind:      "consumable",
		StackMax:  99,
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed stamina potion: %w", err)
	}

	for _, class := range []contentstore.DaggerheartClass{
		{
			ID:              "class.ranger",
			Name:            "Ranger",
			StartingEvasion: 10,
			StartingHP:      6,
			DomainIDs:       []string{"domain.bone", "domain.sage"},
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			ID:              "class.bard",
			Name:            "Bard",
			StartingEvasion: 10,
			StartingHP:      6,
			DomainIDs:       []string{"domain.grace", "domain.codex"},
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	} {
		if err := store.PutDaggerheartClass(ctx, class); err != nil {
			return fmt.Errorf("seed discovery integration class %s: %w", class.ID, err)
		}
	}

	for _, subclass := range []contentstore.DaggerheartSubclass{
		{
			ID:        "subclass.wayfinder",
			Name:      "Wayfinder",
			ClassID:   "class.ranger",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:             "subclass.wordsmith",
			Name:           "Wordsmith",
			ClassID:        "class.bard",
			SpellcastTrait: "presence",
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	} {
		if err := store.PutDaggerheartSubclass(ctx, subclass); err != nil {
			return fmt.Errorf("seed discovery integration subclass %s: %w", subclass.ID, err)
		}
	}

	for _, heritage := range []contentstore.DaggerheartHeritage{
		{
			ID:   "heritage.elf",
			Name: "Elf",
			Kind: "ancestry",
			Features: []contentstore.DaggerheartFeature{
				{
					ID:          "feature.elf-quick-reactions",
					Name:        "Quick Reactions",
					Description: "Integration seed primary elf ancestry feature.",
					Level:       1,
				},
				{
					ID:          "feature.elf-celestial-trance",
					Name:        "Celestial Trance",
					Description: "Integration seed secondary elf ancestry feature.",
					Level:       1,
				},
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "heritage.seaborne",
			Name:      "Seaborne",
			Kind:      "community",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "heritage.wildborne",
			Name:      "Wildborne",
			Kind:      "community",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "heritage.loreborne",
			Name:      "Loreborne",
			Kind:      "community",
			CreatedAt: now,
			UpdatedAt: now,
		},
	} {
		if err := store.PutDaggerheartHeritage(ctx, heritage); err != nil {
			return fmt.Errorf("seed discovery integration heritage %s: %w", heritage.ID, err)
		}
	}

	for _, domain := range []contentstore.DaggerheartDomain{
		{ID: "domain.blade", Name: "Blade", Description: "Integration seed blade domain.", CreatedAt: now, UpdatedAt: now},
		{ID: "domain.bone", Name: "Bone", Description: "Integration seed bone domain.", CreatedAt: now, UpdatedAt: now},
		{ID: "domain.sage", Name: "Sage", Description: "Integration seed sage domain.", CreatedAt: now, UpdatedAt: now},
		{ID: "domain.grace", Name: "Grace", Description: "Integration seed grace domain.", CreatedAt: now, UpdatedAt: now},
		{ID: "domain.codex", Name: "Codex", Description: "Integration seed codex domain.", CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutDaggerheartDomain(ctx, domain); err != nil {
			return fmt.Errorf("seed discovery integration domain %s: %w", domain.ID, err)
		}
	}

	for _, card := range []contentstore.DaggerheartDomainCard{
		{
			ID:          "domain_card.valor-i-am-your-shield",
			Name:        "I Am Your Shield",
			DomainID:    "domain.valor",
			Level:       1,
			Type:        "ability",
			UsageLimit:  "None",
			FeatureText: "Integration seed discovery starter card.",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "domain_card.blade-get-back-up",
			Name:        "Get Back Up",
			DomainID:    "domain.blade",
			Level:       1,
			Type:        "ability",
			UsageLimit:  "None",
			FeatureText: "Integration seed discovery starter card.",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "domain_card.bone-i-see-it-coming",
			Name:        "I See It Coming",
			DomainID:    "domain.bone",
			Level:       1,
			Type:        "ability",
			UsageLimit:  "None",
			FeatureText: "Integration seed discovery starter card.",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "domain_card.sage-gifted-tracker",
			Name:        "Gifted Tracker",
			DomainID:    "domain.sage",
			Level:       1,
			Type:        "ability",
			UsageLimit:  "None",
			FeatureText: "Integration seed discovery starter card.",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "domain_card.grace-inspirational-words",
			Name:        "Inspirational Words",
			DomainID:    "domain.grace",
			Level:       1,
			Type:        "spell",
			UsageLimit:  "None",
			FeatureText: "Integration seed discovery starter card.",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "domain_card.book-of-illiat",
			Name:        "Book of Illiat",
			DomainID:    "domain.codex",
			Level:       1,
			Type:        "spell",
			UsageLimit:  "None",
			FeatureText: "Integration seed discovery starter card.",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	} {
		if err := store.PutDaggerheartDomainCard(ctx, card); err != nil {
			return fmt.Errorf("seed discovery integration domain card %s: %w", card.ID, err)
		}
	}

	for _, weapon := range []contentstore.DaggerheartWeapon{
		{
			ID:         "weapon.shortbow",
			Name:       "Shortbow",
			Category:   "primary",
			Tier:       1,
			Trait:      "Agility",
			Range:      "far",
			DamageDice: []contentstore.DaggerheartDamageDie{{Sides: 8, Count: 1}},
			DamageType: "physical",
			Burden:     2,
			Feature:    "Integration seed discovery starter weapon.",
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			ID:         "weapon.dagger",
			Name:       "Dagger",
			Category:   "primary",
			Tier:       1,
			Trait:      "Finesse",
			Range:      "very_close",
			DamageDice: []contentstore.DaggerheartDamageDie{{Sides: 6, Count: 1}},
			DamageType: "physical",
			Burden:     1,
			Feature:    "Integration seed discovery starter weapon.",
			CreatedAt:  now,
			UpdatedAt:  now,
		},
		{
			ID:         "weapon.small-dagger",
			Name:       "Small Dagger",
			Category:   "secondary",
			Tier:       1,
			Trait:      "Finesse",
			Range:      "very_close",
			DamageDice: []contentstore.DaggerheartDamageDie{{Sides: 6, Count: 1}},
			DamageType: "physical",
			Burden:     1,
			Feature:    "Integration seed discovery starter weapon.",
			CreatedAt:  now,
			UpdatedAt:  now,
		},
	} {
		if err := store.PutDaggerheartWeapon(ctx, weapon); err != nil {
			return fmt.Errorf("seed discovery integration weapon %s: %w", weapon.ID, err)
		}
	}

	for _, armor := range []contentstore.DaggerheartArmor{
		{
			ID:                  "armor.leather-armor",
			Name:                "Leather Armor",
			Tier:                1,
			BaseMajorThreshold:  6,
			BaseSevereThreshold: 13,
			ArmorScore:          3,
			CreatedAt:           now,
			UpdatedAt:           now,
		},
		{
			ID:                  "armor.chainmail-armor",
			Name:                "Chainmail Armor",
			Tier:                1,
			BaseMajorThreshold:  7,
			BaseSevereThreshold: 15,
			ArmorScore:          4,
			Feature:             "Heavy: -1 to Evasion",
			Rules: contentstore.DaggerheartArmorRules{
				AutomationStatus:       contentstore.DaggerheartArmorAutomationStatusSupported,
				MitigationMode:         contentstore.DaggerheartArmorMitigationModeAny,
				EvasionDelta:           -1,
				SeverityReductionSteps: 1,
			},
			CreatedAt: now,
			UpdatedAt: now,
		},
	} {
		if err := store.PutDaggerheartArmor(ctx, armor); err != nil {
			return fmt.Errorf("seed discovery integration armor %s: %w", armor.ID, err)
		}
	}

	return nil
}

// writeScenarioSeedData extends the shared seed with scenario-specific IDs.
func writeScenarioSeedData(ctx context.Context, store contentstore.DaggerheartContentWriteStore, now time.Time) error {
	putAdversary := func(entry contentstore.DaggerheartAdversaryEntry) error {
		if err := store.PutDaggerheartAdversaryEntry(ctx, entry); err != nil {
			return fmt.Errorf("seed scenario adversary entry %s: %w", entry.ID, err)
		}
		return nil
	}
	putArmor := func(armor contentstore.DaggerheartArmor) error {
		if err := store.PutDaggerheartArmor(ctx, armor); err != nil {
			return fmt.Errorf("seed scenario armor %s: %w", armor.ID, err)
		}
		return nil
	}

	if err := store.PutDaggerheartClass(ctx, contentstore.DaggerheartClass{
		ID:              "class.ranger",
		Name:            "Ranger",
		StartingEvasion: 10,
		StartingHP:      6,
		DomainIDs:       []string{"domain.valor"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		return fmt.Errorf("seed ranger class: %w", err)
	}

	if err := store.PutDaggerheartSubclass(ctx, contentstore.DaggerheartSubclass{
		ID:                   "subclass.beastbound",
		Name:                 "Beastbound",
		ClassID:              "class.ranger",
		CreationRequirements: []contentstore.DaggerheartSubclassCreationRequirement{contentstore.DaggerheartSubclassCreationRequirementCompanionSheet},
		CreatedAt:            now,
		UpdatedAt:            now,
	}); err != nil {
		return fmt.Errorf("seed beastbound subclass: %w", err)
	}
	for _, experience := range []contentstore.DaggerheartCompanionExperienceEntry{
		{ID: "companion-experience.navigation", Name: "Navigation", CreatedAt: now, UpdatedAt: now},
		{ID: "companion-experience.scout", Name: "Scout", CreatedAt: now, UpdatedAt: now},
		{ID: "companion-experience.vigilant", Name: "Vigilant", CreatedAt: now, UpdatedAt: now},
	} {
		if err := store.PutDaggerheartCompanionExperience(ctx, experience); err != nil {
			return fmt.Errorf("seed scenario companion experience %s: %w", experience.ID, err)
		}
	}

	if err := store.PutDaggerheartClass(ctx, contentstore.DaggerheartClass{
		ID:              "class.bard",
		Name:            "Bard",
		StartingEvasion: 10,
		StartingHP:      6,
		DomainIDs:       []string{"domain.valor", "domain.codex"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		return fmt.Errorf("seed bard class: %w", err)
	}

	if err := store.PutDaggerheartSubclass(ctx, contentstore.DaggerheartSubclass{
		ID:             "subclass.wordsmith",
		Name:           "Wordsmith",
		ClassID:        "class.bard",
		SpellcastTrait: "presence",
		FoundationFeatures: []contentstore.DaggerheartFeature{{
			ID:          "feature.wordsmith-foundation",
			Name:        "Cutting Verse",
			Description: "Scenario seed multiclass foundation feature.",
			Level:       1,
		}},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed wordsmith subclass: %w", err)
	}

	if err := store.PutDaggerheartDomain(ctx, contentstore.DaggerheartDomain{
		ID:          "domain.codex",
		Name:        "Codex",
		Description: "Scenario seed codex domain.",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		return fmt.Errorf("seed codex domain: %w", err)
	}

	if err := store.PutDaggerheartDomainCard(ctx, contentstore.DaggerheartDomainCard{
		ID:          "domain_card.codex-pattern-study",
		Name:        "Pattern Study",
		DomainID:    "domain.codex",
		Level:       1,
		Type:        "spell",
		UsageLimit:  "None",
		FeatureText: "Scenario seed codex card.",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		return fmt.Errorf("seed codex domain card: %w", err)
	}

	if err := store.PutDaggerheartClass(ctx, contentstore.DaggerheartClass{
		ID:              "class.rogue",
		Name:            "Rogue",
		StartingEvasion: 10,
		StartingHP:      6,
		Features: []contentstore.DaggerheartFeature{{
			ID:               "feature.rogue-cloaked",
			Name:             "Cloaked",
			Description:      "Scenario seed Rogue feature.",
			Level:            1,
			AutomationStatus: contentstore.DaggerheartFeatureAutomationStatusSupported,
			ClassRule: &contentstore.DaggerheartClassFeatureRule{
				Kind: contentstore.DaggerheartClassFeatureRuleKindCloaked,
			},
		}},
		HopeFeature: contentstore.DaggerheartHopeFeature{
			Name:             "Rogue's Dodge",
			Description:      "Spend Hope to increase evasion until hit or rest.",
			HopeCost:         3,
			AutomationStatus: contentstore.DaggerheartFeatureAutomationStatusSupported,
			HopeFeatureRule: &contentstore.DaggerheartHopeFeatureRule{
				Kind:     contentstore.DaggerheartHopeFeatureRuleKindRoguesDodge,
				Bonus:    2,
				HopeCost: 3,
			},
		},
		DomainIDs: []string{"domain.valor"},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed rogue class: %w", err)
	}

	if err := store.PutDaggerheartSubclass(ctx, contentstore.DaggerheartSubclass{
		ID:        "subclass.nightwalker",
		Name:      "Nightwalker",
		ClassID:   "class.rogue",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed nightwalker subclass: %w", err)
	}

	if err := store.PutDaggerheartClass(ctx, contentstore.DaggerheartClass{
		ID:              "class.warrior",
		Name:            "Warrior",
		StartingEvasion: 9,
		StartingHP:      7,
		Features: []contentstore.DaggerheartFeature{{
			ID:               "feature.warrior-combat-training",
			Name:             "Combat Training",
			Description:      "Scenario seed Warrior feature.",
			Level:            1,
			AutomationStatus: contentstore.DaggerheartFeatureAutomationStatusSupported,
			ClassRule: &contentstore.DaggerheartClassFeatureRule{
				Kind:              contentstore.DaggerheartClassFeatureRuleKindCombatTraining,
				UseCharacterLevel: true,
			},
		}},
		HopeFeature: contentstore.DaggerheartHopeFeature{
			Name:             "No Mercy",
			Description:      "Spend Hope to gain +1 on attack rolls until rest.",
			HopeCost:         3,
			AutomationStatus: contentstore.DaggerheartFeatureAutomationStatusSupported,
			HopeFeatureRule: &contentstore.DaggerheartHopeFeatureRule{
				Kind:     contentstore.DaggerheartHopeFeatureRuleKindNoMercy,
				Bonus:    1,
				HopeCost: 3,
			},
		},
		DomainIDs: []string{"domain.valor"},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed warrior class: %w", err)
	}

	if err := store.PutDaggerheartSubclass(ctx, contentstore.DaggerheartSubclass{
		ID:        "subclass.call-of-the-brave",
		Name:      "Call of the Brave",
		ClassID:   "class.warrior",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed call of the brave subclass: %w", err)
	}

	if err := store.PutDaggerheartClass(ctx, contentstore.DaggerheartClass{
		ID:              "class.wizard",
		Name:            "Wizard",
		StartingEvasion: 10,
		StartingHP:      5,
		Features: []contentstore.DaggerheartFeature{{
			ID:               "feature.wizard-strange-patterns",
			Name:             "Strange Patterns",
			Description:      "Scenario seed Wizard feature.",
			Level:            1,
			AutomationStatus: contentstore.DaggerheartFeatureAutomationStatusSupported,
			ClassRule: &contentstore.DaggerheartClassFeatureRule{
				Kind: contentstore.DaggerheartClassFeatureRuleKindStrangePatterns,
			},
		}},
		HopeFeature: contentstore.DaggerheartHopeFeature{
			Name:             "Not This Time",
			Description:      "Scenario seed Wizard hope feature.",
			HopeCost:         3,
			AutomationStatus: contentstore.DaggerheartFeatureAutomationStatusUnsupported,
		},
		DomainIDs: []string{"domain.codex"},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed wizard class: %w", err)
	}

	if err := store.PutDaggerheartSubclass(ctx, contentstore.DaggerheartSubclass{
		ID:        "subclass.school-of-war",
		Name:      "School of War",
		ClassID:   "class.wizard",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed school of war subclass: %w", err)
	}

	if err := store.PutDaggerheartHeritage(ctx, contentstore.DaggerheartHeritage{
		ID:   "heritage.dwarf",
		Name: "Dwarf",
		Kind: "ancestry",
		Features: []contentstore.DaggerheartFeature{
			{
				ID:          "feature.dwarf-thick-skin",
				Name:        "Thick Skin",
				Description: "Scenario seed primary dwarf ancestry feature.",
				Level:       1,
			},
			{
				ID:          "feature.dwarf-increased-fortitude",
				Name:        "Increased Fortitude",
				Description: "Scenario seed secondary dwarf ancestry feature.",
				Level:       1,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed dwarf heritage: %w", err)
	}

	if err := store.PutDaggerheartHeritage(ctx, contentstore.DaggerheartHeritage{
		ID:   "heritage.elf",
		Name: "Elf",
		Kind: "ancestry",
		Features: []contentstore.DaggerheartFeature{
			{
				ID:          "feature.elf-quick-reactions",
				Name:        "Quick Reactions",
				Description: "Scenario seed primary elf ancestry feature.",
				Level:       1,
			},
			{
				ID:          "feature.elf-celestial-trance",
				Name:        "Celestial Trance",
				Description: "Scenario seed secondary elf ancestry feature.",
				Level:       1,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed elf heritage: %w", err)
	}

	if err := store.PutDaggerheartDomainCard(ctx, contentstore.DaggerheartDomainCard{
		ID:          "domain_card.valor-forceful-push",
		Name:        "Forceful Push",
		DomainID:    "domain.valor",
		Level:       1,
		Type:        "ability",
		UsageLimit:  "See text",
		FeatureText: "Scenario seed card",
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		return fmt.Errorf("seed domain card 2: %w", err)
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.leather-armor",
		Name:                "Leather Armor",
		Tier:                1,
		BaseMajorThreshold:  6,
		BaseSevereThreshold: 13,
		ArmorScore:          3,
		CreatedAt:           now,
		UpdatedAt:           now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.chainmail-armor",
		Name:                "Chainmail Armor",
		Tier:                1,
		BaseMajorThreshold:  7,
		BaseSevereThreshold: 15,
		ArmorScore:          4,
		Feature:             "Heavy: -1 to Evasion",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:       contentstore.DaggerheartArmorAutomationStatusSupported,
			MitigationMode:         contentstore.DaggerheartArmorMitigationModeAny,
			EvasionDelta:           -1,
			SeverityReductionSteps: 1,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.channeling-armor",
		Name:                "Channeling Armor",
		Tier:                4,
		BaseMajorThreshold:  13,
		BaseSevereThreshold: 36,
		ArmorScore:          5,
		Feature:             "Channeling: +1 to Spellcast Rolls",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:       contentstore.DaggerheartArmorAutomationStatusSupported,
			MitigationMode:         contentstore.DaggerheartArmorMitigationModeAny,
			SpellcastRollBonus:     1,
			SeverityReductionSteps: 1,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.elundrian-chain-armor",
		Name:                "Elundrian Chain Armor",
		Tier:                2,
		BaseMajorThreshold:  9,
		BaseSevereThreshold: 21,
		ArmorScore:          4,
		Feature:             "Warded: You reduce incoming magic damage by your Armor Score before applying it to your damage thresholds.",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:       contentstore.DaggerheartArmorAutomationStatusSupported,
			MitigationMode:         contentstore.DaggerheartArmorMitigationModeAny,
			SeverityReductionSteps: 1,
			WardedMagicReduction:   true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.full-fortified-armor",
		Name:                "Full Fortified Armor",
		Tier:                4,
		BaseMajorThreshold:  15,
		BaseSevereThreshold: 40,
		ArmorScore:          4,
		Feature:             "Fortified: When you mark an Armor Slot, you reduce the severity of an attack by two thresholds instead of one.",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:       contentstore.DaggerheartArmorAutomationStatusSupported,
			MitigationMode:         contentstore.DaggerheartArmorMitigationModeAny,
			SeverityReductionSteps: 2,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.runes-of-fortification",
		Name:                "Runes of Fortification",
		Tier:                3,
		BaseMajorThreshold:  17,
		BaseSevereThreshold: 43,
		ArmorScore:          6,
		Feature:             "Painful: Each time you mark an Armor Slot, you must mark a Stress.",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:       contentstore.DaggerheartArmorAutomationStatusSupported,
			MitigationMode:         contentstore.DaggerheartArmorMitigationModeAny,
			SeverityReductionSteps: 1,
			StressOnMark:           true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.irontree-breastplate-armor",
		Name:                "Irontree Breastplate Armor",
		Tier:                2,
		BaseMajorThreshold:  9,
		BaseSevereThreshold: 20,
		ArmorScore:          4,
		Feature:             "Reinforced: When you mark your last Armor Slot, increase your damage thresholds by +2 until you clear at least 1 Armor Slot.",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:                contentstore.DaggerheartArmorAutomationStatusSupported,
			MitigationMode:                  contentstore.DaggerheartArmorMitigationModeAny,
			SeverityReductionSteps:          1,
			ThresholdBonusWhenArmorDepleted: 2,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.tyris-soft-armor",
		Name:                "Tyris Soft Armor",
		Tier:                2,
		BaseMajorThreshold:  8,
		BaseSevereThreshold: 18,
		ArmorScore:          5,
		Feature:             "Quiet: You gain a +2 bonus to rolls you make to move silently.",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:       contentstore.DaggerheartArmorAutomationStatusSupported,
			MitigationMode:         contentstore.DaggerheartArmorMitigationModeAny,
			SeverityReductionSteps: 1,
			SilentMovementBonus:    2,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.rosewild-armor",
		Name:                "Rosewild Armor",
		Tier:                2,
		BaseMajorThreshold:  11,
		BaseSevereThreshold: 23,
		ArmorScore:          5,
		Feature:             "Hopeful: When you would spend a Hope, you can mark an Armor Slot instead.",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:            contentstore.DaggerheartArmorAutomationStatusSupported,
			MitigationMode:              contentstore.DaggerheartArmorMitigationModeAny,
			SeverityReductionSteps:      1,
			HopefulReplaceHopeWithArmor: true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.spiked-plate-armor",
		Name:                "Spiked Plate Armor",
		Tier:                3,
		BaseMajorThreshold:  10,
		BaseSevereThreshold: 25,
		ArmorScore:          5,
		Feature:             "Sharp: On a successful attack against a target within Melee range, add a d4 to the damage roll.",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:         contentstore.DaggerheartArmorAutomationStatusSupported,
			MitigationMode:           contentstore.DaggerheartArmorMitigationModeAny,
			SeverityReductionSteps:   1,
			SharpDamageBonusDieSides: 4,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.runetan-floating-armor",
		Name:                "Runetan Floating Armor",
		Tier:                2,
		BaseMajorThreshold:  9,
		BaseSevereThreshold: 20,
		ArmorScore:          4,
		Feature:             "Shifting: When you are targeted for an attack, you can mark an Armor Slot to give the attack roll against you disadvantage.",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:           contentstore.DaggerheartArmorAutomationStatusSupported,
			MitigationMode:             contentstore.DaggerheartArmorMitigationModeAny,
			SeverityReductionSteps:     1,
			ShiftingAttackDisadvantage: 1,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.dunamis-silkchain",
		Name:                "Dunamis Silkchain",
		Tier:                4,
		BaseMajorThreshold:  13,
		BaseSevereThreshold: 36,
		ArmorScore:          7,
		Feature:             "Timeslowing: Mark an Armor Slot to roll a d4 and add its result as a bonus to your Evasion against an incoming attack.",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:                contentstore.DaggerheartArmorAutomationStatusSupported,
			MitigationMode:                  contentstore.DaggerheartArmorMitigationModeAny,
			SeverityReductionSteps:          1,
			TimeslowingEvasionBonusDieSides: 4,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.emberwoven-armor",
		Name:                "Emberwoven Armor",
		Tier:                4,
		BaseMajorThreshold:  13,
		BaseSevereThreshold: 36,
		ArmorScore:          6,
		Feature:             "Burning: When an adversary attacks you within Melee range, they mark a Stress.",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:       contentstore.DaggerheartArmorAutomationStatusSupported,
			MitigationMode:         contentstore.DaggerheartArmorMitigationModeAny,
			SeverityReductionSteps: 1,
			BurningAttackerStress:  1,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.harrowbone-armor",
		Name:                "Harrowbone Armor",
		Tier:                2,
		BaseMajorThreshold:  9,
		BaseSevereThreshold: 21,
		ArmorScore:          4,
		Feature:             "Resilient: Before you mark your last Armor Slot, roll a d6. On a result of 6, reduce the severity by one threshold without marking an Armor Slot.",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:          contentstore.DaggerheartArmorAutomationStatusSupported,
			MitigationMode:            contentstore.DaggerheartArmorMitigationModeAny,
			SeverityReductionSteps:    1,
			ResilientDieSides:         6,
			ResilientSuccessOnOrAbove: 6,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.veritas-opal-armor",
		Name:                "Veritas Opal Armor",
		Tier:                4,
		BaseMajorThreshold:  13,
		BaseSevereThreshold: 36,
		ArmorScore:          6,
		Feature:             "Truthseeking: This armor glows when another creature within Close range tells a lie.",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:       contentstore.DaggerheartArmorAutomationStatusUnsupported,
			MitigationMode:         contentstore.DaggerheartArmorMitigationModeAny,
			SeverityReductionSteps: 1,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putArmor(contentstore.DaggerheartArmor{
		ID:                  "armor.dragonscale-armor",
		Name:                "Dragonscale Armor",
		Tier:                3,
		BaseMajorThreshold:  11,
		BaseSevereThreshold: 27,
		ArmorScore:          5,
		Feature:             "Impenetrable: Once per short rest, when you would mark your last Hit Point, you can instead mark a Stress.",
		Rules: contentstore.DaggerheartArmorRules{
			AutomationStatus:             contentstore.DaggerheartArmorAutomationStatusSupported,
			MitigationMode:               contentstore.DaggerheartArmorMitigationModeAny,
			SeverityReductionSteps:       1,
			ImpenetrableStressCost:       1,
			ImpenetrableUsesPerShortRest: 1,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putAdversary(contentstore.DaggerheartAdversaryEntry{
		ID:              "adversary.shadow-hound",
		Name:            "Shadow Hound",
		Tier:            1,
		Role:            "skulk",
		Description:     "Scenario seed adversary for typed fear-buy validation.",
		Motives:         "Isolate prey from the pack.",
		Difficulty:      12,
		MajorThreshold:  7,
		SevereThreshold: 13,
		HP:              4,
		Stress:          3,
		Armor:           0,
		AttackModifier:  2,
		StandardAttack: contentstore.DaggerheartAdversaryAttack{
			Name:        "Raking Claws",
			Range:       "melee",
			DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 8, Count: 1}},
			DamageBonus: 1,
			DamageType:  "physical",
		},
		Experiences: []contentstore.DaggerheartAdversaryExperience{
			{Name: "Pack Hunter", Modifier: 2},
		},
		Features: []contentstore.DaggerheartAdversaryFeature{
			{
				ID:          "feature.shadow-hound-pounce",
				Name:        "Pounce from Shadow",
				Kind:        "fear",
				Description: "Spend Fear to force an immediate lunge from the dark.",
				CostType:    "fear",
				Cost:        1,
			},
			{
				ID:          "feature.shadow-hound-pack-tactics",
				Name:        "Pack Tactics",
				Kind:        "passive",
				Description: "When another packmate joins the attack, the strike deals 2d10+3 physical damage instead.",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putAdversary(contentstore.DaggerheartAdversaryEntry{
		ID:              "adversary.nazgul",
		Name:            "Nazgul",
		Tier:            2,
		Role:            "solo",
		Description:     "Scenario seed relentless foe for spotlight smoke coverage.",
		Motives:         "Hunt, terrify, pursue the ring-bearer.",
		Difficulty:      14,
		MajorThreshold:  6,
		SevereThreshold: 12,
		HP:              8,
		Stress:          4,
		Armor:           0,
		AttackModifier:  3,
		StandardAttack: contentstore.DaggerheartAdversaryAttack{
			Name:        "Morgul Blade",
			Range:       "melee",
			DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 10, Count: 1}},
			DamageBonus: 2,
			DamageType:  "magic",
		},
		Features: []contentstore.DaggerheartAdversaryFeature{
			{
				ID:          "feature.nazgul-relentless-2",
				Name:        "Relentless (2)",
				Kind:        "passive",
				Description: "This adversary can be spotlighted up to twice per GM turn.",
				CostType:    "fear",
				Cost:        1,
			},
			{
				ID:          "feature.nazgul-terrifying",
				Name:        "Terrifying",
				Kind:        "passive",
				Description: "When the Nazgul makes a successful attack, all PCs within Far range lose a Hope and you gain a Fear.",
			},
		},
		RelentlessRule: &contentstore.DaggerheartAdversaryRelentlessRule{MaxSpotlightsPerGMTurn: 2},
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		return err
	}

	if err := putAdversary(contentstore.DaggerheartAdversaryEntry{
		ID:              "adversary.golum",
		Name:            "Golum",
		Tier:            1,
		Role:            "skulk",
		Description:     "Scenario seed lurker for attack-outcome smoke coverage.",
		Motives:         "Snatch, hide, flee.",
		Difficulty:      12,
		MajorThreshold:  6,
		SevereThreshold: 12,
		HP:              5,
		Stress:          2,
		Armor:           0,
		AttackModifier:  1,
		StandardAttack: contentstore.DaggerheartAdversaryAttack{
			Name:        "Jagged Bite",
			Range:       "melee",
			DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 8, Count: 1}},
			DamageBonus: 1,
			DamageType:  "physical",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putAdversary(contentstore.DaggerheartAdversaryEntry{
		ID:              "adversary.golum-shadow",
		Name:            "Golum Shadow",
		Tier:            1,
		Role:            "skulk",
		Description:     "Scenario seed skulk for cloaked backstab coverage.",
		Motives:         "Lurk, strike, scurry away.",
		Difficulty:      12,
		MajorThreshold:  6,
		SevereThreshold: 12,
		HP:              5,
		Stress:          2,
		Armor:           0,
		AttackModifier:  1,
		StandardAttack: contentstore.DaggerheartAdversaryAttack{
			Name:        "Jagged Bite",
			Range:       "melee",
			DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 8, Count: 1}},
			DamageBonus: 1,
			DamageType:  "physical",
		},
		Features: []contentstore.DaggerheartAdversaryFeature{
			{
				ID:          "feature.golum-cloaked",
				Name:        "Cloaked",
				Kind:        "passive",
				Description: "Until the next attack, this skulk remains hidden in the dark.",
			},
			{
				ID:          "feature.golum-backstab",
				Name:        "Backstab",
				Kind:        "passive",
				Description: "When this adversary attacks with advantage, its attack deals 2d8+3 physical damage instead.",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putAdversary(contentstore.DaggerheartAdversaryEntry{
		ID:              "adversary.vault-guardian-sentinel",
		Name:            "Vault Guardian Sentinel",
		Tier:            2,
		Role:            "standard",
		Description:     "Scenario seed adversary for Box In focus-target disadvantage coverage.",
		Motives:         "Guard the vault entrance; pin intruders.",
		Difficulty:      17,
		MajorThreshold:  7,
		SevereThreshold: 14,
		HP:              8,
		Stress:          3,
		Armor:           1,
		AttackModifier:  2,
		StandardAttack: contentstore.DaggerheartAdversaryAttack{
			Name:        "Halberd Sweep",
			Range:       "melee",
			DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 10, Count: 1}},
			DamageBonus: 2,
			DamageType:  "physical",
		},
		Features: []contentstore.DaggerheartAdversaryFeature{
			{
				ID:          "adversary-feature.vault-guardian-sentinel-box-in",
				Name:        "Box In",
				Kind:        "passive",
				Description: "Mark a target. That target has disadvantage on their next action roll against this adversary.",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putAdversary(contentstore.DaggerheartAdversaryEntry{
		ID:              "adversary.saruman",
		Name:            "Saruman",
		Tier:            2,
		Role:            "leader",
		Description:     "Scenario seed spellcaster for critical-damage smoke coverage.",
		Motives:         "Manipulate, command, corrupt.",
		Difficulty:      15,
		MajorThreshold:  6,
		SevereThreshold: 12,
		HP:              8,
		Stress:          4,
		Armor:           0,
		AttackModifier:  2,
		StandardAttack: contentstore.DaggerheartAdversaryAttack{
			Name:        "Arcane Bolt",
			Range:       "far",
			DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 10, Count: 1}},
			DamageBonus: 2,
			DamageType:  "magic",
		},
		Features: []contentstore.DaggerheartAdversaryFeature{
			{
				ID:          "feature.saruman-warding-sphere",
				Name:        "Warding Sphere",
				Kind:        "reaction",
				Description: "When a nearby foe lands a hit, a ward lashes back with magic.",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putAdversary(contentstore.DaggerheartAdversaryEntry{
		ID:              "adversary.galadriel",
		Name:            "Galadriel",
		Tier:            2,
		Role:            "leader",
		Description:     "Scenario seed adversary for condition and spotlight smoke coverage.",
		Motives:         "Rebuke, dazzle, reposition.",
		Difficulty:      14,
		MajorThreshold:  6,
		SevereThreshold: 12,
		HP:              7,
		Stress:          3,
		Armor:           0,
		AttackModifier:  2,
		StandardAttack: contentstore.DaggerheartAdversaryAttack{
			Name:        "Radiant Rebuke",
			Range:       "close",
			DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 8, Count: 1}},
			DamageBonus: 2,
			DamageType:  "magic",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putAdversary(contentstore.DaggerheartAdversaryEntry{
		ID:              "adversary.orc-archer",
		Name:            "Orc Archer",
		Tier:            1,
		Role:            "ranged",
		Description:     "Scenario seed ranged foe for spotlight sequencing.",
		Motives:         "Pin down, harry, reposition.",
		Difficulty:      11,
		MajorThreshold:  5,
		SevereThreshold: 10,
		HP:              4,
		Stress:          2,
		Armor:           0,
		AttackModifier:  1,
		StandardAttack: contentstore.DaggerheartAdversaryAttack{
			Name:        "Shortbow",
			Range:       "far",
			DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 8, Count: 1}},
			DamageBonus: 1,
			DamageType:  "physical",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putAdversary(contentstore.DaggerheartAdversaryEntry{
		ID:              "adversary.orc-raiders",
		Name:            "Orc Raiders",
		Tier:            1,
		Role:            "standard",
		Description:     "Scenario seed raider pack for spotlight sequencing.",
		Motives:         "Swarm, corner, overwhelm.",
		Difficulty:      11,
		MajorThreshold:  5,
		SevereThreshold: 10,
		HP:              5,
		Stress:          2,
		Armor:           0,
		AttackModifier:  1,
		StandardAttack: contentstore.DaggerheartAdversaryAttack{
			Name:        "Rusty Blade",
			Range:       "melee",
			DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 8, Count: 1}},
			DamageBonus: 1,
			DamageType:  "physical",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := putAdversary(contentstore.DaggerheartAdversaryEntry{
		ID:              "adversary.giant-rat",
		Name:            "Giant Rat",
		Tier:            1,
		Role:            "minion",
		Description:     "Scenario seed minion for spillover smoke coverage.",
		Motives:         "Swarm, bite, scatter.",
		Difficulty:      10,
		MajorThreshold:  0,
		SevereThreshold: 0,
		HP:              1,
		Stress:          1,
		Armor:           0,
		AttackModifier:  -1,
		StandardAttack: contentstore.DaggerheartAdversaryAttack{
			Name:        "Claws",
			Range:       "melee",
			DamageDice:  []contentstore.DaggerheartDamageDie{},
			DamageBonus: 1,
			DamageType:  "physical",
		},
		Features: []contentstore.DaggerheartAdversaryFeature{
			{
				ID:          "feature.giant-rat-minion-3",
				Name:        "Minion (3)",
				Kind:        "passive",
				Description: "This adversary is defeated by any damage, with spillover every 3 damage.",
			},
		},
		MinionRule: &contentstore.DaggerheartAdversaryMinionRule{SpilloverDamageStep: 3},
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		return err
	}

	if err := putAdversary(contentstore.DaggerheartAdversaryEntry{
		ID:              "adversary.swarm-of-rats",
		Name:            "Swarm of Rats",
		Tier:            1,
		Role:            "horde",
		Description:     "Scenario seed horde for bloodied-attack smoke coverage.",
		Motives:         "Swarm, consume, obscure.",
		Difficulty:      10,
		MajorThreshold:  4,
		SevereThreshold: 8,
		HP:              2,
		Stress:          1,
		Armor:           0,
		AttackModifier:  0,
		StandardAttack: contentstore.DaggerheartAdversaryAttack{
			Name:        "Ravenous Claws",
			Range:       "melee",
			DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 12, Count: 1}},
			DamageBonus: 5,
			DamageType:  "physical",
		},
		Features: []contentstore.DaggerheartAdversaryFeature{
			{
				ID:          "feature.swarm-of-rats-horde-1d4-1",
				Name:        "Horde (1d4+1)",
				Kind:        "passive",
				Description: "When bloodied, the swarm's standard attack weakens to 1d4+1.",
			},
		},
		HordeRule: &contentstore.DaggerheartAdversaryHordeRule{
			BloodiedAttack: contentstore.DaggerheartAdversaryAttack{
				Name:        "Ravenous Claws",
				Range:       "melee",
				DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 4, Count: 1}},
				DamageBonus: 1,
				DamageType:  "physical",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return err
	}

	if err := store.PutDaggerheartEnvironment(ctx, contentstore.DaggerheartEnvironment{
		ID:         "environment.crumbling-bridge",
		Name:       "Crumbling Bridge",
		Tier:       1,
		Type:       "hazard",
		Difficulty: 12,
		Impulses:   []string{"Split the party", "Turn stable footing into danger"},
		Features: []contentstore.DaggerheartFeature{
			{
				ID:          "feature.crumbling-bridge-falling-stones",
				Name:        "Falling Stones",
				Description: "Spend Fear to make the bridge shed dangerous debris.",
				Level:       1,
			},
		},
		Prompts:   []string{"Who is stranded on the weakest span?"},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed scenario environment: %w", err)
	}

	if err := seedScenarioGenericAdversaries(ctx, putAdversary, now); err != nil {
		return err
	}

	return nil
}

func seedScenarioGenericAdversaries(
	ctx context.Context,
	putAdversary func(contentstore.DaggerheartAdversaryEntry) error,
	now time.Time,
) error {
	_ = ctx
	for _, name := range []string{
		"Barrow Wight",
		"Bilbo",
		"Bree Merchant",
		"Chaos Elemental",
		"Corrupt Steward",
		"Elf Warden",
		"Ent Sapling Reinforcement",
		"Ent Saplings",
		"Ent Warden",
		"Fell Beast",
		"Goblin",
		"Goblin A",
		"Goblin B",
		"Gondor Archers",
		"Gondor Captain",
		"Gondor Guard Reinforcement",
		"Gondor Guards",
		"Gondor Knight",
		"Gondor Practice Dummy",
		"Great Eagle Scout 1",
		"Great Eagle Scout 2",
		"Great Eagles",
		"Mirkwood Archer",
		"Mirkwood Warden",
		"Moria Rat A",
		"Moria Rat B",
		"Moria Rat C",
		"Moria Rats",
		"Nameless Horror",
		"Orc Boss",
		"Orc Captain",
		"Orc Lackey",
		"Orc Lackey Reinforcement 1",
		"Orc Lackey Reinforcement 2",
		"Orc Lackeys",
		"Orc Lieutenant",
		"Orc Minions",
		"Orc Pack A",
		"Orc Pack B",
		"Orc Rabble",
		"Orc Raider",
		"Orc Raider A",
		"Orc Raider B",
		"Orc Shock Troops",
		"Orc Shock Troops Reinforcement 1",
		"Orc Shock Troops Reinforcement 2",
		"Orc Sniper",
		"Orc Stalker",
		"Orc Waylayers",
		"Ranger of the North",
		"Rotted Zombie Reinforcement",
		"Shadow Corruptor",
		"Shadow Thrall Reinforcement 1",
		"Shadow Thrall Reinforcement 2",
		"Shadow Thralls",
		"Shadow Wraith",
		"Shire Elder",
		"Uruk-hai",
		"Uruk-hai Brute",
		"Uruk-hai Minions",
		"Uruk-hai Reinforcement",
		"Uruk-hai Vanguard",
		"Warg",
		"Warg Hunter",
		"Woodland Elves",
	} {
		if err := putAdversary(genericScenarioAdversaryEntry(name, now)); err != nil {
			return err
		}
	}
	return nil
}

func genericScenarioAdversaryEntry(name string, now time.Time) contentstore.DaggerheartAdversaryEntry {
	role := genericScenarioAdversaryRole(name)
	attack := contentstore.DaggerheartAdversaryAttack{
		Name:        "Scenario Strike",
		Range:       "melee",
		DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 8, Count: 1}},
		DamageBonus: 1,
		DamageType:  "physical",
	}
	if role == "ranged" {
		attack.Name = "Scenario Shot"
		attack.Range = "far"
	}
	if strings.Contains(strings.ToLower(name), "wight") || strings.Contains(strings.ToLower(name), "shadow") || strings.Contains(strings.ToLower(name), "corrupt") {
		attack.DamageType = "magic"
	}
	entry := contentstore.DaggerheartAdversaryEntry{
		ID:              "adversary.scenario." + genericScenarioAdversarySlug(name),
		Name:            name,
		Tier:            1,
		Role:            role,
		Description:     "Scenario-only catalog entry for runtime adversary coverage.",
		Motives:         "Advance the scenario under test.",
		Difficulty:      11,
		MajorThreshold:  5,
		SevereThreshold: 10,
		HP:              4,
		Stress:          2,
		Armor:           0,
		AttackModifier:  1,
		StandardAttack:  attack,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if strings.Contains(strings.ToLower(name), "captain") || strings.Contains(strings.ToLower(name), "warden") {
		entry.Role = "leader"
	}
	if genericScenarioAdversaryIsMinion(name) {
		entry.Role = "minion"
		entry.HP = 1
		entry.MajorThreshold = 0
		entry.SevereThreshold = 0
		entry.AttackModifier = 0
		entry.StandardAttack = contentstore.DaggerheartAdversaryAttack{
			Name:        "Scenario Swarm",
			Range:       "melee",
			DamageDice:  []contentstore.DaggerheartDamageDie{{Sides: 4, Count: 1}},
			DamageBonus: 1,
			DamageType:  "physical",
		}
		entry.Features = []contentstore.DaggerheartAdversaryFeature{
			{
				ID:          entry.ID + ".minion",
				Name:        "Minion (3)",
				Kind:        "passive",
				Description: "Scenario seed minion rule.",
			},
		}
		entry.MinionRule = &contentstore.DaggerheartAdversaryMinionRule{SpilloverDamageStep: 3}
	}
	return entry
}

func genericScenarioAdversaryRole(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "archer"), strings.Contains(lower, "ranger"), strings.Contains(lower, "sniper"):
		return "ranged"
	case strings.Contains(lower, "captain"), strings.Contains(lower, "warden"), strings.Contains(lower, "boss"), strings.Contains(lower, "lieutenant"):
		return "leader"
	case strings.Contains(lower, "wight"), strings.Contains(lower, "shadow"), strings.Contains(lower, "fell beast"), strings.Contains(lower, "warg hunter"), strings.Contains(lower, "stalker"):
		return "skulk"
	default:
		return "standard"
	}
}

func genericScenarioAdversaryIsMinion(name string) bool {
	lower := strings.ToLower(name)
	for _, marker := range []string{
		"goblin",
		"rat",
		"minions",
		"lackey",
		"rabble",
		"thrall",
		"sapling",
		"reinforcement",
	} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func genericScenarioAdversarySlug(name string) string {
	slug := strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer(
		" ", "-",
		"'", "",
		",", "",
		".", "",
		"/", "-",
	)
	slug = replacer.Replace(slug)
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	return strings.Trim(slug, "-")
}

// profileSeedText preserves the prior suite-specific catalog labels during extraction.
func profileSeedText(profile ContentSeedProfile, kind string) string {
	prefix := "Shared"
	switch profile {
	case ContentSeedProfileIntegration:
		prefix = "Integration"
	case ContentSeedProfileScenario:
		prefix = "Scenario"
	}

	switch kind {
	case "domain":
		return prefix + " seed domain"
	case "card":
		return prefix + " seed card"
	case "weapon":
		return prefix + " seed weapon"
	case "armor":
		return prefix + " seed armor"
	default:
		return prefix + " seed"
	}
}

// copyFile duplicates the cached template database into the active test location.
func copyFile(src, dst string) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		_ = output.Close()
	}()

	if _, err := io.Copy(output, input); err != nil {
		return err
	}
	return output.Sync()
}
