package workflow

import (
	"context"
	"testing"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	"golang.org/x/text/language"
)

func TestAppAdaptersRequireConfiguredServices(t *testing.T) {
	t.Parallel()

	if _, err := NewPageAppService(nil); err == nil {
		t.Fatal("NewPageAppService(nil) error = nil, want error")
	}
	if _, err := NewMutationAppService(nil); err == nil {
		t.Fatal("NewMutationAppService(nil) error = nil, want error")
	}
}

func TestPageAppAdapterMapsAppOwnedCreationShapes(t *testing.T) {
	t.Parallel()

	app := workflowCreationAppStub{
		progress: campaignapp.CampaignCharacterCreationProgress{
			Steps:        []campaignapp.CampaignCharacterCreationStep{{Step: 1, Key: "class", Complete: true}},
			NextStep:     2,
			Ready:        true,
			UnmetReasons: []string{"missing details"},
		},
		catalog: campaignapp.CampaignCharacterCreationCatalog{
			AssetTheme: "aurora",
			Classes: []campaignapp.CatalogClass{{
				ID:              "warrior",
				Name:            "Warrior",
				DomainIDs:       []string{"valor", "bone"},
				StartingHP:      6,
				StartingEvasion: 11,
				HopeFeature:     campaignapp.CatalogFeature{Name: "Brave", Description: "Gain hope"},
				Features:        []campaignapp.CatalogFeature{{Name: "Strike", Description: "Hit hard"}},
				Illustration:    campaignapp.CatalogAssetReference{URL: "https://img/class.png", Status: "ready", SetID: "base", AssetID: "class-1"},
				Icon:            campaignapp.CatalogAssetReference{URL: "https://img/icon.png", Status: "ready", SetID: "base", AssetID: "icon-1"},
			}},
			Subclasses: []campaignapp.CatalogSubclass{{
				ID:                   "guardian",
				Name:                 "Guardian",
				ClassID:              "warrior",
				SpellcastTrait:       "presence",
				CreationRequirements: []string{"level-2"},
				Foundation:           []campaignapp.CatalogFeature{{Name: "Wall", Description: "Protect allies"}},
				Illustration:         campaignapp.CatalogAssetReference{URL: "https://img/subclass.png", Status: "ready", SetID: "base", AssetID: "subclass-1"},
			}},
			Heritages: []campaignapp.CatalogHeritage{{
				ID:           "elf",
				Name:         "Elf",
				Kind:         "ancestry",
				Features:     []campaignapp.CatalogFeature{{Name: "Grace", Description: "Move lightly"}},
				Illustration: campaignapp.CatalogAssetReference{URL: "https://img/heritage.png", Status: "ready", SetID: "base", AssetID: "heritage-1"},
			}},
			CompanionExperiences: []campaignapp.CatalogCompanionExperience{{ID: "bond", Name: "Bonded", Description: "Trusted ally"}},
			Domains: []campaignapp.CatalogDomain{{
				ID:           "valor",
				Name:         "Valor",
				Illustration: campaignapp.CatalogAssetReference{URL: "https://img/domain.png", Status: "ready", SetID: "base", AssetID: "domain-1"},
				Icon:         campaignapp.CatalogAssetReference{URL: "https://img/domain-icon.png", Status: "ready", SetID: "base", AssetID: "domain-icon-1"},
			}},
			Weapons: []campaignapp.CatalogWeapon{{
				ID:           "blade",
				Name:         "Blade",
				Category:     "primary",
				Tier:         1,
				Burden:       1,
				Trait:        "agility",
				Range:        "melee",
				Damage:       "d8",
				Feature:      "Sharp",
				DisplayOrder: 1,
				DisplayGroup: "swords",
				Illustration: campaignapp.CatalogAssetReference{URL: "https://img/weapon.png", Status: "ready", SetID: "base", AssetID: "weapon-1"},
			}},
			Armor: []campaignapp.CatalogArmor{{
				ID:             "chain",
				Name:           "Chain",
				Tier:           1,
				ArmorScore:     2,
				BaseThresholds: "6/12",
				Feature:        "Sturdy",
				Illustration:   campaignapp.CatalogAssetReference{URL: "https://img/armor.png", Status: "ready", SetID: "base", AssetID: "armor-1"},
			}},
			Items: []campaignapp.CatalogItem{{
				ID:           "potion",
				Name:         "Potion",
				Description:  "Heal",
				Illustration: campaignapp.CatalogAssetReference{URL: "https://img/item.png", Status: "ready", SetID: "base", AssetID: "item-1"},
			}},
			DomainCards: []campaignapp.CatalogDomainCard{{
				ID:           "card-1",
				Name:         "Heroic Strike",
				DomainID:     "valor",
				DomainName:   "Valor",
				Level:        1,
				Type:         "attack",
				RecallCost:   1,
				FeatureText:  "Deal damage",
				Illustration: campaignapp.CatalogAssetReference{URL: "https://img/card.png", Status: "ready", SetID: "base", AssetID: "card-1"},
			}},
			Adversaries: []campaignapp.CatalogAdversary{{
				ID:           "wolf",
				Name:         "Wolf",
				Illustration: campaignapp.CatalogAssetReference{URL: "https://img/adversary.png", Status: "ready", SetID: "base", AssetID: "adversary-1"},
			}},
			Environments: []campaignapp.CatalogEnvironment{{
				ID:           "forest",
				Name:         "Forest",
				Illustration: campaignapp.CatalogAssetReference{URL: "https://img/environment.png", Status: "ready", SetID: "base", AssetID: "environment-1"},
			}},
		},
		profile: campaignapp.CampaignCharacterCreationProfile{
			CharacterName:                "Nox",
			ClassID:                      "warrior",
			SubclassID:                   "guardian",
			SubclassCreationRequirements: []string{"level-2"},
			Heritage: campaignapp.CampaignCharacterCreationHeritageSelection{
				AncestryLabel:           "Elf",
				FirstFeatureAncestryID:  "elf",
				FirstFeatureID:          "grace",
				SecondFeatureAncestryID: "elf",
				SecondFeatureID:         "sight",
				CommunityID:             "wildborn",
			},
			CompanionSheet: &campaignapp.CampaignCharacterCreationCompanionSheet{
				AnimalKind:        "wolf",
				Name:              "Ash",
				Evasion:           12,
				Experiences:       []campaignapp.CampaignCharacterCreationExperience{{ID: "exp-1", Name: "Tracking", Modifier: "+2"}},
				AttackDescription: "Bite",
				AttackRange:       "melee",
				DamageDieSides:    8,
				DamageType:        "physical",
			},
			Agility:           "+2",
			Strength:          "+1",
			Finesse:           "0",
			Instinct:          "+1",
			Presence:          "+2",
			Knowledge:         "0",
			PrimaryWeaponID:   "blade",
			SecondaryWeaponID: "dagger",
			ArmorID:           "chain",
			PotionItemID:      "potion",
			Background:        "Wanderer",
			Description:       "Guard",
			Experiences:       []campaignapp.CampaignCharacterCreationExperience{{ID: "exp-2", Name: "Lore", Modifier: "+1"}},
			DomainCardIDs:     []string{"card-1"},
			Connections:       "Old debts",
		},
	}

	pageApp, err := NewPageAppService(app)
	if err != nil {
		t.Fatalf("NewPageAppService() error = %v", err)
	}

	progress, err := pageApp.CampaignCharacterCreationProgress(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("CampaignCharacterCreationProgress() error = %v", err)
	}
	if len(progress.Steps) != 1 || progress.Steps[0].Key != "class" || !progress.Ready {
		t.Fatalf("progress = %+v", progress)
	}

	catalog, err := pageApp.CampaignCharacterCreationCatalog(context.Background(), language.AmericanEnglish)
	if err != nil {
		t.Fatalf("CampaignCharacterCreationCatalog() error = %v", err)
	}
	if len(catalog.Classes) != 1 || catalog.Classes[0].HopeFeature.Name != "Brave" || len(catalog.DomainCards) != 1 {
		t.Fatalf("catalog = %+v", catalog)
	}
	if len(catalog.CompanionExperiences) != 1 || catalog.CompanionExperiences[0].ID != "bond" {
		t.Fatalf("catalog companion experiences = %+v", catalog.CompanionExperiences)
	}

	profile, err := pageApp.CampaignCharacterCreationProfile(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("CampaignCharacterCreationProfile() error = %v", err)
	}
	if profile.CharacterName != "Nox" || profile.Heritage.CommunityID != "wildborn" || profile.CompanionSheet == nil {
		t.Fatalf("profile = %+v", profile)
	}
	if len(profile.Experiences) != 1 || profile.Experiences[0].Name != "Lore" {
		t.Fatalf("profile experiences = %+v", profile.Experiences)
	}
}

func TestMutationAppAdapterDelegatesAndMapsProgress(t *testing.T) {
	t.Parallel()

	app := &workflowCreationMutationStub{
		progress: campaignapp.CampaignCharacterCreationProgress{NextStep: 3},
	}

	mutationApp, err := NewMutationAppService(app)
	if err != nil {
		t.Fatalf("NewMutationAppService() error = %v", err)
	}

	progress, err := mutationApp.CampaignCharacterCreationProgress(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("CampaignCharacterCreationProgress() error = %v", err)
	}
	if progress.NextStep != 3 {
		t.Fatalf("progress.NextStep = %d, want 3", progress.NextStep)
	}

	step := &StepInput{Details: &campaignapp.CampaignCharacterCreationStepDetails{Description: "ready"}}
	if err := mutationApp.ApplyCharacterCreationStep(context.Background(), "camp-1", "char-1", step); err != nil {
		t.Fatalf("ApplyCharacterCreationStep() error = %v", err)
	}
	if app.lastStep == nil || app.lastStep.Details == nil || app.lastStep.Details.Description != "ready" {
		t.Fatalf("lastStep = %#v", app.lastStep)
	}

	if err := mutationApp.ResetCharacterCreationWorkflow(context.Background(), "camp-1", "char-1"); err != nil {
		t.Fatalf("ResetCharacterCreationWorkflow() error = %v", err)
	}
	if !app.resetCalled {
		t.Fatal("resetCalled = false, want true")
	}
}

type workflowCreationAppStub struct {
	progress campaignapp.CampaignCharacterCreationProgress
	catalog  campaignapp.CampaignCharacterCreationCatalog
	profile  campaignapp.CampaignCharacterCreationProfile
}

func (s workflowCreationAppStub) CampaignCharacterCreationProgress(context.Context, string, string) (campaignapp.CampaignCharacterCreationProgress, error) {
	return s.progress, nil
}

func (s workflowCreationAppStub) CampaignCharacterCreationCatalog(context.Context, language.Tag) (campaignapp.CampaignCharacterCreationCatalog, error) {
	return s.catalog, nil
}

func (s workflowCreationAppStub) CampaignCharacterCreationProfile(context.Context, string, string) (campaignapp.CampaignCharacterCreationProfile, error) {
	return s.profile, nil
}

type workflowCreationMutationStub struct {
	progress    campaignapp.CampaignCharacterCreationProgress
	lastStep    *campaignapp.CampaignCharacterCreationStepInput
	resetCalled bool
}

func (s *workflowCreationMutationStub) CampaignCharacterCreationProgress(context.Context, string, string) (campaignapp.CampaignCharacterCreationProgress, error) {
	return s.progress, nil
}

func (s *workflowCreationMutationStub) ApplyCharacterCreationStep(_ context.Context, _ string, _ string, step *campaignapp.CampaignCharacterCreationStepInput) error {
	s.lastStep = step
	return nil
}

func (s *workflowCreationMutationStub) ResetCharacterCreationWorkflow(context.Context, string, string) error {
	s.resetCalled = true
	return nil
}
