package mcpbridge

import "strings"

var productionToolNames = map[string]struct{}{
	"campaign_artifact_list":                     {},
	"campaign_artifact_get":                      {},
	"campaign_artifact_upsert":                   {},
	"scene_create":                               {},
	"interaction_active_scene_set":               {},
	"interaction_scene_player_phase_start":       {},
	"interaction_scene_player_phase_accept":      {},
	"interaction_scene_player_revisions_request": {},
	"interaction_scene_player_phase_end":         {},
	"interaction_scene_gm_output_commit":         {},
	"interaction_ooc_pause":                      {},
	"interaction_ooc_post":                       {},
	"interaction_ooc_ready_mark":                 {},
	"interaction_ooc_ready_clear":                {},
	"interaction_ooc_resume":                     {},
	"duality_action_roll":                        {},
	"roll_dice":                                  {},
	"duality_outcome":                            {},
	"duality_explain":                            {},
	"duality_probability":                        {},
	"duality_rules_version":                      {},
	"system_reference_search":                    {},
	"system_reference_read":                      {},
}

// ProductionToolAllowed reports whether name belongs to the GM-safe MCP bridge profile.
func ProductionToolAllowed(name string) bool {
	_, ok := productionToolNames[strings.TrimSpace(name)]
	return ok
}
