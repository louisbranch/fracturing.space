package charactermutationtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandlerApplyLevelUpRejectsInvalidIncrement(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteCharacterCommand: func(context.Context, CharacterCommandInput) error { return nil },
	})

	_, err := handler.ApplyLevelUp(testContext(), &pb.DaggerheartApplyLevelUpRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LevelAfter:  5,
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerApplyLevelUpSuccess(t *testing.T) {
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
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				Level:       2,
			}
			return nil
		},
	})

	resp, err := handler.ApplyLevelUp(testContext(), &pb.DaggerheartApplyLevelUpRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LevelAfter:  2,
		Advancements: []*pb.DaggerheartLevelUpAdvancement{
			{Type: "trait", Trait: "agility"},
		},
	})
	if err != nil {
		t.Fatalf("ApplyLevelUp returned error: %v", err)
	}
	if resp.Level != 2 || resp.Tier != 2 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if commandInput.CommandType != commandids.DaggerheartLevelUpApply {
		t.Fatalf("command type = %v, want %v", commandInput.CommandType, commandids.DaggerheartLevelUpApply)
	}
}
