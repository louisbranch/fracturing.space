package gametools

import (
	"context"
	"fmt"
	"slices"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

type toolExecutor func(*DirectSession, context.Context, []byte) (orchestration.ToolResult, error)

type productionToolDefinition struct {
	Tool    orchestration.Tool
	Execute toolExecutor
}

type productionToolRegistry struct {
	definitions []productionToolDefinition
	byName      map[string]productionToolDefinition
}

// defaultRegistry is the standard production registry used by NewDirectDialer.
// Kept as a package-level cache since the definitions are static; injection
// happens at the DirectDialer/DirectSession level so tests can substitute.
var defaultRegistry = newProductionToolRegistry()

func newProductionToolRegistry() productionToolRegistry {
	definitions := []productionToolDefinition{
		// Artifact tools
		{
			Tool: orchestration.Tool{
				Name:        "campaign_artifact_list",
				Description: "Lists AI GM campaign artifacts such as skills.md, story.md, memory.md, and working notes",
				InputSchema: schemaObject(nil),
			},
			Execute: (*DirectSession).artifactList,
		},
		{
			Tool: orchestration.Tool{
				Name:        "campaign_artifact_get",
				Description: "Reads one AI GM campaign artifact such as story.md, memory.md, or a working note",
				InputSchema: schemaObject(map[string]schemaProperty{
					"path": {Type: "string", Description: "artifact path such as story.md, memory.md, or working/notes.md"},
				}),
			},
			Execute: (*DirectSession).artifactGet,
		},
		{
			Tool: orchestration.Tool{
				Name:        "campaign_artifact_upsert",
				Description: "Writes one mutable AI GM campaign artifact such as story.md, memory.md, or a working note",
				InputSchema: schemaObject(map[string]schemaProperty{
					"path":    {Type: "string", Description: "artifact path such as story.md, memory.md, or working/notes.md"},
					"content": {Type: "string", Description: "full replacement markdown content"},
				}),
			},
			Execute: (*DirectSession).artifactUpsert,
		},

		// Memory section tools
		{
			Tool: orchestration.Tool{
				Name:        "campaign_memory_section_read",
				Description: "Reads one ## heading section from memory.md without fetching the full document",
				InputSchema: schemaObject(map[string]schemaProperty{
					"heading": {Type: "string", Description: "section heading to read, e.g. NPCs, Plot Hooks, World State"},
				}),
			},
			Execute: (*DirectSession).memorySectionRead,
		},
		{
			Tool: orchestration.Tool{
				Name:        "campaign_memory_section_update",
				Description: "Replaces or appends one ## heading section in memory.md without disturbing other sections",
				InputSchema: schemaObject(map[string]schemaProperty{
					"heading": {Type: "string", Description: "section heading to update or create, e.g. NPCs, Plot Hooks, World State"},
					"content": {Type: "string", Description: "new section body content (replaces existing body for this heading)"},
				}),
			},
			Execute: (*DirectSession).memorySectionUpdate,
		},

		// Scene tools
		{
			Tool: orchestration.Tool{
				Name:        "scene_create",
				Description: "Creates a new scene within the current session and activates it by default unless activate is set to false",
				InputSchema: schemaObject(map[string]schemaProperty{
					"name":          {Type: "string", Description: "scene title"},
					"description":   {Type: "string", Description: "scene framing description"},
					"character_ids": {Type: "array", Description: "optional starting character identifiers", Items: &schemaProperty{Type: "string"}},
					"activate":      {Type: "boolean", Description: "optional explicit activation flag; defaults to true"},
				}),
			},
			Execute: (*DirectSession).sceneCreate,
		},
		{
			Tool: orchestration.Tool{
				Name:        "scene_update",
				Description: "Updates scene metadata (name or description) for an existing scene",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id":    {Type: "string", Description: "scene identifier"},
					"name":        {Type: "string", Description: "updated scene title"},
					"description": {Type: "string", Description: "updated scene framing description"},
				}),
			},
			Execute: (*DirectSession).sceneUpdate,
		},
		{
			Tool: orchestration.Tool{
				Name:        "scene_end",
				Description: "Ends a scene, marking it as no longer active",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id": {Type: "string", Description: "scene identifier"},
					"reason":   {Type: "string", Description: "optional reason for ending the scene"},
				}),
			},
			Execute: (*DirectSession).sceneEnd,
		},
		{
			Tool: orchestration.Tool{
				Name:        "scene_transition",
				Description: "Atomically ends the current scene and creates a new one, carrying characters forward",
				InputSchema: schemaObject(map[string]schemaProperty{
					"source_scene_id": {Type: "string", Description: "scene to end (defaults to active scene)"},
					"name":            {Type: "string", Description: "new scene title"},
					"description":     {Type: "string", Description: "new scene framing description"},
				}),
			},
			Execute: (*DirectSession).sceneTransition,
		},
		{
			Tool: orchestration.Tool{
				Name:        "scene_add_character",
				Description: "Adds a character to a scene",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id":     {Type: "string", Description: "scene identifier"},
					"character_id": {Type: "string", Description: "character identifier to add"},
				}),
			},
			Execute: (*DirectSession).sceneAddCharacter,
		},
		{
			Tool: orchestration.Tool{
				Name:        "scene_remove_character",
				Description: "Removes a character from a scene",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id":     {Type: "string", Description: "scene identifier"},
					"character_id": {Type: "string", Description: "character identifier to remove"},
				}),
			},
			Execute: (*DirectSession).sceneRemoveCharacter,
		},

		// Interaction tools
		{
			Tool: orchestration.Tool{
				Name:        "interaction_state_read",
				Description: "Reads the current interaction state so the GM can diagnose the active scene, review status, acting characters, and OOC/session state before correcting an issue",
				InputSchema: schemaObject(nil),
			},
			Execute: (*DirectSession).interactionStateRead,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_activate_scene",
				Description: "Sets the authoritative active scene for the current session",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id": {Type: "string", Description: "scene identifier"},
				}),
			},
			Execute: (*DirectSession).interactionActivateScene,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_open_scene_player_phase",
				Description: "Commits one structured GM interaction and opens a new player phase on the active scene; use this when players should act next",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id":      {Type: "string", Description: "scene identifier (defaults to active scene)"},
					"interaction":   interactionSchemaProperty("structured GM interaction that opens the player phase; usually fiction first and a final prompt beat for the acting characters"),
					"character_ids": {Type: "array", Description: "acting character identifiers", Items: &schemaProperty{Type: "string"}},
				}),
			},
			Execute: (*DirectSession).interactionOpenScenePlayerPhase,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_resolve_scene_player_review",
				Description: "Resolves the active scene GM review by either opening the next player phase or requesting revisions",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id": {Type: "string", Description: "scene identifier (defaults to active scene)"},
					"open_next_player_phase": {
						Type:        "object",
						Description: "commit a GM interaction and open the next player phase",
						Properties: map[string]schemaProperty{
							"interaction":        interactionSchemaProperty("structured GM interaction committed before the next player phase; usually consequence before a final prompt beat"),
							"next_character_ids": {Type: "array", Description: "acting character identifiers for the next player phase", Items: &schemaProperty{Type: "string"}},
						},
					},
					"request_revisions": {
						Type:        "object",
						Description: "commit a GM interaction and request participant-scoped revisions",
						Properties: map[string]schemaProperty{
							"interaction": interactionSchemaProperty("structured GM interaction shown with the revision request; use guidance beats to explain what must change"),
							"revisions": {
								Type:        "array",
								Description: "participant-scoped revision requests",
								Items: &schemaProperty{
									Type: "object",
									Properties: map[string]schemaProperty{
										"participant_id": {Type: "string", Description: "participant identifier that must revise their slot"},
										"reason":         {Type: "string", Description: "GM review reason shown to the participant"},
										"character_ids":  {Type: "array", Description: "optional character identifiers affected by the review request", Items: &schemaProperty{Type: "string"}},
									},
								},
							},
						},
					},
					"return_to_gm": {
						Type:        "object",
						Description: "commit a GM interaction and return the scene to GM control with no open player phase",
						Properties: map[string]schemaProperty{
							"interaction": interactionSchemaProperty("structured GM interaction committed before returning control to the GM; omit a prompt beat if no player handoff follows"),
						},
					},
				}),
			},
			Execute: (*DirectSession).interactionResolveScenePlayerReview,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_record_scene_gm_interaction",
				Description: "Commits one authoritative beat-based GM interaction for the active scene without opening a player phase",
				InputSchema: schemaObject(map[string]schemaProperty{
					"scene_id":    {Type: "string", Description: "scene identifier (defaults to active scene)"},
					"interaction": interactionSchemaProperty("structured GM interaction to commit on the active scene"),
				}),
			},
			Execute: (*DirectSession).interactionRecordSceneGMInteraction,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_open_session_ooc",
				Description: "Opens the session-level out-of-character pause overlay",
				InputSchema: schemaObject(map[string]schemaProperty{
					"reason": {Type: "string", Description: "optional OOC pause reason"},
				}),
			},
			Execute: (*DirectSession).interactionPauseOOC,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_session_ooc_resolve",
				Description: "Resolves the current OOC pause by resuming the interrupted phase, returning control to the GM, or replacing it with a newly opened player phase",
				InputSchema: schemaObject(map[string]schemaProperty{
					"resume_interrupted_phase": {Type: "boolean", Description: "set true to restore the interrupted phase for players"},
					"return_to_gm": {
						Type:        "object",
						Description: "return to GM control, optionally on a different scene in the active session",
						Properties: map[string]schemaProperty{
							"scene_id": {Type: "string", Description: "target scene identifier; defaults to the interrupted scene"},
						},
					},
					"open_player_phase": {
						Type:        "object",
						Description: "replace the interrupted phase with a new GM interaction and acting set",
						Properties: map[string]schemaProperty{
							"scene_id":      {Type: "string", Description: "target scene identifier; defaults to the interrupted scene"},
							"interaction":   interactionSchemaProperty("structured GM interaction committed for the replacement player phase; re-anchor the fiction and end with a prompt beat"),
							"character_ids": {Type: "array", Description: "acting character identifiers for the replacement phase", Items: &schemaProperty{Type: "string"}},
						},
					},
				}),
			},
			Execute: (*DirectSession).interactionResolveSessionOOC,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_post_session_ooc",
				Description: "Posts one append-only out-of-character transcript message",
				InputSchema: schemaObject(map[string]schemaProperty{
					"body": {Type: "string", Description: "out-of-character message body"},
				}),
			},
			Execute: (*DirectSession).interactionPostOOC,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_mark_ooc_ready_to_resume",
				Description: "Marks the caller as ready to resume from the current OOC pause",
				InputSchema: schemaObject(nil),
			},
			Execute: (*DirectSession).interactionMarkOOCReady,
		},
		{
			Tool: orchestration.Tool{
				Name:        "interaction_clear_ooc_ready_to_resume",
				Description: "Clears the caller's ready-to-resume state for the current OOC pause",
				InputSchema: schemaObject(nil),
			},
			Execute: (*DirectSession).interactionClearOOCReady,
		},

		// Daggerheart read tools
		{
			Tool: orchestration.Tool{
				Name:        "character_sheet_read",
				Description: "Reads one authoritative character sheet with traits, equipment, domain cards, active features, Hope, and current state",
				InputSchema: schemaObject(map[string]schemaProperty{
					"character_id": {Type: "string", Description: "character identifier to inspect"},
				}),
			},
			Execute: (*DirectSession).characterSheetRead,
		},
		{
			Tool: orchestration.Tool{
				Name:        "daggerheart_combat_board_read",
				Description: "Reads the current Daggerheart combat board for the bound session, including GM Fear, active-scene readiness diagnostics, spotlight, visible countdowns, and active adversaries",
				InputSchema: schemaObject(nil),
			},
			Execute: (*DirectSession).daggerheartCombatBoardRead,
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
			Execute: (*DirectSession).daggerheartActionRollResolve,
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
			Execute: (*DirectSession).daggerheartGmMoveApply,
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
			Execute: (*DirectSession).daggerheartAdversaryCreate,
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
			Execute: (*DirectSession).daggerheartCountdownCreate,
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
			Execute: (*DirectSession).daggerheartCountdownUpdate,
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
			Execute: (*DirectSession).daggerheartCountdownResolveTrigger,
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
			Execute: (*DirectSession).daggerheartAdversaryUpdate,
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
			Execute: (*DirectSession).daggerheartAttackFlowResolve,
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
			Execute: (*DirectSession).daggerheartAdversaryAttackFlowResolve,
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
			Execute: (*DirectSession).daggerheartGroupActionFlowResolve,
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
			Execute: (*DirectSession).daggerheartReactionFlowResolve,
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
			Execute: (*DirectSession).daggerheartTagTeamFlowResolve,
		},

		// Daggerheart tools
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
			Execute: (*DirectSession).dualityActionRoll,
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
			Execute: (*DirectSession).rollDice,
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
			Execute: (*DirectSession).dualityOutcome,
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
			Execute: (*DirectSession).dualityExplain,
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
			Execute: (*DirectSession).dualityProbability,
		},
		{
			Tool: orchestration.Tool{
				Name:        "duality_rules_version",
				Description: "Describes the Duality ruleset semantics",
				InputSchema: schemaObject(nil),
			},
			Execute: (*DirectSession).dualityRulesVersion,
		},

		// Reference tools
		{
			Tool: orchestration.Tool{
				Name:        "system_reference_search",
				Description: "Searches the configured read-only game-system reference corpus when exact wording or procedure choice is unclear",
				InputSchema: schemaObject(map[string]schemaProperty{
					"system":      {Type: "string", Description: "system identifier; defaults to daggerheart"},
					"query":       {Type: "string", Description: "search query"},
					"max_results": {Type: "integer", Description: "optional max result count"},
				}),
			},
			Execute: (*DirectSession).referenceSearch,
		},
		{
			Tool: orchestration.Tool{
				Name:        "system_reference_read",
				Description: "Reads one full document from the configured read-only game-system reference corpus after a search result still needs exact wording",
				InputSchema: schemaObject(map[string]schemaProperty{
					"system":      {Type: "string", Description: "system identifier; defaults to daggerheart"},
					"document_id": {Type: "string", Description: "document identifier from search results"},
				}),
			},
			Execute: (*DirectSession).referenceRead,
		},
	}

	byName := make(map[string]productionToolDefinition, len(definitions))
	for i, definition := range definitions {
		name := definition.Tool.Name
		if name == "" {
			panic("gametools: production tool name is required")
		}
		if definition.Execute == nil {
			panic(fmt.Sprintf("gametools: production tool %q is missing an executor", name))
		}
		definitions[i] = definition
		if _, exists := byName[name]; exists {
			panic(fmt.Sprintf("gametools: duplicate production tool %q", name))
		}
		byName[name] = definition
	}

	return productionToolRegistry{
		definitions: definitions,
		byName:      byName,
	}
}

func interactionSchemaProperty(description string) schemaProperty {
	return schemaProperty{
		Type:        "object",
		Description: description,
		Properties: map[string]schemaProperty{
			"title":         {Type: "string", Description: "short interaction title"},
			"character_ids": {Type: "array", Description: "characters addressed by the interaction", Items: &schemaProperty{Type: "string"}},
			"beats": {
				Type:        "array",
				Description: "ordered beats that make up the GM interaction; keep related prose in one beat even across paragraphs, and start a new beat only when the function or information context materially changes; end with a prompt beat when players should act next",
				Items: &schemaProperty{
					Type: "object",
					Properties: map[string]schemaProperty{
						"beat_id": {Type: "string", Description: "optional stable beat identifier"},
						"type":    {Type: "string", Description: "beat type: fiction, prompt, resolution, consequence, or guidance"},
						"text":    {Type: "string", Description: "beat body text; may span multiple paragraphs when it serves one coherent beat"},
					},
				},
			},
		},
	}
}

func (r productionToolRegistry) tools() []orchestration.Tool {
	tools := make([]orchestration.Tool, 0, len(r.definitions))
	for _, definition := range r.definitions {
		tools = append(tools, definition.Tool)
	}
	return tools
}

func (r productionToolRegistry) lookup(name string) (productionToolDefinition, bool) {
	definition, ok := r.byName[name]
	return definition, ok
}

// ProductionToolNames returns the concrete production tool profile owned by
// the direct game-tools bridge.
func ProductionToolNames() []string {
	names := make([]string, 0, len(defaultRegistry.definitions))
	for _, definition := range defaultRegistry.definitions {
		names = append(names, definition.Tool.Name)
	}
	return names
}

// schemaProperty is a minimal JSON Schema property definition.
type schemaProperty struct {
	Type        string                    `json:"type"`
	Description string                    `json:"description,omitempty"`
	Items       *schemaProperty           `json:"items,omitempty"`
	Properties  map[string]schemaProperty `json:"properties,omitempty"`
}

// schemaObject builds a JSON-Schema-like map for an object type.
// All properties are marked required for OpenAI strict mode compatibility;
// optional fields accept empty strings and are defaulted server-side.
func schemaObject(properties map[string]schemaProperty, _ ...string) map[string]any {
	schema := map[string]any{
		"type": "object",
	}
	if len(properties) > 0 {
		schema["properties"] = properties
		required := make([]string, 0, len(properties))
		for name := range properties {
			required = append(required, name)
		}
		slices.Sort(required)
		schema["required"] = required
	}
	return schema
}

func rngSchemaProperty(description string) schemaProperty {
	return schemaProperty{
		Type:        "object",
		Description: description,
		Properties: map[string]schemaProperty{
			"seed":      {Type: "integer", Description: "optional seed for deterministic rolls"},
			"roll_mode": {Type: "string", Description: "roll mode (LIVE or REPLAY)"},
		},
	}
}
