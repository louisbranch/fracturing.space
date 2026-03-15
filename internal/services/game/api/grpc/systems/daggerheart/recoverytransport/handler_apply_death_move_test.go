package recoverytransport

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

func TestHandlerApplyDeathMoveRiskItAllDeathAppendsDeleteEvent(t *testing.T) {
	seed := findRiskItAllDeathSeed(t)
	store := &testDaggerheartStore{
		profiles: map[string]projectionstore.DaggerheartCharacterProfile{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", HpMax: 5, StressMax: 3, Level: 1},
		},
		states: map[string]projectionstore.DaggerheartCharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Hp: 0, Hope: 2, HopeMax: 2, Stress: 1},
		},
	}
	var deleted CharacterDeleteInput
	var stressInput StressConditionInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ResolveSeed: func(*commonv1.RngRequest, func() (int64, error), func(commonv1.RollMode) bool) (int64, string, commonv1.RollMode, error) {
			return seed, "replay", commonv1.RollMode_REPLAY, nil
		},
		AppendCharacterDeletedEvent: func(_ context.Context, in CharacterDeleteInput) error {
			deleted = in
			return nil
		},
		ApplyStressConditionChange: func(_ context.Context, in StressConditionInput) error {
			stressInput = in
			return nil
		},
	})

	result, err := handler.ApplyDeathMove(testContext(), &pb.DaggerheartApplyDeathMoveRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		SceneId:     "scene-1",
		Move:        pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL,
	})
	if err != nil {
		t.Fatalf("ApplyDeathMove returned error: %v", err)
	}
	if result.CharacterID != "char-1" {
		t.Fatalf("character id = %q, want char-1", result.CharacterID)
	}
	if result.Outcome.LifeState != daggerheart.LifeStateDead {
		t.Fatalf("life state = %q, want dead", result.Outcome.LifeState)
	}
	if deleted.CharacterID != "char-1" {
		t.Fatalf("deleted character id = %q, want char-1", deleted.CharacterID)
	}
	if stressInput.CharacterID != "char-1" {
		t.Fatalf("stress callback character id = %q, want char-1", stressInput.CharacterID)
	}
}

func findRiskItAllDeathSeed(t *testing.T) int64 {
	t.Helper()
	for seed := int64(1); seed <= 256; seed++ {
		outcome, err := daggerheart.ResolveDeathMove(daggerheart.DeathMoveInput{
			Move:      daggerheart.DeathMoveRiskItAll,
			Level:     1,
			HP:        0,
			HPMax:     5,
			Hope:      2,
			HopeMax:   2,
			Stress:    1,
			StressMax: 3,
			Seed:      seed,
		})
		if err != nil {
			t.Fatalf("ResolveDeathMove seed=%d: %v", seed, err)
		}
		if outcome.LifeState == daggerheart.LifeStateDead {
			return seed
		}
	}
	t.Fatal("could not find deterministic risk_it_all death seed")
	return 0
}
