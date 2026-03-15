package recoverytransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

func TestHandlerResolveBlazeOfGlorySuccess(t *testing.T) {
	store := &testDaggerheartStore{
		states: map[string]projectionstore.DaggerheartCharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 0, Hope: 0, HopeMax: 2, Stress: 0, LifeState: daggerheart.LifeStateBlazeOfGlory},
		},
	}
	var deleted CharacterDeleteInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		AppendCharacterDeletedEvent: func(_ context.Context, in CharacterDeleteInput) error {
			deleted = in
			return nil
		},
	})

	result, err := handler.ResolveBlazeOfGlory(testContext(), &pb.DaggerheartResolveBlazeOfGloryRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
	})
	if err != nil {
		t.Fatalf("ResolveBlazeOfGlory returned error: %v", err)
	}
	if result.CharacterID != "char-1" {
		t.Fatalf("character id = %q, want char-1", result.CharacterID)
	}
	if deleted.CharacterID != "char-1" {
		t.Fatalf("deleted character id = %q, want char-1", deleted.CharacterID)
	}
}
