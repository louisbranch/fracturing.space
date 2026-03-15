package gmmovetransport

import (
	"context"
	"errors"
	"testing"

	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
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
	if err := requireDaggerheartSystem(record, "unsupported"); err != nil {
		t.Fatalf("requireDaggerheartSystem returned error: %v", err)
	}
}

func TestRequireDaggerheartSystemRejectsOtherSystems(t *testing.T) {
	record := storage.CampaignRecord{System: systembridge.SystemIDUnspecified}
	err := requireDaggerheartSystem(record, "unsupported")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestEnsureNoOpenSessionGateAllowsMissingGate(t *testing.T) {
	err := ensureNoOpenSessionGate(context.Background(), testGateStore{err: storage.ErrNotFound}, "camp-1", "sess-1")
	if err != nil {
		t.Fatalf("ensureNoOpenSessionGate returned error: %v", err)
	}
}

func TestEnsureNoOpenSessionGateRejectsOpenGate(t *testing.T) {
	err := ensureNoOpenSessionGate(context.Background(), testGateStore{gate: storage.SessionGate{GateID: "gate-1"}}, "camp-1", "sess-1")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestEnsureNoOpenSessionGateWrapsStoreErrors(t *testing.T) {
	err := ensureNoOpenSessionGate(context.Background(), testGateStore{err: errors.New("boom")}, "camp-1", "sess-1")
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}
