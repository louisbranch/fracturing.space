//go:build integration

package integration

import (
	"context"
	"testing"

	dualitydomain "github.com/louisbranch/duality-engine/internal/duality/domain"
	"github.com/louisbranch/duality-engine/internal/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// runDualityToolsTests exercises duality-related MCP tools.
func runDualityToolsTests(t *testing.T, suite *integrationSuite) {
	t.Helper()

	t.Run("duality outcome", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		params := &mcp.CallToolParams{
			Name: "duality_outcome",
			Arguments: map[string]any{
				"hope":       10,
				"fear":       4,
				"modifier":   1,
				"difficulty": 10,
			},
		}
		result, err := suite.client.CallTool(ctx, params)
		if err != nil {
			t.Fatalf("call duality_outcome: %v", err)
		}
		if result == nil {
			t.Fatal("call duality_outcome returned nil")
		}
		if result.IsError {
			t.Fatalf("call duality_outcome returned error content: %+v", result.Content)
		}

		output := decodeStructuredContent[domain.DualityOutcomeResult](t, result.StructuredContent)
		expected, err := dualitydomain.EvaluateOutcome(dualitydomain.OutcomeRequest{
			Hope:       10,
			Fear:       4,
			Modifier:   1,
			Difficulty: intPointer(10),
		})
		if err != nil {
			t.Fatalf("evaluate outcome: %v", err)
		}
		if output.Total != expected.Total || output.MeetsDifficulty != expected.MeetsDifficulty {
			t.Fatalf("unexpected outcome totals: %+v", output)
		}
	})

	t.Run("rules metadata", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		params := &mcp.CallToolParams{Name: "duality_rules_version"}
		result, err := suite.client.CallTool(ctx, params)
		if err != nil {
			t.Fatalf("call duality_rules_version: %v", err)
		}
		if result == nil {
			t.Fatal("call duality_rules_version returned nil")
		}
		if result.IsError {
			t.Fatalf("duality_rules_version returned error content: %+v", result.Content)
		}

		output := decodeStructuredContent[domain.RulesVersionResult](t, result.StructuredContent)
		expected := dualitydomain.RulesVersion()
		if output.System != expected.System {
			t.Fatalf("expected system %q, got %q", expected.System, output.System)
		}
		if output.Module != expected.Module {
			t.Fatalf("expected module %q, got %q", expected.Module, output.Module)
		}
		if output.RulesVersion != expected.RulesVersion {
			t.Fatalf("expected rules version %q, got %q", expected.RulesVersion, output.RulesVersion)
		}
		if output.DiceModel != expected.DiceModel {
			t.Fatalf("expected dice model %q, got %q", expected.DiceModel, output.DiceModel)
		}

		expectedOutcomes := []string{
			"ROLL_WITH_HOPE",
			"ROLL_WITH_FEAR",
			"SUCCESS_WITH_HOPE",
			"SUCCESS_WITH_FEAR",
			"FAILURE_WITH_HOPE",
			"FAILURE_WITH_FEAR",
			"CRITICAL_SUCCESS",
		}
		if len(output.Outcomes) != len(expectedOutcomes) {
			t.Fatalf("expected %d outcomes, got %d", len(expectedOutcomes), len(output.Outcomes))
		}
		for i, expectedOutcome := range expectedOutcomes {
			if output.Outcomes[i] != expectedOutcome {
				t.Fatalf("expected outcome %q at index %d, got %q", expectedOutcome, i, output.Outcomes[i])
			}
		}
	})
}
