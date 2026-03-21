package adversarytransport

import (
	"context"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestHandlerDeleteAdversarySuccess(t *testing.T) {
	store := &testDaggerheartStore{adversaries: map[string]projectionstore.DaggerheartAdversary{
		"adv-1": {AdversaryID: "adv-1", CampaignID: "camp-1", Name: "Old", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}}
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			delete(store.adversaries, in.EntityID)
			return nil
		},
	})

	resp, err := handler.DeleteAdversary(testContext(), &pb.DaggerheartDeleteAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("DeleteAdversary returned error: %v", err)
	}
	if resp.GetAdversary().GetId() != "adv-1" {
		t.Fatalf("adversary id = %q, want adv-1", resp.GetAdversary().GetId())
	}
}
