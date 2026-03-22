package countdowntransport

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerAdvanceSceneCountdownRejectsMissingMutation(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error { return nil },
	})

	_, err := handler.AdvanceSceneCountdown(testContext(), &pb.DaggerheartAdvanceSceneCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CountdownId: "cd-1",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerAdvanceSceneCountdownRejectsNilRequest(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.AdvanceSceneCountdown(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerAdvanceSceneCountdownSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		countdowns: map[string]projectionstore.DaggerheartCountdown{
			"camp-1:cd-1": {
				CampaignID:        "camp-1",
				SessionID:         "sess-1",
				SceneID:           "scene-1",
				CountdownID:       "cd-1",
				Name:              "Clock",
				Tone:              "progress",
				AdvancementPolicy: "manual",
				StartingValue:     4,
				RemainingValue:    3,
				LoopBehavior:      "none",
				Status:            "active",
			},
		},
	}
	var commandInput DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commandInput = in
			var payload daggerheartpayload.SceneCountdownAdvancedPayload
			if err := json.Unmarshal(in.PayloadJSON, &payload); err != nil {
				return err
			}
			store.countdowns["camp-1:cd-1"] = projectionstore.DaggerheartCountdown{
				CampaignID:        "camp-1",
				SessionID:         "sess-1",
				SceneID:           "scene-1",
				CountdownID:       "cd-1",
				Name:              "Clock",
				Tone:              "progress",
				AdvancementPolicy: "manual",
				StartingValue:     4,
				RemainingValue:    payload.AfterRemaining,
				LoopBehavior:      "none",
				Status:            payload.StatusAfter,
			}
			return nil
		},
	})

	resp, err := handler.AdvanceSceneCountdown(testContext(), &pb.DaggerheartAdvanceSceneCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CountdownId: "cd-1",
		Amount:      1,
	})
	if err != nil {
		t.Fatalf("AdvanceSceneCountdown returned error: %v", err)
	}
	if commandInput.CommandType != commandids.DaggerheartSceneCountdownAdvance {
		t.Fatalf("command type = %q, want %q", commandInput.CommandType, commandids.DaggerheartSceneCountdownAdvance)
	}
	if resp.Summary.BeforeRemaining != 3 || resp.Summary.AfterRemaining != 2 || resp.Summary.AdvancedBy != 1 {
		t.Fatalf("advance summary = (%d,%d,%d), want (3,2,1)", resp.Summary.BeforeRemaining, resp.Summary.AfterRemaining, resp.Summary.AdvancedBy)
	}
	if resp.Countdown.RemainingValue != 2 {
		t.Fatalf("countdown remaining value = %d, want 2", resp.Countdown.RemainingValue)
	}
}

func TestHandlerAdvanceSceneCountdownWrapsCountdownStoreErrors(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Daggerheart: testDaggerheartStore{getErr: errors.New("boom")},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.AdvanceSceneCountdown(testContext(), &pb.DaggerheartAdvanceSceneCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CountdownId: "cd-1",
		Amount:      1,
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerAdvanceSceneCountdownRejectsInactiveSession(t *testing.T) {
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

	_, err := handler.AdvanceSceneCountdown(testContext(), &pb.DaggerheartAdvanceSceneCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CountdownId: "cd-1",
		Amount:      1,
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestHandlerAdvanceSceneCountdownRejectsUnsupportedSystem(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Campaign: testCampaignStore{record: storage.CampaignRecord{
			ID:     "camp-1",
			System: systembridge.SystemIDUnspecified,
			Status: campaign.StatusActive,
		}},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.AdvanceSceneCountdown(testContext(), &pb.DaggerheartAdvanceSceneCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CountdownId: "cd-1",
		Amount:      1,
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}
