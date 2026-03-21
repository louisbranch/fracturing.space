package workfloweffects

import (
	"context"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type countdownStoreStub struct {
	err error
}

func (s countdownStoreStub) GetDaggerheartCountdown(context.Context, string, string) (projectionstore.DaggerheartCountdown, error) {
	return projectionstore.DaggerheartCountdown{}, s.err
}

func TestAdvanceBreathCountdown_NoCountdownID(t *testing.T) {
	handler := NewHandler(Dependencies{})
	if err := handler.AdvanceBreathCountdown(context.Background(), "camp-1", "sess-1", "", true); err != nil {
		t.Fatalf("AdvanceBreathCountdown returned error: %v", err)
	}
}

func TestAdvanceBreathCountdown_CreatesAndUpdates(t *testing.T) {
	created := false
	updated := false
	handler := NewHandler(Dependencies{
		Daggerheart: countdownStoreStub{err: storage.ErrNotFound},
		CreateCountdown: func(_ context.Context, in *pb.DaggerheartCreateCountdownRequest) error {
			created = true
			if in.GetCountdownId() != "countdown-1" {
				t.Fatalf("countdown id = %q, want countdown-1", in.GetCountdownId())
			}
			return nil
		},
		UpdateCountdown: func(_ context.Context, in *pb.DaggerheartUpdateCountdownRequest) error {
			updated = true
			if in.GetCountdownId() != "countdown-1" {
				t.Fatalf("countdown id = %q, want countdown-1", in.GetCountdownId())
			}
			return nil
		},
	})

	if err := handler.AdvanceBreathCountdown(context.Background(), "camp-1", "sess-1", "countdown-1", true); err != nil {
		t.Fatalf("AdvanceBreathCountdown returned error: %v", err)
	}
	if !created || !updated {
		t.Fatalf("created=%t updated=%t, want both true", created, updated)
	}
}

func TestAdvanceBreathCountdown_IgnoresFailedPreconditionCreate(t *testing.T) {
	updated := false
	handler := NewHandler(Dependencies{
		Daggerheart: countdownStoreStub{err: storage.ErrNotFound},
		CreateCountdown: func(context.Context, *pb.DaggerheartCreateCountdownRequest) error {
			return status.Error(codes.FailedPrecondition, "already exists")
		},
		UpdateCountdown: func(context.Context, *pb.DaggerheartUpdateCountdownRequest) error {
			updated = true
			return nil
		},
	})

	if err := handler.AdvanceBreathCountdown(context.Background(), "camp-1", "sess-1", "countdown-1", false); err != nil {
		t.Fatalf("AdvanceBreathCountdown returned error: %v", err)
	}
	if !updated {
		t.Fatal("expected update after failed-precondition create")
	}
}

func TestAdvanceBreathCountdown_MapsStoreErrors(t *testing.T) {
	handler := NewHandler(Dependencies{
		Daggerheart:     countdownStoreStub{err: errors.New("boom")},
		CreateCountdown: func(context.Context, *pb.DaggerheartCreateCountdownRequest) error { return nil },
		UpdateCountdown: func(context.Context, *pb.DaggerheartUpdateCountdownRequest) error { return nil },
	})

	err := handler.AdvanceBreathCountdown(context.Background(), "camp-1", "sess-1", "countdown-1", false)
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}
