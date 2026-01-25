//go:build integration

package integration

import (
	"context"
	"fmt"
	"sort"
	"testing"
)

// runMCPToolsTests exercises MCP tool discovery.
func runMCPToolsTests(t *testing.T, suite *integrationSuite) {
	t.Helper()

	t.Run("list tools", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		result, err := suite.client.ListTools(ctx, nil)
		if err != nil {
			t.Fatalf("list tools: %v", err)
		}
		if result == nil {
			t.Fatal("list tools returned nil result")
		}

		expected := []string{
			"actor_control_set",
			"actor_create",
			"campaign_create",
			"duality_action_roll",
			"duality_explain",
			"duality_outcome",
			"duality_probability",
			"duality_rules_version",
			"participant_create",
			"roll_dice",
			"session_start",
			"set_context",
		}

		actual := make([]string, 0, len(result.Tools))
		for _, tool := range result.Tools {
			actual = append(actual, tool.Name)
		}

		assertStringSet(t, "tools", actual, expected)
	})
}

// assertStringSet compares unordered string sets and reports differences.
func assertStringSet(t *testing.T, label string, actual []string, expected []string) {
	t.Helper()

	actualSet := make(map[string]int, len(actual))
	for _, item := range actual {
		actualSet[item]++
	}

	expectedSet := make(map[string]int, len(expected))
	for _, item := range expected {
		expectedSet[item]++
	}

	missing := make([]string, 0)
	for item := range expectedSet {
		if actualSet[item] == 0 {
			missing = append(missing, item)
		}
	}

	extra := make([]string, 0)
	for item := range actualSet {
		if expectedSet[item] == 0 {
			extra = append(extra, item)
		}
	}

	if len(missing) == 0 && len(extra) == 0 {
		return
	}

	sort.Strings(missing)
	sort.Strings(extra)
	if len(missing) > 0 || len(extra) > 0 {
		message := ""
		if len(missing) > 0 {
			message = fmt.Sprintf("missing %s: %v", label, missing)
		}
		if len(extra) > 0 {
			if message != "" {
				message += "; "
			}
			message += fmt.Sprintf("unexpected %s: %v", label, extra)
		}
		t.Fatalf("%s", message)
	}
}
