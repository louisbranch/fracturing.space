package contenttransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func TestToProtoDaggerheartClass(t *testing.T) {
	proto := toProtoDaggerheartClass(contentstore.DaggerheartClass{
		ID:              "class-1",
		Name:            "Guardian",
		StartingEvasion: 12,
		StartingHP:      16,
		StartingItems:   []string{"shield", "blade"},
		Features: []contentstore.DaggerheartFeature{{
			ID:          "feat-1",
			Name:        "Hold the Line",
			Description: "Stand firm",
			Level:       1,
		}},
		HopeFeature: contentstore.DaggerheartHopeFeature{
			Name:        "Resolve",
			Description: "Keep fighting",
			HopeCost:    2,
		},
		DomainIDs: []string{"valor", "shield"},
	})

	if proto.GetId() != "class-1" || proto.GetName() != "Guardian" {
		t.Fatalf("class metadata mismatch: %v", proto)
	}
	if proto.GetStartingEvasion() != 12 || proto.GetStartingHp() != 16 {
		t.Fatalf("starting stats mismatch: %v", proto)
	}
	if len(proto.GetStartingItems()) != 2 || proto.GetStartingItems()[0] != "shield" {
		t.Fatalf("starting items mismatch: %v", proto.GetStartingItems())
	}
	if len(proto.GetDomainIds()) != 2 || proto.GetDomainIds()[1] != "shield" {
		t.Fatalf("domain ids mismatch: %v", proto.GetDomainIds())
	}
	if len(proto.GetFeatures()) != 1 || proto.GetFeatures()[0].GetId() != "feat-1" {
		t.Fatalf("features mismatch: %v", proto.GetFeatures())
	}
	if proto.GetHopeFeature().GetHopeCost() != 2 {
		t.Fatalf("hope feature mismatch: %v", proto.GetHopeFeature())
	}
}

func TestToProtoDaggerheartClasses(t *testing.T) {
	protos := toProtoDaggerheartClasses([]contentstore.DaggerheartClass{
		{ID: "c1", Name: "A"},
		{ID: "c2", Name: "B"},
	})
	if len(protos) != 2 || protos[0].GetId() != "c1" || protos[1].GetId() != "c2" {
		t.Fatalf("classes slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartSubclass(t *testing.T) {
	proto := toProtoDaggerheartSubclass(contentstore.DaggerheartSubclass{
		ID:             "sub-1",
		Name:           "Bladeweaver",
		SpellcastTrait: "agility",
		FoundationFeatures: []contentstore.DaggerheartFeature{
			{ID: "f1", Name: "Foundation", Description: "Base", Level: 1},
		},
		SpecializationFeatures: []contentstore.DaggerheartFeature{
			{ID: "f2", Name: "Specialization", Description: "Mid", Level: 5},
		},
		MasteryFeatures: []contentstore.DaggerheartFeature{
			{ID: "f3", Name: "Mastery", Description: "Top", Level: 10},
		},
	})

	if proto.GetId() != "sub-1" || proto.GetName() != "Bladeweaver" {
		t.Fatalf("subclass metadata mismatch: %v", proto)
	}
	if proto.GetSpellcastTrait() != "agility" {
		t.Fatalf("spellcast trait mismatch: %v", proto.GetSpellcastTrait())
	}
	if len(proto.GetFoundationFeatures()) != 1 || proto.GetFoundationFeatures()[0].GetId() != "f1" {
		t.Fatalf("foundation features mismatch: %v", proto.GetFoundationFeatures())
	}
	if len(proto.GetSpecializationFeatures()) != 1 || proto.GetSpecializationFeatures()[0].GetId() != "f2" {
		t.Fatalf("specialization features mismatch: %v", proto.GetSpecializationFeatures())
	}
	if len(proto.GetMasteryFeatures()) != 1 || proto.GetMasteryFeatures()[0].GetId() != "f3" {
		t.Fatalf("mastery features mismatch: %v", proto.GetMasteryFeatures())
	}
}

func TestToProtoDaggerheartSubclasses(t *testing.T) {
	protos := toProtoDaggerheartSubclasses([]contentstore.DaggerheartSubclass{
		{ID: "s1", Name: "A"},
		{ID: "s2", Name: "B"},
	})
	if len(protos) != 2 || protos[0].GetId() != "s1" || protos[1].GetId() != "s2" {
		t.Fatalf("subclasses slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartHeritage(t *testing.T) {
	proto := toProtoDaggerheartHeritage(contentstore.DaggerheartHeritage{
		ID:   "her-1",
		Name: "Elf",
		Kind: "ancestry",
		Features: []contentstore.DaggerheartFeature{
			{ID: "f1", Name: "Keen Eyes", Description: "See far", Level: 1},
		},
	})

	if proto.GetId() != "her-1" || proto.GetName() != "Elf" {
		t.Fatalf("heritage metadata mismatch: %v", proto)
	}
	if proto.GetKind() != pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY {
		t.Fatalf("heritage kind mismatch: %v", proto.GetKind())
	}
	if len(proto.GetFeatures()) != 1 || proto.GetFeatures()[0].GetId() != "f1" {
		t.Fatalf("heritage features mismatch: %v", proto.GetFeatures())
	}
}

func TestToProtoDaggerheartHeritages(t *testing.T) {
	protos := toProtoDaggerheartHeritages([]contentstore.DaggerheartHeritage{
		{ID: "h1", Name: "A", Kind: "ancestry"},
		{ID: "h2", Name: "B", Kind: "community"},
	})
	if len(protos) != 2 || protos[0].GetId() != "h1" {
		t.Fatalf("heritages slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartExperience(t *testing.T) {
	proto := toProtoDaggerheartExperience(contentstore.DaggerheartExperienceEntry{
		ID:          "exp-1",
		Name:        "Wanderer",
		Description: "Traveled far",
	})

	if proto.GetId() != "exp-1" || proto.GetName() != "Wanderer" || proto.GetDescription() != "Traveled far" {
		t.Fatalf("experience mismatch: %v", proto)
	}
}

func TestToProtoDaggerheartExperiences(t *testing.T) {
	protos := toProtoDaggerheartExperiences([]contentstore.DaggerheartExperienceEntry{
		{ID: "e1", Name: "A"},
		{ID: "e2", Name: "B"},
	})
	if len(protos) != 2 || protos[1].GetId() != "e2" {
		t.Fatalf("experiences slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartFeatures(t *testing.T) {
	protos := toProtoDaggerheartFeatures([]contentstore.DaggerheartFeature{
		{ID: "f1", Name: "Shield Bash", Description: "Bash with shield", Level: 1},
		{ID: "f2", Name: "Rally", Description: "Rally allies", Level: 5},
	})

	if len(protos) != 2 {
		t.Fatalf("expected 2 features, got %d", len(protos))
	}
	if protos[0].GetId() != "f1" || protos[0].GetName() != "Shield Bash" {
		t.Fatalf("feature 0 mismatch: %v", protos[0])
	}
	if protos[1].GetLevel() != 5 || protos[1].GetDescription() != "Rally allies" {
		t.Fatalf("feature 1 mismatch: %v", protos[1])
	}
}

func TestToProtoDaggerheartHopeFeature(t *testing.T) {
	proto := toProtoDaggerheartHopeFeature(contentstore.DaggerheartHopeFeature{
		Name:        "Inspiring",
		Description: "Inspire allies",
		HopeCost:    3,
	})

	if proto.GetName() != "Inspiring" || proto.GetDescription() != "Inspire allies" || proto.GetHopeCost() != 3 {
		t.Fatalf("hope feature mismatch: %v", proto)
	}
}

func TestToProtoDaggerheartDamageDice(t *testing.T) {
	dice := toProtoDaggerheartDamageDice([]contentstore.DaggerheartDamageDie{{Sides: 6, Count: 2}})
	if len(dice) != 1 {
		t.Fatalf("expected 1 dice spec, got %d", len(dice))
	}
	if dice[0].GetSides() != 6 || dice[0].GetCount() != 2 {
		t.Fatalf("dice mapping mismatch: %v", dice[0])
	}
}

func TestHeritageKindToProto(t *testing.T) {
	if heritageKindToProto(" Ancestry ") != pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY {
		t.Fatal("expected ancestry heritage kind")
	}
	if heritageKindToProto("community") != pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_COMMUNITY {
		t.Fatal("expected community heritage kind")
	}
	if heritageKindToProto("unknown") != pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_UNSPECIFIED {
		t.Fatal("expected unspecified heritage kind")
	}
}
