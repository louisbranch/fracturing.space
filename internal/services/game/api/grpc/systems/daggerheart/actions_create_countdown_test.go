package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

func TestCreateSceneCountdown_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.CreateSceneCountdown(context.Background(), &pb.DaggerheartCreateSceneCountdownRequest{
		CampaignId: "c1",
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestCreateSceneCountdown_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateSceneCountdown(context.Background(), &pb.DaggerheartCreateSceneCountdownRequest{
		SessionId: "sess-1",
		SceneId:   "scene-1",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestCreateSceneCountdown_MissingSessionOrScene(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateSceneCountdown(context.Background(), &pb.DaggerheartCreateSceneCountdownRequest{
		CampaignId: "camp-1",
		SceneId:    "scene-1",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)

	_, err = svc.CreateSceneCountdown(context.Background(), &pb.DaggerheartCreateSceneCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestCreateSceneCountdown_InvalidShape(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateSceneCountdown(context.Background(), &pb.DaggerheartCreateSceneCountdownRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		SceneId:           "scene-1",
		Name:              "Test Countdown",
		Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS,
		AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
		StartingValue:     &pb.DaggerheartCreateSceneCountdownRequest_FixedStartingValue{FixedStartingValue: 0},
		LoopBehavior:      pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestCreateSceneCountdown_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.CreateSceneCountdown(context.Background(), &pb.DaggerheartCreateSceneCountdownRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		SceneId:           "scene-1",
		Name:              "Test Countdown",
		Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS,
		AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
		StartingValue:     &pb.DaggerheartCreateSceneCountdownRequest_FixedStartingValue{FixedStartingValue: 4},
		LoopBehavior:      pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestCreateSceneCountdown_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	countdownPayload := daggerheartpayload.SceneCountdownCreatedPayload{
		SessionID:         "sess-1",
		SceneID:           "scene-1",
		CountdownID:       "cd-1",
		Name:              "Test Countdown",
		Tone:              "progress",
		AdvancementPolicy: "manual",
		StartingValue:     4,
		RemainingValue:    4,
		LoopBehavior:      "none",
		Status:            "active",
	}
	countdownPayloadJSON, err := json.Marshal(countdownPayload)
	if err != nil {
		t.Fatalf("encode countdown payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.scene_countdown.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.scene_countdown_created"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				SceneID:       "scene-1",
				RequestID:     "req-scene-countdown-create-success",
				EntityType:    "scene_countdown",
				EntityID:      "cd-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   countdownPayloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = serviceDomain
	resp, err := svc.CreateSceneCountdown(context.Background(), &pb.DaggerheartCreateSceneCountdownRequest{
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		SceneId:           "scene-1",
		Name:              "Test Countdown",
		Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS,
		AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
		StartingValue:     &pb.DaggerheartCreateSceneCountdownRequest_FixedStartingValue{FixedStartingValue: 4},
		LoopBehavior:      pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
		CountdownId:       "cd-1",
	})
	if err != nil {
		t.Fatalf("CreateSceneCountdown returned error: %v", err)
	}
	if resp.Countdown == nil || resp.Countdown.Name != "Test Countdown" || resp.Countdown.StartingValue != 4 || resp.Countdown.RemainingValue != 4 {
		t.Fatalf("unexpected response countdown: %#v", resp.Countdown)
	}
}
