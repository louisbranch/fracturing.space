package daggerheart

import (
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
)

func TestIsDaggerheartSystemAndNewWorkflow(t *testing.T) {
	t.Parallel()

	if !IsDaggerheartSystem(" daggerheart ") {
		t.Fatalf("IsDaggerheartSystem() = false, want true for canonical label")
	}
	if !IsDaggerheartSystem("DAGGERHEART") {
		t.Fatalf("IsDaggerheartSystem() = false, want true for case-insensitive label")
	}
	if IsDaggerheartSystem("fate") {
		t.Fatalf("IsDaggerheartSystem() = true, want false for different system")
	}
	if got := New(""); got != (Workflow{}) {
		t.Fatalf("New() = %+v, want zero-value Workflow", got)
	}
}

func TestCreationViewMapsDomainModelAndCopiesSlices(t *testing.T) {
	t.Parallel()

	creation := campaignapp.CampaignCharacterCreation{
		Progress: campaignapp.CampaignCharacterCreationProgress{
			Ready:        true,
			NextStep:     4,
			UnmetReasons: []string{"choose-class"},
			Steps: []campaignapp.CampaignCharacterCreationStep{
				{Step: 1, Key: "class_subclass", Complete: true},
			},
		},
		Profile: campaignapp.CampaignCharacterCreationProfile{
			ClassID:           "class-1",
			SubclassID:        "subclass-1",
			AncestryID:        "ancestry-1",
			CommunityID:       "community-1",
			Agility:           "2",
			Strength:          "1",
			Finesse:           "0",
			Instinct:          "1",
			Presence:          "2",
			Knowledge:         "-1",
			PrimaryWeaponID:   "weapon-1",
			SecondaryWeaponID: "weapon-2",
			ArmorID:           "armor-1",
			PotionItemID:      "item-1",
			Background:        "Scholar",
			Experiences: []campaignapp.CampaignCharacterCreationExperience{
				{Name: "Wanderer", Modifier: "2"},
			},
			DomainCardIDs: []string{"card-1"},
			Connections:   "Known ally",
		},
		Classes: []campaignapp.CatalogClass{{
			ID:          "class-1",
			Name:        "Bard",
			HopeFeature: campaignapp.CatalogFeature{Name: "Make a Scene", Description: "Spend 3 Hope to force an NPC to make a scene."},
			Features: []campaignapp.CatalogFeature{
				{Name: "Bardic Knowledge", Description: "You have advantage on knowledge checks related to lore."},
			},
		}},
		Subclasses: []campaignapp.CatalogSubclass{{
			ID:      "subclass-1",
			Name:    "Lore",
			ClassID: "class-1",
			Foundation: []campaignapp.CatalogFeature{
				{Name: "Lore Master", Description: "You gain advantage on recall checks."},
			},
		}},
		Ancestries: []campaignapp.CatalogHeritage{{
			ID:   "ancestry-1",
			Name: "Elf",
			Features: []campaignapp.CatalogFeature{
				{Name: "Darkvision", Description: "You can see in darkness as if it were dim light."},
			},
		}},
		Communities: []campaignapp.CatalogHeritage{{
			ID:   "community-1",
			Name: "Loreborne",
			Features: []campaignapp.CatalogFeature{
				{Name: "Bookworm", Description: "You gain advantage on knowledge recall."},
			},
		}},
		PrimaryWeapons:   []campaignapp.CatalogWeapon{{ID: "weapon-1", Name: "Sword"}},
		SecondaryWeapons: []campaignapp.CatalogWeapon{{ID: "weapon-2", Name: "Dagger"}},
		Armor:            []campaignapp.CatalogArmor{{ID: "armor-1", Name: "Leather"}},
		PotionItems:      []campaignapp.CatalogItem{{ID: "item-1", Name: "Minor Potion"}},
		DomainCards:      []campaignapp.CatalogDomainCard{{ID: "card-1", Name: "Arcane Bolt", DomainID: "arcana", Level: 1}},
	}

	view := Workflow{}.CreationView(creation)

	if !view.Ready || view.NextStep != 4 {
		t.Fatalf("progress mapping mismatch: %+v", view)
	}
	if view.ClassID != "class-1" || view.SecondaryWeaponID != "weapon-2" {
		t.Fatalf("profile mapping mismatch: %+v", view)
	}
	if len(view.Experiences) != 1 || view.Experiences[0].Name != "Wanderer" || view.Experiences[0].Modifier != "2" {
		t.Fatalf("experience mapping mismatch: %+v", view.Experiences)
	}
	if len(view.Steps) != 1 || view.Steps[0].Step != 1 || view.Steps[0].Key != "class_subclass" || !view.Steps[0].Complete {
		t.Fatalf("steps mapping mismatch: %+v", view.Steps)
	}
	if len(view.Classes) != 1 || view.Classes[0].Name != "Bard" {
		t.Fatalf("classes mapping mismatch: %+v", view.Classes)
	}
	if view.Classes[0].HopeFeature.Name != "Make a Scene" || view.Classes[0].HopeFeature.Description == "" {
		t.Fatalf("hope feature mapping mismatch: %+v", view.Classes[0].HopeFeature)
	}
	if len(view.Classes[0].Features) != 1 || view.Classes[0].Features[0].Name != "Bardic Knowledge" || view.Classes[0].Features[0].Description == "" {
		t.Fatalf("class features mapping mismatch: %+v", view.Classes[0].Features)
	}
	if len(view.Subclasses) != 1 || view.Subclasses[0].ClassID != "class-1" {
		t.Fatalf("subclasses mapping mismatch: %+v", view.Subclasses)
	}
	if len(view.Subclasses[0].Foundation) != 1 || view.Subclasses[0].Foundation[0].Name != "Lore Master" || view.Subclasses[0].Foundation[0].Description == "" {
		t.Fatalf("subclass foundation mapping mismatch: %+v", view.Subclasses[0].Foundation)
	}
	if len(view.Ancestries) != 1 || len(view.Communities) != 1 {
		t.Fatalf("heritage mapping mismatch ancestries=%+v communities=%+v", view.Ancestries, view.Communities)
	}
	if len(view.Ancestries[0].Features) != 1 || view.Ancestries[0].Features[0].Name != "Darkvision" || view.Ancestries[0].Features[0].Description == "" {
		t.Fatalf("ancestry features mapping mismatch: %+v", view.Ancestries[0].Features)
	}
	if len(view.Communities[0].Features) != 1 || view.Communities[0].Features[0].Name != "Bookworm" || view.Communities[0].Features[0].Description == "" {
		t.Fatalf("community features mapping mismatch: %+v", view.Communities[0].Features)
	}
	if len(view.PrimaryWeapons) != 1 || len(view.SecondaryWeapons) != 1 || len(view.Armor) != 1 || len(view.PotionItems) != 1 {
		t.Fatalf("equipment mapping mismatch: primary=%+v secondary=%+v armor=%+v potions=%+v", view.PrimaryWeapons, view.SecondaryWeapons, view.Armor, view.PotionItems)
	}
	if len(view.DomainCards) != 1 || view.DomainCards[0].DomainID != "arcana" || view.DomainCards[0].Level != 1 {
		t.Fatalf("domain card mapping mismatch: %+v", view.DomainCards)
	}

	creation.Progress.UnmetReasons[0] = "changed"
	creation.Profile.DomainCardIDs[0] = "changed"
	if view.UnmetReasons[0] != "choose-class" {
		t.Fatalf("UnmetReasons should be copied, got %+v", view.UnmetReasons)
	}
	if view.DomainCardIDs[0] != "card-1" {
		t.Fatalf("DomainCardIDs should be copied, got %+v", view.DomainCardIDs)
	}
}

func TestCreationViewResolvesClassImageURL(t *testing.T) {
	t.Parallel()

	creation := campaignapp.CampaignCharacterCreation{
		Classes: []campaignapp.CatalogClass{{ID: "class.bard", Name: "Bard"}},
	}

	// Without AssetBaseURL, ImageURL should be empty.
	viewNoURL := New("").CreationView(creation)
	if len(viewNoURL.Classes) != 1 || viewNoURL.Classes[0].ImageURL != "" {
		t.Fatalf("expected empty ImageURL without AssetBaseURL, got %q", viewNoURL.Classes[0].ImageURL)
	}

	// With AssetBaseURL, ImageURL should be populated for a known class ID.
	viewWithURL := New("https://res.cloudinary.com/test/image/upload").CreationView(creation)
	if len(viewWithURL.Classes) != 1 || viewWithURL.Classes[0].ImageURL == "" {
		t.Fatalf("expected non-empty ImageURL with AssetBaseURL, got %q", viewWithURL.Classes[0].ImageURL)
	}
}

func TestCreationViewResolvesAncestryImageURL(t *testing.T) {
	t.Parallel()

	creation := campaignapp.CampaignCharacterCreation{
		Ancestries: []campaignapp.CatalogHeritage{{ID: "heritage.elf", Name: "Elf", Kind: "ancestry"}},
	}

	viewNoURL := New("").CreationView(creation)
	if len(viewNoURL.Ancestries) != 1 || viewNoURL.Ancestries[0].ImageURL != "" {
		t.Fatalf("expected empty ImageURL without AssetBaseURL, got %q", viewNoURL.Ancestries[0].ImageURL)
	}

	viewWithURL := New("https://res.cloudinary.com/test/image/upload").CreationView(creation)
	if len(viewWithURL.Ancestries) != 1 || viewWithURL.Ancestries[0].ImageURL == "" {
		t.Fatalf("expected non-empty ImageURL with AssetBaseURL, got %q", viewWithURL.Ancestries[0].ImageURL)
	}
}

func TestCreationViewResolvesCommunityImageURL(t *testing.T) {
	t.Parallel()

	creation := campaignapp.CampaignCharacterCreation{
		Communities: []campaignapp.CatalogHeritage{{ID: "heritage.loreborne", Name: "Loreborne", Kind: "community"}},
	}

	viewNoURL := New("").CreationView(creation)
	if len(viewNoURL.Communities) != 1 || viewNoURL.Communities[0].ImageURL != "" {
		t.Fatalf("expected empty ImageURL without AssetBaseURL, got %q", viewNoURL.Communities[0].ImageURL)
	}

	viewWithURL := New("https://res.cloudinary.com/test/image/upload").CreationView(creation)
	if len(viewWithURL.Communities) != 1 || viewWithURL.Communities[0].ImageURL == "" {
		t.Fatalf("expected non-empty ImageURL with AssetBaseURL, got %q", viewWithURL.Communities[0].ImageURL)
	}
}

func TestCreationViewResolvesSubclassImageURL(t *testing.T) {
	t.Parallel()

	creation := campaignapp.CampaignCharacterCreation{
		Subclasses: []campaignapp.CatalogSubclass{{ID: "subclass.stalwart", Name: "Stalwart", ClassID: "class.guardian"}},
	}

	// Without AssetBaseURL, ImageURL should be empty.
	viewNoURL := New("").CreationView(creation)
	if len(viewNoURL.Subclasses) != 1 || viewNoURL.Subclasses[0].ImageURL != "" {
		t.Fatalf("expected empty ImageURL without AssetBaseURL, got %q", viewNoURL.Subclasses[0].ImageURL)
	}

	// With AssetBaseURL, ImageURL should be populated for a known subclass ID.
	viewWithURL := New("https://res.cloudinary.com/test/image/upload").CreationView(creation)
	if len(viewWithURL.Subclasses) != 1 || viewWithURL.Subclasses[0].ImageURL == "" {
		t.Fatalf("expected non-empty ImageURL with AssetBaseURL, got %q", viewWithURL.Subclasses[0].ImageURL)
	}
}
