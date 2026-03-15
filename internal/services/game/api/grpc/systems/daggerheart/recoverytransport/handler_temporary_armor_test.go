package recoverytransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

func TestHandlerApplyTemporaryArmorSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", HpMax: 5, StressMax: 3, ArmorMax: 2},
		},
		states: map[string]projectionstore.DaggerheartCharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 5, Hope: 1, HopeMax: 2, Stress: 0, Armor: 2},
		},
	}
	var command SystemCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteSystemCommand: func(_ context.Context, in SystemCommandInput) error {
			command = in
			return nil
		},
	})

	result, err := handler.ApplyTemporaryArmor(testContext(), &pb.DaggerheartApplyTemporaryArmorRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		SceneId:     "scene-1",
		Armor: &pb.DaggerheartTemporaryArmor{
			Source:   "ritual",
			Duration: "short_rest",
			Amount:   2,
			SourceId: "blessing:1",
		},
	})
	if err != nil {
		t.Fatalf("ApplyTemporaryArmor returned error: %v", err)
	}
	if result.CharacterID != "char-1" {
		t.Fatalf("character id = %q, want char-1", result.CharacterID)
	}
	if command.CommandType != "sys.daggerheart.character_temporary_armor.apply" {
		t.Fatalf("command type = %q, want temporary armor apply", command.CommandType)
	}
	if command.SceneID != "scene-1" {
		t.Fatalf("scene id = %q, want scene-1", command.SceneID)
	}
}
