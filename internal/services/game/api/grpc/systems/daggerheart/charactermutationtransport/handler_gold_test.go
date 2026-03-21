package charactermutationtransport

import (
	"context"
	"errors"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerUpdateGoldRequiresExecutor(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.UpdateGold(testContext(), &pb.DaggerheartUpdateGoldRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}

func TestHandlerUpdateGoldSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"camp-1:char-1": testProfile("camp-1", "char-1"),
		},
	}
	var commandInput CharacterCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteCharacterCommand: func(_ context.Context, in CharacterCommandInput) error {
			commandInput = in
			store.profiles["camp-1:char-1"] = projectionstore.DaggerheartCharacterProfile{
				CampaignID:   "camp-1",
				CharacterID:  "char-1",
				Level:        1,
				GoldHandfuls: 4,
				GoldBags:     5,
				GoldChests:   6,
			}
			return nil
		},
	})

	resp, err := handler.UpdateGold(testContext(), &pb.DaggerheartUpdateGoldRequest{
		CampaignId:     "camp-1",
		CharacterId:    "char-1",
		HandfulsBefore: 1,
		HandfulsAfter:  4,
		BagsBefore:     2,
		BagsAfter:      5,
		ChestsBefore:   3,
		ChestsAfter:    6,
		Reason:         "loot",
	})
	if err != nil {
		t.Fatalf("UpdateGold returned error: %v", err)
	}
	if resp.Handfuls != 4 || resp.Bags != 5 || resp.Chests != 6 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if commandInput.CommandType != commandids.DaggerheartGoldUpdate {
		t.Fatalf("command type = %v, want %v", commandInput.CommandType, commandids.DaggerheartGoldUpdate)
	}
}

func TestHandlerMapsProfileLookupErrors(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Daggerheart: &testDaggerheartStore{getErr: errors.New("boom")},
		ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.UpdateGold(testContext(), &pb.DaggerheartUpdateGoldRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}
