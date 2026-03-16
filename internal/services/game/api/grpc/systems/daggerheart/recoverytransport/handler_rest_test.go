package recoverytransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

func TestHandlerApplyRestSuccess(t *testing.T) {
	store := &testDaggerheartStore{
		snapshot: projectionstore.DaggerheartSnapshot{CampaignID: "camp-1", GMFear: 2, ConsecutiveShortRests: 1},
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", HpMax: 5, StressMax: 3},
		},
		states: map[string]projectionstore.DaggerheartCharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 3, Hope: 2, HopeMax: 2, Stress: 0},
		},
	}
	var commandCount int
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteSystemCommand: func(context.Context, SystemCommandInput) error {
			commandCount++
			return nil
		},
	})

	result, err := handler.ApplyRest(testContext(), &pb.DaggerheartApplyRestRequest{
		CampaignId: "camp-1",
		Rest: &pb.DaggerheartRestRequest{
			RestType: pb.DaggerheartRestType_DAGGERHEART_REST_TYPE_SHORT,
			Participants: []*pb.DaggerheartRestParticipant{
				{CharacterId: "char-1"},
			},
		},
	})
	if err != nil {
		t.Fatalf("ApplyRest returned error: %v", err)
	}
	if commandCount != 1 {
		t.Fatalf("command count = %d, want 1", commandCount)
	}
	if len(result.CharacterStates) != 1 {
		t.Fatalf("character states = %d, want 1", len(result.CharacterStates))
	}
}
