package adversarytransport

import (
	"context"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestHandlerUpdateAdversarySuccess(t *testing.T) {
	store := &testDaggerheartStore{adversaries: map[string]projectionstore.DaggerheartAdversary{
		"adv-1": {AdversaryID: "adv-1", CampaignID: "camp-1", Name: "Old", HP: 4, HPMax: 6, Stress: 1, StressMax: 2, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}}
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			current := store.adversaries[in.EntityID]
			current.Name = "New"
			current.HP = 5
			current.UpdatedAt = time.Now()
			store.adversaries[in.EntityID] = current
			return nil
		},
	})

	resp, err := handler.UpdateAdversary(testContext(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Name:        wrapperspb.String("New"),
		Hp:          wrapperspb.Int32(5),
	})
	if err != nil {
		t.Fatalf("UpdateAdversary returned error: %v", err)
	}
	if resp.GetAdversary().GetName() != "New" {
		t.Fatalf("name = %q, want New", resp.GetAdversary().GetName())
	}
}
