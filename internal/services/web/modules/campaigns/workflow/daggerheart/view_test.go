package daggerheart

import (
	"strings"
	"testing"

	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
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

	creation := catalogCreation{
		Progress: campaignworkflow.Progress{
			Ready:        true,
			NextStep:     4,
			UnmetReasons: []string{"choose-class"},
			Steps: []campaignworkflow.Step{
				{Step: 1, Key: "class_subclass", Complete: true},
			},
		},
		Profile: campaignworkflow.Profile{
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
			Experiences: []campaignworkflow.Experience{
				{Name: "Wanderer", Modifier: "2"},
			},
			DomainCardIDs: []string{"card-1"},
			Connections:   "Known ally",
		},
		Classes: []campaignworkflow.Class{{
			ID:          "class-1",
			Name:        "Bard",
			DomainIDs:   []string{"domain.sage", "domain.arcana"},
			HopeFeature: campaignworkflow.Feature{Name: "Make a Scene", Description: "Spend 3 Hope to force an NPC to make a scene."},
			Features: []campaignworkflow.Feature{
				{Name: "Bardic Knowledge", Description: "You have advantage on knowledge checks related to lore."},
			},
		}},
		Subclasses: []campaignworkflow.Subclass{{
			ID:      "subclass-1",
			Name:    "Lore",
			ClassID: "class-1",
			Foundation: []campaignworkflow.Feature{
				{Name: "Lore Master", Description: "You gain advantage on recall checks."},
			},
		}},
		Ancestries: []campaignworkflow.Heritage{{
			ID:   "ancestry-1",
			Name: "Elf",
			Features: []campaignworkflow.Feature{
				{Name: "Darkvision", Description: "You can see in darkness as if it were dim light."},
			},
		}},
		Communities: []campaignworkflow.Heritage{{
			ID:   "community-1",
			Name: "Loreborne",
			Features: []campaignworkflow.Feature{
				{Name: "Bookworm", Description: "You gain advantage on knowledge recall."},
			},
		}},
		PrimaryWeapons:   []campaignworkflow.Weapon{{ID: "weapon-1", Name: "Sword", Burden: 2}},
		SecondaryWeapons: []campaignworkflow.Weapon{{ID: "weapon-2", Name: "Dagger", Burden: 1}},
		Armor:            []campaignworkflow.Armor{{ID: "armor-1", Name: "Leather"}},
		PotionItems:      []campaignworkflow.Item{{ID: "item-1", Name: "Minor Potion"}},
		DomainCards:      []campaignworkflow.DomainCard{{ID: "card-1", Name: "Arcane Bolt", DomainID: "arcana", Level: 1}},
		Domains: []campaignworkflow.Domain{
			{ID: "domain.sage", Name: "Sage", Icon: campaignworkflow.AssetReference{URL: "https://cdn.example.com/domain/sage-icon.png"}},
			{ID: "domain.arcana", Name: "Arcana", Icon: campaignworkflow.AssetReference{URL: "https://cdn.example.com/domain/arcana-icon.png"}},
		},
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
	if len(view.Classes[0].DomainNames) != 2 || view.Classes[0].DomainNames[0] != "Sage" || view.Classes[0].DomainNames[1] != "Arcana" {
		t.Fatalf("class domain names mapping mismatch: %+v", view.Classes[0].DomainNames)
	}
	if len(view.Classes[0].DomainWatermarks) != 2 {
		t.Fatalf("class domain watermarks = %d, want 2", len(view.Classes[0].DomainWatermarks))
	}
	if view.Classes[0].DomainWatermarks[0].ID != "domain.sage" || view.Classes[0].DomainWatermarks[0].IconURL != "https://cdn.example.com/domain/sage-icon.png" {
		t.Fatalf("first class domain watermark mismatch: %+v", view.Classes[0].DomainWatermarks[0])
	}
	if view.Classes[0].DomainWatermarks[1].ID != "domain.arcana" || view.Classes[0].DomainWatermarks[1].IconURL != "https://cdn.example.com/domain/arcana-icon.png" {
		t.Fatalf("second class domain watermark mismatch: %+v", view.Classes[0].DomainWatermarks[1])
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
	if view.PrimaryWeapons[0].Burden != 2 || view.SecondaryWeapons[0].Burden != 1 {
		t.Fatalf("weapon burden mapping mismatch: primary=%+v secondary=%+v", view.PrimaryWeapons, view.SecondaryWeapons)
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

	creation := catalogCreation{
		Classes: []campaignworkflow.Class{{ID: "class.bard", Name: "Bard"}},
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

	creation := catalogCreation{
		Ancestries: []campaignworkflow.Heritage{{ID: "heritage.elf", Name: "Elf", Kind: "ancestry"}},
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

	creation := catalogCreation{
		Communities: []campaignworkflow.Heritage{{ID: "heritage.loreborne", Name: "Loreborne", Kind: "community"}},
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

func TestCreationViewAddsHeritagePrefetchURLsForClassStep(t *testing.T) {
	t.Parallel()

	creation := catalogCreation{
		Progress: campaignworkflow.Progress{NextStep: 1},
		Ancestries: []campaignworkflow.Heritage{
			{ID: "heritage.elf", Name: "Elf", Kind: "ancestry"},
		},
		Communities: []campaignworkflow.Heritage{
			{ID: "heritage.loreborne", Name: "Loreborne", Kind: "community"},
		},
	}

	view := New("https://res.cloudinary.com/test/image/upload").CreationView(creation)
	if len(view.NextStepPrefetchURLs) != 2 {
		t.Fatalf("len(NextStepPrefetchURLs) = %d, want 2", len(view.NextStepPrefetchURLs))
	}
	for _, got := range view.NextStepPrefetchURLs {
		if got == "" {
			t.Fatalf("NextStepPrefetchURLs contained empty entry: %+v", view.NextStepPrefetchURLs)
		}
	}
}

func TestCreationViewAddsEquipmentPrefetchURLsForTraitsStep(t *testing.T) {
	t.Parallel()

	creation := catalogCreation{
		Progress: campaignworkflow.Progress{NextStep: 3},
		PrimaryWeapons: []campaignworkflow.Weapon{
			{ID: "weapon-1", Name: "Sword", Illustration: campaignworkflow.AssetReference{URL: "https://cdn.example.com/weapon-1.png"}},
		},
		SecondaryWeapons: []campaignworkflow.Weapon{
			{ID: "weapon-2", Name: "Dagger", Illustration: campaignworkflow.AssetReference{URL: "https://cdn.example.com/weapon-2.png"}},
		},
		Armor: []campaignworkflow.Armor{
			{ID: "armor-1", Name: "Leather", Illustration: campaignworkflow.AssetReference{URL: "https://cdn.example.com/armor-1.png"}},
		},
		PotionItems: []campaignworkflow.Item{
			{ID: "item-1", Name: "Potion", Illustration: campaignworkflow.AssetReference{URL: "https://cdn.example.com/item-1.png"}},
		},
	}

	view := New("").CreationView(creation)
	if len(view.NextStepPrefetchURLs) != 4 {
		t.Fatalf("len(NextStepPrefetchURLs) = %d, want 4", len(view.NextStepPrefetchURLs))
	}
}

func TestCreationViewAddsDomainCardPrefetchURLsForExperiencesStep(t *testing.T) {
	t.Parallel()

	creation := catalogCreation{
		Progress: campaignworkflow.Progress{NextStep: 5},
		DomainCards: []campaignworkflow.DomainCard{
			{ID: "card-1", Name: "Arcane Bolt", Illustration: campaignworkflow.AssetReference{URL: "https://cdn.example.com/card-1.png"}},
			{ID: "card-2", Name: "Arcane Shield", Illustration: campaignworkflow.AssetReference{URL: "https://cdn.example.com/card-2.png"}},
		},
	}

	view := New("").CreationView(creation)
	if len(view.NextStepPrefetchURLs) != 2 {
		t.Fatalf("len(NextStepPrefetchURLs) = %d, want 2", len(view.NextStepPrefetchURLs))
	}
}

func TestCreationViewResolvesSubclassImageURL(t *testing.T) {
	t.Parallel()

	creation := catalogCreation{
		Subclasses: []campaignworkflow.Subclass{{ID: "subclass.stalwart", Name: "Stalwart", ClassID: "class.guardian"}},
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

func TestCreationViewUsesCatalogEquipmentIllustrationURLs(t *testing.T) {
	t.Parallel()

	creation := catalogCreation{
		PrimaryWeapons: []campaignworkflow.Weapon{{
			ID:   "weapon.battleaxe",
			Name: "Battleaxe",
			Illustration: campaignworkflow.AssetReference{
				URL: "https://cdn.example.com/weapons/battleaxe.png",
			},
		}},
		Armor: []campaignworkflow.Armor{{
			ID:   "armor.chainmail-armor",
			Name: "Chainmail Armor",
			Illustration: campaignworkflow.AssetReference{
				URL: "https://cdn.example.com/armor/chainmail.png",
			},
		}},
		PotionItems: []campaignworkflow.Item{{
			ID:   "item.minor-health-potion",
			Name: "Minor Health Potion",
			Illustration: campaignworkflow.AssetReference{
				URL: "https://cdn.example.com/items/minor-health-potion.png",
			},
		}},
	}

	view := New("").CreationView(creation)
	if len(view.PrimaryWeapons) != 1 || view.PrimaryWeapons[0].ImageURL != "https://cdn.example.com/weapons/battleaxe.png" {
		t.Fatalf("weapon image url = %q, want %q", view.PrimaryWeapons[0].ImageURL, "https://cdn.example.com/weapons/battleaxe.png")
	}
	if len(view.Armor) != 1 || view.Armor[0].ImageURL != "https://cdn.example.com/armor/chainmail.png" {
		t.Fatalf("armor image url = %q, want %q", view.Armor[0].ImageURL, "https://cdn.example.com/armor/chainmail.png")
	}
	if len(view.PotionItems) != 1 || view.PotionItems[0].ImageURL != "https://cdn.example.com/items/minor-health-potion.png" {
		t.Fatalf("item image url = %q, want %q", view.PotionItems[0].ImageURL, "https://cdn.example.com/items/minor-health-potion.png")
	}
}

func TestCreationViewResolvesEquipmentImageURLFallback(t *testing.T) {
	t.Parallel()

	creation := catalogCreation{
		PrimaryWeapons: []campaignworkflow.Weapon{{ID: "weapon.battleaxe", Name: "Battleaxe"}},
		Armor:          []campaignworkflow.Armor{{ID: "armor.chainmail-armor", Name: "Chainmail Armor"}},
		PotionItems:    []campaignworkflow.Item{{ID: "item.minor-health-potion", Name: "Minor Health Potion"}},
	}

	viewNoURL := New("").CreationView(creation)
	if len(viewNoURL.PrimaryWeapons) != 1 || viewNoURL.PrimaryWeapons[0].ImageURL != "" {
		t.Fatalf("expected empty weapon image url without AssetBaseURL, got %q", viewNoURL.PrimaryWeapons[0].ImageURL)
	}
	if len(viewNoURL.Armor) != 1 || viewNoURL.Armor[0].ImageURL != "" {
		t.Fatalf("expected empty armor image url without AssetBaseURL, got %q", viewNoURL.Armor[0].ImageURL)
	}
	if len(viewNoURL.PotionItems) != 1 || viewNoURL.PotionItems[0].ImageURL != "" {
		t.Fatalf("expected empty item image url without AssetBaseURL, got %q", viewNoURL.PotionItems[0].ImageURL)
	}

	viewWithURL := New("https://res.cloudinary.com/test/image/upload").CreationView(creation)
	if len(viewWithURL.PrimaryWeapons) != 1 || viewWithURL.PrimaryWeapons[0].ImageURL == "" {
		t.Fatalf("expected non-empty weapon image url with AssetBaseURL, got %q", viewWithURL.PrimaryWeapons[0].ImageURL)
	}
	if len(viewWithURL.Armor) != 1 || viewWithURL.Armor[0].ImageURL == "" {
		t.Fatalf("expected non-empty armor image url with AssetBaseURL, got %q", viewWithURL.Armor[0].ImageURL)
	}
	if len(viewWithURL.PotionItems) != 1 || viewWithURL.PotionItems[0].ImageURL == "" {
		t.Fatalf("expected non-empty item image url with AssetBaseURL, got %q", viewWithURL.PotionItems[0].ImageURL)
	}
}

func TestCreationViewResolvesSecondaryWeaponNoneImageURL(t *testing.T) {
	t.Parallel()

	viewNoURL := New("").CreationView(catalogCreation{})
	if viewNoURL.SecondaryWeaponNoneImageURL != "" {
		t.Fatalf("expected empty secondary none image url without AssetBaseURL, got %q", viewNoURL.SecondaryWeaponNoneImageURL)
	}

	viewWithURL := New("https://res.cloudinary.com/test/image/upload").CreationView(catalogCreation{})
	if viewWithURL.SecondaryWeaponNoneImageURL == "" {
		t.Fatal("expected non-empty secondary none image url with AssetBaseURL")
	}
	if !strings.Contains(viewWithURL.SecondaryWeaponNoneImageURL, "no_secondary_weapon") {
		t.Fatalf("secondary none image url = %q, want substring %q", viewWithURL.SecondaryWeaponNoneImageURL, "no_secondary_weapon")
	}
}

func TestCreationViewClassDomainWatermarksSkipMissingIconsAndCapAtTwo(t *testing.T) {
	t.Parallel()

	creation := catalogCreation{
		Classes: []campaignworkflow.Class{
			{
				ID:        "class.druid",
				Name:      "Druid",
				DomainIDs: []string{"domain.sage", "domain.arcana", "domain.bone"},
			},
		},
		Domains: []campaignworkflow.Domain{
			{ID: "domain.sage", Name: "Sage", Icon: campaignworkflow.AssetReference{URL: "https://cdn.example.com/domain/sage.png"}},
			{ID: "domain.arcana", Name: "Arcana", Icon: campaignworkflow.AssetReference{URL: ""}},
			{ID: "domain.bone", Name: "Bone", Icon: campaignworkflow.AssetReference{URL: "https://cdn.example.com/domain/bone.png"}},
		},
	}

	view := Workflow{}.CreationView(creation)
	if len(view.Classes) != 1 {
		t.Fatalf("classes = %d, want 1", len(view.Classes))
	}
	if len(view.Classes[0].DomainWatermarks) != 2 {
		t.Fatalf("domain watermarks = %d, want 2", len(view.Classes[0].DomainWatermarks))
	}
	if view.Classes[0].DomainWatermarks[0].ID != "domain.sage" {
		t.Fatalf("first domain watermark id = %q, want %q", view.Classes[0].DomainWatermarks[0].ID, "domain.sage")
	}
	if view.Classes[0].DomainWatermarks[1].ID != "domain.bone" {
		t.Fatalf("second domain watermark id = %q, want %q", view.Classes[0].DomainWatermarks[1].ID, "domain.bone")
	}
}

func TestCreationViewUsesCatalogDomainCardIllustrationURL(t *testing.T) {
	t.Parallel()

	creation := catalogCreation{
		DomainCards: []campaignworkflow.DomainCard{
			{
				ID:   "domain_card.arcana-runeward",
				Name: "Runeward",
				Illustration: campaignworkflow.AssetReference{
					URL: "https://cdn.example.com/domain-cards/runeward.png",
				},
			},
		},
	}

	view := New("").CreationView(creation)
	if len(view.DomainCards) != 1 {
		t.Fatalf("domain cards = %d, want 1", len(view.DomainCards))
	}
	if view.DomainCards[0].ImageURL != "https://cdn.example.com/domain-cards/runeward.png" {
		t.Fatalf("domain card image url = %q, want %q", view.DomainCards[0].ImageURL, "https://cdn.example.com/domain-cards/runeward.png")
	}
}

func TestCreationViewResolvesDomainCardImageURLFallback(t *testing.T) {
	t.Parallel()

	creation := catalogCreation{
		DomainCards: []campaignworkflow.DomainCard{
			{
				ID:   "domain_card.arcana-runeward",
				Name: "Runeward",
			},
		},
	}

	viewNoURL := New("").CreationView(creation)
	if len(viewNoURL.DomainCards) != 1 || viewNoURL.DomainCards[0].ImageURL != "" {
		t.Fatalf("expected empty domain card image url without AssetBaseURL, got %q", viewNoURL.DomainCards[0].ImageURL)
	}

	viewWithURL := New("https://res.cloudinary.com/test/image/upload").CreationView(creation)
	if len(viewWithURL.DomainCards) != 1 || viewWithURL.DomainCards[0].ImageURL == "" {
		t.Fatalf("expected non-empty domain card image url with AssetBaseURL, got %q", viewWithURL.DomainCards[0].ImageURL)
	}
}
