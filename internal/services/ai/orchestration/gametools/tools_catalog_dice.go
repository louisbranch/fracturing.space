package gametools

import (
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration/daggerhearttools"
)

func diceUtilityToolDefinitions() []productionToolDefinition {
	return []productionToolDefinition{
		{
			Tool: orchestration.Tool{
				Name:        "duality_action_roll",
				Description: "Rolls Duality dice for an action",
				InputSchema: schemaObject(map[string]schemaProperty{
					"modifier":   {Type: "integer", Description: "modifier applied to the roll"},
					"difficulty": {Type: "integer", Description: "optional difficulty target"},
					"rng": {Type: "object", Description: "optional rng configuration", Properties: map[string]schemaProperty{
						"seed":      {Type: "integer", Description: "optional seed for deterministic rolls"},
						"roll_mode": {Type: "string", Description: "roll mode (LIVE or REPLAY)"},
					}},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.DualityActionRoll),
		},
		{
			Tool: orchestration.Tool{
				Name:        "roll_dice",
				Description: "Rolls arbitrary dice pools",
				InputSchema: schemaObject(map[string]schemaProperty{
					"dice": {Type: "array", Description: "dice specifications to roll", Items: &schemaProperty{
						Type: "object", Properties: map[string]schemaProperty{
							"sides": {Type: "integer", Description: "number of sides for the die"},
							"count": {Type: "integer", Description: "number of dice to roll"},
						},
					}},
					"rng": {Type: "object", Description: "optional rng configuration", Properties: map[string]schemaProperty{
						"seed":      {Type: "integer", Description: "optional seed for deterministic rolls"},
						"roll_mode": {Type: "string", Description: "roll mode (LIVE or REPLAY)"},
					}},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.RollDice),
		},
		{
			Tool: orchestration.Tool{
				Name:        "duality_outcome",
				Description: "Evaluates a duality outcome from known dice",
				InputSchema: schemaObject(map[string]schemaProperty{
					"hope":       {Type: "integer", Description: "hope die result"},
					"fear":       {Type: "integer", Description: "fear die result"},
					"modifier":   {Type: "integer", Description: "modifier applied to the roll"},
					"difficulty": {Type: "integer", Description: "optional difficulty target"},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.DualityOutcome),
		},
		{
			Tool: orchestration.Tool{
				Name:        "duality_explain",
				Description: "Explains a duality outcome from known dice",
				InputSchema: schemaObject(map[string]schemaProperty{
					"hope":       {Type: "integer", Description: "hope die result"},
					"fear":       {Type: "integer", Description: "fear die result"},
					"modifier":   {Type: "integer", Description: "modifier applied to the roll"},
					"difficulty": {Type: "integer", Description: "optional difficulty target"},
					"request_id": {Type: "string", Description: "optional correlation identifier"},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.DualityExplain),
		},
		{
			Tool: orchestration.Tool{
				Name:        "duality_probability",
				Description: "Computes outcome probabilities across duality dice",
				InputSchema: schemaObject(map[string]schemaProperty{
					"modifier":   {Type: "integer", Description: "modifier applied to the roll"},
					"difficulty": {Type: "integer", Description: "difficulty target"},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.DualityProbability),
		},
		{
			Tool: orchestration.Tool{
				Name:        "duality_rules_version",
				Description: "Describes the Duality ruleset semantics",
				InputSchema: schemaObject(nil),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.DualityRulesVersion),
		},
	}
}
