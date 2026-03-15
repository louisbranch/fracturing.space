package recoverytransport

import (
	"context"
	"errors"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRequireDaggerheartSystem(t *testing.T) {
	record := storage.CampaignRecord{System: systembridge.SystemIDDaggerheart}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "unsupported"); err != nil {
		t.Fatalf("RequireDaggerheartSystem returned error: %v", err)
	}

	err := daggerheartguard.RequireDaggerheartSystem(storage.CampaignRecord{System: systembridge.SystemID("other")}, "unsupported")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestEnsureNoOpenSessionGate(t *testing.T) {
	store := testGateStore{err: storage.ErrNotFound}
	if err := daggerheartguard.EnsureNoOpenSessionGate(context.Background(), store, "camp-1", "sess-1"); err != nil {
		t.Fatalf("EnsureNoOpenSessionGate returned error: %v", err)
	}

	err := daggerheartguard.EnsureNoOpenSessionGate(context.Background(), testGateStore{gate: storage.SessionGate{GateID: "gate-1"}}, "camp-1", "sess-1")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestResolveSeedUsesReplayMode(t *testing.T) {
	seed, err := resolveSeed(&commonv1.RngRequest{}, func() (int64, error) { return 0, nil }, func(_ *commonv1.RngRequest, _ func() (int64, error), allow func(commonv1.RollMode) bool) (int64, string, commonv1.RollMode, error) {
		if !allow(commonv1.RollMode_REPLAY) {
			t.Fatal("expected replay mode to be allowed")
		}
		return 7, "generated", commonv1.RollMode_LIVE, nil
	})
	if err != nil {
		t.Fatalf("resolveSeed returned error: %v", err)
	}
	if seed != 7 {
		t.Fatalf("seed = %d, want 7", seed)
	}
}

func TestRestTypeFromProto(t *testing.T) {
	if _, err := restTypeFromProto(pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_UNSPECIFIED); err == nil {
		t.Fatal("expected error for unspecified rest type")
	}
	if got, err := restTypeFromProto(pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG); err != nil || got != daggerheart.RestTypeLong {
		t.Fatalf("restTypeFromProto(long) = %v, %v", got, err)
	}
}

func TestDowntimeMoveHelpers(t *testing.T) {
	move, err := downtimeMoveFromProto(pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_PREPARE)
	if err != nil {
		t.Fatalf("downtimeMoveFromProto returned error: %v", err)
	}
	if move != daggerheart.DowntimePrepare {
		t.Fatalf("move = %v, want %v", move, daggerheart.DowntimePrepare)
	}
	if got := downtimeMoveToString(daggerheart.DowntimeWorkOnProject); got != "work_on_project" {
		t.Fatalf("downtimeMoveToString = %q, want work_on_project", got)
	}
}

func TestCountdownFromStorage(t *testing.T) {
	countdown := countdownFromStorage(projectionstore.DaggerheartCountdown{
		CampaignID:  "camp-1",
		CountdownID: "count-1",
		Name:        "Impending Doom",
		Kind:        "long_term",
		Current:     2,
		Max:         6,
		Direction:   "up",
		Looping:     true,
	})
	if countdown.ID != "count-1" {
		t.Fatalf("countdown id = %q, want count-1", countdown.ID)
	}
	if countdown.Current != 2 || countdown.Max != 6 {
		t.Fatalf("countdown progress = %d/%d, want 2/6", countdown.Current, countdown.Max)
	}
	if !countdown.Looping {
		t.Fatal("expected looping countdown")
	}
}

func TestDeathMoveFromProto(t *testing.T) {
	if _, err := deathMoveFromProto(pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED); err == nil {
		t.Fatal("expected error for unspecified death move")
	}
	move, err := deathMoveFromProto(pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL)
	if err != nil {
		t.Fatalf("deathMoveFromProto returned error: %v", err)
	}
	if move != daggerheart.DeathMoveRiskItAll {
		t.Fatalf("move = %q, want %q", move, daggerheart.DeathMoveRiskItAll)
	}
}

func TestDeathMoveToProto(t *testing.T) {
	if got := DeathMoveToProto(daggerheart.DeathMoveAvoidDeath); got != pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH {
		t.Fatalf("DeathMoveToProto(avoid_death) = %v, want avoid_death", got)
	}
	if got := DeathMoveToProto("unknown"); got != pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED {
		t.Fatalf("DeathMoveToProto(unknown) = %v, want unspecified", got)
	}
}

func TestHandleDomainError(t *testing.T) {
	err := grpcerror.HandleDomainError(errors.New("boom"))
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}
