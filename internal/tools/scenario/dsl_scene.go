package scenario

import (
	"github.com/Shopify/go-lua"
)

func scenarioCreateScene(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "create_scene", data)
	return 0
}

func scenarioEndScene(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "end_scene", data)
	return 0
}

func scenarioSceneAddCharacter(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "scene_add_character", data)
	return 0
}

func scenarioSceneRemoveCharacter(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "scene_remove_character", data)
	return 0
}

func scenarioSceneTransferCharacter(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "scene_transfer_character", data)
	return 0
}

func scenarioSceneTransition(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "scene_transition", data)
	return 0
}

func scenarioSceneGateOpen(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "scene_gate_open", data)
	return 0
}

func scenarioSceneGateResolve(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "scene_gate_resolve", data)
	return 0
}

func scenarioSceneGateAbandon(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "scene_gate_abandon", data)
	return 0
}

func scenarioSceneSetSpotlight(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "scene_set_spotlight", data)
	return 0
}

func scenarioSceneClearSpotlight(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "scene_clear_spotlight", data)
	return 0
}

func scenarioUpdateScene(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "update_scene", data)
	return 0
}
