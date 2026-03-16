package adversarytransport

import (
	"context"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

func TestHandlerCreateAdversarySuccess(t *testing.T) {
	store := &testDaggerheartStore{adversaries: map[string]projectionstore.DaggerheartAdversary{}}
	var command DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		GenerateID:  func() (string, error) { return "adv-1", nil },
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			command = in
			store.adversaries[in.EntityID] = projectionstore.DaggerheartAdversary{
				AdversaryID: in.EntityID,
				CampaignID:  in.CampaignID,
				Name:        "Rival",
				HP:          4,
				HPMax:       6,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			return nil
		},
	})

	resp, err := handler.CreateAdversary(testContext(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId:       "camp-1",
		SessionId:        "sess-1",
		SceneId:          "scene-1",
		AdversaryEntryId: "adversary.rival",
	})
	if err != nil {
		t.Fatalf("CreateAdversary returned error: %v", err)
	}
	if resp.GetAdversary().GetId() != "adv-1" {
		t.Fatalf("adversary id = %q, want adv-1", resp.GetAdversary().GetId())
	}
	if command.CommandType == "" {
		t.Fatal("expected command callback to be invoked")
	}
}
