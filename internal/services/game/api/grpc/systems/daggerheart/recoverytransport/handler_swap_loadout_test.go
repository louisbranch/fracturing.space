package recoverytransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestHandlerSwapLoadoutWithRecallCostExecutesStressSpend(t *testing.T) {
	store := &testDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", HpMax: 5, StressMax: 3, ArmorMax: 2},
		},
		states: map[string]projectionstore.DaggerheartCharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 5, Hope: 1, HopeMax: 2, Stress: 2, Armor: 0},
		},
	}
	var commands []SystemCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteSystemCommand: func(_ context.Context, in SystemCommandInput) error {
			commands = append(commands, in)
			return nil
		},
	})

	_, err := handler.SwapLoadout(testContext(), &pb.DaggerheartSwapLoadoutRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Swap: &pb.DaggerheartLoadoutSwapRequest{
			CardId:     "card-1",
			RecallCost: 1,
		},
	})
	if err != nil {
		t.Fatalf("SwapLoadout returned error: %v", err)
	}
	if len(commands) != 2 {
		t.Fatalf("command count = %d, want 2", len(commands))
	}
}
