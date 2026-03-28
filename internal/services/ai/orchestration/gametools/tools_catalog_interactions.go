package gametools

import "github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"

func interactionToolDefinitions() []productionToolDefinition {
	return []productionToolDefinition{
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
		{
			Tool: orchestration.Tool{
				Name:        "interaction_conclude_session",
				Description: "Commits the final session-closing GM interaction, stores the structured recap, ends all open scenes, ends the session, and optionally completes the campaign",
				InputSchema: schemaObject(map[string]schemaProperty{
					"conclusion":   {Type: "string", Description: "final fiction beats wrapping the session's story"},
					"summary":      {Type: "string", Description: "session recap markdown with the required headings: Key Events, NPCs Met, Decisions Made, Unresolved Threads, Next Session Hooks"},
					"end_campaign": {Type: "boolean", Description: "set true only when this session also ends the campaign"},
					"epilogue":     {Type: "string", Description: "required when end_campaign is true; otherwise leave empty"},
				}),
			},
			Execute: (*DirectSession).interactionConcludeSession,
		},
	}
}
