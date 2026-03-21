package gmmovetransport

import (
	"context"
	"errors"
	"testing"

	daggerheartguard "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/guard"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
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

func TestRequireDaggerheartSystem(t *testing.T) {
	record := storage.CampaignRecord{System: systembridge.SystemIDDaggerheart}
	if err := daggerheartguard.RequireDaggerheartSystem(record, "unsupported"); err != nil {
		t.Fatalf("RequireDaggerheartSystem returned error: %v", err)
	}
}

func TestRequireDaggerheartSystemRejectsOtherSystems(t *testing.T) {
	record := storage.CampaignRecord{System: systembridge.SystemIDUnspecified}
	err := daggerheartguard.RequireDaggerheartSystem(record, "unsupported")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestEnsureNoOpenSessionGateAllowsMissingGate(t *testing.T) {
	err := daggerheartguard.EnsureNoOpenSessionGate(context.Background(), testGateStore{err: storage.ErrNotFound}, "camp-1", "sess-1")
	if err != nil {
		t.Fatalf("EnsureNoOpenSessionGate returned error: %v", err)
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
