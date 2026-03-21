package state

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestCharacterProfileNormalized_SeedsCreationArmorInvariant(t *testing.T) {
	t.Parallel()

	profile := CharacterProfile{
		HpMax:           7,
		StressMax:       6,
		Evasion:         9,
		MajorThreshold:  8,
		SevereThreshold: 12,
		Proficiency:     1,
		ArmorScore:      4,
		StartingArmorID: "armor.chainmail-armor",
	}

	got := profile.Normalized()

	if got.EquippedArmorID != "armor.chainmail-armor" {
		t.Fatalf("equipped armor id = %q, want %q", got.EquippedArmorID, "armor.chainmail-armor")
	}
	if got.ArmorMax != 4 {
		t.Fatalf("armor max = %d, want 4", got.ArmorMax)
	}
}

func TestCharacterProfileNormalized_DoesNotReequipUnarmoredProfile(t *testing.T) {
	t.Parallel()

	profile := CharacterProfile{
		HpMax:           7,
		StressMax:       6,
		Evasion:         9,
		MajorThreshold:  1,
		SevereThreshold: 2,
		Proficiency:     1,
		ArmorScore:      0,
		ArmorMax:        0,
		StartingArmorID: "armor.chainmail-armor",
	}

	got := profile.Normalized()

	if got.EquippedArmorID != "" {
		t.Fatalf("equipped armor id = %q, want empty", got.EquippedArmorID)
	}
	if got.ArmorMax != 0 {
		t.Fatalf("armor max = %d, want 0", got.ArmorMax)
	}
}

func TestCharacterHeritageValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		heritage CharacterHeritage
		wantErr  string
	}{
		{name: "zero value is allowed"},
		{
			name: "missing first feature ancestry is rejected",
			heritage: CharacterHeritage{
				FirstFeatureID:          "feature.one",
				SecondFeatureAncestryID: "ancestry.elf",
				SecondFeatureID:         "feature.two",
				CommunityID:             "community.wanderer",
			},
			wantErr: "heritage first feature is required",
		},
		{
			name: "missing second feature ancestry is rejected",
			heritage: CharacterHeritage{
				FirstFeatureAncestryID: "ancestry.elf",
				FirstFeatureID:         "feature.one",
				CommunityID:            "community.wanderer",
			},
			wantErr: "heritage second feature is required",
		},
		{
			name: "missing community is rejected",
			heritage: CharacterHeritage{
				FirstFeatureAncestryID:  "ancestry.elf",
				FirstFeatureID:          "feature.one",
				SecondFeatureAncestryID: "ancestry.orc",
				SecondFeatureID:         "feature.two",
			},
			wantErr: "heritage community is required",
		},
		{
			name: "same ancestry cannot carry mixed label",
			heritage: CharacterHeritage{
				AncestryLabel:           "Mixed",
				FirstFeatureAncestryID:  "ancestry.elf",
				FirstFeatureID:          "feature.one",
				SecondFeatureAncestryID: "ancestry.elf",
				SecondFeatureID:         "feature.two",
				CommunityID:             "community.wanderer",
			},
			wantErr: "ancestry label is only allowed for mixed ancestry",
		},
		{
			name: "mixed ancestry label is allowed",
			heritage: CharacterHeritage{
				AncestryLabel:           "Mixed",
				FirstFeatureAncestryID:  "ancestry.elf",
				FirstFeatureID:          "feature.one",
				SecondFeatureAncestryID: "ancestry.orc",
				SecondFeatureID:         "feature.two",
				CommunityID:             "community.wanderer",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.heritage.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() returned error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestCharacterSubclassTrackValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		track   CharacterSubclassTrack
		wantErr string
	}{
		{
			name: "primary track is valid",
			track: CharacterSubclassTrack{
				Origin:     SubclassTrackOriginPrimary,
				ClassID:    "class.guardian",
				SubclassID: "subclass.stalwart",
				Rank:       SubclassTrackRankFoundation,
			},
		},
		{
			name: "multiclass track requires domain",
			track: CharacterSubclassTrack{
				Origin:     SubclassTrackOriginMulticlass,
				ClassID:    "class.seraph",
				SubclassID: "subclass.winged-sentinel",
				Rank:       SubclassTrackRankSpecialization,
			},
			wantErr: "multiclass subclass track domain is required",
		},
		{
			name: "invalid origin is rejected",
			track: CharacterSubclassTrack{
				Origin:     "sidecar",
				ClassID:    "class.guardian",
				SubclassID: "subclass.stalwart",
				Rank:       SubclassTrackRankFoundation,
			},
			wantErr: "subclass track origin is invalid",
		},
		{
			name: "missing class is rejected",
			track: CharacterSubclassTrack{
				Origin:     SubclassTrackOriginPrimary,
				SubclassID: "subclass.stalwart",
				Rank:       SubclassTrackRankFoundation,
			},
			wantErr: "subclass track class is required",
		},
		{
			name: "invalid rank is rejected",
			track: CharacterSubclassTrack{
				Origin:     SubclassTrackOriginPrimary,
				ClassID:    "class.guardian",
				SubclassID: "subclass.stalwart",
				Rank:       "grandmaster",
			},
			wantErr: "subclass track rank is invalid",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.track.Validate()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() returned error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestCharacterCompanionSheetValidate(t *testing.T) {
	t.Parallel()

	valid := CharacterCompanionSheet{
		AnimalKind: "fox",
		Name:       "Ash",
		Evasion:    CompanionSheetDefaultEvasion,
		Experiences: []CharacterCompanionExperience{
			{ExperienceID: "experience.keen-ears", Modifier: CompanionSheetExperienceModifier},
			{ExperienceID: "experience.shadow-step", Modifier: CompanionSheetExperienceModifier},
		},
		AttackDescription: "A blur of teeth and claws.",
		AttackRange:       CompanionSheetDefaultAttackRange,
		DamageDieSides:    CompanionSheetDefaultDamageDieSides,
		DamageType:        CompanionDamageTypeMagic,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() returned error for valid sheet: %v", err)
	}

	tests := []struct {
		name    string
		sheet   CharacterCompanionSheet
		wantErr string
	}{
		{
			name: "requires two experiences",
			sheet: CharacterCompanionSheet{
				AnimalKind:        "fox",
				Name:              "Ash",
				Evasion:           CompanionSheetDefaultEvasion,
				Experiences:       []CharacterCompanionExperience{{ExperienceID: "experience.keen-ears", Modifier: CompanionSheetExperienceModifier}},
				AttackDescription: "A blur of teeth and claws.",
				AttackRange:       CompanionSheetDefaultAttackRange,
				DamageDieSides:    CompanionSheetDefaultDamageDieSides,
				DamageType:        CompanionDamageTypePhysical,
			},
			wantErr: "companion requires exactly two experiences",
		},
		{
			name: "requires attack description",
			sheet: CharacterCompanionSheet{
				AnimalKind:     "fox",
				Name:           "Ash",
				Evasion:        CompanionSheetDefaultEvasion,
				Experiences:    valid.Experiences,
				AttackRange:    CompanionSheetDefaultAttackRange,
				DamageDieSides: CompanionSheetDefaultDamageDieSides,
				DamageType:     CompanionDamageTypePhysical,
			},
			wantErr: "companion attack description is required",
		},
		{
			name: "requires supported damage type",
			sheet: CharacterCompanionSheet{
				AnimalKind:        "fox",
				Name:              "Ash",
				Evasion:           CompanionSheetDefaultEvasion,
				Experiences:       valid.Experiences,
				AttackDescription: "A blur of teeth and claws.",
				AttackRange:       CompanionSheetDefaultAttackRange,
				DamageDieSides:    CompanionSheetDefaultDamageDieSides,
				DamageType:        "psychic",
			},
			wantErr: "companion damage type must be physical or magic",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.sheet.Validate()
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestCharacterProfileRoundTripAndCreationProfile(t *testing.T) {
	t.Parallel()

	profile := validCharacterProfile()
	if err := profile.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}

	stored := profile.ToStorage(" camp-1 ", " char-1 ")
	if stored.CampaignID != "camp-1" || stored.CharacterID != "char-1" {
		t.Fatalf("trimmed storage ids = %q/%q, want camp-1/char-1", stored.CampaignID, stored.CharacterID)
	}
	if stored.EquippedArmorID != "armor.chainmail-armor" {
		t.Fatalf("EquippedArmorID = %q, want armor.chainmail-armor", stored.EquippedArmorID)
	}
	if stored.ArmorMax != 4 {
		t.Fatalf("ArmorMax = %d, want 4", stored.ArmorMax)
	}
	if len(stored.SubclassTracks) != 2 {
		t.Fatalf("len(SubclassTracks) = %d, want 2", len(stored.SubclassTracks))
	}
	if stored.CompanionSheet == nil || stored.CompanionSheet.Evasion != CompanionSheetDefaultEvasion {
		t.Fatalf("stored companion sheet = %+v, want normalized defaults", stored.CompanionSheet)
	}

	roundTrip := CharacterProfileFromStorage(stored)
	if !reflect.DeepEqual(roundTrip.StartingWeaponIDs, profile.StartingWeaponIDs) {
		t.Fatalf("StartingWeaponIDs = %#v, want %#v", roundTrip.StartingWeaponIDs, profile.StartingWeaponIDs)
	}
	if roundTrip.CompanionSheet == nil || roundTrip.CompanionSheet.DamageType != CompanionDamageTypePhysical {
		t.Fatalf("roundTrip.CompanionSheet = %+v, want physical companion", roundTrip.CompanionSheet)
	}
	if roundTrip.Heritage.CommunityID != profile.Heritage.CommunityID {
		t.Fatalf("CommunityID = %q, want %q", roundTrip.Heritage.CommunityID, profile.Heritage.CommunityID)
	}

	creation := profile.CreationProfile()
	if creation.Level != daggerheartprofile.PCLevelDefault {
		t.Fatalf("creation.Level = %d, want default level %d", creation.Level, daggerheartprofile.PCLevelDefault)
	}
	if creation.EquippedArmorID != "armor.chainmail-armor" {
		t.Fatalf("creation.EquippedArmorID = %q, want armor.chainmail-armor", creation.EquippedArmorID)
	}
	if creation.CompanionSheet == nil || creation.CompanionSheet.AttackRange != CompanionSheetDefaultAttackRange {
		t.Fatalf("creation.CompanionSheet = %+v, want normalized companion defaults", creation.CompanionSheet)
	}
	if len(creation.Experiences) != 1 || creation.Experiences[0].Name != "Trail Scout" {
		t.Fatalf("creation.Experiences = %#v, want Trail Scout entry", creation.Experiences)
	}
}

func TestValidateSubclassRequirementsAndTracks(t *testing.T) {
	t.Parallel()

	if err := ValidateSubclassCreationRequirements([]string{"", SubclassCreationRequirementCompanionSheet}); err != nil {
		t.Fatalf("ValidateSubclassCreationRequirements() returned error: %v", err)
	}
	if !RequiresCompanionSheet([]string{"other", SubclassCreationRequirementCompanionSheet}) {
		t.Fatal("RequiresCompanionSheet() = false, want true")
	}
	if err := ValidateSubclassCreationRequirements([]string{"unknown"}); err == nil {
		t.Fatal("ValidateSubclassCreationRequirements() error = nil, want error")
	}

	validTracks := []CharacterSubclassTrack{
		{
			Origin:     SubclassTrackOriginPrimary,
			ClassID:    "class.guardian",
			SubclassID: "subclass.stalwart",
			Rank:       SubclassTrackRankFoundation,
		},
	}
	if err := ValidateSubclassTracks("class.guardian", "subclass.stalwart", validTracks); err != nil {
		t.Fatalf("ValidateSubclassTracks() returned error: %v", err)
	}

	duplicatePrimary := append(validTracks, CharacterSubclassTrack{
		Origin:     SubclassTrackOriginPrimary,
		ClassID:    "class.guardian",
		SubclassID: "subclass.stalwart",
		Rank:       SubclassTrackRankSpecialization,
	})
	if err := ValidateSubclassTracks("class.guardian", "subclass.stalwart", duplicatePrimary); err == nil || !strings.Contains(err.Error(), "duplicate primary") {
		t.Fatalf("ValidateSubclassTracks() error = %v, want duplicate primary error", err)
	}

	missingPrimary := []CharacterSubclassTrack{
		{
			Origin:     SubclassTrackOriginMulticlass,
			ClassID:    "class.seraph",
			SubclassID: "subclass.winged-sentinel",
			Rank:       SubclassTrackRankFoundation,
			DomainID:   "domain.grace",
		},
	}
	if err := ValidateSubclassTracks("class.guardian", "subclass.stalwart", missingPrimary); err == nil || !strings.Contains(err.Error(), "primary subclass track is required") {
		t.Fatalf("ValidateSubclassTracks() error = %v, want missing primary error", err)
	}
}

func validCharacterProfile() CharacterProfile {
	return CharacterProfile{
		Level:           0,
		HpMax:           6,
		StressMax:       6,
		Evasion:         10,
		MajorThreshold:  8,
		SevereThreshold: 12,
		Proficiency:     1,
		ArmorScore:      4,
		Experiences: []CharacterProfileExperience{
			{Name: "Trail Scout", Modifier: 2},
		},
		Agility:    1,
		Strength:   2,
		Finesse:    0,
		Instinct:   1,
		Presence:   -1,
		Knowledge:  0,
		ClassID:    "class.guardian",
		SubclassID: "subclass.stalwart",
		SubclassTracks: []CharacterSubclassTrack{
			{
				Origin:     SubclassTrackOriginPrimary,
				ClassID:    "class.guardian",
				SubclassID: "subclass.stalwart",
				Rank:       SubclassTrackRankFoundation,
			},
			{
				Origin:     SubclassTrackOriginMulticlass,
				ClassID:    "class.seraph",
				SubclassID: "subclass.winged-sentinel",
				Rank:       SubclassTrackRankSpecialization,
				DomainID:   "domain.grace",
			},
		},
		SubclassCreationRequirements: []string{SubclassCreationRequirementCompanionSheet},
		Heritage: CharacterHeritage{
			AncestryLabel:           "Mixed",
			FirstFeatureAncestryID:  "ancestry.elf",
			FirstFeatureID:          "ancestry.elf.feature.keen-eyes",
			SecondFeatureAncestryID: "ancestry.orc",
			SecondFeatureID:         "ancestry.orc.feature.enduring",
			CommunityID:             "community.wanderer",
		},
		CompanionSheet: &CharacterCompanionSheet{
			AnimalKind: "wolf",
			Name:       "Ember",
			Experiences: []CharacterCompanionExperience{
				{ExperienceID: "experience.guard", Modifier: 0},
				{ExperienceID: "experience.track", Modifier: 99},
			},
			AttackDescription: "A lunging bite.",
			DamageType:        CompanionDamageTypePhysical,
		},
		SpellcastRollBonus:   1,
		TraitsAssigned:       true,
		DetailsRecorded:      true,
		StartingWeaponIDs:    []string{"weapon.bastard-sword", "weapon.round-shield"},
		StartingArmorID:      "armor.chainmail-armor",
		StartingPotionItemID: "item.minor-health-potion",
		Background:           "Former caravan guard.",
		Description:          "Keeps watch with methodical patience.",
		DomainCardIDs:        []string{"domain.valor.card-1"},
		Connections:          "Sworn to protect the lantern bearer.",
		GoldHandfuls:         1,
		GoldBags:             2,
		GoldChests:           3,
	}
}

func TestCharacterProfileFromStorageHandlesNilCollections(t *testing.T) {
	t.Parallel()

	profile := CharacterProfileFromStorage(projectionstore.DaggerheartCharacterProfile{
		Level:           1,
		HpMax:           6,
		StressMax:       6,
		Evasion:         10,
		MajorThreshold:  8,
		SevereThreshold: 12,
		Proficiency:     1,
	})

	if profile.CompanionSheet != nil {
		t.Fatalf("CompanionSheet = %+v, want nil", profile.CompanionSheet)
	}
	if profile.SubclassTracks != nil {
		t.Fatalf("SubclassTracks = %#v, want nil", profile.SubclassTracks)
	}
}

func TestCharacterClassStateHelpers(t *testing.T) {
	t.Parallel()

	raw := CharacterClassState{
		AttackBonusUntilRest:       -1,
		EvasionBonusUntilHitOrRest: -2,
		DifficultyPenaltyUntilRest: 1,
		FocusTargetID:              " adv-1 ",
		ActiveBeastform: &CharacterActiveBeastformState{
			BeastformID:            " beastform.bear ",
			BaseTrait:              " strength ",
			AttackTrait:            " agility ",
			TraitBonus:             -1,
			EvasionBonus:           -2,
			AttackRange:            " melee ",
			DamageDice:             []CharacterDamageDie{{Count: 2, Sides: 8}, {Count: 0, Sides: 6}},
			DamageBonus:            -3,
			DamageType:             " physical ",
			EvolutionTraitOverride: " instinct ",
		},
		StrangePatternsNumber: -1,
		RallyDice:             []int{0, 6, -1, 8},
		PrayerDice:            []int{-1, 10},
		Unstoppable: CharacterUnstoppableState{
			Active:           true,
			CurrentValue:     -1,
			DieSides:         -12,
			UsedThisLongRest: true,
		},
	}

	normalized := raw.Normalized()
	if normalized.AttackBonusUntilRest != 0 || normalized.EvasionBonusUntilHitOrRest != 0 || normalized.DifficultyPenaltyUntilRest != 0 {
		t.Fatalf("normalized combat bonuses = %+v, want clamped zeros", normalized)
	}
	if normalized.FocusTargetID != "adv-1" {
		t.Fatalf("FocusTargetID = %q, want adv-1", normalized.FocusTargetID)
	}
	if normalized.ActiveBeastform == nil || normalized.ActiveBeastform.BeastformID != "beastform.bear" {
		t.Fatalf("ActiveBeastform = %+v, want trimmed beastform", normalized.ActiveBeastform)
	}
	if !reflect.DeepEqual(normalized.RallyDice, []int{6, 8}) || !reflect.DeepEqual(normalized.PrayerDice, []int{10}) {
		t.Fatalf("normalized dice = rally:%v prayer:%v, want [6 8] and [10]", normalized.RallyDice, normalized.PrayerDice)
	}
	if normalized.Unstoppable.CurrentValue != 0 || normalized.Unstoppable.DieSides != 0 {
		t.Fatalf("normalized unstoppable = %+v, want non-negative values", normalized.Unstoppable)
	}

	equivalent := CharacterClassState{
		FocusTargetID: "adv-1",
		ActiveBeastform: &CharacterActiveBeastformState{
			BeastformID:            "beastform.bear",
			BaseTrait:              "strength",
			AttackTrait:            "agility",
			AttackRange:            "melee",
			DamageDice:             []CharacterDamageDie{{Count: 2, Sides: 8}},
			DamageType:             "physical",
			EvolutionTraitOverride: "instinct",
		},
		RallyDice:  []int{6, 8},
		PrayerDice: []int{10},
		Unstoppable: CharacterUnstoppableState{
			Active:           true,
			UsedThisLongRest: true,
		},
	}
	if !raw.Equal(equivalent) {
		t.Fatal("Equal() = false, want true after normalization")
	}
	if raw.Equal(CharacterClassState{}) {
		t.Fatal("Equal() = true, want false for distinct state")
	}
	if (CharacterClassState{}).IsZero() != true {
		t.Fatal("IsZero() = false, want true for zero state")
	}
	if raw.IsZero() {
		t.Fatal("IsZero() = true, want false for populated state")
	}
	if got := NormalizedDiceValues([]int{-1, 0}); got != nil {
		t.Fatalf("NormalizedDiceValues() = %v, want nil", got)
	}
	if got := NormalizedDamageDice([]CharacterDamageDie{{Count: 0, Sides: 6}}); got != nil {
		t.Fatalf("NormalizedDamageDice() = %v, want nil", got)
	}
	if got := NormalizedActiveBeastformPtr(&CharacterActiveBeastformState{BeastformID: " "}); got != nil {
		t.Fatalf("NormalizedActiveBeastformPtr() = %+v, want nil", got)
	}
	withBeastform := WithActiveBeastform(CharacterClassState{}, &CharacterActiveBeastformState{BeastformID: " bear "})
	if withBeastform.ActiveBeastform == nil || withBeastform.ActiveBeastform.BeastformID != "bear" {
		t.Fatalf("WithActiveBeastform() = %+v, want trimmed beastform", withBeastform.ActiveBeastform)
	}
}

func TestCharacterCompanionStateHelpers(t *testing.T) {
	t.Parallel()

	if got := (CharacterCompanionState{}).Normalized(); got.Status != CompanionStatusPresent || got.ActiveExperienceID != "" {
		t.Fatalf("Normalized() = %+v, want present/idle", got)
	}
	if got := (CharacterCompanionState{Status: CompanionStatusAway}).Normalized(); got.Status != CompanionStatusPresent {
		t.Fatalf("Normalized() = %+v, want away without experience to reset", got)
	}
	if got := (CharacterCompanionState{Status: "invalid", ActiveExperienceID: "exp-1"}).Normalized(); got.Status != CompanionStatusPresent || got.ActiveExperienceID != "" {
		t.Fatalf("Normalized() invalid status = %+v, want present/idle", got)
	}

	away := WithActiveCompanionExperience(CharacterCompanionState{}, " exp-1 ")
	if away.Status != CompanionStatusAway || away.ActiveExperienceID != "exp-1" {
		t.Fatalf("WithActiveCompanionExperience() = %+v, want away on exp-1", away)
	}
	if !away.Equal(CharacterCompanionState{Status: "away", ActiveExperienceID: "exp-1"}) {
		t.Fatal("Equal() = false, want normalized equality")
	}
	if away.IsZero() {
		t.Fatal("IsZero() = true, want false for away companion")
	}
	if got := WithCompanionPresent(away); !got.IsZero() {
		t.Fatalf("WithCompanionPresent() = %+v, want zero/present state", got)
	}
	if got := NormalizedCompanionStatePtr(nil); got != nil {
		t.Fatalf("NormalizedCompanionStatePtr(nil) = %+v, want nil", got)
	}
}

func TestSnapshotStateHelpers(t *testing.T) {
	t.Parallel()

	var snapshot SnapshotState
	snapshot.EnsureMaps()
	if snapshot.CharacterProfiles == nil || snapshot.CountdownStates == nil || snapshot.CharacterStatModifiers == nil {
		t.Fatalf("EnsureMaps() left nil maps: %+v", snapshot)
	}

	valueState, ok := SnapshotOrDefault(SnapshotState{})
	if !ok || valueState.CharacterProfiles == nil {
		t.Fatalf("SnapshotOrDefault(value) = (%+v, %v), want initialized value and true", valueState, ok)
	}
	ptrState, ok := SnapshotOrDefault(&SnapshotState{})
	if !ok || ptrState.CharacterStates == nil {
		t.Fatalf("SnapshotOrDefault(pointer) = (%+v, %v), want initialized value and true", ptrState, ok)
	}
	defaultState, ok := SnapshotOrDefault("unsupported")
	if ok || defaultState.GMFear != GMFearDefault || defaultState.CharacterProfiles == nil {
		t.Fatalf("SnapshotOrDefault(unsupported) = (%+v, %v), want default state and false", defaultState, ok)
	}

	if _, err := AssertSnapshotState("unsupported"); err == nil {
		t.Fatal("AssertSnapshotState() error = nil, want error for unsupported type")
	}
	asserted, err := AssertSnapshotState(nil)
	if err != nil {
		t.Fatalf("AssertSnapshotState(nil) returned error: %v", err)
	}
	if asserted.GMFear != GMFearDefault || asserted.CharacterProfiles == nil {
		t.Fatalf("AssertSnapshotState(nil) = %+v, want default initialized snapshot", asserted)
	}

	if got := AppendUnique([]string{"one"}, "one"); !reflect.DeepEqual(got, []string{"one"}) {
		t.Fatalf("AppendUnique(existing) = %#v, want unchanged slice", got)
	}
	if got := AppendUnique([]string{"one"}, "two"); !reflect.DeepEqual(got, []string{"one", "two"}) {
		t.Fatalf("AppendUnique(new) = %#v, want appended slice", got)
	}

	newSnapshot := NewSnapshotState(" camp-1 ")
	if string(newSnapshot.CampaignID) != "camp-1" {
		t.Fatalf("NewSnapshotState() CampaignID = %q, want camp-1", newSnapshot.CampaignID)
	}
	if newSnapshot.CharacterProfiles == nil || newSnapshot.EnvironmentStates == nil {
		t.Fatalf("NewSnapshotState() left nil maps: %+v", newSnapshot)
	}
}

func TestCharacterSubclassStateHelpers(t *testing.T) {
	t.Parallel()

	raw := CharacterSubclassState{
		GiftedPerformerRelaxingSongUses:        -1,
		GiftedPerformerEpicSongUses:            -1,
		GiftedPerformerHeartbreakingSongUses:   -1,
		ContactsEverywhereUsesThisSession:      -1,
		ContactsEverywhereActionDieBonus:       -1,
		ContactsEverywhereDamageDiceBonusCount: -1,
		SparingTouchUsesThisLongRest:           -1,
		ElementalistActionBonus:                -1,
		ElementalistDamageBonus:                -1,
		TranscendenceTraitBonusTarget:          " agility ",
		TranscendenceTraitBonusValue:           -1,
		TranscendenceProficiencyBonus:          -1,
		TranscendenceEvasionBonus:              -1,
		TranscendenceSevereThresholdBonus:      -1,
		ElementalChannel:                       " storm ",
		NemesisTargetID:                        " adv-1 ",
	}

	normalized := raw.Normalized()
	if normalized.GiftedPerformerRelaxingSongUses != 0 || normalized.ElementalistDamageBonus != 0 {
		t.Fatalf("Normalized() = %+v, want non-negative uses and bonuses", normalized)
	}
	if normalized.TranscendenceTraitBonusTarget != "" || normalized.TranscendenceTraitBonusValue != 0 {
		t.Fatalf("Normalized() transcendence = %+v, want cleared inactive bonuses", normalized)
	}
	if normalized.ElementalChannel != "" || normalized.NemesisTargetID != "adv-1" {
		t.Fatalf("Normalized() channel/nemesis = %+v, want empty invalid channel and trimmed nemesis id", normalized)
	}
	if !raw.Equal(normalized) {
		t.Fatal("Equal() = false, want normalized equality")
	}
	if raw.IsZero() {
		t.Fatal("IsZero() = true, want false for populated subclass state")
	}
	if !(CharacterSubclassState{}).IsZero() {
		t.Fatal("IsZero() = false, want true for zero subclass state")
	}
	if got := NormalizedSubclassStatePtr(nil); got != nil {
		t.Fatalf("NormalizedSubclassStatePtr(nil) = %+v, want nil", got)
	}
}

func TestSubclassTrackHelpers(t *testing.T) {
	t.Parallel()

	primary := CharacterSubclassTrack{
		Origin:     SubclassTrackOriginPrimary,
		ClassID:    "class.guardian",
		SubclassID: "subclass.stalwart",
		Rank:       SubclassTrackRankFoundation,
	}
	tracks := []CharacterSubclassTrack{primary}

	gotPrimary, idx, ok := PrimarySubclassTrack(tracks)
	if !ok || idx != 0 || gotPrimary.SubclassID != "subclass.stalwart" {
		t.Fatalf("PrimarySubclassTrack() = (%+v, %d, %v), want primary track at index 0", gotPrimary, idx, ok)
	}

	advanced, promoted, err := AdvancePrimarySubclassTrack(tracks)
	if err != nil {
		t.Fatalf("AdvancePrimarySubclassTrack() returned error: %v", err)
	}
	if promoted.Rank != SubclassTrackRankSpecialization || advanced[0].Rank != SubclassTrackRankSpecialization {
		t.Fatalf("AdvancePrimarySubclassTrack() = %#v / %+v, want specialization rank", advanced, promoted)
	}
	if _, _, err := AdvancePrimarySubclassTrack([]CharacterSubclassTrack{{
		Origin:     SubclassTrackOriginPrimary,
		ClassID:    "class.guardian",
		SubclassID: "subclass.stalwart",
		Rank:       SubclassTrackRankMastery,
	}}); err == nil {
		t.Fatal("AdvancePrimarySubclassTrack() error = nil, want no-next-rank error")
	}

	withMulticlass, multiclass, err := AddMulticlassSubclassTrack(tracks, "class.seraph", "subclass.winged-sentinel", "domain.grace")
	if err != nil {
		t.Fatalf("AddMulticlassSubclassTrack() returned error: %v", err)
	}
	if multiclass.Origin != SubclassTrackOriginMulticlass || len(withMulticlass) != 2 {
		t.Fatalf("AddMulticlassSubclassTrack() = %#v / %+v, want appended multiclass track", withMulticlass, multiclass)
	}
	if _, _, err := AddMulticlassSubclassTrack(withMulticlass, "class.seraph", "subclass.winged-sentinel", "domain.grace"); err == nil {
		t.Fatal("AddMulticlassSubclassTrack() duplicate error = nil, want error")
	}

	replaced := EnsurePrimarySubclassTrack(withMulticlass, "class.ranger", "subclass.wayfinder")
	if got, _, _ := PrimarySubclassTrack(replaced); got.ClassID != "class.ranger" || got.SubclassID != "subclass.wayfinder" {
		t.Fatalf("EnsurePrimarySubclassTrack() primary = %+v, want replaced primary identity", got)
	}

	featureFoundation := contentstore.DaggerheartFeature{
		ID:   "feature.foundation",
		Name: "Foundation",
		SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
			Kind:  contentstore.DaggerheartSubclassFeatureRuleKindHPSlotBonus,
			Bonus: 1,
		},
	}
	featureSpecialization := contentstore.DaggerheartFeature{
		ID:   "feature.specialization",
		Name: "Specialization",
		SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
			Kind:            contentstore.DaggerheartSubclassFeatureRuleKindBonusMagicDamageOnSuccessWithFear,
			DamageDiceCount: 2,
			DamageDieSides:  8,
		},
	}
	featureMastery := contentstore.DaggerheartFeature{
		ID:   "feature.mastery",
		Name: "Mastery",
		SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{
			Kind:            contentstore.DaggerheartSubclassFeatureRuleKindEvasionBonusWhileHopeAtLeast,
			Bonus:           2,
			RequiredHopeMin: 3,
		},
	}
	subclass := contentstore.DaggerheartSubclass{
		ID:                     "subclass.stalwart",
		FoundationFeatures:     []contentstore.DaggerheartFeature{featureFoundation},
		SpecializationFeatures: []contentstore.DaggerheartFeature{featureSpecialization},
		MasteryFeatures:        []contentstore.DaggerheartFeature{featureMastery},
	}
	loadCalls := 0
	loaded, err := ActiveSubclassTrackFeaturesFromLoader(context.Background(), func(_ context.Context, id string) (contentstore.DaggerheartSubclass, error) {
		loadCalls++
		if id != "subclass.stalwart" {
			t.Fatalf("loader received id %q, want subclass.stalwart", id)
		}
		return subclass, nil
	}, []CharacterSubclassTrack{{
		Origin:     SubclassTrackOriginPrimary,
		ClassID:    "class.guardian",
		SubclassID: "subclass.stalwart",
		Rank:       SubclassTrackRankMastery,
	}})
	if err != nil {
		t.Fatalf("ActiveSubclassTrackFeaturesFromLoader() returned error: %v", err)
	}
	if loadCalls != 1 || len(loaded) != 1 || len(loaded[0].MasteryFeatures) != 1 {
		t.Fatalf("ActiveSubclassTrackFeaturesFromLoader() = %#v, want one loaded mastery track", loaded)
	}
	if _, err := ActiveSubclassTrackFeaturesFromLoader(context.Background(), nil, tracks); err == nil {
		t.Fatal("ActiveSubclassTrackFeaturesFromLoader(nil) error = nil, want loader error")
	}

	if got := UnlockedSubclassStageFeatures(subclass, SubclassTrackRankFoundation); len(got) != 1 || got[0].ID != "feature.foundation" {
		t.Fatalf("UnlockedSubclassStageFeatures(foundation) = %#v, want foundation feature", got)
	}
	if got := UnlockedSubclassStageFeatures(subclass, SubclassTrackRankSpecialization); len(got) != 1 || got[0].ID != "feature.specialization" {
		t.Fatalf("UnlockedSubclassStageFeatures(specialization) = %#v, want specialization feature", got)
	}
	if got := UnlockedSubclassStageFeatures(subclass, SubclassTrackRankMastery); len(got) != 1 || got[0].ID != "feature.mastery" {
		t.Fatalf("UnlockedSubclassStageFeatures(mastery) = %#v, want mastery feature", got)
	}
	if got := UnlockedSubclassStageFeatures(subclass, "unknown"); got != nil {
		t.Fatalf("UnlockedSubclassStageFeatures(unknown) = %#v, want nil", got)
	}

	flattened := FlattenActiveSubclassFeatures(loaded)
	if len(flattened) != 3 {
		t.Fatalf("FlattenActiveSubclassFeatures() = %#v, want 3 active features", flattened)
	}

	bonusFeatures := []contentstore.DaggerheartFeature{
		{SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{Kind: contentstore.DaggerheartSubclassFeatureRuleKindHPSlotBonus, Bonus: 1}},
		{SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{Kind: contentstore.DaggerheartSubclassFeatureRuleKindStressSlotBonus, Bonus: 2}},
		{SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{Kind: contentstore.DaggerheartSubclassFeatureRuleKindEvasionBonus, Bonus: 1}},
		{SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{Kind: contentstore.DaggerheartSubclassFeatureRuleKindThresholdBonus, Bonus: 2}},
		{SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{Kind: contentstore.DaggerheartSubclassFeatureRuleKindThresholdBonus, Bonus: 1, ThresholdScope: contentstore.DaggerheartSubclassThresholdScopeSevereOnly}},
	}
	bonuses := SubclassStatBonusesFromFeatures(bonusFeatures)
	if bonuses.HpMaxDelta != 1 || bonuses.StressMaxDelta != 2 || bonuses.EvasionDelta != 1 || bonuses.MajorThresholdDelta != 2 || bonuses.SevereThresholdDelta != 3 {
		t.Fatalf("SubclassStatBonusesFromFeatures() = %+v, want aggregated permanent bonuses", bonuses)
	}

	profile := &CharacterProfile{HpMax: 6, StressMax: 6, Evasion: 10, MajorThreshold: 8, SevereThreshold: 12}
	ApplySubclassStatBonuses(profile, bonuses)
	if profile.HpMax != 7 || profile.StressMax != 8 || profile.Evasion != 11 || profile.MajorThreshold != 10 || profile.SevereThreshold != 15 {
		t.Fatalf("ApplySubclassStatBonuses() profile = %+v, want applied bonuses", profile)
	}
	ApplySubclassStatBonuses(nil, bonuses)

	ruleSummary := SummarizeActiveSubclassRules([]contentstore.DaggerheartFeature{
		{SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{Kind: contentstore.DaggerheartSubclassFeatureRuleKindGainHopeOnFailureWithFear, Bonus: 1}},
		{SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{Kind: contentstore.DaggerheartSubclassFeatureRuleKindGainHopeOnFailureWithFear, Bonus: 2}},
		{SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{Kind: contentstore.DaggerheartSubclassFeatureRuleKindBonusMagicDamageOnSuccessWithFear, DamageDiceCount: 1, DamageDieSides: 10}},
		{SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{Kind: contentstore.DaggerheartSubclassFeatureRuleKindBonusMagicDamageOnSuccessWithFear, DamageDiceCount: 2, DamageDieSides: 8}},
		{SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{Kind: contentstore.DaggerheartSubclassFeatureRuleKindEvasionBonusWhileHopeAtLeast, Bonus: 2, RequiredHopeMin: 3}},
		{SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{Kind: contentstore.DaggerheartSubclassFeatureRuleKindBonusDamageWhileVulnerable, Bonus: 1}},
		{SubclassRule: &contentstore.DaggerheartSubclassFeatureRule{Kind: contentstore.DaggerheartSubclassFeatureRuleKindBonusDamageWhileVulnerable, UseCharacterLevel: true}},
	})
	if ruleSummary.GainHopeOnFailureWithFearAmount != 2 || ruleSummary.BonusMagicDamageDiceCount != 2 || ruleSummary.BonusMagicDamageDieSides != 8 || ruleSummary.EvasionBonusWhileHopeAtLeast != 2 || !ruleSummary.BonusDamageWhileVulnerableLevel {
		t.Fatalf("SummarizeActiveSubclassRules() = %+v, want summarized maxima and level-based vulnerable damage", ruleSummary)
	}

	if got, ok := NextSubclassTrackRank(SubclassTrackRankFoundation); !ok || got != SubclassTrackRankSpecialization {
		t.Fatalf("NextSubclassTrackRank(foundation) = (%q, %v), want specialization/true", got, ok)
	}
	if got, ok := NextSubclassTrackRank(SubclassTrackRankSpecialization); !ok || got != SubclassTrackRankMastery {
		t.Fatalf("NextSubclassTrackRank(specialization) = (%q, %v), want mastery/true", got, ok)
	}
	if got, ok := NextSubclassTrackRank(SubclassTrackRankMastery); ok || got != "" {
		t.Fatalf("NextSubclassTrackRank(mastery) = (%q, %v), want empty/false", got, ok)
	}
}
