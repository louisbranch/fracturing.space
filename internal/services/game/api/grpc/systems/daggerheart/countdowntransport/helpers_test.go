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

func TestCountdownToneFromProto(t *testing.T) {
	got, err := countdownToneFromProto(pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS)
	if err != nil {
		t.Fatalf("countdownToneFromProto returned error: %v", err)
	}
	if got != "progress" {
		t.Fatalf("countdown tone = %q, want progress", got)
	}
}

func TestCountdownPolicyFromProto(t *testing.T) {
	got, err := countdownPolicyFromProto(pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST)
	if err != nil {
		t.Fatalf("countdownPolicyFromProto returned error: %v", err)
	}
	if got != "long_rest" {
		t.Fatalf("countdown policy = %q, want long_rest", got)
	}
}

func TestCountdownLoopBehaviorFromProto(t *testing.T) {
	got, err := countdownLoopBehaviorFromProto(pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_DECREASE_START)
	if err != nil {
		t.Fatalf("countdownLoopBehaviorFromProto returned error: %v", err)
	}
	if got != "reset_decrease_start" {
		t.Fatalf("countdown loop behavior = %q, want reset_decrease_start", got)
	}
}

func TestCountdownStatusFromProtoRejectsUnknown(t *testing.T) {
	if _, err := countdownStatusFromProto(pb.DaggerheartCountdownStatus(99)); err == nil {
		t.Fatal("expected error for invalid countdown status")
	}
}

func TestCountdownToneFromProtoRejectsUnspecified(t *testing.T) {
	if _, err := countdownToneFromProto(pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_UNSPECIFIED); err == nil {
		t.Fatal("expected error for unspecified countdown tone")
	}
}

func TestCountdownPolicyFromProtoRejectsUnspecified(t *testing.T) {
	if _, err := countdownPolicyFromProto(pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_UNSPECIFIED); err == nil {
		t.Fatal("expected error for unspecified countdown advancement policy")
	}
}

func TestCountdownToneToProto(t *testing.T) {
	if got := countdownToneToProto("unknown"); got != pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_UNSPECIFIED {
		t.Fatalf("countdownToneToProto(unknown) = %v, want unspecified", got)
	}
	if got := countdownToneToProto("consequence"); got != pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE {
		t.Fatalf("countdownToneToProto(consequence) = %v, want consequence", got)
	}
}

func TestCountdownLoopBehaviorToProto(t *testing.T) {
	if got := countdownLoopBehaviorToProto("unknown"); got != pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_UNSPECIFIED {
		t.Fatalf("countdownLoopBehaviorToProto(unknown) = %v, want unspecified", got)
	}
	if got := countdownLoopBehaviorToProto("reset"); got != pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET {
		t.Fatalf("countdownLoopBehaviorToProto(reset) = %v, want reset", got)
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
		CampaignID:        "camp-1",
		CountdownID:       "cd-1",
		Name:              "Clock",
		Tone:              "progress",
		AdvancementPolicy: "manual",
		StartingValue:     4,
		RemainingValue:    2,
		LoopBehavior:      "reset",
		Status:            "active",
	})
	if countdown.ID != "cd-1" || countdown.RemainingValue != 2 || countdown.LoopBehavior != "reset" {
		t.Fatalf("countdown = %+v", countdown)
	}
}

func TestSceneCountdownToProto(t *testing.T) {
	proto := SceneCountdownToProto(projectionstore.DaggerheartCountdown{
		CountdownID:       "count-1",
		CampaignID:        "camp-1",
		SessionID:         "sess-1",
		SceneID:           "scene-1",
		Name:              "Threat",
		Tone:              "progress",
		AdvancementPolicy: "action_standard",
		StartingValue:     5,
		RemainingValue:    2,
		LoopBehavior:      "reset",
		Status:            "active",
	})
	if proto.GetCountdownId() != "count-1" || proto.GetName() != "Threat" {
		t.Fatal("expected countdown id and name to map")
	}
	if proto.GetTone() != pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS {
		t.Fatal("expected progress tone")
	}
	if proto.GetAdvancementPolicy() != pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_STANDARD {
		t.Fatal("expected action_standard policy")
	}
	if proto.GetRemainingValue() != 2 || proto.GetStartingValue() != 5 {
		t.Fatal("expected remaining/starting values to map")
	}
	if proto.GetLoopBehavior() != pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET {
		t.Fatal("expected loop behavior to map")
	}
}
