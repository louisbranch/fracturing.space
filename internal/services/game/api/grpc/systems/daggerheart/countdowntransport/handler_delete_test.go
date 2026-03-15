package countdowntransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerDeleteCountdownRequiresExecutor(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.DeleteCountdown(testContext(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerDeleteCountdownRejectsNilRequest(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.DeleteCountdown(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerDeleteCountdownSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		countdowns: map[string]projectionstore.DaggerheartCountdown{
			"camp-1:cd-1": {
				CampaignID:  "camp-1",
				CountdownID: "cd-1",
			},
		},
	}
	var commandInput DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commandInput = in
			delete(store.countdowns, "camp-1:cd-1")
			return nil
		},
	})

	resp, err := handler.DeleteCountdown(testContext(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
	})
	if err != nil {
		t.Fatalf("DeleteCountdown returned error: %v", err)
	}
	if commandInput.CommandType != commandids.DaggerheartCountdownDelete {
		t.Fatalf("command type = %q, want %q", commandInput.CommandType, commandids.DaggerheartCountdownDelete)
	}
	if resp.CountdownID != "cd-1" {
		t.Fatalf("countdown_id = %q, want cd-1", resp.CountdownID)
	}
}

func TestHandlerDeleteCountdownRejectsMissingCountdown(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Daggerheart: testDaggerheartStore{},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.DeleteCountdown(testContext(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func TestHandlerDeleteCountdownRejectsOpenSessionGate(t *testing.T) {
	handler := newTestHandler(Dependencies{
		SessionGate: testGateStore{gate: storage.SessionGate{GateID: "gate-1"}},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.DeleteCountdown(testContext(), &pb.DaggerheartDeleteCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}
