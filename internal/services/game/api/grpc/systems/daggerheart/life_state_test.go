package daggerheart

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func TestDaggerheartLifeStateFromProto(t *testing.T) {
	if _, err := daggerheartLifeStateFromProto(pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED); err == nil {
		t.Fatal("expected error for unspecified life state")
	}

	state, err := daggerheartLifeStateFromProto(pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state != daggerheart.LifeStateAlive {
		t.Fatalf("expected alive, got %s", state)
	}

	if _, err := daggerheartLifeStateFromProto(pb.DaggerheartLifeState(99)); err == nil {
		t.Fatal("expected error for invalid life state")
	}
}

func TestDaggerheartLifeStateToProto(t *testing.T) {
	if daggerheartLifeStateToProto(daggerheart.LifeStateDead) != pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD {
		t.Fatal("expected dead life state")
	}
	if daggerheartLifeStateToProto("unknown") != pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED {
		t.Fatal("expected unspecified for unknown life state")
	}
}

func TestDaggerheartDeathMoveFromProto(t *testing.T) {
	if _, err := daggerheartDeathMoveFromProto(pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED); err == nil {
		t.Fatal("expected error for unspecified death move")
	}

	move, err := daggerheartDeathMoveFromProto(pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if move != daggerheart.DeathMoveAvoidDeath {
		t.Fatalf("expected avoid_death, got %s", move)
	}

	if _, err := daggerheartDeathMoveFromProto(pb.DaggerheartDeathMove(99)); err == nil {
		t.Fatal("expected error for invalid death move")
	}
}

func TestDaggerheartDeathMoveToProto(t *testing.T) {
	if daggerheartDeathMoveToProto(daggerheart.DeathMoveRiskItAll) != pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL {
		t.Fatal("expected risk_it_all death move")
	}
	if daggerheartDeathMoveToProto("unknown") != pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED {
		t.Fatal("expected unspecified for unknown death move")
	}
}
