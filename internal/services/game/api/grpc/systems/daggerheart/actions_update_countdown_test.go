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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

func TestAdvanceSceneCountdown_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.AdvanceSceneCountdown(context.Background(), &pb.DaggerheartAdvanceSceneCountdownRequest{
		CampaignId: "c1",
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestAdvanceSceneCountdown_ValidatesShape(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.AdvanceSceneCountdown(context.Background(), &pb.DaggerheartAdvanceSceneCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		SceneId:    "scene-1",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestAdvanceSceneCountdown_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Countdowns["camp-1:cd-1"] = projectionstore.DaggerheartCountdown{
		CampaignID:        "camp-1",
		SessionID:         "sess-1",
		SceneID:           "scene-1",
		CountdownID:       "cd-1",
		Name:              "Update",
		Tone:              "progress",
		AdvancementPolicy: "manual",
		StartingValue:     4,
		RemainingValue:    2,
		LoopBehavior:      "none",
		Status:            "active",
	}
	advancePayload := daggerheartpayload.SceneCountdownAdvancedPayload{
		CountdownID:     "cd-1",
		BeforeRemaining: 2,
		AfterRemaining:  1,
		AdvancedBy:      1,
		StatusBefore:    "active",
		StatusAfter:     "active",
		Reason:          "advance",
	}
	advancePayloadJSON, err := json.Marshal(advancePayload)
	if err != nil {
		t.Fatalf("encode countdown advance payload: %v", err)
	}
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.scene_countdown.advance"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.scene_countdown_advanced"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				SceneID:       "scene-1",
				RequestID:     "req-scene-countdown-advance",
				EntityType:    "scene_countdown",
				EntityID:      "cd-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   advancePayloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	resp, err := svc.AdvanceSceneCountdown(context.Background(), &pb.DaggerheartAdvanceSceneCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CountdownId: "cd-1",
		Amount:      1,
		Reason:      "advance",
	})
	if err != nil {
		t.Fatalf("AdvanceSceneCountdown returned error: %v", err)
	}
	if resp.GetAdvance().GetRemainingBefore() != 2 || resp.GetAdvance().GetRemainingAfter() != 1 {
		t.Fatalf("unexpected advance summary: %#v", resp.GetAdvance())
	}
	if resp.Countdown == nil || resp.Countdown.RemainingValue != 1 {
		t.Fatalf("unexpected advance response: %#v", resp)
	}
}
