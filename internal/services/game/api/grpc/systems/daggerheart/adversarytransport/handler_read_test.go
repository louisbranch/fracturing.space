package adversarytransport

import (
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

func TestHandlerReadOperations(t *testing.T) {
	store := &testDaggerheartStore{adversaries: map[string]projectionstore.DaggerheartAdversary{
		"adv-1": {AdversaryID: "adv-1", CampaignID: "camp-1", Name: "One", SessionID: "sess-1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		"adv-2": {AdversaryID: "adv-2", CampaignID: "camp-1", Name: "Two", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}}
	handler := newTestHandler(Dependencies{Daggerheart: store})

	getResp, err := handler.GetAdversary(testContext(), &pb.DaggerheartGetAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("GetAdversary returned error: %v", err)
	}
	if getResp.GetAdversary().GetId() != "adv-1" {
		t.Fatalf("adversary id = %q, want adv-1", getResp.GetAdversary().GetId())
	}

	listResp, err := handler.ListAdversaries(testContext(), &pb.DaggerheartListAdversariesRequest{
		CampaignId: "camp-1",
	})
	if err != nil {
		t.Fatalf("ListAdversaries returned error: %v", err)
	}
	if len(listResp.GetAdversaries()) != 2 {
		t.Fatalf("adversaries = %d, want 2", len(listResp.GetAdversaries()))
	}
}
