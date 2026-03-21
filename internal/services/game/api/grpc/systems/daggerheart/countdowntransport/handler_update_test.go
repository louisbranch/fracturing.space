package countdowntransport

import (
	"context"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerUpdateCountdownRejectsMissingMutation(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error { return nil },
	})

	_, err := handler.UpdateCountdown(testContext(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerUpdateCountdownRejectsNilRequest(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.UpdateCountdown(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerUpdateCountdownSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		countdowns: map[string]projectionstore.DaggerheartCountdown{
			"camp-1:cd-1": {
				CampaignID:  "camp-1",
				CountdownID: "cd-1",
				Name:        "Clock",
				Kind:        daggerheart.CountdownKindProgress,
				Current:     1,
				Max:         4,
				Direction:   daggerheart.CountdownDirectionIncrease,
			},
		},
	}
	var commandInput DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commandInput = in
			store.countdowns["camp-1:cd-1"] = projectionstore.DaggerheartCountdown{
				CampaignID:  "camp-1",
				CountdownID: "cd-1",
				Name:        "Clock",
				Kind:        daggerheart.CountdownKindProgress,
				Current:     2,
				Max:         4,
				Direction:   daggerheart.CountdownDirectionIncrease,
			}
			return nil
		},
	})

	resp, err := handler.UpdateCountdown(testContext(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
		Delta:       1,
	})
	if err != nil {
		t.Fatalf("UpdateCountdown returned error: %v", err)
	}
	if commandInput.CommandType != commandids.DaggerheartCountdownUpdate {
		t.Fatalf("command type = %q, want %q", commandInput.CommandType, commandids.DaggerheartCountdownUpdate)
	}
	if resp.After != 2 || resp.Delta != 1 || resp.Before != 1 {
		t.Fatalf("update summary = (%d,%d,%d), want (1,2,1)", resp.Before, resp.After, resp.Delta)
	}
	if resp.Countdown.Current != 2 {
		t.Fatalf("countdown current = %d, want 2", resp.Countdown.Current)
	}
}

func TestHandlerUpdateCountdownWrapsCountdownStoreErrors(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Daggerheart: testDaggerheartStore{getErr: errors.New("boom")},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.UpdateCountdown(testContext(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
		Delta:       1,
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerUpdateCountdownRejectsInactiveSession(t *testing.T) {
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

	_, err := handler.UpdateCountdown(testContext(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
		Delta:       1,
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestHandlerUpdateCountdownRejectsUnsupportedSystem(t *testing.T) {
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

	_, err := handler.UpdateCountdown(testContext(), &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
		Delta:       1,
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}
