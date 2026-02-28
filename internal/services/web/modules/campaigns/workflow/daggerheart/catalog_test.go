package daggerheart

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
)

func TestAssembleCatalogSortsClassesByName(t *testing.T) {
	t.Parallel()

	creation := Workflow{}.AssembleCatalog(
		campaigns.CampaignCharacterCreationProgress{},
		campaigns.CampaignCharacterCreationCatalog{
			Classes: []campaigns.CatalogClass{
				{ID: "c2", Name: "Warrior"},
				{ID: "c1", Name: "Bard"},
			},
		},
		campaigns.CampaignCharacterCreationProfile{},
	)
	if len(creation.Classes) != 2 {
		t.Fatalf("classes count = %d, want 2", len(creation.Classes))
	}
	if creation.Classes[0].ID != "c1" || creation.Classes[1].ID != "c2" {
		t.Fatalf("classes = %v, want sorted by name [Bard, Warrior]", creation.Classes)
	}
}

func TestAssembleCatalogSkipsEmptyIDs(t *testing.T) {
	t.Parallel()

	creation := Workflow{}.AssembleCatalog(
		campaigns.CampaignCharacterCreationProgress{},
		campaigns.CampaignCharacterCreationCatalog{
			Classes:    []campaigns.CatalogClass{{ID: "", Name: "Ghost"}, {ID: "c1", Name: "Bard"}},
			Subclasses: []campaigns.CatalogSubclass{{ID: "", Name: "None"}, {ID: "s1", Name: "Lore", ClassID: "c1"}},
			Heritages:  []campaigns.CatalogHeritage{{ID: "", Name: "None"}, {ID: "h1", Name: "Elf", Kind: "ancestry"}},
			Weapons:    []campaigns.CatalogWeapon{{ID: "", Name: "None", Tier: 1, Category: "primary"}},
			Armor:      []campaigns.CatalogArmor{{ID: "", Name: "None", Tier: 1}},
		},
		campaigns.CampaignCharacterCreationProfile{},
	)
	if len(creation.Classes) != 1 {
		t.Fatalf("classes = %d, want 1", len(creation.Classes))
	}
	if len(creation.Subclasses) != 1 {
		t.Fatalf("subclasses = %d, want 1", len(creation.Subclasses))
	}
	if len(creation.Ancestries) != 1 {
		t.Fatalf("ancestries = %d, want 1", len(creation.Ancestries))
	}
	if len(creation.PrimaryWeapons) != 0 {
		t.Fatalf("primary weapons = %d, want 0", len(creation.PrimaryWeapons))
	}
	if len(creation.Armor) != 0 {
		t.Fatalf("armor = %d, want 0", len(creation.Armor))
	}
}

func TestAssembleCatalogFiltersSubclassesBySelectedClass(t *testing.T) {
	t.Parallel()

	creation := Workflow{}.AssembleCatalog(
		campaigns.CampaignCharacterCreationProgress{},
		campaigns.CampaignCharacterCreationCatalog{
			Subclasses: []campaigns.CatalogSubclass{
				{ID: "s1", Name: "Lore", ClassID: "bard"},
				{ID: "s2", Name: "Battle", ClassID: "warrior"},
			},
		},
		campaigns.CampaignCharacterCreationProfile{ClassID: "bard"},
	)
	if len(creation.Subclasses) != 1 {
		t.Fatalf("subclasses = %d, want 1", len(creation.Subclasses))
	}
	if creation.Subclasses[0].ID != "s1" {
		t.Fatalf("subclass = %q, want s1", creation.Subclasses[0].ID)
	}
}

func TestAssembleCatalogIncludesAllSubclassesWhenNoClassSelected(t *testing.T) {
	t.Parallel()

	creation := Workflow{}.AssembleCatalog(
		campaigns.CampaignCharacterCreationProgress{},
		campaigns.CampaignCharacterCreationCatalog{
			Subclasses: []campaigns.CatalogSubclass{
				{ID: "s1", Name: "Lore", ClassID: "bard"},
				{ID: "s2", Name: "Battle", ClassID: "warrior"},
			},
		},
		campaigns.CampaignCharacterCreationProfile{},
	)
	if len(creation.Subclasses) != 2 {
		t.Fatalf("subclasses = %d, want 2", len(creation.Subclasses))
	}
}

func TestAssembleCatalogSplitsHeritagesByKind(t *testing.T) {
	t.Parallel()

	creation := Workflow{}.AssembleCatalog(
		campaigns.CampaignCharacterCreationProgress{},
		campaigns.CampaignCharacterCreationCatalog{
			Heritages: []campaigns.CatalogHeritage{
				{ID: "h1", Name: "Elf", Kind: "ancestry"},
				{ID: "h2", Name: "Dwarf", Kind: "ancestry"},
				{ID: "h3", Name: "Loreborne", Kind: "community"},
			},
		},
		campaigns.CampaignCharacterCreationProfile{},
	)
	if len(creation.Ancestries) != 2 {
		t.Fatalf("ancestries = %d, want 2", len(creation.Ancestries))
	}
	if len(creation.Communities) != 1 {
		t.Fatalf("communities = %d, want 1", len(creation.Communities))
	}
}

func TestAssembleCatalogFiltersWeaponsByTierAndCategory(t *testing.T) {
	t.Parallel()

	creation := Workflow{}.AssembleCatalog(
		campaigns.CampaignCharacterCreationProgress{},
		campaigns.CampaignCharacterCreationCatalog{
			Weapons: []campaigns.CatalogWeapon{
				{ID: "w1", Name: "Sword", Tier: 1, Category: "primary"},
				{ID: "w2", Name: "Dagger", Tier: 1, Category: "secondary"},
				{ID: "w3", Name: "Greatsword", Tier: 2, Category: "primary"},
			},
		},
		campaigns.CampaignCharacterCreationProfile{},
	)
	if len(creation.PrimaryWeapons) != 1 {
		t.Fatalf("primary weapons = %d, want 1", len(creation.PrimaryWeapons))
	}
	if creation.PrimaryWeapons[0].ID != "w1" {
		t.Fatalf("primary weapon = %q, want w1", creation.PrimaryWeapons[0].ID)
	}
	if len(creation.SecondaryWeapons) != 1 {
		t.Fatalf("secondary weapons = %d, want 1", len(creation.SecondaryWeapons))
	}
}

func TestAssembleCatalogFiltersArmorByTier(t *testing.T) {
	t.Parallel()

	creation := Workflow{}.AssembleCatalog(
		campaigns.CampaignCharacterCreationProgress{},
		campaigns.CampaignCharacterCreationCatalog{
			Armor: []campaigns.CatalogArmor{
				{ID: "a1", Name: "Leather", Tier: 1},
				{ID: "a2", Name: "Plate", Tier: 2},
			},
		},
		campaigns.CampaignCharacterCreationProfile{},
	)
	if len(creation.Armor) != 1 {
		t.Fatalf("armor = %d, want 1", len(creation.Armor))
	}
	if creation.Armor[0].ID != "a1" {
		t.Fatalf("armor = %q, want a1", creation.Armor[0].ID)
	}
}

func TestAssembleCatalogFiltersPotionsByAllowlist(t *testing.T) {
	t.Parallel()

	creation := Workflow{}.AssembleCatalog(
		campaigns.CampaignCharacterCreationProgress{},
		campaigns.CampaignCharacterCreationCatalog{
			Items: []campaigns.CatalogItem{
				{ID: allowedPotionMinorHealth, Name: "Minor Health Potion"},
				{ID: "item.elixir-of-power", Name: "Elixir"},
				{ID: allowedPotionMinorStamina, Name: "Minor Stamina Potion"},
			},
		},
		campaigns.CampaignCharacterCreationProfile{},
	)
	if len(creation.PotionItems) != 2 {
		t.Fatalf("potion items = %d, want 2", len(creation.PotionItems))
	}
}

func TestAssembleCatalogFiltersDomainCardsByClassDomains(t *testing.T) {
	t.Parallel()

	creation := Workflow{}.AssembleCatalog(
		campaigns.CampaignCharacterCreationProgress{},
		campaigns.CampaignCharacterCreationCatalog{
			Classes: []campaigns.CatalogClass{
				{ID: "bard", Name: "Bard", DomainIDs: []string{"arcana", "grace"}},
			},
			DomainCards: []campaigns.CatalogDomainCard{
				{ID: "dc1", Name: "Arcane Bolt", DomainID: "arcana", Level: 1},
				{ID: "dc2", Name: "Shield Wall", DomainID: "valor", Level: 1},
				{ID: "dc3", Name: "Grace Step", DomainID: "grace", Level: 1},
			},
		},
		campaigns.CampaignCharacterCreationProfile{ClassID: "bard"},
	)
	if len(creation.DomainCards) != 2 {
		t.Fatalf("domain cards = %d, want 2", len(creation.DomainCards))
	}
	if creation.DomainCards[0].ID != "dc1" || creation.DomainCards[1].ID != "dc3" {
		t.Fatalf("domain cards = %v, want [dc1, dc3]", creation.DomainCards)
	}
}

func TestAssembleCatalogIncludesAllDomainCardsWhenNoClassSelected(t *testing.T) {
	t.Parallel()

	creation := Workflow{}.AssembleCatalog(
		campaigns.CampaignCharacterCreationProgress{},
		campaigns.CampaignCharacterCreationCatalog{
			DomainCards: []campaigns.CatalogDomainCard{
				{ID: "dc1", Name: "Arcane Bolt", DomainID: "arcana", Level: 1},
				{ID: "dc2", Name: "Shield Wall", DomainID: "valor", Level: 1},
			},
		},
		campaigns.CampaignCharacterCreationProfile{},
	)
	if len(creation.DomainCards) != 2 {
		t.Fatalf("domain cards = %d, want 2", len(creation.DomainCards))
	}
}

func TestAssembleCatalogSortsDomainCardsByLevelThenName(t *testing.T) {
	t.Parallel()

	creation := Workflow{}.AssembleCatalog(
		campaigns.CampaignCharacterCreationProgress{},
		campaigns.CampaignCharacterCreationCatalog{
			DomainCards: []campaigns.CatalogDomainCard{
				{ID: "dc3", Name: "Zephyr", DomainID: "grace", Level: 1},
				{ID: "dc2", Name: "Arcane Bolt", DomainID: "arcana", Level: 2},
				{ID: "dc1", Name: "Arcane Shield", DomainID: "arcana", Level: 1},
			},
		},
		campaigns.CampaignCharacterCreationProfile{},
	)
	if len(creation.DomainCards) != 3 {
		t.Fatalf("domain cards = %d, want 3", len(creation.DomainCards))
	}
	// Level 1 first (Arcane Shield, Zephyr), then level 2 (Arcane Bolt)
	if creation.DomainCards[0].ID != "dc1" {
		t.Fatalf("domain cards[0] = %q, want dc1 (Arcane Shield, level 1)", creation.DomainCards[0].ID)
	}
	if creation.DomainCards[1].ID != "dc3" {
		t.Fatalf("domain cards[1] = %q, want dc3 (Zephyr, level 1)", creation.DomainCards[1].ID)
	}
	if creation.DomainCards[2].ID != "dc2" {
		t.Fatalf("domain cards[2] = %q, want dc2 (Arcane Bolt, level 2)", creation.DomainCards[2].ID)
	}
}

func TestAssembleCatalogTrimsProfileFields(t *testing.T) {
	t.Parallel()

	creation := Workflow{}.AssembleCatalog(
		campaigns.CampaignCharacterCreationProgress{},
		campaigns.CampaignCharacterCreationCatalog{},
		campaigns.CampaignCharacterCreationProfile{
			ClassID:       "  bard  ",
			SubclassID:    "  lore  ",
			Background:    "  noble  ",
			DomainCardIDs: []string{"  dc1  ", "", "dc2"},
		},
	)
	if creation.Profile.ClassID != "bard" {
		t.Fatalf("ClassID = %q, want %q", creation.Profile.ClassID, "bard")
	}
	if creation.Profile.SubclassID != "lore" {
		t.Fatalf("SubclassID = %q, want %q", creation.Profile.SubclassID, "lore")
	}
	if creation.Profile.Background != "noble" {
		t.Fatalf("Background = %q, want %q", creation.Profile.Background, "noble")
	}
	if len(creation.Profile.DomainCardIDs) != 2 {
		t.Fatalf("DomainCardIDs = %v, want 2 items", creation.Profile.DomainCardIDs)
	}
	if creation.Profile.DomainCardIDs[0] != "dc1" {
		t.Fatalf("DomainCardIDs[0] = %q, want %q", creation.Profile.DomainCardIDs[0], "dc1")
	}
}

func TestAssembleCatalogUsesIDAsNameFallback(t *testing.T) {
	t.Parallel()

	creation := Workflow{}.AssembleCatalog(
		campaigns.CampaignCharacterCreationProgress{},
		campaigns.CampaignCharacterCreationCatalog{
			Classes: []campaigns.CatalogClass{{ID: "bard", Name: ""}},
		},
		campaigns.CampaignCharacterCreationProfile{},
	)
	if len(creation.Classes) != 1 {
		t.Fatalf("classes = %d, want 1", len(creation.Classes))
	}
	if creation.Classes[0].Name != "bard" {
		t.Fatalf("class name = %q, want %q (fallback to ID)", creation.Classes[0].Name, "bard")
	}
}

func TestAssembleCatalogCopiesProgressFields(t *testing.T) {
	t.Parallel()

	progress := campaigns.CampaignCharacterCreationProgress{
		Steps:        []campaigns.CampaignCharacterCreationStep{{Step: 1, Key: "class_subclass", Complete: true}},
		NextStep:     2,
		Ready:        false,
		UnmetReasons: []string{"missing heritage"},
	}
	creation := Workflow{}.AssembleCatalog(
		progress,
		campaigns.CampaignCharacterCreationCatalog{},
		campaigns.CampaignCharacterCreationProfile{},
	)
	if creation.Progress.NextStep != 2 {
		t.Fatalf("NextStep = %d, want 2", creation.Progress.NextStep)
	}
	if len(creation.Progress.Steps) != 1 || creation.Progress.Steps[0].Key != "class_subclass" {
		t.Fatalf("Steps = %v, want [{1 class_subclass true}]", creation.Progress.Steps)
	}
	if len(creation.Progress.UnmetReasons) != 1 || creation.Progress.UnmetReasons[0] != "missing heritage" {
		t.Fatalf("UnmetReasons = %v, want [missing heritage]", creation.Progress.UnmetReasons)
	}
}
