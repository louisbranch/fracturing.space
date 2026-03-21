package protocol

import (
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestDaggerheartCardFromSheetBasicMapping(t *testing.T) {
	t.Parallel()

	char := &gamev1.Character{
		Id:   "char-1",
		Name: "Lark",
		Kind: gamev1.CharacterKind_PC,
	}
	profile := &daggerheartv1.DaggerheartProfile{
		Level:     3,
		HpMax:     12,
		StressMax: wrapperspb.Int32(6),
		Evasion:   wrapperspb.Int32(10),
		ArmorMax:  wrapperspb.Int32(3),
		Agility:   wrapperspb.Int32(2),
		Strength:  wrapperspb.Int32(-1),
		Finesse:   wrapperspb.Int32(0),
		Heritage: &daggerheartv1.DaggerheartHeritageSelection{
			AncestryLabel: "Elf",
		},
		ActiveClassFeatures: []*daggerheartv1.DaggerheartActiveClassFeature{
			{Name: "Bard", Level: 1, HopeFeature: false},
			{Name: "Inspiring Song", Level: 1, HopeFeature: true},
		},
	}
	state := &daggerheartv1.DaggerheartCharacterState{
		Hp:      8,
		Stress:  2,
		Armor:   1,
		Hope:    3,
		HopeMax: 5,
	}

	card := DaggerheartCardFromSheet(char, profile, state)

	if card.ID != "char-1" || card.Name != "Lark" {
		t.Fatalf("card identity = %q / %q", card.ID, card.Name)
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
	if summary.ClassName != "Bard" {
		t.Fatalf("className = %q, want %q", summary.ClassName, "Bard")
	}
	if summary.AncestryName != "Elf" {
		t.Fatalf("ancestryName = %q, want %q", summary.AncestryName, "Elf")
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
	if summary.Feature != "Inspiring Song" {
		t.Fatalf("feature = %q, want %q", summary.Feature, "Inspiring Song")
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

	sheet := DaggerheartSheetFromResponse(char, profile, state)

	if sheet.LifeState != "unconscious" {
		t.Fatalf("lifeState = %q, want %q", sheet.LifeState, "unconscious")
	}
	if len(sheet.Conditions) != 2 || sheet.Conditions[0] != "Hidden" || sheet.Conditions[1] != "restrained" {
		t.Fatalf("conditions = %#v", sheet.Conditions)
	}
	if sheet.Kind != "npc" {
		t.Fatalf("kind = %q, want %q", sheet.Kind, "npc")
	}
}

func TestDaggerheartCardFromSheetNilProfileAndState(t *testing.T) {
	t.Parallel()

	char := &gamev1.Character{Id: "c", Name: "Blank"}
	card := DaggerheartCardFromSheet(char, nil, nil)

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
	sheet := DaggerheartSheetFromResponse(char, profile, nil)

	if len(sheet.Experiences) != 2 {
		t.Fatalf("experiences count = %d, want 2", len(sheet.Experiences))
	}
	if sheet.Experiences[0].Name != "Athletics" || sheet.Experiences[0].Modifier != 2 {
		t.Fatalf("experience[0] = %#v", sheet.Experiences[0])
	}
	if len(sheet.DomainCards) != 2 {
		t.Fatalf("domain cards count = %d, want 2", len(sheet.DomainCards))
	}
}
