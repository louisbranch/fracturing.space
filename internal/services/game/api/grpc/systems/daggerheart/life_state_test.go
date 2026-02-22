package daggerheart

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
)

func TestDaggerheartLifeStateFromProto(t *testing.T) {
	if _, err := daggerheartLifeStateFromProto(pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED); err == nil {
		t.Fatal("expected error for unspecified life state")
	}

	tests := []struct {
		proto pb.DaggerheartLifeState
		want  string
	}{
		{pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE, daggerheart.LifeStateAlive},
		{pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS, daggerheart.LifeStateUnconscious},
		{pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY, daggerheart.LifeStateBlazeOfGlory},
		{pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD, daggerheart.LifeStateDead},
	}
	for _, tc := range tests {
		state, err := daggerheartLifeStateFromProto(tc.proto)
		if err != nil {
			t.Fatalf("unexpected error for %v: %v", tc.proto, err)
		}
		if state != tc.want {
			t.Fatalf("expected %s, got %s", tc.want, state)
		}
	}

	if _, err := daggerheartLifeStateFromProto(pb.DaggerheartLifeState(99)); err == nil {
		t.Fatal("expected error for invalid life state")
	}
}

func TestDaggerheartLifeStateToProto(t *testing.T) {
	tests := []struct {
		input string
		want  pb.DaggerheartLifeState
	}{
		{daggerheart.LifeStateAlive, pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE},
		{daggerheart.LifeStateUnconscious, pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS},
		{daggerheart.LifeStateBlazeOfGlory, pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY},
		{daggerheart.LifeStateDead, pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD},
		{"unknown", pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED},
	}
	for _, tc := range tests {
		if got := daggerheartLifeStateToProto(tc.input); got != tc.want {
			t.Fatalf("daggerheartLifeStateToProto(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestDaggerheartDeathMoveFromProto(t *testing.T) {
	if _, err := daggerheartDeathMoveFromProto(pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED); err == nil {
		t.Fatal("expected error for unspecified death move")
	}

	tests := []struct {
		proto pb.DaggerheartDeathMove
		want  string
	}{
		{pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_BLAZE_OF_GLORY, daggerheart.DeathMoveBlazeOfGlory},
		{pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH, daggerheart.DeathMoveAvoidDeath},
		{pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL, daggerheart.DeathMoveRiskItAll},
	}
	for _, tc := range tests {
		move, err := daggerheartDeathMoveFromProto(tc.proto)
		if err != nil {
			t.Fatalf("unexpected error for %v: %v", tc.proto, err)
		}
		if move != tc.want {
			t.Fatalf("expected %s, got %s", tc.want, move)
		}
	}

	if _, err := daggerheartDeathMoveFromProto(pb.DaggerheartDeathMove(99)); err == nil {
		t.Fatal("expected error for invalid death move")
	}
}

func TestDaggerheartDeathMoveToProto(t *testing.T) {
	tests := []struct {
		input string
		want  pb.DaggerheartDeathMove
	}{
		{daggerheart.DeathMoveBlazeOfGlory, pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_BLAZE_OF_GLORY},
		{daggerheart.DeathMoveAvoidDeath, pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH},
		{daggerheart.DeathMoveRiskItAll, pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL},
		{"unknown", pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED},
	}
	for _, tc := range tests {
		if got := daggerheartDeathMoveToProto(tc.input); got != tc.want {
			t.Fatalf("daggerheartDeathMoveToProto(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}
