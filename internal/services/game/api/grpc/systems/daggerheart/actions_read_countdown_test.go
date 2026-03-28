package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

func TestGetSceneCountdown_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.GetSceneCountdown(context.Background(), &pb.DaggerheartGetSceneCountdownRequest{
		CampaignId: "camp-1",
	})
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestGetSceneCountdown_Success(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Countdowns["camp-1:cd-1"] = projectionstore.DaggerheartCountdown{
		CampaignID:        "camp-1",
		SessionID:         "sess-1",
		SceneID:           "scene-1",
		CountdownID:       "cd-1",
		Name:              "Breach",
		Tone:              "progress",
		AdvancementPolicy: "action_standard",
		StartingValue:     4,
		RemainingValue:    1,
		LoopBehavior:      "none",
		Status:            "active",
	}

	resp, err := svc.GetSceneCountdown(context.Background(), &pb.DaggerheartGetSceneCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CountdownId: "cd-1",
	})
	if err != nil {
		t.Fatalf("GetSceneCountdown returned error: %v", err)
	}
	if resp.GetCountdown().GetCountdownId() != "cd-1" || resp.GetCountdown().GetName() != "Breach" {
		t.Fatalf("unexpected scene countdown: %#v", resp.GetCountdown())
	}
}

func TestListCampaignCountdowns_Success(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	dhStore.Countdowns["camp-1:cd-2"] = projectionstore.DaggerheartCountdown{
		CampaignID:        "camp-1",
		CountdownID:       "cd-2",
		Name:              "Storm Front",
		Tone:              "consequence",
		AdvancementPolicy: "long_rest",
		StartingValue:     6,
		RemainingValue:    2,
		LoopBehavior:      "none",
		Status:            "active",
	}
	dhStore.Countdowns["camp-1:cd-1"] = projectionstore.DaggerheartCountdown{
		CampaignID:        "camp-1",
		CountdownID:       "cd-1",
		Name:              "Breach",
		Tone:              "progress",
		AdvancementPolicy: "manual",
		StartingValue:     4,
		RemainingValue:    1,
		LoopBehavior:      "none",
		Status:            "active",
	}

	resp, err := svc.ListCampaignCountdowns(context.Background(), &pb.DaggerheartListCampaignCountdownsRequest{
		CampaignId: "camp-1",
	})
	if err != nil {
		t.Fatalf("ListCampaignCountdowns returned error: %v", err)
	}
	if len(resp.GetCountdowns()) != 2 {
		t.Fatalf("countdowns = %d, want 2", len(resp.GetCountdowns()))
	}
	if resp.GetCountdowns()[0].GetCountdownId() != "cd-1" || resp.GetCountdowns()[1].GetCountdownId() != "cd-2" {
		t.Fatalf("countdown order = %#v", resp.GetCountdowns())
	}
}
