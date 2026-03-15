package charactermutationtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
)

func TestHandlerSwapEquipmentSuccess(t *testing.T) {
	var commandInput CharacterCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			commandInput = in
			return nil
		},
	})

	resp, err := handler.SwapEquipment(testContext(), &pb.DaggerheartSwapEquipmentRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		ItemId:      "item-1",
		ItemType:    "weapon",
		From:        "inventory",
		To:          "active",
		StressCost:  1,
	})
	if err != nil {
		t.Fatalf("SwapEquipment returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character id = %q, want %q", resp.CharacterId, "char-1")
	}
	if commandInput.CommandType != commandids.DaggerheartEquipmentSwap {
		t.Fatalf("command type = %v, want %v", commandInput.CommandType, commandids.DaggerheartEquipmentSwap)
	}
}
