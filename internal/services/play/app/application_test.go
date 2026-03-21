package app

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

func TestPlayApplicationSystemMetadata(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	server.campaign = fakePlayCampaignClient{response: &gamev1.GetCampaignResponse{
		Campaign: &gamev1.Campaign{Id: "c1", System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
	}}
	server.system = fakePlaySystemClient{response: &gamev1.GetGameSystemResponse{
		System: &gamev1.GameSystemInfo{Id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, Name: "Daggerheart", Version: "v1"},
	}}

	system, err := server.application().systemMetadata(context.Background(), playRequest{
		campaignRequest: campaignRequest{CampaignID: "c1"},
		UserID:          "user-1",
	})
	if err != nil {
		t.Fatalf("systemMetadata() error = %v", err)
	}
	if system.ID != "daggerheart" || system.Name != "Daggerheart" || system.Version != "v1" {
		t.Fatalf("system = %#v", system)
	}
}
