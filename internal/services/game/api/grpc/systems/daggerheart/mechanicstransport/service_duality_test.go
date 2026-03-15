package mechanicstransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func TestHandlerDualityOutcomeAndExplain(t *testing.T) {
	handler := newTestHandler(42)
	difficulty := int32(10)

	outcomeResp, err := handler.DualityOutcome(context.Background(), &pb.DualityOutcomeRequest{
		Hope:       10,
		Fear:       4,
		Modifier:   1,
		Difficulty: &difficulty,
	})
	if err != nil {
		t.Fatalf("DualityOutcome: %v", err)
	}
	if outcomeResp.GetOutcome() == pb.Outcome_OUTCOME_UNSPECIFIED {
		t.Fatal("expected concrete outcome")
	}

	explainResp, err := handler.DualityExplain(context.Background(), &pb.DualityExplainRequest{
		Hope:       10,
		Fear:       4,
		Modifier:   1,
		Difficulty: &difficulty,
		RequestId:  stringPointer("trace-123"),
	})
	if err != nil {
		t.Fatalf("DualityExplain: %v", err)
	}
	if len(explainResp.GetSteps()) == 0 || explainResp.GetRulesVersion() == "" {
		t.Fatalf("unexpected explain response: %+v", explainResp)
	}
}

func TestHandlerDualityProbabilityAndRulesVersion(t *testing.T) {
	handler := newTestHandler(42)

	probabilityResp, err := handler.DualityProbability(context.Background(), &pb.DualityProbabilityRequest{
		Modifier:   0,
		Difficulty: 10,
	})
	if err != nil {
		t.Fatalf("DualityProbability: %v", err)
	}
	if probabilityResp.GetTotalOutcomes() == 0 || len(probabilityResp.GetOutcomeCounts()) == 0 {
		t.Fatalf("unexpected probability response: %+v", probabilityResp)
	}

	rulesResp, err := handler.RulesVersion(context.Background(), &pb.RulesVersionRequest{})
	if err != nil {
		t.Fatalf("RulesVersion: %v", err)
	}
	if rulesResp.GetRulesVersion() == "" || len(rulesResp.GetOutcomes()) == 0 {
		t.Fatalf("unexpected rules version response: %+v", rulesResp)
	}
}
