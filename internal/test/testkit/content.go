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

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
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
		DomainIDs:       []string{"domain.valor"},
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		return fmt.Errorf("seed class: %w", err)
	}

	if err := store.PutDaggerheartSubclass(ctx, contentstore.DaggerheartSubclass{
		ID:        "subclass.stalwart",
		Name:      "Stalwart",
		ClassID:   "class.guardian",
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		return fmt.Errorf("seed subclass: %w", err)
	}

	if err := store.PutDaggerheartHeritage(ctx, contentstore.DaggerheartHeritage{
		ID:        "heritage.human",
		Name:      "Human",
		Kind:      "ancestry",
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

	return nil
}

// writeScenarioSeedData extends the shared seed with scenario-specific IDs.
func writeScenarioSeedData(ctx context.Context, store contentstore.DaggerheartContentWriteStore, now time.Time) error {
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

	return nil
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
