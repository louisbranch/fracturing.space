package render

import (
	"testing"

	campaignworkflow "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/workflow"
)

func TestNewCharacterCreationPageViewAdaptsWorkflowPageData(t *testing.T) {
	t.Parallel()

	page := campaignworkflow.PageData{
		CharacterName: "Aria",
		Creation: campaignworkflow.CharacterCreationView{
			NextStep: 2,
			ClassID:  "class-1",
			Classes: []campaignworkflow.CreationClassView{
				{ID: "class-1", Name: "Bard"},
			},
		},
	}

	view := NewCharacterCreationPageView("campaign-1", "character-1", page)
	if view.CampaignID != "campaign-1" || view.CharacterID != "character-1" {
		t.Fatalf("page ids = %#v, want campaign-1/character-1", view)
	}
	if view.Creation.NextStep != 2 || view.Creation.ClassID != "class-1" {
		t.Fatalf("creation = %#v, want adapted workflow view", view.Creation)
	}
	if len(view.Creation.Classes) != 1 || view.Creation.Classes[0].Name != "Bard" {
		t.Fatalf("creation classes = %#v, want Bard option", view.Creation.Classes)
	}
}

func TestNewCharacterCreationViewAdaptsNestedWorkflowFields(t *testing.T) {
	t.Parallel()

	source := campaignworkflow.CharacterCreationView{
		Ready:                        true,
		NextStep:                     5,
		UnmetReasons:                 []string{"need-domain"},
		ClassID:                      "class.guardian",
		SubclassID:                   "subclass.stalwart",
		SubclassCreationRequirements: []string{"companion_sheet_required"},
		Heritage: campaignworkflow.CreationHeritageSelectionView{
			AncestryLabel:           "Stoneborn",
			FirstFeatureAncestryID:  "ancestry.dwarf",
			FirstFeatureID:          "feature.thick_skin",
			SecondFeatureAncestryID: "ancestry.elf",
			SecondFeatureID:         "feature.celestial_trance",
			CommunityID:             "community.highborne",
		},
		CompanionSheet: &campaignworkflow.CreationCompanionView{
			AnimalKind: "Fox",
			Name:       "Ash",
			Evasion:    10,
			Experiences: []campaignworkflow.CreationExperienceView{
				{Name: "Scout", Modifier: "+2"},
			},
			AttackDescription: "Lunges from shadow",
			AttackRange:       "melee",
			DamageDieSides:    6,
			DamageType:        "physical",
		},
		Agility:           "2",
		Strength:          "1",
		Finesse:           "0",
		Instinct:          "1",
		Presence:          "2",
		Knowledge:         "-1",
		PrimaryWeaponID:   "weapon.spear",
		SecondaryWeaponID: "weapon.dagger",
		ArmorID:           "armor.scale",
		PotionItemID:      "item.health_potion",
		Background:        "Scout",
		Description:       "Quiet watcher",
		Experiences: []campaignworkflow.CreationExperienceView{
			{Name: "Trail Sense", Modifier: "+2"},
		},
		DomainCardIDs:        []string{"domain-card.shadow_1"},
		Connections:          "Owes the guild",
		NextStepPrefetchURLs: []string{"/next"},
		Steps: []campaignworkflow.CharacterCreationStepView{
			{Step: 1, Key: "class_subclass", Complete: true},
		},
		Classes: []campaignworkflow.CreationClassView{{
			ID:              "class.guardian",
			Name:            "Guardian",
			ImageURL:        "/guardian.png",
			StartingHP:      7,
			StartingEvasion: 11,
			HopeFeature: campaignworkflow.CreationClassFeatureView{
				Name:        "Stand Firm",
				Description: "Hold the line.",
			},
			Features: []campaignworkflow.CreationClassFeatureView{{
				Name:        "Armor Training",
				Description: "You know heavy armor.",
			}},
			DomainNames: []string{"Valor"},
			DomainWatermarks: []campaignworkflow.CreationDomainWatermarkView{{
				ID:      "domain.valor",
				Name:    "Valor",
				IconURL: "/valor.svg",
			}},
		}},
		Subclasses: []campaignworkflow.CreationSubclassView{{
			ID:             "subclass.stalwart",
			Name:           "Stalwart",
			ImageURL:       "/stalwart.png",
			ClassID:        "class.guardian",
			SpellcastTrait: "presence",
			CreationRequirements: []string{
				"companion_sheet_required",
			},
			Foundation: []campaignworkflow.CreationClassFeatureView{{
				Name:        "Unwavering",
				Description: "Gain threshold bonus.",
			}},
		}},
		Ancestries: []campaignworkflow.CreationHeritageView{{
			ID:       "ancestry.dwarf",
			Name:     "Dwarf",
			ImageURL: "/dwarf.png",
			Features: []campaignworkflow.CreationClassFeatureView{{
				Name:        "Thick Skin",
				Description: "Reduce harm.",
			}},
		}},
		Communities: []campaignworkflow.CreationHeritageView{{
			ID:       "community.highborne",
			Name:     "Highborne",
			ImageURL: "/highborne.png",
			Features: []campaignworkflow.CreationClassFeatureView{{
				Name:        "Noble Bearing",
				Description: "Read a room quickly.",
			}},
		}},
		PrimaryWeapons: []campaignworkflow.CreationWeaponView{{
			ID:       "weapon.spear",
			Name:     "Spear",
			ImageURL: "/spear.png",
			Burden:   1,
			Trait:    "finesse",
			Range:    "melee",
			Damage:   "d8",
			Feature:  "Reach",
		}},
		SecondaryWeapons: []campaignworkflow.CreationWeaponView{{
			ID:       "weapon.dagger",
			Name:     "Dagger",
			ImageURL: "/dagger.png",
			Burden:   1,
			Trait:    "agility",
			Range:    "melee",
			Damage:   "d6",
			Feature:  "Thrown",
		}},
		SecondaryWeaponNoneImageURL: "/none.png",
		Armor: []campaignworkflow.CreationArmorView{{
			ID:             "armor.scale",
			Name:           "Scale",
			ImageURL:       "/scale.png",
			ArmorScore:     5,
			BaseThresholds: "7 / 14",
			Feature:        "Fortified",
		}},
		PotionItems: []campaignworkflow.CreationItemView{{
			ID:          "item.health_potion",
			Name:        "Health Potion",
			ImageURL:    "/potion.png",
			Description: "Restore 1d4 HP.",
		}},
		DomainCards: []campaignworkflow.CreationDomainCardView{{
			ID:          "domain-card.shadow_1",
			Name:        "Shadow Step",
			ImageURL:    "/shadow-step.png",
			DomainID:    "domain.shadow",
			DomainName:  "Shadow",
			Level:       1,
			Type:        "spell",
			RecallCost:  1,
			FeatureText: "Blink between cover.",
		}},
	}

	view := NewCharacterCreationView(source)

	if !view.Ready || view.NextStep != 5 || view.ClassID != "class.guardian" || view.SubclassID != "subclass.stalwart" {
		t.Fatalf("view identity = %#v, want adapted ready creation view", view)
	}
	if view.Heritage.AncestryLabel != "Stoneborn" || view.Heritage.SecondFeatureID != "feature.celestial_trance" {
		t.Fatalf("heritage = %#v, want adapted heritage fields", view.Heritage)
	}
	if view.CompanionSheet == nil || view.CompanionSheet.Name != "Ash" || view.CompanionSheet.Experiences[0].Name != "Scout" {
		t.Fatalf("companion = %#v, want adapted companion sheet", view.CompanionSheet)
	}
	if len(view.Classes) != 1 || view.Classes[0].HopeFeature.Name != "Stand Firm" || view.Classes[0].DomainWatermarks[0].ID != "domain.valor" {
		t.Fatalf("classes = %#v, want adapted class card", view.Classes)
	}
	if len(view.Subclasses) != 1 || view.Subclasses[0].Foundation[0].Name != "Unwavering" {
		t.Fatalf("subclasses = %#v, want adapted subclass card", view.Subclasses)
	}
	if len(view.Ancestries) != 1 || view.Ancestries[0].Features[0].Name != "Thick Skin" {
		t.Fatalf("ancestries = %#v, want adapted ancestry card", view.Ancestries)
	}
	if len(view.PrimaryWeapons) != 1 || view.PrimaryWeapons[0].Feature != "Reach" {
		t.Fatalf("primary weapons = %#v, want adapted weapon card", view.PrimaryWeapons)
	}
	if len(view.Armor) != 1 || view.Armor[0].ArmorScore != 5 {
		t.Fatalf("armor = %#v, want adapted armor card", view.Armor)
	}
	if len(view.PotionItems) != 1 || view.PotionItems[0].Description != "Restore 1d4 HP." {
		t.Fatalf("potion items = %#v, want adapted item card", view.PotionItems)
	}
	if len(view.DomainCards) != 1 || view.DomainCards[0].FeatureText != "Blink between cover." {
		t.Fatalf("domain cards = %#v, want adapted domain card", view.DomainCards)
	}

	source.UnmetReasons[0] = "changed"
	source.SubclassCreationRequirements[0] = "changed"
	source.DomainCardIDs[0] = "changed"
	source.NextStepPrefetchURLs[0] = "changed"
	source.Classes[0].DomainNames[0] = "changed"
	if view.UnmetReasons[0] != "need-domain" ||
		view.SubclassCreationRequirements[0] != "companion_sheet_required" ||
		view.DomainCardIDs[0] != "domain-card.shadow_1" ||
		view.NextStepPrefetchURLs[0] != "/next" ||
		view.Classes[0].DomainNames[0] != "Valor" {
		t.Fatalf("view slices changed with source mutation: %#v", view)
	}
}
