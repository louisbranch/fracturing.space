package conditiontransport

import (
	"context"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestConditionStatesFromProto(t *testing.T) {
	got, err := ConditionStatesFromProto([]*pb.DaggerheartConditionState{
		{
			Id:       daggerheart.ConditionHidden,
			Class:    pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD,
			Standard: pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN,
			Code:     daggerheart.ConditionHidden,
		},
		{
			Id:       daggerheart.ConditionVulnerable,
			Class:    pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD,
			Standard: pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE,
			Code:     daggerheart.ConditionVulnerable,
		},
	})
	if err != nil {
		t.Fatalf("ConditionStatesFromProto returned error: %v", err)
	}
	if codes := ConditionStateViewsToCodes(got); len(codes) != 2 || codes[0] != daggerheart.ConditionHidden || codes[1] != daggerheart.ConditionVulnerable {
		t.Fatalf("ConditionStateViewsToCodes = %v, want hidden/vulnerable", codes)
	}
	if _, err := ConditionStatesFromProto([]*pb.DaggerheartConditionState{{}}); err == nil {
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
	if len(proto) != 3 {
		t.Fatalf("expected 3 proto conditions, got %d", len(proto))
	}
	if proto[0].GetStandard() != pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN {
		t.Fatalf("unexpected first condition: %v", proto[0])
	}
	if proto[1].GetStandard() != pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE {
		t.Fatalf("unexpected second condition: %v", proto[1])
	}
	if proto[2].GetCode() != "unknown" {
		t.Fatalf("unexpected third condition: %v", proto[2])
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
	if err := daggerheartguard.RequireDaggerheartSystem(storage.CampaignRecord{System: systembridge.SystemIDDaggerheart}, "unsupported"); err != nil {
		t.Fatalf("RequireDaggerheartSystem returned error for daggerheart: %v", err)
	}
	err := daggerheartguard.RequireDaggerheartSystem(storage.CampaignRecord{System: systembridge.SystemID("other")}, "unsupported")
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
	if err := daggerheartguard.EnsureNoOpenSessionGate(context.Background(), gateStoreStub{err: storage.ErrNotFound}, "camp-1", "sess-1"); err != nil {
		t.Fatalf("EnsureNoOpenSessionGate returned error for missing gate: %v", err)
	}
	err := daggerheartguard.EnsureNoOpenSessionGate(context.Background(), gateStoreStub{gate: storage.SessionGate{GateID: "gate-1"}}, "camp-1", "sess-1")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
	err = daggerheartguard.EnsureNoOpenSessionGate(context.Background(), gateStoreStub{err: errors.New("boom")}, "camp-1", "sess-1")
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}
