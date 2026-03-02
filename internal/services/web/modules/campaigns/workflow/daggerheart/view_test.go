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
	if got := New(); got != (Workflow{}) {
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
			ClassID:            "class-1",
			SubclassID:         "subclass-1",
			AncestryID:         "ancestry-1",
			CommunityID:        "community-1",
			Agility:            "2",
			Strength:           "1",
			Finesse:            "0",
			Instinct:           "1",
			Presence:           "2",
			Knowledge:          "-1",
			PrimaryWeaponID:    "weapon-1",
			SecondaryWeaponID:  "weapon-2",
			ArmorID:            "armor-1",
			PotionItemID:       "item-1",
			Background:         "Scholar",
			ExperienceName:     "Wanderer",
			ExperienceModifier: "2",
			DomainCardIDs:      []string{"card-1"},
			Connections:        "Known ally",
		},
		Classes:          []campaignapp.CatalogClass{{ID: "class-1", Name: "Bard"}},
		Subclasses:       []campaignapp.CatalogSubclass{{ID: "subclass-1", Name: "Lore", ClassID: "class-1"}},
		Ancestries:       []campaignapp.CatalogHeritage{{ID: "ancestry-1", Name: "Elf"}},
		Communities:      []campaignapp.CatalogHeritage{{ID: "community-1", Name: "Loreborne"}},
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
	if view.ClassID != "class-1" || view.SecondaryWeaponID != "weapon-2" || view.ExperienceModifier != "2" {
		t.Fatalf("profile mapping mismatch: %+v", view)
	}
	if len(view.Steps) != 1 || view.Steps[0].Step != 1 || view.Steps[0].Key != "class_subclass" || !view.Steps[0].Complete {
		t.Fatalf("steps mapping mismatch: %+v", view.Steps)
	}
	if len(view.Classes) != 1 || view.Classes[0].Name != "Bard" {
		t.Fatalf("classes mapping mismatch: %+v", view.Classes)
	}
	if len(view.Subclasses) != 1 || view.Subclasses[0].ClassID != "class-1" {
		t.Fatalf("subclasses mapping mismatch: %+v", view.Subclasses)
	}
	if len(view.Ancestries) != 1 || len(view.Communities) != 1 {
		t.Fatalf("heritage mapping mismatch ancestries=%+v communities=%+v", view.Ancestries, view.Communities)
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
