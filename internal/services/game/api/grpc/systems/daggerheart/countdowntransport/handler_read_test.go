package countdowntransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestHandlerReadOperations(t *testing.T) {
	store := testDaggerheartStore{countdowns: map[string]projectionstore.DaggerheartCountdown{
		"camp-1:cd-3": {CampaignID: "camp-1", CountdownID: "cd-3", Name: "Long Project", Current: 2, Max: 6, Kind: "consequence", Direction: "increase"},
		"camp-1:cd-2": {CampaignID: "camp-1", SessionID: "sess-1", SceneID: "scene-1", CountdownID: "cd-2", Name: "Storm Front", Current: 2, Max: 6, Kind: "consequence", Direction: "increase"},
		"camp-1:cd-1": {CampaignID: "camp-1", SessionID: "sess-1", SceneID: "scene-1", CountdownID: "cd-1", Name: "Breach", Current: 1, Max: 4, Kind: "progress", Direction: "increase"},
	}}
	handler := newTestHandler(Dependencies{Daggerheart: store})

	getResp, err := handler.GetSceneCountdown(testContext(), &pb.DaggerheartGetSceneCountdownRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		CountdownId: "cd-1",
	})
	if err != nil {
		t.Fatalf("GetSceneCountdown returned error: %v", err)
	}
	if getResp.GetCountdown().GetCountdownId() != "cd-1" {
		t.Fatalf("countdown id = %q, want cd-1", getResp.GetCountdown().GetCountdownId())
	}

	listResp, err := handler.ListSceneCountdowns(testContext(), &pb.DaggerheartListSceneCountdownsRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		SceneId:    "scene-1",
	})
	if err != nil {
		t.Fatalf("ListSceneCountdowns returned error: %v", err)
	}
	if len(listResp.GetCountdowns()) != 2 {
		t.Fatalf("countdowns = %d, want 2", len(listResp.GetCountdowns()))
	}
	if listResp.GetCountdowns()[0].GetCountdownId() != "cd-1" || listResp.GetCountdowns()[1].GetCountdownId() != "cd-2" {
		t.Fatalf("countdown order = %#v", listResp.GetCountdowns())
	}

	campaignGetResp, err := handler.GetCampaignCountdown(testContext(), &pb.DaggerheartGetCampaignCountdownRequest{
		CampaignId:  "camp-1",
		CountdownId: "cd-3",
	})
	if err != nil {
		t.Fatalf("GetCampaignCountdown returned error: %v", err)
	}
	if campaignGetResp.GetCountdown().GetCountdownId() != "cd-3" {
		t.Fatalf("campaign countdown id = %q, want cd-3", campaignGetResp.GetCountdown().GetCountdownId())
	}

	campaignListResp, err := handler.ListCampaignCountdowns(testContext(), &pb.DaggerheartListCampaignCountdownsRequest{
		CampaignId: "camp-1",
	})
	if err != nil {
		t.Fatalf("ListCampaignCountdowns returned error: %v", err)
	}
	if len(campaignListResp.GetCountdowns()) != 1 || campaignListResp.GetCountdowns()[0].GetCountdownId() != "cd-3" {
		t.Fatalf("campaign countdowns = %#v", campaignListResp.GetCountdowns())
	}
}
