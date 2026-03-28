package guard

import (
	"context"
	"errors"
	"testing"

	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeSessionGateStore is a minimal fake for SessionGateStore.
type fakeSessionGateStore struct {
	gate storage.SessionGate
	err  error
}

func (f *fakeSessionGateStore) GetOpenSessionGate(_ context.Context, _, _ string) (storage.SessionGate, error) {
	return f.gate, f.err
}

func TestCampaignSupportsDaggerheart_CorrectSystem(t *testing.T) {
	record := storage.CampaignRecord{System: bridge.SystemIDDaggerheart}
	if !CampaignSupportsDaggerheart(record) {
		t.Fatal("expected true for daggerheart system")
	}
}

func TestCampaignSupportsDaggerheart_WrongSystem(t *testing.T) {
	record := storage.CampaignRecord{System: bridge.SystemID("other_system")}
	if CampaignSupportsDaggerheart(record) {
		t.Fatal("expected false for non-daggerheart system")
	}
}

func TestCampaignSupportsDaggerheart_EmptySystem(t *testing.T) {
	record := storage.CampaignRecord{System: bridge.SystemID("")}
	if CampaignSupportsDaggerheart(record) {
		t.Fatal("expected false for empty system")
	}
}

func TestRequireDaggerheartSystem_CorrectSystem(t *testing.T) {
	record := storage.CampaignRecord{System: bridge.SystemIDDaggerheart}
	if err := RequireDaggerheartSystem(record, "unsupported"); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestRequireDaggerheartSystem_WrongSystem(t *testing.T) {
	record := storage.CampaignRecord{System: bridge.SystemID("other")}
	err := RequireDaggerheartSystem(record, "campaign does not support daggerheart")
	if err == nil {
		t.Fatal("expected error for wrong system")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", st.Code())
	}
	if st.Message() != "campaign does not support daggerheart" {
		t.Fatalf("unexpected message: %q", st.Message())
	}
}

func TestEnsureNoOpenSessionGate_NoOpenGate(t *testing.T) {
	store := &fakeSessionGateStore{err: storage.ErrNotFound}
	err := EnsureNoOpenSessionGate(context.Background(), store, "camp-1", "sess-1")
	if err != nil {
		t.Fatalf("expected nil error for no open gate, got %v", err)
	}
}

func TestEnsureNoOpenSessionGate_OpenGate(t *testing.T) {
	store := &fakeSessionGateStore{
		gate: storage.SessionGate{GateID: "gate-42"},
		err:  nil,
	}
	err := EnsureNoOpenSessionGate(context.Background(), store, "camp-1", "sess-1")
	if err == nil {
		t.Fatal("expected error for open gate")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status error")
	}
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", st.Code())
	}
}

func TestEnsureNoOpenSessionGate_StoreErrorPropagates(t *testing.T) {
	storeErr := status.Error(codes.Internal, "database failure")
	store := &fakeSessionGateStore{err: storeErr}
	err := EnsureNoOpenSessionGate(context.Background(), store, "camp-1", "sess-1")
	if err == nil {
		t.Fatal("expected error to propagate")
	}
	// Invariant: non-not-found store errors must not be silently swallowed.
	if !errors.Is(err, storeErr) {
		t.Fatalf("expected store error to propagate, got %v", err)
	}
}

func TestEnsureNoOpenSessionGate_NilStore(t *testing.T) {
	err := EnsureNoOpenSessionGate(context.Background(), nil, "camp-1", "sess-1")
	if err != nil {
		t.Fatalf("expected nil error for nil store, got %v", err)
	}
}

func TestEnsureNoOpenSessionGate_EmptyCampaignID(t *testing.T) {
	store := &fakeSessionGateStore{err: storage.ErrNotFound}
	err := EnsureNoOpenSessionGate(context.Background(), store, "", "sess-1")
	if err != nil {
		t.Fatalf("expected nil error for empty campaign id, got %v", err)
	}
}

func TestEnsureNoOpenSessionGate_EmptySessionID(t *testing.T) {
	store := &fakeSessionGateStore{err: storage.ErrNotFound}
	err := EnsureNoOpenSessionGate(context.Background(), store, "camp-1", "")
	if err != nil {
		t.Fatalf("expected nil error for empty session id, got %v", err)
	}
}
