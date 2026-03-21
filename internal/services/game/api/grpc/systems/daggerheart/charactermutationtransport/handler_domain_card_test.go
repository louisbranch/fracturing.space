package charactermutationtransport

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
)

func TestHandlerAcquireDomainCardDefaultsDestination(t *testing.T) {
	var payload daggerheartpayload.DomainCardAcquirePayload
	handler := newTestHandler(Dependencies{
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			if err := json.Unmarshal(in.PayloadJSON, &payload); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
			return nil
		},
	})

	resp, err := handler.AcquireDomainCard(testContext(), &pb.DaggerheartAcquireDomainCardRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		CardId:      "card-1",
		CardLevel:   2,
	})
	if err != nil {
		t.Fatalf("AcquireDomainCard returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character id = %q, want %q", resp.CharacterId, "char-1")
	}
	if payload.Destination != "vault" {
		t.Fatalf("destination = %q, want vault", payload.Destination)
	}
}
