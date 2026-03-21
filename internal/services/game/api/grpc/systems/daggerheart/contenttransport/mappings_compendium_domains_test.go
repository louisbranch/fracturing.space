package contenttransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func TestToProtoDaggerheartDamageTypeEntry(t *testing.T) {
	proto := toProtoDaggerheartDamageType(contentstore.DaggerheartDamageTypeEntry{
		ID:          "dt-1",
		Name:        "Fire",
		Description: "Burns things",
	})

	if proto.GetId() != "dt-1" || proto.GetName() != "Fire" || proto.GetDescription() != "Burns things" {
		t.Fatalf("damage type entry mismatch: %v", proto)
	}
}

func TestToProtoDaggerheartDamageTypes(t *testing.T) {
	protos := toProtoDaggerheartDamageTypes([]contentstore.DaggerheartDamageTypeEntry{
		{ID: "d1", Name: "A"},
		{ID: "d2", Name: "B"},
	})
	if len(protos) != 2 || protos[0].GetId() != "d1" {
		t.Fatalf("damage types slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartDomain(t *testing.T) {
	proto := toProtoDaggerheartDomain(contentstore.DaggerheartDomain{
		ID:          "dom-1",
		Name:        "Valor",
		Description: "Bravery domain",
	})

	if proto.GetId() != "dom-1" || proto.GetName() != "Valor" || proto.GetDescription() != "Bravery domain" {
		t.Fatalf("domain mismatch: %v", proto)
	}
}

func TestToProtoDaggerheartDomains(t *testing.T) {
	protos := toProtoDaggerheartDomains([]contentstore.DaggerheartDomain{
		{ID: "d1", Name: "A"},
		{ID: "d2", Name: "B"},
	})
	if len(protos) != 2 || protos[1].GetId() != "d2" {
		t.Fatalf("domains slice mismatch: got %d items", len(protos))
	}
}

func TestToProtoDaggerheartDomainCard(t *testing.T) {
	proto := toProtoDaggerheartDomainCard(contentstore.DaggerheartDomainCard{
		ID:          "card-1",
		Name:        "Fireball",
		DomainID:    "dom-1",
		Level:       3,
		Type:        "spell",
		RecallCost:  2,
		UsageLimit:  "once",
		FeatureText: "Deals fire damage",
	})

	if proto.GetId() != "card-1" || proto.GetName() != "Fireball" {
		t.Fatalf("domain card metadata mismatch: %v", proto)
	}
	if proto.GetDomainId() != "dom-1" || proto.GetLevel() != 3 {
		t.Fatalf("domain card domain/level mismatch")
	}
	if proto.GetType() != pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_SPELL {
		t.Fatalf("domain card type mismatch: %v", proto.GetType())
	}
	if proto.GetRecallCost() != 2 || proto.GetUsageLimit() != "once" || proto.GetFeatureText() != "Deals fire damage" {
		t.Fatalf("domain card fields mismatch")
	}
}

func TestToProtoDaggerheartDomainCards(t *testing.T) {
	protos := toProtoDaggerheartDomainCards([]contentstore.DaggerheartDomainCard{
		{ID: "c1", Name: "A"},
		{ID: "c2", Name: "B"},
	})
	if len(protos) != 2 || protos[0].GetId() != "c1" {
		t.Fatalf("domain cards slice mismatch: got %d items", len(protos))
	}
}
