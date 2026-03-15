package conditiontransport

import (
	"context"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestConditionsFromProto(t *testing.T) {
	got, err := ConditionsFromProto([]pb.DaggerheartCondition{
		pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN,
		pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE,
	})
	if err != nil {
		t.Fatalf("ConditionsFromProto returned error: %v", err)
	}
	if len(got) != 2 || got[0] != daggerheart.ConditionHidden || got[1] != daggerheart.ConditionVulnerable {
		t.Fatalf("ConditionsFromProto = %v, want hidden/vulnerable", got)
	}
	if _, err := ConditionsFromProto([]pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_UNSPECIFIED}); err == nil {
		t.Fatal("expected unspecified condition error")
	}
}

func TestConditionsToProto(t *testing.T) {
	if ConditionsToProto(nil) != nil {
		t.Fatal("expected nil proto slice")
	}

	proto := ConditionsToProto([]string{
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

func TestLifeStateFromProto(t *testing.T) {
	got, err := lifeStateFromProto(pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY)
	if err != nil {
		t.Fatalf("lifeStateFromProto returned error: %v", err)
	}
	if got != daggerheart.LifeStateBlazeOfGlory {
		t.Fatalf("lifeStateFromProto = %q, want %q", got, daggerheart.LifeStateBlazeOfGlory)
	}
	if _, err := lifeStateFromProto(pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED); err == nil {
		t.Fatal("expected unspecified life_state error")
	}
}

func TestLifeStateToProto(t *testing.T) {
	if got := LifeStateToProto(daggerheart.LifeStateDead); got != pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD {
		t.Fatalf("LifeStateToProto(dead) = %v, want dead", got)
	}
	if got := LifeStateToProto("unknown"); got != pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED {
		t.Fatalf("LifeStateToProto(unknown) = %v, want unspecified", got)
	}
}

func TestRequireDaggerheartSystem(t *testing.T) {
	if err := requireDaggerheartSystem(storage.CampaignRecord{System: systembridge.SystemIDDaggerheart}, "unsupported"); err != nil {
		t.Fatalf("requireDaggerheartSystem returned error for daggerheart: %v", err)
	}
	err := requireDaggerheartSystem(storage.CampaignRecord{System: systembridge.SystemID("other")}, "unsupported")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

type gateStoreStub struct {
	gate storage.SessionGate
	err  error
}

func (s gateStoreStub) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return s.gate, s.err
}

func TestEnsureNoOpenSessionGate(t *testing.T) {
	if err := ensureNoOpenSessionGate(context.Background(), gateStoreStub{err: storage.ErrNotFound}, "camp-1", "sess-1"); err != nil {
		t.Fatalf("ensureNoOpenSessionGate returned error for missing gate: %v", err)
	}
	err := ensureNoOpenSessionGate(context.Background(), gateStoreStub{gate: storage.SessionGate{GateID: "gate-1"}}, "camp-1", "sess-1")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
	err = ensureNoOpenSessionGate(context.Background(), gateStoreStub{err: errors.New("boom")}, "camp-1", "sess-1")
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}
