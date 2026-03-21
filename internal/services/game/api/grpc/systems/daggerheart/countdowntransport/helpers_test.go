package countdowntransport

import (
	"context"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testGateStore struct {
	gate storage.SessionGate
	err  error
}

func (s testGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	if s.err != nil {
		return storage.SessionGate{}, s.err
	}
	return s.gate, nil
}

func TestCountdownKindFromProto(t *testing.T) {
	got, err := countdownKindFromProto(pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS)
	if err != nil {
		t.Fatalf("countdownKindFromProto returned error: %v", err)
	}
	if got == "" {
		t.Fatal("expected countdown kind")
	}
}

func TestCountdownKindFromProtoConsequence(t *testing.T) {
	got, err := countdownKindFromProto(pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_CONSEQUENCE)
	if err != nil {
		t.Fatalf("countdownKindFromProto returned error: %v", err)
	}
	if got == "" {
		t.Fatal("expected countdown kind")
	}
}

func TestCountdownDirectionFromProto(t *testing.T) {
	got, err := countdownDirectionFromProto(pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_DECREASE)
	if err != nil {
		t.Fatalf("countdownDirectionFromProto returned error: %v", err)
	}
	if got == "" {
		t.Fatal("expected countdown direction")
	}
}

func TestCountdownKindFromProtoRejectsUnknown(t *testing.T) {
	if _, err := countdownKindFromProto(pb.DaggerheartCountdownKind(99)); err == nil {
		t.Fatal("expected error for invalid countdown kind")
	}
}

func TestCountdownDirectionFromProtoRejectsUnknown(t *testing.T) {
	if _, err := countdownDirectionFromProto(pb.DaggerheartCountdownDirection(99)); err == nil {
		t.Fatal("expected error for invalid countdown direction")
	}
}

func TestCountdownDirectionFromProtoRejectsUnspecified(t *testing.T) {
	if _, err := countdownDirectionFromProto(pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_UNSPECIFIED); err == nil {
		t.Fatal("expected error for unspecified countdown direction")
	}
}

func TestCountdownKindToProto(t *testing.T) {
	if got := countdownKindToProto("unknown"); got != pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_UNSPECIFIED {
		t.Fatalf("countdownKindToProto(unknown) = %v, want unspecified", got)
	}
	if got := countdownKindToProto("consequence"); got != pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_CONSEQUENCE {
		t.Fatalf("countdownKindToProto(consequence) = %v, want consequence", got)
	}
}

func TestCountdownDirectionToProto(t *testing.T) {
	if got := countdownDirectionToProto("unknown"); got != pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_UNSPECIFIED {
		t.Fatalf("countdownDirectionToProto(unknown) = %v, want unspecified", got)
	}
	if got := countdownDirectionToProto("increase"); got != pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE {
		t.Fatalf("countdownDirectionToProto(increase) = %v, want increase", got)
	}
}

func TestRequireDaggerheartSystemRejectsOtherSystems(t *testing.T) {
	record := storage.CampaignRecord{System: "unspecified"}
	err := daggerheartguard.RequireDaggerheartSystem(record, "unsupported")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestEnsureNoOpenSessionGateRejectsOpenGate(t *testing.T) {
	err := daggerheartguard.EnsureNoOpenSessionGate(context.Background(), testGateStore{gate: storage.SessionGate{GateID: "gate-1"}}, "camp-1", "sess-1")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestEnsureNoOpenSessionGateWrapsStoreErrors(t *testing.T) {
	err := daggerheartguard.EnsureNoOpenSessionGate(context.Background(), testGateStore{err: errors.New("boom")}, "camp-1", "sess-1")
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestCountdownFromStorage(t *testing.T) {
	countdown := countdownFromStorage(projectionstore.DaggerheartCountdown{
		CampaignID:  "camp-1",
		CountdownID: "cd-1",
		Name:        "Clock",
		Kind:        "progress",
		Current:     2,
		Max:         4,
		Direction:   "increase",
		Looping:     true,
	})
	if countdown.ID != "cd-1" || countdown.Current != 2 || !countdown.Looping {
		t.Fatalf("countdown = %+v", countdown)
	}
}

func TestCountdownToProto(t *testing.T) {
	proto := CountdownToProto(projectionstore.DaggerheartCountdown{
		CountdownID: "count-1",
		Name:        "Threat",
		Kind:        "progress",
		Current:     2,
		Max:         5,
		Direction:   "increase",
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
