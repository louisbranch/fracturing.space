package daggerheart

import (
	"testing"

	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/profile"
)

func TestCharacterHeritageValidate(t *testing.T) {
	tests := []struct {
		name    string
		input   CharacterHeritage
		wantErr bool
	}{
		{name: "empty heritage allowed", input: CharacterHeritage{}},
		{
			name: "single ancestry valid",
			input: CharacterHeritage{
				FirstFeatureAncestryID:  "heritage.human",
				FirstFeatureID:          "feature.human-high-stamina",
				SecondFeatureAncestryID: "heritage.human",
				SecondFeatureID:         "feature.human-adaptable",
				CommunityID:             "heritage.highborne",
			},
		},
		{
			name: "missing first feature rejected",
			input: CharacterHeritage{
				SecondFeatureAncestryID: "heritage.human",
				SecondFeatureID:         "feature.human-adaptable",
				CommunityID:             "heritage.highborne",
			},
			wantErr: true,
		},
		{
			name: "missing second feature rejected",
			input: CharacterHeritage{
				FirstFeatureAncestryID: "heritage.human",
				FirstFeatureID:         "feature.human-high-stamina",
				CommunityID:            "heritage.highborne",
			},
			wantErr: true,
		},
		{
			name: "missing community rejected",
			input: CharacterHeritage{
				FirstFeatureAncestryID:  "heritage.human",
				FirstFeatureID:          "feature.human-high-stamina",
				SecondFeatureAncestryID: "heritage.human",
				SecondFeatureID:         "feature.human-adaptable",
			},
			wantErr: true,
		},
		{
			name: "mixed ancestry valid without label",
			input: CharacterHeritage{
				FirstFeatureAncestryID:  "heritage.faerie",
				FirstFeatureID:          "feature.faerie-luckbender",
				SecondFeatureAncestryID: "heritage.human",
				SecondFeatureID:         "feature.human-adaptable",
				CommunityID:             "heritage.highborne",
			},
		},
		{
			name: "single ancestry rejects ancestry label",
			input: CharacterHeritage{
				AncestryLabel:           "Human Variant",
				FirstFeatureAncestryID:  "heritage.human",
				FirstFeatureID:          "feature.human-high-stamina",
				SecondFeatureAncestryID: "heritage.human",
				SecondFeatureID:         "feature.human-adaptable",
				CommunityID:             "heritage.highborne",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}

func TestCharacterProfileNormalized_NormalizesCompanionDefaults(t *testing.T) {
	profile := validCharacterProfile()
	profile.Level = 0
	profile.CompanionSheet = &CharacterCompanionSheet{
		AnimalKind: "wolf",
		Name:       "Ash",
		Experiences: []CharacterCompanionExperience{
			{ExperienceID: " companion-experience.tracking  ", Modifier: 9},
			{ExperienceID: "companion-experience.guarding", Modifier: -3},
		},
		AttackDescription: "Savage bite",
		DamageType:        CompanionDamageTypePhysical,
	}

	normalized := profile.Normalized()
	if normalized.Level != daggerheartprofile.PCLevelDefault {
		t.Fatalf("normalized level = %d, want %d", normalized.Level, daggerheartprofile.PCLevelDefault)
	}
	if normalized.CompanionSheet == nil {
		t.Fatal("normalized companion sheet = nil")
	}
	if normalized.CompanionSheet.Evasion != CompanionSheetDefaultEvasion {
		t.Fatalf("normalized companion evasion = %d, want %d", normalized.CompanionSheet.Evasion, CompanionSheetDefaultEvasion)
	}
	if normalized.CompanionSheet.AttackRange != CompanionSheetDefaultAttackRange {
		t.Fatalf("normalized companion range = %q, want %q", normalized.CompanionSheet.AttackRange, CompanionSheetDefaultAttackRange)
	}
	if normalized.CompanionSheet.DamageDieSides != CompanionSheetDefaultDamageDieSides {
		t.Fatalf("normalized companion damage die = %d, want %d", normalized.CompanionSheet.DamageDieSides, CompanionSheetDefaultDamageDieSides)
	}
	if normalized.CompanionSheet.Experiences[0].ExperienceID != "companion-experience.tracking" {
		t.Fatalf("normalized first companion experience = %q, want %q", normalized.CompanionSheet.Experiences[0].ExperienceID, "companion-experience.tracking")
	}
	if normalized.CompanionSheet.Experiences[0].Modifier != CompanionSheetExperienceModifier {
		t.Fatalf("normalized first companion experience modifier = %d, want %d", normalized.CompanionSheet.Experiences[0].Modifier, CompanionSheetExperienceModifier)
	}
	if err := normalized.CompanionSheet.Validate(); err != nil {
		t.Fatalf("normalized companion validate: %v", err)
	}
}

func TestCharacterCompanionSheetValidate_RejectedShapes(t *testing.T) {
	base := CharacterCompanionSheet{
		AnimalKind: "wolf",
		Name:       "Ash",
		Experiences: []CharacterCompanionExperience{
			{ExperienceID: "companion-experience.tracking", Modifier: CompanionSheetExperienceModifier},
			{ExperienceID: "companion-experience.guarding", Modifier: CompanionSheetExperienceModifier},
		},
		AttackDescription: "Savage bite",
		Evasion:           CompanionSheetDefaultEvasion,
		AttackRange:       CompanionSheetDefaultAttackRange,
		DamageDieSides:    CompanionSheetDefaultDamageDieSides,
		DamageType:        CompanionDamageTypePhysical,
	}

	tests := []struct {
		name   string
		mutate func(*CharacterCompanionSheet)
	}{
		{
			name: "missing animal kind",
			mutate: func(sheet *CharacterCompanionSheet) {
				sheet.AnimalKind = ""
			},
		},
		{
			name: "missing name",
			mutate: func(sheet *CharacterCompanionSheet) {
				sheet.Name = ""
			},
		},
		{
			name: "wrong experience count",
			mutate: func(sheet *CharacterCompanionSheet) {
				sheet.Experiences = sheet.Experiences[:1]
			},
		},
		{
			name: "blank experience id",
			mutate: func(sheet *CharacterCompanionSheet) {
				sheet.Experiences[0].ExperienceID = ""
			},
		},
		{
			name: "wrong experience modifier",
			mutate: func(sheet *CharacterCompanionSheet) {
				sheet.Experiences[0].Modifier = 1
			},
		},
		{
			name: "missing attack description",
			mutate: func(sheet *CharacterCompanionSheet) {
				sheet.AttackDescription = ""
			},
		},
		{
			name: "wrong evasion",
			mutate: func(sheet *CharacterCompanionSheet) {
				sheet.Evasion = 9
			},
		},
		{
			name: "wrong attack range",
			mutate: func(sheet *CharacterCompanionSheet) {
				sheet.AttackRange = "far"
			},
		},
		{
			name: "wrong damage die",
			mutate: func(sheet *CharacterCompanionSheet) {
				sheet.DamageDieSides = 8
			},
		},
		{
			name: "wrong damage type",
			mutate: func(sheet *CharacterCompanionSheet) {
				sheet.DamageType = "void"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sheet := base
			sheet.Experiences = append([]CharacterCompanionExperience(nil), base.Experiences...)
			tt.mutate(&sheet)
			if err := sheet.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}

	base.DamageType = CompanionDamageTypeMagic
	if err := base.Validate(); err != nil {
		t.Fatalf("magic companion Validate() error = %v, want nil", err)
	}
}

func TestCharacterProfileValidate_CompanionRequirements(t *testing.T) {
	t.Run("unsupported requirement rejected", func(t *testing.T) {
		profile := validCharacterProfile()
		profile.SubclassCreationRequirements = []string{"unknown_requirement"}
		if err := profile.Validate(); err == nil {
			t.Fatal("expected unsupported requirement error")
		}
	})

	t.Run("missing required companion rejected", func(t *testing.T) {
		profile := validCharacterProfile()
		profile.SubclassCreationRequirements = []string{SubclassCreationRequirementCompanionSheet}
		profile.CompanionSheet = nil
		if err := profile.Validate(); err == nil {
			t.Fatal("expected missing companion error")
		}
	})

	t.Run("invalid companion rejected", func(t *testing.T) {
		profile := validCharacterProfile()
		profile.SubclassCreationRequirements = []string{SubclassCreationRequirementCompanionSheet}
		profile.CompanionSheet = &CharacterCompanionSheet{
			AnimalKind:        "wolf",
			Name:              "Ash",
			Experiences:       []CharacterCompanionExperience{{ExperienceID: "companion-experience.tracking", Modifier: 2}},
			AttackDescription: "Savage bite",
			DamageType:        "void",
		}
		if err := profile.Validate(); err == nil {
			t.Fatal("expected invalid companion error")
		}
	})

	t.Run("valid companion accepted", func(t *testing.T) {
		profile := validCharacterProfile()
		profile.SubclassCreationRequirements = []string{SubclassCreationRequirementCompanionSheet}
		profile.CompanionSheet = &CharacterCompanionSheet{
			AnimalKind: "wolf",
			Name:       "Ash",
			Experiences: []CharacterCompanionExperience{
				{ExperienceID: "companion-experience.tracking", Modifier: CompanionSheetExperienceModifier},
				{ExperienceID: "companion-experience.guarding", Modifier: CompanionSheetExperienceModifier},
			},
			AttackDescription: "Savage bite",
			Evasion:           CompanionSheetDefaultEvasion,
			AttackRange:       CompanionSheetDefaultAttackRange,
			DamageDieSides:    CompanionSheetDefaultDamageDieSides,
			DamageType:        CompanionDamageTypePhysical,
		}
		if err := profile.Validate(); err != nil {
			t.Fatalf("Validate() error = %v, want nil", err)
		}
	})
}

func TestCharacterProfileStorageAndCreationProfile_PreserveStructuredFields(t *testing.T) {
	profile := validCharacterProfile()
	profile.SubclassCreationRequirements = []string{SubclassCreationRequirementCompanionSheet}
	profile.CompanionSheet = &CharacterCompanionSheet{
		AnimalKind: "wolf",
		Name:       "Ash",
		Experiences: []CharacterCompanionExperience{
			{ExperienceID: "companion-experience.tracking", Modifier: CompanionSheetExperienceModifier},
			{ExperienceID: "companion-experience.guarding", Modifier: CompanionSheetExperienceModifier},
		},
		AttackDescription: "Savage bite",
		Evasion:           CompanionSheetDefaultEvasion,
		AttackRange:       CompanionSheetDefaultAttackRange,
		DamageDieSides:    CompanionSheetDefaultDamageDieSides,
		DamageType:        CompanionDamageTypeMagic,
	}

	storageProfile := profile.ToStorage(" camp-1 ", " char-1 ")
	if storageProfile.CampaignID != "camp-1" || storageProfile.CharacterID != "char-1" {
		t.Fatalf("storage ids = (%q, %q), want trimmed ids", storageProfile.CampaignID, storageProfile.CharacterID)
	}
	if len(storageProfile.SubclassCreationRequirements) != 1 {
		t.Fatalf("storage subclass requirements = %v, want one requirement", storageProfile.SubclassCreationRequirements)
	}
	if storageProfile.CompanionSheet == nil || storageProfile.CompanionSheet.DamageType != CompanionDamageTypeMagic {
		t.Fatalf("storage companion = %+v, want magic companion", storageProfile.CompanionSheet)
	}

	roundTrip := CharacterProfileFromStorage(storageProfile)
	if len(roundTrip.SubclassCreationRequirements) != 1 || roundTrip.SubclassCreationRequirements[0] != SubclassCreationRequirementCompanionSheet {
		t.Fatalf("round-trip subclass requirements = %v", roundTrip.SubclassCreationRequirements)
	}
	if roundTrip.CompanionSheet == nil || roundTrip.CompanionSheet.Name != "Ash" {
		t.Fatalf("round-trip companion = %+v, want Ash", roundTrip.CompanionSheet)
	}

	creation := profile.CreationProfile()
	if len(creation.SubclassCreationRequirements) != 1 {
		t.Fatalf("creation subclass requirements = %v, want one requirement", creation.SubclassCreationRequirements)
	}
	if creation.CompanionSheet == nil || creation.CompanionSheet.Evasion != CompanionSheetDefaultEvasion {
		t.Fatalf("creation companion = %+v, want normalized companion", creation.CompanionSheet)
	}
}

func TestEvaluateCreationReadiness_RequiresCompanionForSubclass(t *testing.T) {
	profile := validCharacterProfile()
	profile.SubclassCreationRequirements = []string{SubclassCreationRequirementCompanionSheet}
	profile.CompanionSheet = nil

	ready, reason := EvaluateCreationReadiness(profile)
	if ready {
		t.Fatal("ready = true, want false")
	}
	if reason != "class and subclass selection is required" {
		t.Fatalf("reason = %q, want %q", reason, "class and subclass selection is required")
	}

	profile.CompanionSheet = &CharacterCompanionSheet{
		AnimalKind: "wolf",
		Name:       "Ash",
		Experiences: []CharacterCompanionExperience{
			{ExperienceID: "companion-experience.tracking", Modifier: CompanionSheetExperienceModifier},
			{ExperienceID: "companion-experience.guarding", Modifier: CompanionSheetExperienceModifier},
		},
		AttackDescription: "Savage bite",
		Evasion:           CompanionSheetDefaultEvasion,
		AttackRange:       CompanionSheetDefaultAttackRange,
		DamageDieSides:    CompanionSheetDefaultDamageDieSides,
		DamageType:        CompanionDamageTypePhysical,
	}

	ready, reason = EvaluateCreationReadiness(profile)
	if !ready || reason != "" {
		t.Fatalf("ready, reason = (%t, %q), want (true, %q)", ready, reason, "")
	}
}

func validCharacterProfile() CharacterProfile {
	return CharacterProfile{
		Level:           1,
		HpMax:           6,
		StressMax:       6,
		Evasion:         10,
		MajorThreshold:  3,
		SevereThreshold: 6,
		Proficiency:     1,
		ArmorScore:      0,
		ArmorMax:        2,
		Agility:         2,
		Strength:        1,
		Finesse:         1,
		Instinct:        0,
		Presence:        0,
		Knowledge:       -1,
		Experiences: []CharacterProfileExperience{
			{Name: "Tactics", Modifier: 2},
			{Name: "Patrol Routes", Modifier: 2},
		},
		ClassID:    "class.guardian",
		SubclassID: "subclass.stalwart",
		Heritage: CharacterHeritage{
			FirstFeatureAncestryID:  "heritage.human",
			FirstFeatureID:          "feature.human-high-stamina",
			SecondFeatureAncestryID: "heritage.human",
			SecondFeatureID:         "feature.human-adaptable",
			CommunityID:             "heritage.highborne",
		},
		TraitsAssigned:       true,
		DetailsRecorded:      true,
		StartingWeaponIDs:    []string{"weapon.longsword"},
		StartingArmorID:      "armor.gambeson-armor",
		StartingPotionItemID: StartingPotionMinorHealthID,
		Background:           "Former sentinel",
		Description:          "Calm and relentless.",
		DomainCardIDs:        []string{"domain-card.valor-bare-bones", "domain-card.valor-shield-wall"},
		Connections:          "Owes the captain a favor",
	}
}
