package gmmovetransport

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerApplyGmMoveRejectsNilRequest(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.ApplyGmMove(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerApplyGmMoveRequiresExecutor(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.ApplyGmMove(testContext(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Move:       "change_environment",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerApplyGmMoveSuccessWithoutFearSpend(t *testing.T) {
	var commandCalls int
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			commandCalls++
			return nil
		},
	})

	resp, err := handler.ApplyGmMove(testContext(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Move:       "change_environment",
	})
	if err != nil {
		t.Fatalf("ApplyGmMove returned error: %v", err)
	}
	if resp.CampaignID != "camp-1" {
		t.Fatalf("campaign_id = %q, want camp-1", resp.CampaignID)
	}
	if resp.GMFearBefore != 0 || resp.GMFearAfter != 0 {
		t.Fatalf("gm fear = (%d,%d), want (0,0)", resp.GMFearBefore, resp.GMFearAfter)
	}
	if commandCalls != 0 {
		t.Fatalf("command calls = %d, want 0", commandCalls)
	}
}

func TestHandlerApplyGmMoveWithFearSpent(t *testing.T) {
	var commandInput DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: testDaggerheartStore{
			snapshot: projectionstore.DaggerheartSnapshot{
				CampaignID: "camp-1",
				GMFear:     3,
			},
		},
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commandInput = in
			return nil
		},
	})

	resp, err := handler.ApplyGmMove(testContext(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		SceneId:    "scene-1",
		Move:       "change_environment",
		FearSpent:  1,
	})
	if err != nil {
		t.Fatalf("ApplyGmMove returned error: %v", err)
	}
	if resp.GMFearBefore != 3 || resp.GMFearAfter != 2 {
		t.Fatalf("gm fear = (%d,%d), want (3,2)", resp.GMFearBefore, resp.GMFearAfter)
	}
	if commandInput.CommandType != commandids.DaggerheartGMFearSet {
		t.Fatalf("command type = %q, want %q", commandInput.CommandType, commandids.DaggerheartGMFearSet)
	}
	if commandInput.SceneID != "scene-1" {
		t.Fatalf("scene_id = %q, want scene-1", commandInput.SceneID)
	}
	if commandInput.RequestID != "req-1" || commandInput.InvocationID != "inv-1" {
		t.Fatalf("request metadata = (%q,%q), want (req-1,inv-1)", commandInput.RequestID, commandInput.InvocationID)
	}
	var payload daggerheart.GMFearSetPayload
	if err := json.Unmarshal(commandInput.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.After == nil || *payload.After != 2 {
		t.Fatalf("payload after = %v, want 2", payload.After)
	}
	if payload.Reason != "gm_move" {
		t.Fatalf("payload reason = %q, want gm_move", payload.Reason)
	}
}

func TestHandlerApplyGmMoveRejectsNegativeFearSpent(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error { return nil },
	})

	_, err := handler.ApplyGmMove(testContext(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Move:       "change_environment",
		FearSpent:  -1,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerApplyGmMoveRejectsInactiveSession(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Session: testSessionStore{record: storage.SessionRecord{
			ID:         "sess-1",
			CampaignID: "camp-1",
			Status:     session.StatusEnded,
		}},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.ApplyGmMove(testContext(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Move:       "change_environment",
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestHandlerApplyGmMoveCampaignNotFound(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Campaign: testCampaignStore{err: storage.ErrNotFound},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.ApplyGmMove(testContext(), &pb.DaggerheartApplyGmMoveRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Move:       "change_environment",
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.NotFound)
	}
}
