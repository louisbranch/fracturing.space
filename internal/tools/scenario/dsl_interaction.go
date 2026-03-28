package scenario

import "github.com/Shopify/go-lua"

// interactionStep appends a root-level interaction step backed by one optional
// Lua table so scenario scripts can stay terse for GM/player beats.
func interactionStep(state *lua.State, kind string) int {
	scenario := checkScenario(state)
	appendStep(scenario, kind, optionalTable(state, 2))
	return 0
}

func scenarioInteractionSetGMAuthority(state *lua.State) int {
	return interactionStep(state, "interaction_set_session_gm_authority")
}

func scenarioInteractionActivateScene(state *lua.State) int {
	return interactionStep(state, "interaction_activate_scene")
}

func scenarioInteractionRecordGMInteraction(state *lua.State) int {
	return interactionStep(state, "interaction_record_scene_gm_interaction")
}

func scenarioInteractionStartPlayerPhase(state *lua.State) int {
	return interactionStep(state, "interaction_open_scene_player_phase")
}

func scenarioInteractionPost(state *lua.State) int {
	return interactionStep(state, "interaction_submit_scene_player_action")
}

func scenarioInteractionYield(state *lua.State) int {
	return interactionStep(state, "interaction_yield_scene_player_phase")
}

func scenarioInteractionUnyield(state *lua.State) int {
	return interactionStep(state, "interaction_withdraw_scene_player_yield")
}

func scenarioInteractionEndPlayerPhase(state *lua.State) int {
	return interactionStep(state, "interaction_interrupt_scene_player_phase")
}

func scenarioInteractionResolveReview(state *lua.State) int {
	return interactionStep(state, "interaction_resolve_scene_player_review")
}

func scenarioInteractionPauseOOC(state *lua.State) int {
	return interactionStep(state, "interaction_open_session_ooc")
}

func scenarioInteractionPostOOC(state *lua.State) int {
	return interactionStep(state, "interaction_post_session_ooc")
}

func scenarioInteractionReadyOOC(state *lua.State) int {
	return interactionStep(state, "interaction_mark_ooc_ready_to_resume")
}

func scenarioInteractionClearReadyOOC(state *lua.State) int {
	return interactionStep(state, "interaction_clear_ooc_ready_to_resume")
}

func scenarioInteractionResolveSessionOOC(state *lua.State) int {
	return interactionStep(state, "interaction_resolve_session_ooc")
}

func scenarioInteractionExpect(state *lua.State) int {
	return interactionStep(state, "interaction_expect")
}

func scenarioInteractionConcludeSession(state *lua.State) int {
	return interactionStep(state, "interaction_conclude_session")
}
