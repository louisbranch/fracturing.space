package workfloweffects

import (
	"context"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func TestApplyStressVulnerableCondition_NoThresholdCrossing(t *testing.T) {
	called := false
	handler := NewHandler(Dependencies{
		ExecuteConditionChange: func(context.Context, ConditionChangeCommandInput) error {
			called = true
			return nil
		},
	})

	err := handler.ApplyStressVulnerableCondition(context.Background(), ApplyStressVulnerableConditionInput{
		CampaignID:   "camp-1",
		SessionID:    "sess-1",
		CharacterID:  "char-1",
		StressBefore: 2,
		StressAfter:  2,
		StressMax:    6,
	})
	if err != nil {
		t.Fatalf("ApplyStressVulnerableCondition returned error: %v", err)
	}
	if called {
		t.Fatal("expected no condition change execution")
	}
}

func TestApplyStressVulnerableCondition_ExecutesRepair(t *testing.T) {
	var got ConditionChangeCommandInput
	handler := NewHandler(Dependencies{
		ConditionChangeAlreadyApplied: func(context.Context, ConditionChangeReplayCheckInput) (bool, error) {
			return false, nil
		},
		ExecuteConditionChange: func(_ context.Context, in ConditionChangeCommandInput) error {
			got = in
			return nil
		},
	})

	err := handler.ApplyStressVulnerableCondition(context.Background(), ApplyStressVulnerableConditionInput{
		CampaignID:   "camp-1",
		SessionID:    "sess-1",
		CharacterID:  "char-1",
		Conditions:   []projectionstore.DaggerheartConditionState{{Standard: daggerheart.ConditionHidden}},
		StressBefore: 5,
		StressAfter:  6,
		StressMax:    6,
		RequestID:    "req-1",
	})
	if err != nil {
		t.Fatalf("ApplyStressVulnerableCondition returned error: %v", err)
	}
	if got.CharacterID != "char-1" || got.CampaignID != "camp-1" || got.SessionID != "sess-1" {
		t.Fatalf("unexpected command input: %+v", got)
	}
	if len(got.PayloadJSON) == 0 {
		t.Fatal("expected payload JSON")
	}
}

func TestApplyStressVulnerableCondition_SkipsWhenReplayAlreadyApplied(t *testing.T) {
	called := false
	handler := NewHandler(Dependencies{
		ConditionChangeAlreadyApplied: func(context.Context, ConditionChangeReplayCheckInput) (bool, error) {
			return true, nil
		},
		ExecuteConditionChange: func(context.Context, ConditionChangeCommandInput) error {
			called = true
			return nil
		},
	})

	rollSeq := uint64(22)
	err := handler.ApplyStressVulnerableCondition(context.Background(), ApplyStressVulnerableConditionInput{
		CampaignID:   "camp-1",
		SessionID:    "sess-1",
		CharacterID:  "char-1",
		Conditions:   []projectionstore.DaggerheartConditionState{{Standard: daggerheart.ConditionHidden}},
		StressBefore: 5,
		StressAfter:  6,
		StressMax:    6,
		RollSeq:      &rollSeq,
		RequestID:    "req-1",
	})
	if err != nil {
		t.Fatalf("ApplyStressVulnerableCondition returned error: %v", err)
	}
	if called {
		t.Fatal("expected replay-applied repair to be skipped")
	}
}
