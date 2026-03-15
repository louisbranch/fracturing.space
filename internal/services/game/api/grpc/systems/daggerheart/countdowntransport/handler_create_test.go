package countdowntransport

import (
	"context"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerCreateCountdownRejectsNilRequest(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.CreateCountdown(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerCreateCountdownRequiresExecutor(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.CreateCountdown(testContext(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Clock",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        4,
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerCreateCountdownRejectsInvalidMax(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error { return nil },
	})

	_, err := handler.CreateCountdown(testContext(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Clock",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        0,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerCreateCountdownRejectsCurrentOutOfRange(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error { return nil },
	})

	_, err := handler.CreateCountdown(testContext(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Clock",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        4,
		Current:    5,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerCreateCountdownRejectsDuplicate(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Daggerheart: testDaggerheartStore{
			countdowns: map[string]projectionstore.DaggerheartCountdown{
				"camp-1:cd-1": {CampaignID: "camp-1", CountdownID: "cd-1"},
			},
		},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.CreateCountdown(testContext(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		CountdownId: "cd-1",
		Name:        "Clock",
		Kind:        pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:   pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:         4,
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestHandlerCreateCountdownPropagatesIDGenerationFailure(t *testing.T) {
	handler := newTestHandler(Dependencies{
		NewID: func() (string, error) { return "", errors.New("boom") },
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.CreateCountdown(testContext(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Clock",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        4,
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerCreateCountdownSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		countdowns: map[string]projectionstore.DaggerheartCountdown{},
	}
	var commandInput DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commandInput = in
			store.countdowns["camp-1:generated-id"] = projectionstore.DaggerheartCountdown{
				CampaignID:  "camp-1",
				CountdownID: "generated-id",
				Name:        "Clock",
				Kind:        daggerheart.CountdownKindProgress,
				Current:     0,
				Max:         4,
				Direction:   daggerheart.CountdownDirectionIncrease,
			}
			return nil
		},
	})

	resp, err := handler.CreateCountdown(testContext(), &pb.DaggerheartCreateCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		Name:       "Clock",
		Kind:       pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Direction:  pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Max:        4,
	})
	if err != nil {
		t.Fatalf("CreateCountdown returned error: %v", err)
	}
	if resp.Countdown.CountdownID != "generated-id" {
		t.Fatalf("countdown_id = %q, want generated-id", resp.Countdown.CountdownID)
	}
	if commandInput.CommandType != commandids.DaggerheartCountdownCreate {
		t.Fatalf("command type = %q, want %q", commandInput.CommandType, commandids.DaggerheartCountdownCreate)
	}
	if commandInput.RequestID != "req-1" || commandInput.InvocationID != "inv-1" {
		t.Fatalf("request metadata = (%q,%q), want (req-1,inv-1)", commandInput.RequestID, commandInput.InvocationID)
	}
}
