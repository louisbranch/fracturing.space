package countdowntransport

import (
	"context"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerCreateSceneCountdownRejectsNilRequest(t *testing.T) {
	handler := newTestHandler(Dependencies{})
	_, err := handler.CreateSceneCountdown(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerCreateSceneCountdownRequiresExecutor(t *testing.T) {
	handler := newTestHandler(Dependencies{})
	_, err := handler.CreateSceneCountdown(testContext(), &pb.DaggerheartCreateSceneCountdownRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		SceneId:           "scene-1",
		Name:              "Clock",
		Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS,
		AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
		StartingValue:     &pb.DaggerheartCreateSceneCountdownRequest_FixedStartingValue{FixedStartingValue: 4},
		LoopBehavior:      pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerCreateSceneCountdownRejectsInvalidStart(t *testing.T) {
	handler := newTestHandler(Dependencies{ExecuteDomainCommand: func(context.Context, DomainCommandInput) error { return nil }})
	_, err := handler.CreateSceneCountdown(testContext(), &pb.DaggerheartCreateSceneCountdownRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		SceneId:           "scene-1",
		Name:              "Clock",
		Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS,
		AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
		StartingValue:     &pb.DaggerheartCreateSceneCountdownRequest_FixedStartingValue{FixedStartingValue: 0},
		LoopBehavior:      pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerCreateSceneCountdownRejectsDuplicate(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Daggerheart:          testDaggerheartStore{countdowns: map[string]projectionstore.DaggerheartCountdown{"camp-1:cd-1": {CampaignID: "camp-1", CountdownID: "cd-1"}}},
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error { t.Fatal("unexpected command execution"); return nil },
	})
	_, err := handler.CreateSceneCountdown(testContext(), &pb.DaggerheartCreateSceneCountdownRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		SceneId:           "scene-1",
		CountdownId:       "cd-1",
		Name:              "Clock",
		Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS,
		AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
		StartingValue:     &pb.DaggerheartCreateSceneCountdownRequest_FixedStartingValue{FixedStartingValue: 4},
		LoopBehavior:      pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestHandlerCreateSceneCountdownPropagatesIDGenerationFailure(t *testing.T) {
	handler := newTestHandler(Dependencies{
		NewID:                func() (string, error) { return "", errors.New("boom") },
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error { t.Fatal("unexpected command execution"); return nil },
	})
	_, err := handler.CreateSceneCountdown(testContext(), &pb.DaggerheartCreateSceneCountdownRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		SceneId:           "scene-1",
		Name:              "Clock",
		Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS,
		AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
		StartingValue:     &pb.DaggerheartCreateSceneCountdownRequest_FixedStartingValue{FixedStartingValue: 4},
		LoopBehavior:      pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerCreateSceneCountdownSuccess(t *testing.T) {
	store := &testDaggerheartStore{countdowns: map[string]projectionstore.DaggerheartCountdown{}}
	var commandInput DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commandInput = in
			store.countdowns["camp-1:generated-id"] = projectionstore.DaggerheartCountdown{
				CampaignID:        "camp-1",
				SessionID:         "sess-1",
				SceneID:           "scene-1",
				CountdownID:       "generated-id",
				Name:              "Clock",
				Tone:              rules.CountdownToneProgress,
				AdvancementPolicy: rules.CountdownAdvancementPolicyManual,
				StartingValue:     4,
				RemainingValue:    4,
				LoopBehavior:      rules.CountdownLoopBehaviorNone,
				Status:            rules.CountdownStatusActive,
			}
			return nil
		},
	})
	resp, err := handler.CreateSceneCountdown(testContext(), &pb.DaggerheartCreateSceneCountdownRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		SceneId:           "scene-1",
		Name:              "Clock",
		Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS,
		AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
		StartingValue:     &pb.DaggerheartCreateSceneCountdownRequest_FixedStartingValue{FixedStartingValue: 4},
		LoopBehavior:      pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
	})
	if err != nil {
		t.Fatalf("CreateSceneCountdown returned error: %v", err)
	}
	if resp.Countdown.CountdownID != "generated-id" {
		t.Fatalf("countdown_id = %q, want generated-id", resp.Countdown.CountdownID)
	}
	if commandInput.CommandType != commandids.DaggerheartSceneCountdownCreate {
		t.Fatalf("command type = %q, want %q", commandInput.CommandType, commandids.DaggerheartSceneCountdownCreate)
	}
}
