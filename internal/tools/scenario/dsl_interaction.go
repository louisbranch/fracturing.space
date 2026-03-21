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
	return interactionStep(state, "interaction_set_gm_authority")
}

func scenarioInteractionSetActiveScene(state *lua.State) int {
	return interactionStep(state, "interaction_set_active_scene")
}

func scenarioInteractionStartPlayerPhase(state *lua.State) int {
	return interactionStep(state, "interaction_start_player_phase")
}

func scenarioInteractionPost(state *lua.State) int {
	return interactionStep(state, "interaction_post")
}

func scenarioInteractionYield(state *lua.State) int {
	return interactionStep(state, "interaction_yield")
}

func scenarioInteractionUnyield(state *lua.State) int {
	return interactionStep(state, "interaction_unyield")
}

func scenarioInteractionEndPlayerPhase(state *lua.State) int {
	return interactionStep(state, "interaction_end_player_phase")
}

func scenarioInteractionResolveReview(state *lua.State) int {
	return interactionStep(state, "interaction_resolve_review")
}

func scenarioInteractionPauseOOC(state *lua.State) int {
	return interactionStep(state, "interaction_pause_ooc")
}

func scenarioInteractionPostOOC(state *lua.State) int {
	return interactionStep(state, "interaction_post_ooc")
}

func scenarioInteractionReadyOOC(state *lua.State) int {
	return interactionStep(state, "interaction_ready_ooc")
}

func scenarioInteractionClearReadyOOC(state *lua.State) int {
	return interactionStep(state, "interaction_clear_ready_ooc")
}

func scenarioInteractionResumeOOC(state *lua.State) int {
	return interactionStep(state, "interaction_resume_ooc")
}

func scenarioInteractionResolveInterruptedPhase(state *lua.State) int {
	return interactionStep(state, "interaction_resolve_interrupted_phase")
}

func scenarioInteractionExpect(state *lua.State) int {
	return interactionStep(state, "interaction_expect")
}
