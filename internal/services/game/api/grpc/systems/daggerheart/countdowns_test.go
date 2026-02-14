package daggerheart

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestDaggerheartCountdownKindFromProto(t *testing.T) {
	if _, err := daggerheartCountdownKindFromProto(pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_UNSPECIFIED); err == nil {
		t.Fatal("expected error for unspecified kind")
	}

	kind, err := daggerheartCountdownKindFromProto(pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kind != daggerheart.CountdownKindProgress {
		t.Fatalf("expected progress, got %s", kind)
	}

	if _, err := daggerheartCountdownKindFromProto(pb.DaggerheartCountdownKind(99)); err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

func TestDaggerheartCountdownKindToProto(t *testing.T) {
	if daggerheartCountdownKindToProto(daggerheart.CountdownKindConsequence) != pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_CONSEQUENCE {
		t.Fatal("expected consequence kind")
	}
	if daggerheartCountdownKindToProto("unknown") != pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_UNSPECIFIED {
		t.Fatal("expected unspecified for unknown kind")
	}
}

func TestDaggerheartCountdownDirectionFromProto(t *testing.T) {
	if _, err := daggerheartCountdownDirectionFromProto(pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_UNSPECIFIED); err == nil {
		t.Fatal("expected error for unspecified direction")
	}

	direction, err := daggerheartCountdownDirectionFromProto(pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_DECREASE)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if direction != daggerheart.CountdownDirectionDecrease {
		t.Fatalf("expected decrease, got %s", direction)
	}

	if _, err := daggerheartCountdownDirectionFromProto(pb.DaggerheartCountdownDirection(99)); err == nil {
		t.Fatal("expected error for invalid direction")
	}
}

func TestDaggerheartCountdownDirectionToProto(t *testing.T) {
	if daggerheartCountdownDirectionToProto(daggerheart.CountdownDirectionIncrease) != pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE {
		t.Fatal("expected increase direction")
	}
	if daggerheartCountdownDirectionToProto("unknown") != pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_UNSPECIFIED {
		t.Fatal("expected unspecified for unknown direction")
	}
}

func TestDaggerheartCountdownToProto(t *testing.T) {
	proto := daggerheartCountdownToProto(storage.DaggerheartCountdown{
		CountdownID: "count-1",
		Name:        "Threat",
		Kind:        daggerheart.CountdownKindProgress,
		Current:     2,
		Max:         5,
		Direction:   daggerheart.CountdownDirectionIncrease,
		Looping:     true,
	})

	if proto.GetCountdownId() != "count-1" || proto.GetName() != "Threat" {
		t.Fatal("expected countdown id and name to map")
	}
	if proto.GetKind() != pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS {
		t.Fatal("expected progress kind")
	}
	if proto.GetDirection() != pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE {
		t.Fatal("expected increase direction")
	}
	if proto.GetCurrent() != 2 || proto.GetMax() != 5 {
		t.Fatal("expected current/max to map")
	}
	if !proto.GetLooping() {
		t.Fatal("expected looping to map")
	}
}
