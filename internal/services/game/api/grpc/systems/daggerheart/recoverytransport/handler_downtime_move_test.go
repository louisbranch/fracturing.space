package recoverytransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

func TestHandlerApplyDowntimeMoveInvokesStressCallback(t *testing.T) {
	store := &testDaggerheartStore{
		snapshot: projectionstore.DaggerheartSnapshot{CampaignID: "camp-1"},
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", HpMax: 5, StressMax: 3, ArmorMax: 2},
		},
		states: map[string]projectionstore.DaggerheartCharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 5, Hope: 1, HopeMax: 2, Stress: 2, Armor: 0},
		},
	}
	var stressInput StressConditionInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ApplyStressConditionChange: func(_ context.Context, in StressConditionInput) error {
			stressInput = in
			return nil
		},
	})

	result, err := handler.ApplyDowntimeMove(testContext(), &pb.DaggerheartApplyDowntimeMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Move: &pb.DaggerheartDowntimeRequest{
			Move: pb.DaggerheartDowntimeMove_DAGGERHEART_DOWNTIME_MOVE_CLEAR_ALL_STRESS,
		},
	})
	if err != nil {
		t.Fatalf("ApplyDowntimeMove returned error: %v", err)
	}
	if result.CharacterID != "char-1" {
		t.Fatalf("character id = %q, want char-1", result.CharacterID)
	}
	if stressInput.CharacterID != "char-1" {
		t.Fatalf("stress callback character id = %q, want char-1", stressInput.CharacterID)
	}
}
