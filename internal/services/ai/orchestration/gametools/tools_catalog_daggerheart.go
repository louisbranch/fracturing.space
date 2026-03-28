package gametools

import (
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration/daggerhearttools"
)

func daggerheartToolDefinitions() []productionToolDefinition {
	return []productionToolDefinition{
		{
			Tool: orchestration.Tool{
				Name:        "character_sheet_read",
				Description: "Reads one authoritative character sheet with traits, equipment, domain cards, active features, Hope, and current state",
				InputSchema: schemaObject(map[string]schemaProperty{
					"character_id": {Type: "string", Description: "character identifier to inspect"},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.CharacterSheetRead),
		},
		{
			Tool: orchestration.Tool{
				Name:        "daggerheart_combat_board_read",
				Description: "Reads the current Daggerheart combat board for the bound session, including GM Fear, active-scene readiness diagnostics, spotlight, visible countdowns, and active adversaries",
				InputSchema: schemaObject(nil),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.CombatBoardRead),
		},
		{
			Tool: orchestration.Tool{
				Name:        "daggerheart_action_roll_resolve",
				Description: "Runs an authoritative Daggerheart action roll and applies its outcome in one tool call",
				InputSchema: schemaObject(map[string]schemaProperty{
					"character_id":              {Type: "string", Description: "acting character identifier"},
					"trait":                     {Type: "string", Description: "trait used for the action roll, such as agility or strength"},
					"difficulty":                {Type: "integer", Description: "difficulty target for the action"},
					"modifiers":                 {Type: "array", Description: "optional roll modifiers", Items: &schemaProperty{Type: "object", Properties: map[string]schemaProperty{"source": {Type: "string", Description: "modifier source label"}, "value": {Type: "integer", Description: "signed modifier value"}}}},
					"advantage":                 {Type: "integer", Description: "count of advantage dice"},
					"disadvantage":              {Type: "integer", Description: "count of disadvantage dice"},
					"underwater":                {Type: "boolean", Description: "whether the action is underwater"},
					"breath_scene_countdown_id": {Type: "string", Description: "optional scene breath countdown to advance for underwater actions"},
					"scene_id":                  {Type: "string", Description: "optional explicit scene identifier; defaults to the active scene"},
					"replace_hope_with_armor":   {Type: "boolean", Description: "whether eligible armor may replace a Hope spend"},
					"context":                   {Type: "string", Description: "optional narrow rules context, for example move_silently"},
					"targets":                   {Type: "array", Description: "optional outcome targets; defaults to the rolling character", Items: &schemaProperty{Type: "string"}},
					"swap_hope_fear":            {Type: "boolean", Description: "whether outcome flavor should treat Hope and Fear as swapped"},
					"rng": {Type: "object", Description: "optional rng configuration", Properties: map[string]schemaProperty{
						"seed":      {Type: "integer", Description: "optional seed for deterministic rolls"},
						"roll_mode": {Type: "string", Description: "roll mode (LIVE or REPLAY)"},
					}},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.ActionRollResolve),
		},
		{
			Tool: orchestration.Tool{
				Name:        "daggerheart_gm_move_apply",
				Description: "Spends Fear through one authoritative Daggerheart GM move; provide exactly one spend target",
				InputSchema: schemaObject(map[string]schemaProperty{
					"fear_spent": {Type: "integer", Description: "Fear spent on the move"},
					"scene_id":   {Type: "string", Description: "optional explicit scene identifier; defaults to the active scene"},
					"direct_move": {
						Type:        "object",
						Description: "direct GM move target; common clean path for environment shifts and reveals",
						Properties: map[string]schemaProperty{
							"kind":         {Type: "string", Description: "move kind, for example additional_move or interrupt_and_move"},
							"shape":        {Type: "string", Description: "move shape, for example reveal_danger or shift_environment"},
							"description":  {Type: "string", Description: "optional move description"},
							"adversary_id": {Type: "string", Description: "optional adversary involved in the direct move"},
						},
					},
					"adversary_feature": {
						Type:        "object",
						Description: "adversary feature Fear spend target",
						Properties: map[string]schemaProperty{
							"adversary_id": {Type: "string", Description: "adversary identifier"},
							"feature_id":   {Type: "string", Description: "feature identifier"},
							"description":  {Type: "string", Description: "optional move description"},
						},
					},
					"environment_feature": {
						Type:        "object",
						Description: "environment feature Fear spend target",
						Properties: map[string]schemaProperty{
							"environment_entity_id": {Type: "string", Description: "environment entity identifier"},
							"feature_id":            {Type: "string", Description: "feature identifier"},
							"description":           {Type: "string", Description: "optional move description"},
						},
					},
					"adversary_experience": {
						Type:        "object",
						Description: "adversary experience Fear spend target",
						Properties: map[string]schemaProperty{
							"adversary_id":    {Type: "string", Description: "adversary identifier"},
							"experience_name": {Type: "string", Description: "experience name"},
							"description":     {Type: "string", Description: "optional move description"},
						},
					},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.GmMoveApply),
		},
		{
			Tool: orchestration.Tool{
				Name:        "daggerheart_adversary_create",
				Description: "Creates one Daggerheart adversary on the current session scene",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id":           {Type: "string", Description: "optional explicit scene identifier; defaults to the active scene"},
					"adversary_entry_id": {Type: "string", Description: "adversary entry identifier to instantiate"},
					"notes":              {Type: "string", Description: "optional scene-specific notes for the created adversary"},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.AdversaryCreate),
		},
		{
			Tool: orchestration.Tool{
				Name:        "daggerheart_scene_countdown_create",
				Description: "Creates one Daggerheart scene countdown on the current session scene; prefer fixed_starting_value for normal board-control turns",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id":             {Type: "string", Description: "optional explicit scene identifier; defaults to the active scene"},
					"countdown_id":         {Type: "string", Description: "optional countdown identifier; omit to let the system generate one"},
					"name":                 {Type: "string", Description: "countdown name shown to the table"},
					"tone":                 {Type: "string", Description: "countdown tone: NEUTRAL, PROGRESS, or CONSEQUENCE"},
					"advancement_policy":   {Type: "string", Description: "countdown advancement policy: MANUAL, ACTION_STANDARD, ACTION_DYNAMIC, or LONG_REST"},
					"fixed_starting_value": {Type: "integer", Description: "fixed starting value; omit if using randomized_start"},
					"randomized_start":     {Type: "object", Description: "optional randomized starting range", Properties: map[string]schemaProperty{"min": {Type: "integer", Description: "minimum starting value"}, "max": {Type: "integer", Description: "maximum starting value"}, "seed": {Type: "integer", Description: "optional deterministic seed"}}},
					"loop_behavior":        {Type: "string", Description: "loop behavior: NONE, RESET, RESET_INCREASE_START, or RESET_DECREASE_START"},
					"linked_countdown_id":  {Type: "string", Description: "optional linked countdown identifier for progress/consequence pairs"},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.CountdownCreate),
		},
		{
			Tool: orchestration.Tool{
				Name:        "daggerheart_scene_countdown_advance",
				Description: "Advances one Daggerheart scene countdown by a positive amount",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id":     {Type: "string", Description: "optional explicit scene identifier; defaults to the active scene"},
					"countdown_id": {Type: "string", Description: "countdown identifier to advance"},
					"amount":       {Type: "integer", Description: "positive amount to advance"},
					"reason":       {Type: "string", Description: "optional short reason for the countdown advance"},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.CountdownAdvance),
		},
		{
			Tool: orchestration.Tool{
				Name:        "daggerheart_scene_countdown_resolve_trigger",
				Description: "Resolves one pending Daggerheart scene countdown trigger and applies its loop behavior",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id":     {Type: "string", Description: "optional explicit scene identifier; defaults to the active scene"},
					"countdown_id": {Type: "string", Description: "countdown identifier whose pending trigger should be resolved"},
					"reason":       {Type: "string", Description: "optional short reason for resolving the trigger"},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.CountdownResolveTrigger),
		},
		{
			Tool: orchestration.Tool{
				Name:        "daggerheart_adversary_update",
				Description: "Updates one Daggerheart adversary on the current scene board",
				InputSchema: schemaObject(map[string]schemaProperty{
					"adversary_id": {Type: "string", Description: "adversary identifier to update"},
					"scene_id":     {Type: "string", Description: "optional explicit scene identifier; defaults to the active scene"},
					"notes":        {Type: "string", Description: "scene-specific notes for the adversary; provide an empty string to clear them"},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.AdversaryUpdate),
		},
		{
			Tool: orchestration.Tool{
				Name:        "daggerheart_attack_flow_resolve",
				Description: "Runs an authoritative Daggerheart attack flow from action roll through damage application, inferring the acting character's default attack profile when possible",
				InputSchema: schemaObject(map[string]schemaProperty{
					"character_id":              {Type: "string", Description: "attacking character identifier"},
					"difficulty":                {Type: "integer", Description: "difficulty target for the attack roll"},
					"modifiers":                 {Type: "array", Description: "optional action-roll modifiers", Items: &schemaProperty{Type: "object", Properties: map[string]schemaProperty{"source": {Type: "string", Description: "modifier source label"}, "value": {Type: "integer", Description: "signed modifier value"}}}},
					"underwater":                {Type: "boolean", Description: "whether the attack is underwater"},
					"breath_scene_countdown_id": {Type: "string", Description: "optional scene breath countdown to advance for underwater attacks"},
					"target_id":                 {Type: "string", Description: "primary target identifier; omit only when the active scene board has exactly one visible adversary"},
					"damage":                    {Type: "object", Description: "optional damage application details; omitted values default from the inferred attack profile when available", Properties: map[string]schemaProperty{"damage_type": {Type: "string", Description: "damage type: physical, magic, or mixed"}, "resist_physical": {Type: "boolean", Description: "whether the target resists physical damage"}, "resist_magic": {Type: "boolean", Description: "whether the target resists magic damage"}, "immune_physical": {Type: "boolean", Description: "whether the target is immune to physical damage"}, "immune_magic": {Type: "boolean", Description: "whether the target is immune to magic damage"}, "direct": {Type: "boolean", Description: "whether the damage bypasses thresholds"}, "massive_damage": {Type: "boolean", Description: "whether the hit counts as massive damage"}, "source": {Type: "string", Description: "short source label"}, "source_character_ids": {Type: "array", Description: "optional source character identifiers", Items: &schemaProperty{Type: "string"}}}},
					"require_damage_roll":       {Type: "boolean", Description: "whether damage application should require the recorded damage roll; defaults to true"},
					"action_rng":                rngSchemaProperty("optional rng configuration for the action roll"),
					"damage_rng":                rngSchemaProperty("optional rng configuration for the damage roll"),
					"scene_id":                  {Type: "string", Description: "optional explicit scene identifier; defaults to the active scene"},
					"replace_hope_with_armor":   {Type: "boolean", Description: "whether eligible armor may replace a Hope spend"},
					"target_is_adversary":       {Type: "boolean", Description: "set true when the target is an adversary rather than a character"},
					"nearby_adversary_ids":      {Type: "array", Description: "optional nearby adversary identifiers for attacks with splash-style follow-through", Items: &schemaProperty{Type: "string"}},
					"standard_attack": {Type: "object", Description: "optional explicit standard attack profile; omit to infer the acting character's default primary-weapon attack", Properties: map[string]schemaProperty{
						"trait":           {Type: "string", Description: "attack trait, such as strength or finesse"},
						"damage_dice":     {Type: "array", Description: "damage dice to roll on a hit", Items: &schemaProperty{Type: "object", Properties: map[string]schemaProperty{"sides": {Type: "integer", Description: "die sides"}, "count": {Type: "integer", Description: "dice count"}}}},
						"damage_modifier": {Type: "integer", Description: "flat damage modifier"},
						"attack_range":    {Type: "string", Description: "attack range: melee or ranged"},
						"damage_critical": {Type: "boolean", Description: "whether the attack profile adds critical damage support"},
					}},
					"beastform_attack": {Type: "object", Description: "optional explicit request to use the acting character's active beastform attack profile", Properties: map[string]schemaProperty{}},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.AttackFlowResolve),
		},
		{
			Tool: orchestration.Tool{
				Name:        "daggerheart_adversary_attack_flow_resolve",
				Description: "Runs an authoritative Daggerheart adversary attack flow from attack roll through damage application",
				InputSchema: schemaObject(map[string]schemaProperty{
					"adversary_id":        {Type: "string", Description: "attacking adversary identifier"},
					"target_id":           {Type: "string", Description: "primary target character identifier"},
					"target_ids":          {Type: "array", Description: "optional additional target character identifiers", Items: &schemaProperty{Type: "string"}},
					"difficulty":          {Type: "integer", Description: "base difficulty before runtime adjustments"},
					"advantage":           {Type: "integer", Description: "optional advantage count"},
					"disadvantage":        {Type: "integer", Description: "optional disadvantage count"},
					"damage":              {Type: "object", Description: "damage application details", Properties: map[string]schemaProperty{"damage_type": {Type: "string", Description: "damage type: physical, magic, or mixed"}, "resist_physical": {Type: "boolean", Description: "whether the target resists physical damage"}, "resist_magic": {Type: "boolean", Description: "whether the target resists magic damage"}, "immune_physical": {Type: "boolean", Description: "whether the target is immune to physical damage"}, "immune_magic": {Type: "boolean", Description: "whether the target is immune to magic damage"}, "direct": {Type: "boolean", Description: "whether the damage bypasses thresholds"}, "massive_damage": {Type: "boolean", Description: "whether the hit counts as massive damage"}, "source": {Type: "string", Description: "short source label"}, "source_character_ids": {Type: "array", Description: "optional source character identifiers", Items: &schemaProperty{Type: "string"}}}},
					"require_damage_roll": {Type: "boolean", Description: "whether damage application should require the recorded damage roll; defaults to true"},
					"damage_critical":     {Type: "boolean", Description: "whether to apply critical damage handling if the attack hits critically"},
					"attack_rng":          rngSchemaProperty("optional rng configuration for the adversary attack roll"),
					"damage_rng":          rngSchemaProperty("optional rng configuration for the damage roll"),
					"scene_id":            {Type: "string", Description: "optional explicit scene identifier; defaults to the active scene"},
					"target_armor_reaction": {Type: "object", Description: "optional armor reaction used by the primary target", Properties: map[string]schemaProperty{
						"shifting": {Type: "object", Description: "spend armor for shifting reaction", Properties: map[string]schemaProperty{}},
						"timeslowing": {Type: "object", Description: "spend armor for timeslowing reaction", Properties: map[string]schemaProperty{
							"rng": rngSchemaProperty("optional rng configuration for the timeslowing bonus die"),
						}},
					}},
					"feature_id":                {Type: "string", Description: "optional adversary feature identifier driving the attack"},
					"contributor_adversary_ids": {Type: "array", Description: "optional related adversary identifiers contributing to the attack", Items: &schemaProperty{Type: "string"}},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.AdversaryAttackFlowResolve),
		},
		{
			Tool: orchestration.Tool{
				Name:        "daggerheart_group_action_flow_resolve",
				Description: "Runs an authoritative Daggerheart group action flow with supporter rolls and the final leader roll",
				InputSchema: schemaObject(map[string]schemaProperty{
					"leader_character_id": {Type: "string", Description: "leader character identifier"},
					"leader_trait":        {Type: "string", Description: "leader trait for the final action roll"},
					"difficulty":          {Type: "integer", Description: "difficulty target for supporters and leader"},
					"leader_modifiers":    {Type: "array", Description: "optional leader roll modifiers", Items: &schemaProperty{Type: "object", Properties: map[string]schemaProperty{"source": {Type: "string", Description: "modifier source label"}, "value": {Type: "integer", Description: "signed modifier value"}}}},
					"supporters": {Type: "array", Description: "supporting characters contributing reaction rolls", Items: &schemaProperty{Type: "object", Properties: map[string]schemaProperty{
						"character_id": {Type: "string", Description: "supporter character identifier"},
						"trait":        {Type: "string", Description: "supporter trait"},
						"modifiers":    {Type: "array", Description: "optional supporter modifiers", Items: &schemaProperty{Type: "object", Properties: map[string]schemaProperty{"source": {Type: "string", Description: "modifier source label"}, "value": {Type: "integer", Description: "signed modifier value"}}}},
						"rng":          rngSchemaProperty("optional rng configuration for the supporter roll"),
						"context":      {Type: "string", Description: "optional narrow supporter rules context, for example move_silently"},
					}}},
					"leader_rng":     rngSchemaProperty("optional rng configuration for the leader roll"),
					"scene_id":       {Type: "string", Description: "optional explicit scene identifier; defaults to the active scene"},
					"leader_context": {Type: "string", Description: "optional narrow leader rules context, for example move_silently"},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.GroupActionFlowResolve),
		},
		{
			Tool: orchestration.Tool{
				Name:        "daggerheart_reaction_flow_resolve",
				Description: "Runs an authoritative Daggerheart reaction flow from reaction roll through reaction outcome",
				InputSchema: schemaObject(map[string]schemaProperty{
					"character_id":            {Type: "string", Description: "reacting character identifier"},
					"trait":                   {Type: "string", Description: "trait used for the reaction roll"},
					"difficulty":              {Type: "integer", Description: "difficulty target for the reaction"},
					"modifiers":               {Type: "array", Description: "optional reaction-roll modifiers", Items: &schemaProperty{Type: "object", Properties: map[string]schemaProperty{"source": {Type: "string", Description: "modifier source label"}, "value": {Type: "integer", Description: "signed modifier value"}}}},
					"reaction_rng":            rngSchemaProperty("optional rng configuration for the reaction roll"),
					"advantage":               {Type: "integer", Description: "optional advantage count"},
					"disadvantage":            {Type: "integer", Description: "optional disadvantage count"},
					"scene_id":                {Type: "string", Description: "optional explicit scene identifier; defaults to the active scene"},
					"replace_hope_with_armor": {Type: "boolean", Description: "whether eligible armor may replace a Hope spend"},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.ReactionFlowResolve),
		},
		{
			Tool: orchestration.Tool{
				Name:        "daggerheart_tag_team_flow_resolve",
				Description: "Runs an authoritative Daggerheart tag-team flow with both rolls and the selected combined outcome",
				InputSchema: schemaObject(map[string]schemaProperty{
					"first": {Type: "object", Description: "first tag-team participant", Properties: map[string]schemaProperty{
						"character_id": {Type: "string", Description: "first participant character identifier"},
						"trait":        {Type: "string", Description: "first participant trait"},
						"modifiers":    {Type: "array", Description: "optional first participant modifiers", Items: &schemaProperty{Type: "object", Properties: map[string]schemaProperty{"source": {Type: "string", Description: "modifier source label"}, "value": {Type: "integer", Description: "signed modifier value"}}}},
						"rng":          rngSchemaProperty("optional rng configuration for the first roll"),
					}},
					"second": {Type: "object", Description: "second tag-team participant", Properties: map[string]schemaProperty{
						"character_id": {Type: "string", Description: "second participant character identifier"},
						"trait":        {Type: "string", Description: "second participant trait"},
						"modifiers":    {Type: "array", Description: "optional second participant modifiers", Items: &schemaProperty{Type: "object", Properties: map[string]schemaProperty{"source": {Type: "string", Description: "modifier source label"}, "value": {Type: "integer", Description: "signed modifier value"}}}},
						"rng":          rngSchemaProperty("optional rng configuration for the second roll"),
					}},
					"difficulty":            {Type: "integer", Description: "difficulty target shared by both participants"},
					"selected_character_id": {Type: "string", Description: "participant whose roll should determine the final combined outcome"},
					"scene_id":              {Type: "string", Description: "optional explicit scene identifier; defaults to the active scene"},
				}),
			},
			Execute: wrapDaggerheartExecutor(daggerhearttools.TagTeamFlowResolve),
		},
	}
}
