//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func ensureMCPCharacterCreationReadiness(t *testing.T, ctx context.Context, client *mcp.ClientSession, characterID string) {
	t.Helper()

	result, err := client.CallTool(ctx, &mcp.CallToolParams{
		Name: "character_creation_workflow_apply",
		Arguments: map[string]any{
			"character_id": characterID,
			"class_id":     "class.guardian",
			"subclass_id":  "subclass.stalwart",
			"heritage": map[string]any{
				"first_feature_ancestry_id":  "heritage.human",
				"second_feature_ancestry_id": "heritage.human",
				"community_id":               "heritage.highborne",
			},
			"agility":         2,
			"strength":        1,
			"finesse":         1,
			"instinct":        0,
			"presence":        0,
			"knowledge":       -1,
			"weapon_ids":      []string{"weapon.longsword"},
			"armor_id":        "armor.gambeson-armor",
			"potion_item_id":  "item.minor-health-potion",
			"background":      "integration background",
			"experiences":     []map[string]any{{"name": "integration experience", "modifier": 2}, {"name": "integration patrol", "modifier": 2}},
			"domain_card_ids": []string{"domain_card.valor-bare-bones", "domain_card.valor-shield-wall"},
			"description":     "integration description",
			"connections":     "integration connections",
		},
	})
	if err != nil {
		t.Fatalf("call character_creation_workflow_apply: %v", err)
	}
	if result == nil || result.IsError {
		t.Fatalf("character_creation_workflow_apply failed: %+v", result)
	}
}
