package countdowntransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerDeleteSceneCountdownRequiresExecutor(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.DeleteSceneCountdown(testContext(), &pb.DaggerheartDeleteSceneCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CountdownId: "cd-1",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerDeleteSceneCountdownRejectsNilRequest(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.DeleteSceneCountdown(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerDeleteSceneCountdownSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		countdowns: map[string]projectionstore.DaggerheartCountdown{
			"camp-1:cd-1": {
				CampaignID:  "camp-1",
				SessionID:   "sess-1",
				SceneID:     "scene-1",
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

	resp, err := handler.DeleteSceneCountdown(testContext(), &pb.DaggerheartDeleteSceneCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CountdownId: "cd-1",
	})
	if err != nil {
		t.Fatalf("DeleteSceneCountdown returned error: %v", err)
	}
	if commandInput.CommandType != commandids.DaggerheartSceneCountdownDelete {
		t.Fatalf("command type = %q, want %q", commandInput.CommandType, commandids.DaggerheartSceneCountdownDelete)
	}
	if resp.CountdownID != "cd-1" {
		t.Fatalf("countdown_id = %q, want cd-1", resp.CountdownID)
	}
}

func TestHandlerDeleteSceneCountdownRejectsMissingCountdown(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Daggerheart: testDaggerheartStore{},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.DeleteSceneCountdown(testContext(), &pb.DaggerheartDeleteSceneCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CountdownId: "cd-1",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.NotFound)
	}
}

func TestHandlerDeleteSceneCountdownRejectsOpenSessionGate(t *testing.T) {
	handler := newTestHandler(Dependencies{
		SessionGate: testGateStore{gate: storage.SessionGate{GateID: "gate-1"}},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.DeleteSceneCountdown(testContext(), &pb.DaggerheartDeleteSceneCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CountdownId: "cd-1",
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}
