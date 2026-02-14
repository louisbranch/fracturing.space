package daggerheart

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func TestDaggerheartConditionsFromProto(t *testing.T) {
	if conditions, err := daggerheartConditionsFromProto(nil); err != nil || conditions != nil {
		t.Fatalf("expected nil conditions, got %v (err=%v)", conditions, err)
	}

	conditions, err := daggerheartConditionsFromProto([]pb.DaggerheartCondition{
		pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN,
		pb.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(conditions) != 2 || conditions[0] != daggerheart.ConditionHidden || conditions[1] != daggerheart.ConditionRestrained {
		t.Fatalf("unexpected conditions: %v", conditions)
	}

	if _, err := daggerheartConditionsFromProto([]pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_UNSPECIFIED}); err == nil {
		t.Fatal("expected error for unspecified condition")
	}

	if _, err := daggerheartConditionsFromProto([]pb.DaggerheartCondition{pb.DaggerheartCondition(99)}); err == nil {
		t.Fatal("expected error for invalid condition")
	}
}

func TestDaggerheartConditionsToProto(t *testing.T) {
	if daggerheartConditionsToProto(nil) != nil {
		t.Fatal("expected nil proto slice")
	}

	proto := daggerheartConditionsToProto([]string{
		daggerheart.ConditionHidden,
		daggerheart.ConditionVulnerable,
		"unknown",
	})
	if len(proto) != 2 {
		t.Fatalf("expected 2 proto conditions, got %d", len(proto))
	}
	if proto[0] != pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN {
		t.Fatalf("unexpected first condition: %v", proto[0])
	}
	if proto[1] != pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE {
		t.Fatalf("unexpected second condition: %v", proto[1])
	}
}
