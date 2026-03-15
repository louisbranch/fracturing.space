package charactermutationtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
)

func TestHandlerUseConsumableSuccess(t *testing.T) {
	var commandInput CharacterCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			commandInput = in
			return nil
		},
	})

	resp, err := handler.UseConsumable(testContext(), &pb.DaggerheartUseConsumableRequest{
		CampaignId:     "camp-1",
		CharacterId:    "char-1",
		ConsumableId:   "cons-1",
		QuantityBefore: 2,
		QuantityAfter:  1,
	})
	if err != nil {
		t.Fatalf("UseConsumable returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character id = %q, want %q", resp.CharacterId, "char-1")
	}
	if commandInput.CommandType != commandids.DaggerheartConsumableUse {
		t.Fatalf("command type = %v, want %v", commandInput.CommandType, commandids.DaggerheartConsumableUse)
	}
}

func TestHandlerAcquireConsumableSuccess(t *testing.T) {
	var commandInput CharacterCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			commandInput = in
			return nil
		},
	})

	resp, err := handler.AcquireConsumable(testContext(), &pb.DaggerheartAcquireConsumableRequest{
		CampaignId:     "camp-1",
		CharacterId:    "char-1",
		ConsumableId:   "cons-1",
		QuantityBefore: 1,
		QuantityAfter:  3,
	})
	if err != nil {
		t.Fatalf("AcquireConsumable returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character id = %q, want %q", resp.CharacterId, "char-1")
	}
	if commandInput.CommandType != commandids.DaggerheartConsumableAcquire {
		t.Fatalf("command type = %v, want %v", commandInput.CommandType, commandids.DaggerheartConsumableAcquire)
	}
}
