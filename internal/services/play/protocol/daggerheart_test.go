package protocol

import (
	"strings"
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestDaggerheartCardFromSheetBasicMapping(t *testing.T) {
	t.Parallel()

	char := &gamev1.Character{
		Id:            "char-1",
		CampaignId:    "camp-1",
		Name:          "Lark",
		Kind:          gamev1.CharacterKind_PC,
		AvatarSetId:   "avatar_set_v1",
		AvatarAssetId: "ceremonial_choir_lead",
	}
	profile := &daggerheartv1.DaggerheartProfile{
		Level:      3,
		HpMax:      12,
		ClassId:    "class.guardian",
		SubclassId: "subclass.stalwart",
		StressMax:  wrapperspb.Int32(6),
		Evasion:    wrapperspb.Int32(10),
		ArmorScore: wrapperspb.Int32(3),
		ArmorMax:   wrapperspb.Int32(3),
		Agility:    wrapperspb.Int32(2),
		Strength:   wrapperspb.Int32(-1),
		Finesse:    wrapperspb.Int32(0),
		Heritage: &daggerheartv1.DaggerheartHeritageSelection{
			AncestryName:  "Elf",
			CommunityName: "Loreborne",
		},
		ActiveClassFeatures: []*daggerheartv1.DaggerheartActiveClassFeature{
			{Name: "Unstoppable", Level: 1, HopeFeature: false},
			{Name: "Stand Firm", Level: 1, HopeFeature: true},
		},
	}
	state := &daggerheartv1.DaggerheartCharacterState{
		Hp:      8,
		Stress:  2,
		Armor:   1,
		Hope:    3,
		HopeMax: 5,
	}

	card := DaggerheartCardFromSheet("https://cdn.example.com/assets", char, profile, state)

	if card.ID != "char-1" || card.Name != "Lark" {
		t.Fatalf("card identity = %q / %q", card.ID, card.Name)
	}
	if !strings.Contains(card.Portrait.Src, "ceremonial_choir_lead") {
		t.Fatalf("card portrait src = %q, want asset-backed URL", card.Portrait.Src)
	}
	if card.Identity == nil || card.Identity.Kind != "pc" {
		t.Fatalf("card identity kind = %#v", card.Identity)
	}
	if card.Daggerheart == nil {
		t.Fatal("card daggerheart section missing")
	}
	summary := card.Daggerheart.Summary
	if summary == nil {
		t.Fatal("card summary missing")
	}
	if summary.Level != 3 {
		t.Fatalf("level = %d, want 3", summary.Level)
	}
	if summary.ClassName != "Guardian" {
		t.Fatalf("className = %q, want %q", summary.ClassName, "Guardian")
	}
	if summary.SubclassName != "Stalwart" {
		t.Fatalf("subclassName = %q, want %q", summary.SubclassName, "Stalwart")
	}
	if summary.AncestryName != "Elf" {
		t.Fatalf("ancestryName = %q, want %q", summary.AncestryName, "Elf")
	}
	if summary.CommunityName != "Loreborne" {
		t.Fatalf("communityName = %q, want %q", summary.CommunityName, "Loreborne")
	}
	if summary.HP == nil || summary.HP.Current != 8 || summary.HP.Max != 12 {
		t.Fatalf("hp = %#v", summary.HP)
	}
	if summary.Stress == nil || summary.Stress.Current != 2 || summary.Stress.Max != 6 {
		t.Fatalf("stress = %#v", summary.Stress)
	}
	if summary.Hope == nil || summary.Hope.Current != 3 || summary.Hope.Max != 5 {
		t.Fatalf("hope = %#v", summary.Hope)
	}
	if summary.Armor == nil || summary.Armor.Current != 1 || summary.Armor.Max != 3 {
		t.Fatalf("armor = %#v", summary.Armor)
	}
	if summary.Feature != "Stand Firm" {
		t.Fatalf("feature = %q, want %q", summary.Feature, "Stand Firm")
	}
	traits := card.Daggerheart.Traits
	if traits == nil {
		t.Fatal("traits missing")
	}
	if traits.Agility != "+2" {
		t.Fatalf("agility = %q, want %q", traits.Agility, "+2")
	}
	if traits.Strength != "-1" {
		t.Fatalf("strength = %q, want %q", traits.Strength, "-1")
	}
	if traits.Finesse != "+0" {
		t.Fatalf("finesse = %q, want %q", traits.Finesse, "+0")
	}
}

func TestDaggerheartSheetFromResponseIncludesConditions(t *testing.T) {
	t.Parallel()

	char := &gamev1.Character{
		Id:   "char-2",
		Name: "Riven",
		Kind: gamev1.CharacterKind_NPC,
	}
	profile := &daggerheartv1.DaggerheartProfile{
		Level: 1,
		HpMax: 6,
		DomainCardIds: []string{
			"domain_card.valor-i-am-your-shield",
			"domain_card.blade-get-back-up",
		},
	}
	state := &daggerheartv1.DaggerheartCharacterState{
		Hp:        3,
		HopeMax:   4,
		LifeState: daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
		ConditionStates: []*daggerheartv1.DaggerheartConditionState{
			{Label: "Hidden"},
			{Code: "restrained"},
		},
	}

	sheet := DaggerheartSheetFromResponse("https://cdn.example.com/assets", char, profile, state, nil)

	if sheet.LifeState != "unconscious" {
		t.Fatalf("lifeState = %q, want %q", sheet.LifeState, "unconscious")
	}
	if len(sheet.Conditions) != 2 || sheet.Conditions[0] != "Hidden" || sheet.Conditions[1] != "restrained" {
		t.Fatalf("conditions = %#v", sheet.Conditions)
	}
	if sheet.Kind != "npc" {
		t.Fatalf("kind = %q, want %q", sheet.Kind, "npc")
	}
	if len(sheet.DomainCards) != 2 {
		t.Fatalf("domain cards = %#v, want 2 readable titles", sheet.DomainCards)
	}
	if sheet.DomainCards[0].ID != "domain_card.valor-i-am-your-shield" {
		t.Fatalf("domainCards[0].ID = %q, want %q", sheet.DomainCards[0].ID, "domain_card.valor-i-am-your-shield")
	}
	if sheet.DomainCards[0].Name != "I Am Your Shield" {
		t.Fatalf("domainCards[0].Name = %q, want %q", sheet.DomainCards[0].Name, "I Am Your Shield")
	}
	if sheet.DomainCards[0].Domain != "Valor" {
		t.Fatalf("domainCards[0].Domain = %q, want %q", sheet.DomainCards[0].Domain, "Valor")
	}
	if sheet.DomainCards[1].Name != "Get Back Up" {
		t.Fatalf("domainCards[1].Name = %q, want %q", sheet.DomainCards[1].Name, "Get Back Up")
	}
	if sheet.DomainCards[1].Domain != "Blade" {
		t.Fatalf("domainCards[1].Domain = %q, want %q", sheet.DomainCards[1].Domain, "Blade")
	}
	if len(sheet.Traits) != 0 {
		t.Fatalf("traits = %#v, want none when profile traits are unset", sheet.Traits)
	}
}

func TestDaggerheartSheetFromResponseUsesEnrichedDomainCardLookup(t *testing.T) {
	t.Parallel()

	char := &gamev1.Character{Id: "c", Name: "Lark"}
	profile := &daggerheartv1.DaggerheartProfile{
		DomainCardIds: []string{"domain_card.valor-i-am-your-shield"},
	}

	sheet := DaggerheartSheetFromResponse("", char, profile, nil, map[string]DaggerheartDomainCard{
		"domain_card.valor-i-am-your-shield": {
			ID:          "domain_card.valor-i-am-your-shield",
			Name:        "I Am Your Shield",
			Domain:      "Valor",
			FeatureText: "Protect an ally within very close range.",
		},
	})

	if len(sheet.DomainCards) != 1 {
		t.Fatalf("domain cards = %#v, want 1", sheet.DomainCards)
	}
	if sheet.DomainCards[0].FeatureText != "Protect an ally within very close range." {
		t.Fatalf("featureText = %q", sheet.DomainCards[0].FeatureText)
	}
}

func TestDaggerheartCardFromSheetNilProfileAndState(t *testing.T) {
	t.Parallel()

	char := &gamev1.Character{Id: "c", Name: "Blank"}
	card := DaggerheartCardFromSheet("", char, nil, nil)

	if card.Daggerheart != nil {
		t.Fatalf("expected nil daggerheart section, got %#v", card.Daggerheart)
	}
}

func TestDaggerheartSheetExperiences(t *testing.T) {
	t.Parallel()

	char := &gamev1.Character{Id: "c", Name: "Test"}
	profile := &daggerheartv1.DaggerheartProfile{
		Level: 1,
		HpMax: 6,
		Experiences: []*daggerheartv1.DaggerheartExperience{
			{Name: "Athletics", Modifier: 2},
			{Name: "Stealth", Modifier: -1},
		},
		DomainCardIds: []string{"dc-1", "dc-2"},
	}
	sheet := DaggerheartSheetFromResponse("", char, profile, nil, nil)

	if len(sheet.Experiences) != 2 {
		t.Fatalf("experiences count = %d, want 2", len(sheet.Experiences))
	}
	if sheet.Experiences[0].Name != "Athletics" || sheet.Experiences[0].Modifier != 2 {
		t.Fatalf("experience[0] = %#v", sheet.Experiences[0])
	}
	if len(sheet.DomainCards) != 2 {
		t.Fatalf("domain cards count = %d, want 2", len(sheet.DomainCards))
	}
	if len(sheet.Traits) != 0 {
		t.Fatalf("traits count = %d, want 0 when no trait wrappers are provided", len(sheet.Traits))
	}
}

func TestDaggerheartSheetFromResponseIncludesFullFeatureTextAndEquipment(t *testing.T) {
	t.Parallel()

	char := &gamev1.Character{Id: "c", Name: "Mira"}
	profile := &daggerheartv1.DaggerheartProfile{
		ActiveClassFeatures: []*daggerheartv1.DaggerheartActiveClassFeature{
			{
				Name:        "Rogue's Dodge",
				Description: "Spend 3 Hope to gain +2 Evasion until an attack succeeds against you.",
				HopeFeature: true,
			},
			{
				Name:        "Sneak Attack",
				Description: "When you have advantage on a melee attack, deal an extra 1d8 damage.",
			},
		},
		PrimaryWeapon: &daggerheartv1.DaggerheartSheetWeaponSummary{
			Name:       "Sword",
			Trait:      "Finesse",
			Range:      "melee",
			DamageDice: "1d8",
			DamageType: "physical",
			Feature:    "Versatile",
		},
		SecondaryWeapon: &daggerheartv1.DaggerheartSheetWeaponSummary{
			Name:       "Dagger",
			Trait:      "Finesse",
			Range:      "very close",
			DamageDice: "1d6",
			DamageType: "physical",
		},
		ActiveArmor: &daggerheartv1.DaggerheartSheetArmorSummary{
			Name:      "Leather",
			BaseScore: 2,
			Feature:   "Quiet",
		},
	}

	sheet := DaggerheartSheetFromResponse("", char, profile, nil, nil)

	if sheet.HopeFeature != "Rogue's Dodge: Spend 3 Hope to gain +2 Evasion until an attack succeeds against you." {
		t.Fatalf("hope feature = %q", sheet.HopeFeature)
	}
	if sheet.ClassFeature != "Sneak Attack: When you have advantage on a melee attack, deal an extra 1d8 damage." {
		t.Fatalf("class feature = %q", sheet.ClassFeature)
	}
	if sheet.PrimaryWeapon == nil || sheet.PrimaryWeapon.Name != "Sword" || sheet.PrimaryWeapon.DamageDice != "1d8" {
		t.Fatalf("primary weapon = %#v", sheet.PrimaryWeapon)
	}
	if sheet.SecondaryWeapon == nil || sheet.SecondaryWeapon.Name != "Dagger" {
		t.Fatalf("secondary weapon = %#v", sheet.SecondaryWeapon)
	}
	if sheet.ActiveArmor == nil || sheet.ActiveArmor.Name != "Leather" {
		t.Fatalf("active armor = %#v", sheet.ActiveArmor)
	}
	if sheet.ActiveArmor.BaseScore == nil || *sheet.ActiveArmor.BaseScore != 2 {
		t.Fatalf("active armor base score = %#v", sheet.ActiveArmor.BaseScore)
	}
}

func TestDaggerheartHeritageDisplayFallsBackToAncestryLabel(t *testing.T) {
	t.Parallel()

	char := &gamev1.Character{Id: "c", Name: "Mira"}
	profile := &daggerheartv1.DaggerheartProfile{
		Heritage: &daggerheartv1.DaggerheartHeritageSelection{
			AncestryLabel: "Half-Clank",
		},
	}

	card := DaggerheartCardFromSheet("", char, profile, nil)
	if card.Daggerheart == nil || card.Daggerheart.Summary == nil {
		t.Fatalf("card summary = %#v", card.Daggerheart)
	}
	if got := card.Daggerheart.Summary.AncestryName; got != "Half-Clank" {
		t.Fatalf("card ancestryName = %q, want %q", got, "Half-Clank")
	}
	if got := card.Daggerheart.Summary.CommunityName; got != "" {
		t.Fatalf("card communityName = %q, want empty", got)
	}

	sheet := DaggerheartSheetFromResponse("", char, profile, nil, nil)
	if got := sheet.AncestryName; got != "Half-Clank" {
		t.Fatalf("sheet ancestryName = %q, want %q", got, "Half-Clank")
	}
	if got := sheet.CommunityName; got != "" {
		t.Fatalf("sheet communityName = %q, want empty", got)
	}
}

func TestDaggerheartSheetTraitsIncludeSkillLabels(t *testing.T) {
	t.Parallel()

	char := &gamev1.Character{Id: "c", Name: "Test"}
	profile := &daggerheartv1.DaggerheartProfile{
		Agility: wrapperspb.Int32(2),
	}

	sheet := DaggerheartSheetFromResponse("", char, profile, nil, nil)

	if len(sheet.Traits) != 1 {
		t.Fatalf("traits count = %d, want 1", len(sheet.Traits))
	}
	if len(sheet.Traits[0].Skills) == 0 {
		t.Fatalf("trait skills = %#v, want built-in skill labels", sheet.Traits[0].Skills)
	}
	if sheet.Traits[0].Skills[0] != "Sprint" {
		t.Fatalf("trait skills[0] = %q, want %q", sheet.Traits[0].Skills[0], "Sprint")
	}
}

func TestDaggerheartArmorTrack_UsesBaseArmorSlotsAndArmorScore(t *testing.T) {
	t.Parallel()

	char := &gamev1.Character{Id: "char-3", Name: "Vale"}
	profile := &daggerheartv1.DaggerheartProfile{
		ArmorScore: wrapperspb.Int32(4),
		ArmorMax:   wrapperspb.Int32(4),
	}
	state := &daggerheartv1.DaggerheartCharacterState{
		Armor: 5,
		TemporaryArmorBuckets: []*daggerheartv1.DaggerheartTemporaryArmorBucket{
			{Amount: 2},
		},
	}

	card := DaggerheartCardFromSheet("", char, profile, state)
	if card.Daggerheart == nil || card.Daggerheart.Summary == nil || card.Daggerheart.Summary.Armor == nil {
		t.Fatalf("card armor summary = %#v", card.Daggerheart)
	}
	if card.Daggerheart.Summary.Armor.Current != 3 || card.Daggerheart.Summary.Armor.Max != 4 {
		t.Fatalf("card armor = %#v, want 3/4", card.Daggerheart.Summary.Armor)
	}

	sheet := DaggerheartSheetFromResponse("", char, profile, state, nil)
	if sheet.Armor == nil || sheet.Armor.Current != 3 || sheet.Armor.Max != 4 {
		t.Fatalf("sheet armor = %#v, want 3/4", sheet.Armor)
	}
}
