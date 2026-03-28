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

func TestDeleteSceneCountdown_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.DeleteSceneCountdown(context.Background(), &pb.DaggerheartDeleteSceneCountdownRequest{
		CampaignId: "c1",
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestDeleteSceneCountdown_ValidatesShape(t *testing.T) {
	svc := newActionTestService()
	_, err := svc.DeleteSceneCountdown(context.Background(), &pb.DaggerheartDeleteSceneCountdownRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteSceneCountdown_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Countdowns["camp-1:cd-delete"] = projectionstore.DaggerheartCountdown{
		CampaignID:        "camp-1",
		SessionID:         "sess-1",
		SceneID:           "scene-1",
		CountdownID:       "cd-delete",
		Name:              "Delete Test",
		Tone:              "consequence",
		AdvancementPolicy: "manual",
		StartingValue:     4,
		RemainingValue:    4,
		LoopBehavior:      "none",
		Status:            "active",
	}
	deletePayload := daggerheartpayload.SceneCountdownDeletedPayload{CountdownID: "cd-delete"}
	deletePayloadJSON, err := json.Marshal(deletePayload)
	if err != nil {
		t.Fatalf("encode countdown delete payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.scene_countdown.delete"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.scene_countdown_deleted"),
				Timestamp:     testTimestamp,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				SceneID:       "scene-1",
				RequestID:     "req-scene-countdown-delete-success",
				EntityType:    "scene_countdown",
				EntityID:      "cd-delete",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   deletePayloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = serviceDomain
	resp, err := svc.DeleteSceneCountdown(context.Background(), &pb.DaggerheartDeleteSceneCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CountdownId: "cd-delete",
	})
	if err != nil {
		t.Fatalf("DeleteSceneCountdown returned error: %v", err)
	}
	if resp.CountdownId != "cd-delete" {
		t.Fatalf("countdown_id = %q, want %q", resp.CountdownId, "cd-delete")
	}
}
