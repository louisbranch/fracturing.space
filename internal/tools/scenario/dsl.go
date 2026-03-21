package scenario

import (
	"math"
	"strings"
	"sync"

	"github.com/Shopify/go-lua"
)

const (
	scenarioTypeName    = "scenario"
	gmActionTypeName    = "gm_action"
	participantTypeName = "participant"
	systemTypeName      = "system_handle"
)

type gmAction struct {
	scenario  *Scenario
	stepIndex int
}

type participantHandle struct {
	scenario *Scenario
	name     string
}

type systemHandle struct {
	scenario *Scenario
	system   string
}

func registerLuaTypes(state *lua.State) {
	registerScenarioType(state)
	registerGMActionType(state)
	registerParticipantType(state)
	registerSystemType(state)
	registerScenarioConstructor(state)
	registerModifierHelpers(state)
}

func registerScenarioType(state *lua.State) {
	lua.NewMetaTable(state, scenarioTypeName)
	state.NewTable()
	lua.SetFunctions(state, scenarioTypeMethods(), 0)
	state.SetField(-2, "__index")
	state.Pop(1)
}

func registerGMActionType(state *lua.State) {
	lua.NewMetaTable(state, gmActionTypeName)
	state.NewTable()
	lua.SetFunctions(state, gmActionMethods, 0)
	state.SetField(-2, "__index")
	state.Pop(1)
}

func registerParticipantType(state *lua.State) {
	lua.NewMetaTable(state, participantTypeName)
	state.NewTable()
	lua.SetFunctions(state, participantMethods, 0)
	state.SetField(-2, "__index")
	state.Pop(1)
}

func registerSystemType(state *lua.State) {
	lua.NewMetaTable(state, systemTypeName)
	state.NewTable()
	lua.SetFunctions(state, registeredScenarioSystemMethods(), 0)
	state.SetField(-2, "__index")
	state.Pop(1)
}

func registerScenarioConstructor(state *lua.State) {
	state.NewTable()
	lua.SetFunctions(state, scenarioConstructor, 0)
	state.SetGlobal("Scenario")
}

func registerModifierHelpers(state *lua.State) {
	state.NewTable()
	lua.SetFunctions(state, modifierHelpers, 0)
	state.SetGlobal("Modifiers")
}

var scenarioConstructor = []lua.RegistryFunction{
	{Name: "new", Function: scenarioNew},
}

var modifierHelpers = []lua.RegistryFunction{
	{Name: "mod", Function: modifierHelper},
	{Name: "hope", Function: hopeSpendHelper},
}

func modifierHelper(state *lua.State) int {
	source := lua.CheckString(state, 1)
	value := lua.CheckInteger(state, 2)
	state.NewTable()
	state.PushString(source)
	state.SetField(-2, "source")
	state.PushInteger(value)
	state.SetField(-2, "value")
	return 1
}

func hopeSpendHelper(state *lua.State) int {
	source := lua.CheckString(state, 1)
	state.NewTable()
	state.PushString(source)
	state.SetField(-2, "source")
	return 1
}

func scenarioNew(state *lua.State) int {
	name := lua.OptString(state, 1, "")
	scenario := &Scenario{Name: name}
	state.PushUserData(scenario)
	lua.SetMetaTableNamed(state, scenarioTypeName)
	return 1
}

var scenarioMethods = []lua.RegistryFunction{
	{Name: "system", Function: scenarioSystem},
	{Name: "campaign", Function: scenarioCampaign},
	{Name: "participant", Function: scenarioParticipant},
	{Name: "start_session", Function: scenarioStartSession},
	{Name: "end_session", Function: scenarioEndSession},
	{Name: "pc", Function: scenarioPC},
	{Name: "npc", Function: scenarioNPC},
	{Name: "prefab", Function: scenarioPrefab},
	{Name: "set_spotlight", Function: scenarioSetSpotlight},
	{Name: "clear_spotlight", Function: scenarioClearSpotlight},
	{Name: "create_scene", Function: scenarioCreateScene},
	{Name: "end_scene", Function: scenarioEndScene},
	{Name: "scene_add_character", Function: scenarioSceneAddCharacter},
	{Name: "scene_remove_character", Function: scenarioSceneRemoveCharacter},
	{Name: "scene_transfer_character", Function: scenarioSceneTransferCharacter},
	{Name: "scene_transition", Function: scenarioSceneTransition},
	{Name: "scene_gate_open", Function: scenarioSceneGateOpen},
	{Name: "scene_gate_resolve", Function: scenarioSceneGateResolve},
	{Name: "scene_gate_abandon", Function: scenarioSceneGateAbandon},
	{Name: "scene_set_spotlight", Function: scenarioSceneSetSpotlight},
	{Name: "scene_clear_spotlight", Function: scenarioSceneClearSpotlight},
	{Name: "update_scene", Function: scenarioUpdateScene},
	{Name: "interaction_set_gm_authority", Function: scenarioInteractionSetGMAuthority},
	{Name: "interaction_set_active_scene", Function: scenarioInteractionSetActiveScene},
	{Name: "interaction_start_player_phase", Function: scenarioInteractionStartPlayerPhase},
	{Name: "interaction_post", Function: scenarioInteractionPost},
	{Name: "interaction_yield", Function: scenarioInteractionYield},
	{Name: "interaction_unyield", Function: scenarioInteractionUnyield},
	{Name: "interaction_end_player_phase", Function: scenarioInteractionEndPlayerPhase},
	{Name: "interaction_resolve_review", Function: scenarioInteractionResolveReview},
	{Name: "interaction_pause_ooc", Function: scenarioInteractionPauseOOC},
	{Name: "interaction_post_ooc", Function: scenarioInteractionPostOOC},
	{Name: "interaction_ready_ooc", Function: scenarioInteractionReadyOOC},
	{Name: "interaction_clear_ready_ooc", Function: scenarioInteractionClearReadyOOC},
	{Name: "interaction_resume_ooc", Function: scenarioInteractionResumeOOC},
	{Name: "interaction_resolve_interrupted_phase", Function: scenarioInteractionResolveInterruptedPhase},
	{Name: "interaction_expect", Function: scenarioInteractionExpect},
}

var (
	scenarioTypeMethodsOnce sync.Once
	scenarioTypeMethodList  []lua.RegistryFunction
)

func scenarioTypeMethods() []lua.RegistryFunction {
	scenarioTypeMethodsOnce.Do(func() {
		scenarioTypeMethodList = make([]lua.RegistryFunction, 0, len(scenarioMethods)+len(registeredScenarioSystemMethods()))
		scenarioTypeMethodList = append(scenarioTypeMethodList, scenarioMethods...)

		seen := make(map[string]struct{}, len(scenarioTypeMethodList))
		for _, method := range scenarioTypeMethodList {
			seen[method.Name] = struct{}{}
		}

		for _, method := range registeredScenarioSystemMethods() {
			if _, exists := seen[method.Name]; exists {
				continue
			}
			name := method.Name
			scenarioTypeMethodList = append(scenarioTypeMethodList, lua.RegistryFunction{
				Name:     name,
				Function: legacyScenarioSystemMethod(name),
			})
			seen[name] = struct{}{}
		}
	})
	return scenarioTypeMethodList
}

func legacyScenarioSystemMethod(method string) lua.Function {
	return func(state *lua.State) int {
		lua.Errorf(state, "%s requires a system handle, use <root>:system(\"<SYSTEM_ID>\"):%s(...)", method, method)
		return 0
	}
}

func scenarioSystem(state *lua.State) int {
	scenario := checkScenario(state)
	system := strings.TrimSpace(lua.CheckString(state, 2))
	if system == "" {
		lua.Errorf(state, "system id is required")
		return 0
	}
	registeredSystemID, _, err := registeredScenarioSystemIDForValue(system)
	if err != nil {
		lua.Errorf(state, err.Error())
		return 0
	}
	state.PushUserData(&systemHandle{scenario: scenario, system: registeredSystemID})
	lua.SetMetaTableNamed(state, systemTypeName)
	return 1
}

func scenarioCampaign(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "campaign", data)
	return 0
}

func scenarioParticipant(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	name := optionalString(data, "name", "")
	if strings.TrimSpace(name) == "" {
		lua.Errorf(state, "participant name is required")
		return 0
	}
	appendStep(scenario, "participant", data)
	state.PushUserData(&participantHandle{scenario: scenario, name: name})
	lua.SetMetaTableNamed(state, participantTypeName)
	return 1
}

func scenarioStartSession(state *lua.State) int {
	scenario := checkScenario(state)
	name := lua.OptString(state, 2, "Scenario Session")
	appendStep(scenario, "start_session", map[string]any{"name": name})
	return 0
}

func scenarioEndSession(state *lua.State) int {
	scenario := checkScenario(state)
	appendStep(scenario, "end_session", nil)
	return 0
}

func scenarioPC(state *lua.State) int {
	scenario := checkScenario(state)
	name := lua.CheckString(state, 2)
	opts := optionalTable(state, 3)
	data := map[string]any{"name": name, "kind": "PC"}
	for key, value := range opts {
		data[key] = value
	}
	appendStep(scenario, "character", data)
	return 0
}

func scenarioNPC(state *lua.State) int {
	scenario := checkScenario(state)
	name := lua.CheckString(state, 2)
	opts := optionalTable(state, 3)
	data := map[string]any{"name": name, "kind": "NPC"}
	for key, value := range opts {
		data[key] = value
	}
	appendStep(scenario, "character", data)
	return 0
}

func scenarioPrefab(state *lua.State) int {
	scenario := checkScenario(state)
	name := lua.CheckString(state, 2)
	appendStep(scenario, "prefab", map[string]any{"name": name})
	return 0
}

func scenarioAdversary(state *lua.State) int {
	handle := checkSystemHandle(state, "adversary")
	if handle == nil {
		return 0
	}
	name := lua.CheckString(state, 2)
	opts := optionalTable(state, 3)
	data := map[string]any{"name": name}
	for key, value := range opts {
		data[key] = value
	}
	appendStepWithSystem(handle.scenario, handle.system, "adversary", data)
	return 0
}

// scenarioCreationWorkflow adds an explicit Daggerheart creation workflow step
// so scenarios can drive character creation without relying on readiness
// shortcuts.
func scenarioCreationWorkflow(state *lua.State) int {
	handle := checkSystemHandle(state, "creation_workflow")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "creation_workflow", data)
	return 0
}

// scenarioExpectGMFear adds a read-only GM Fear assertion step for Daggerheart
// scenarios.
func scenarioExpectGMFear(state *lua.State) int {
	handle := checkSystemHandle(state, "expect_gm_fear")
	if handle == nil {
		return 0
	}
	value := int(lua.CheckNumber(state, 2))
	appendStepWithSystem(handle.scenario, handle.system, "expect_gm_fear", map[string]any{"value": value})
	return 0
}

func scenarioGMFear(state *lua.State) int {
	handle := checkSystemHandle(state, "gm_fear")
	if handle == nil {
		return 0
	}
	value := int(lua.CheckNumber(state, 2))
	appendStepWithSystem(handle.scenario, handle.system, "gm_fear", map[string]any{"value": value})
	return 0
}

func scenarioSetSpotlight(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "set_spotlight", data)
	return 0
}

func scenarioClearSpotlight(state *lua.State) int {
	scenario := checkScenario(state)
	appendStep(scenario, "clear_spotlight", nil)
	return 0
}

func scenarioReaction(state *lua.State) int {
	handle := checkSystemHandle(state, "reaction")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "reaction", data)
	return 0
}

func scenarioGroupReaction(state *lua.State) int {
	handle := checkSystemHandle(state, "group_reaction")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "group_reaction", data)
	return 0
}

func scenarioAttack(state *lua.State) int {
	handle := checkSystemHandle(state, "attack")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "attack", data)
	return 0
}

func scenarioMultiAttack(state *lua.State) int {
	handle := checkSystemHandle(state, "multi_attack")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "multi_attack", data)
	return 0
}

func scenarioCombinedDamage(state *lua.State) int {
	handle := checkSystemHandle(state, "combined_damage")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "combined_damage", data)
	return 0
}

func scenarioAdversaryAttack(state *lua.State) int {
	handle := checkSystemHandle(state, "adversary_attack")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "adversary_attack", data)
	return 0
}

func scenarioAdversaryFeature(state *lua.State) int {
	handle := checkSystemHandle(state, "adversary_feature")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "adversary_feature", data)
	return 0
}

func scenarioAdversaryReaction(state *lua.State) int {
	handle := checkSystemHandle(state, "adversary_reaction")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "adversary_reaction", data)
	return 0
}

func scenarioAdversaryUpdate(state *lua.State) int {
	handle := checkSystemHandle(state, "adversary_update")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "adversary_update", data)
	return 0
}

func scenarioApplyCondition(state *lua.State) int {
	handle := checkSystemHandle(state, "apply_condition")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "apply_condition", data)
	return 0
}

func scenarioApplyStatModifier(state *lua.State) int {
	handle := checkSystemHandle(state, "apply_stat_modifier")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "apply_stat_modifier", data)
	return 0
}

func scenarioGroupAction(state *lua.State) int {
	handle := checkSystemHandle(state, "group_action")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "group_action", data)
	return 0
}

func scenarioTagTeam(state *lua.State) int {
	handle := checkSystemHandle(state, "tag_team")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "tag_team", data)
	return 0
}

func scenarioTemporaryArmor(state *lua.State) int {
	handle := checkSystemHandle(state, "temporary_armor")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "temporary_armor", data)
	return 0
}

func scenarioRest(state *lua.State) int {
	handle := checkSystemHandle(state, "rest")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "rest", data)
	return 0
}

func scenarioDeathMove(state *lua.State) int {
	handle := checkSystemHandle(state, "death_move")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "death_move", data)
	return 0
}

func scenarioBlazeOfGlory(state *lua.State) int {
	handle := checkSystemHandle(state, "blaze_of_glory")
	if handle == nil {
		return 0
	}
	name := lua.CheckString(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "blaze_of_glory", map[string]any{"target": name})
	return 0
}

func scenarioGMSpendFear(state *lua.State) int {
	handle := checkSystemHandle(state, "gm_spend_fear")
	if handle == nil {
		return 0
	}
	amount := int(lua.CheckNumber(state, 2))
	stepIndex := appendStepWithSystem(handle.scenario, handle.system, "gm_spend_fear", map[string]any{"amount": amount})
	state.PushUserData(&gmAction{scenario: handle.scenario, stepIndex: stepIndex})
	lua.SetMetaTableNamed(state, gmActionTypeName)
	return 1
}

func scenarioSwapLoadout(state *lua.State) int {
	handle := checkSystemHandle(state, "swap_loadout")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "swap_loadout", data)
	return 0
}

func scenarioCountdownCreate(state *lua.State) int {
	handle := checkSystemHandle(state, "countdown_create")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "countdown_create", data)
	return 0
}

func scenarioCountdownUpdate(state *lua.State) int {
	handle := checkSystemHandle(state, "countdown_update")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "countdown_update", data)
	return 0
}

func scenarioCountdownDelete(state *lua.State) int {
	handle := checkSystemHandle(state, "countdown_delete")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "countdown_delete", data)
	return 0
}

func scenarioActionRoll(state *lua.State) int {
	handle := checkSystemHandle(state, "action_roll")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "action_roll", data)
	return 0
}

func scenarioReactionRoll(state *lua.State) int {
	handle := checkSystemHandle(state, "reaction_roll")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "reaction_roll", data)
	return 0
}

func scenarioDamageRoll(state *lua.State) int {
	handle := checkSystemHandle(state, "damage_roll")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "damage_roll", data)
	return 0
}

func scenarioAdversaryAttackRoll(state *lua.State) int {
	handle := checkSystemHandle(state, "adversary_attack_roll")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "adversary_attack_roll", data)
	return 0
}

func scenarioApplyRollOutcome(state *lua.State) int {
	handle := checkSystemHandle(state, "apply_roll_outcome")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "apply_roll_outcome", data)
	return 0
}

func scenarioApplyAttackOutcome(state *lua.State) int {
	handle := checkSystemHandle(state, "apply_attack_outcome")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "apply_attack_outcome", data)
	return 0
}

func scenarioApplyAdversaryAttackOutcome(state *lua.State) int {
	handle := checkSystemHandle(state, "apply_adversary_attack_outcome")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "apply_adversary_attack_outcome", data)
	return 0
}

func scenarioApplyReactionOutcome(state *lua.State) int {
	handle := checkSystemHandle(state, "apply_reaction_outcome")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "apply_reaction_outcome", data)
	return 0
}

func scenarioMitigateDamage(state *lua.State) int {
	handle := checkSystemHandle(state, "mitigate_damage")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "mitigate_damage", data)
	return 0
}

func scenarioLevelUp(state *lua.State) int {
	handle := checkSystemHandle(state, "level_up")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "level_up", data)
	return 0
}

func scenarioClassFeature(state *lua.State) int {
	handle := checkSystemHandle(state, "class_feature")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "class_feature", data)
	return 0
}

func scenarioUpdateGold(state *lua.State) int {
	handle := checkSystemHandle(state, "update_gold")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "update_gold", data)
	return 0
}

func scenarioAcquireDomainCard(state *lua.State) int {
	handle := checkSystemHandle(state, "acquire_domain_card")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "acquire_domain_card", data)
	return 0
}

func scenarioSwapEquipment(state *lua.State) int {
	handle := checkSystemHandle(state, "swap_equipment")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "swap_equipment", data)
	return 0
}

func scenarioUseConsumable(state *lua.State) int {
	handle := checkSystemHandle(state, "use_consumable")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "use_consumable", data)
	return 0
}

func scenarioAcquireConsumable(state *lua.State) int {
	handle := checkSystemHandle(state, "acquire_consumable")
	if handle == nil {
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStepWithSystem(handle.scenario, handle.system, "acquire_consumable", data)
	return 0
}

var gmActionMethods = []lua.RegistryFunction{
	{Name: "spotlight", Function: gmActionSpotlight},
	{Name: "adversary_spotlight", Function: gmActionAdversarySpotlight},
	{Name: "move", Function: gmActionMove},
	{Name: "adversary_feature", Function: gmActionAdversaryFeature},
	{Name: "environment_feature", Function: gmActionEnvironmentFeature},
	{Name: "adversary_experience", Function: gmActionAdversaryExperience},
}

var participantMethods = []lua.RegistryFunction{
	{Name: "character", Function: participantCharacter},
}

func gmActionSpotlight(state *lua.State) int {
	ud := lua.CheckUserData(state, 1, gmActionTypeName)
	action, ok := ud.(*gmAction)
	if !ok || action == nil {
		lua.Errorf(state, "invalid gm action")
		return 0
	}
	name := lua.CheckString(state, 2)
	opts := optionalTable(state, 3)
	step := gmActionStep(state, action)
	if step == nil {
		return 0
	}
	step.Args["spend_target"] = "direct_move"
	step.Args["move"] = "custom"
	step.Args["description"] = "spotlight " + name
	mergeScenarioStepArgs(step, opts)
	return 0
}

func gmActionAdversarySpotlight(state *lua.State) int {
	ud := lua.CheckUserData(state, 1, gmActionTypeName)
	action, ok := ud.(*gmAction)
	if !ok || action == nil {
		lua.Errorf(state, "invalid gm action")
		return 0
	}
	name := lua.CheckString(state, 2)
	opts := optionalTable(state, 3)
	step := gmActionStep(state, action)
	if step == nil {
		return 0
	}
	step.Args["spend_target"] = "direct_move"
	step.Args["move"] = "spotlight"
	step.Args["target"] = name
	mergeScenarioStepArgs(step, opts)
	return 0
}

func gmActionMove(state *lua.State) int {
	ud := lua.CheckUserData(state, 1, gmActionTypeName)
	action, ok := ud.(*gmAction)
	if !ok || action == nil {
		lua.Errorf(state, "invalid gm action")
		return 0
	}
	move := lua.CheckString(state, 2)
	opts := optionalTable(state, 3)
	step := gmActionStep(state, action)
	if step == nil {
		return 0
	}
	step.Args["spend_target"] = "direct_move"
	step.Args["move"] = move
	mergeScenarioStepArgs(step, opts)
	return 0
}

func gmActionAdversaryFeature(state *lua.State) int {
	ud := lua.CheckUserData(state, 1, gmActionTypeName)
	action, ok := ud.(*gmAction)
	if !ok || action == nil {
		lua.Errorf(state, "invalid gm action")
		return 0
	}
	target := lua.CheckString(state, 2)
	featureID := lua.CheckString(state, 3)
	opts := optionalTable(state, 4)
	step := gmActionStep(state, action)
	if step == nil {
		return 0
	}
	step.Args["spend_target"] = "adversary_feature"
	step.Args["target"] = target
	step.Args["feature_id"] = featureID
	mergeScenarioStepArgs(step, opts)
	return 0
}

func gmActionEnvironmentFeature(state *lua.State) int {
	ud := lua.CheckUserData(state, 1, gmActionTypeName)
	action, ok := ud.(*gmAction)
	if !ok || action == nil {
		lua.Errorf(state, "invalid gm action")
		return 0
	}
	environmentID := lua.CheckString(state, 2)
	featureID := lua.CheckString(state, 3)
	opts := optionalTable(state, 4)
	step := gmActionStep(state, action)
	if step == nil {
		return 0
	}
	step.Args["spend_target"] = "environment_feature"
	step.Args["environment_id"] = environmentID
	step.Args["feature_id"] = featureID
	mergeScenarioStepArgs(step, opts)
	return 0
}

func gmActionAdversaryExperience(state *lua.State) int {
	ud := lua.CheckUserData(state, 1, gmActionTypeName)
	action, ok := ud.(*gmAction)
	if !ok || action == nil {
		lua.Errorf(state, "invalid gm action")
		return 0
	}
	target := lua.CheckString(state, 2)
	experienceName := lua.CheckString(state, 3)
	opts := optionalTable(state, 4)
	step := gmActionStep(state, action)
	if step == nil {
		return 0
	}
	step.Args["spend_target"] = "adversary_experience"
	step.Args["target"] = target
	step.Args["experience_name"] = experienceName
	mergeScenarioStepArgs(step, opts)
	return 0
}

func gmActionStep(state *lua.State, action *gmAction) *Step {
	if action.stepIndex < 0 || action.stepIndex >= len(action.scenario.Steps) {
		lua.Errorf(state, "gm action is out of range")
		return nil
	}
	step := &action.scenario.Steps[action.stepIndex]
	if step.Args == nil {
		step.Args = map[string]any{}
	}
	return step
}

func mergeScenarioStepArgs(step *Step, opts map[string]any) {
	if step == nil || len(opts) == 0 {
		return
	}
	for key, value := range opts {
		step.Args[key] = value
	}
}

func participantCharacter(state *lua.State) int {
	ud := lua.CheckUserData(state, 1, participantTypeName)
	handle, ok := ud.(*participantHandle)
	if !ok || handle == nil {
		lua.Errorf(state, "invalid participant handle")
		return 0
	}
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	name := optionalString(data, "name", "")
	if strings.TrimSpace(name) == "" {
		lua.Errorf(state, "character name is required")
		return 0
	}
	if _, ok := data["kind"]; !ok {
		data["kind"] = "PC"
	}
	if _, ok := data["control"]; !ok {
		data["control"] = "participant"
	}
	data["participant"] = handle.name
	appendStep(handle.scenario, "character", data)
	return 0
}

func checkScenario(state *lua.State) *Scenario {
	ud := lua.CheckUserData(state, 1, scenarioTypeName)
	if scenario, ok := ud.(*Scenario); ok && scenario != nil {
		return scenario
	}
	lua.ArgumentError(state, 1, "scenario expected")
	return nil
}

func checkSystemHandle(state *lua.State, method string) *systemHandle {
	if state.TypeOf(1) == lua.TypeUserData {
		if scenario, ok := state.ToUserData(1).(*Scenario); ok && scenario != nil {
			lua.Errorf(state, "%s requires a system handle, use <root>:system(\"<SYSTEM_ID>\"):%s(...)", method, method)
			return nil
		}
	}
	ud := lua.CheckUserData(state, 1, systemTypeName)
	handle, ok := ud.(*systemHandle)
	if !ok || handle == nil || handle.scenario == nil || strings.TrimSpace(handle.system) == "" {
		lua.Errorf(state, "invalid system handle")
		return nil
	}
	return handle
}

func appendStep(scenario *Scenario, kind string, data map[string]any) int {
	return appendStepWithSystem(scenario, "", kind, data)
}

func appendStepWithSystem(scenario *Scenario, system string, kind string, data map[string]any) int {
	if scenario == nil {
		return -1
	}
	if data == nil {
		data = map[string]any{}
	}
	scenario.Steps = append(scenario.Steps, Step{System: strings.ToUpper(strings.TrimSpace(system)), Kind: kind, Args: data})
	return len(scenario.Steps) - 1
}

func optionalTable(state *lua.State, index int) map[string]any {
	if state.IsNoneOrNil(index) || state.TypeOf(index) != lua.TypeTable {
		return map[string]any{}
	}
	return tableToMap(state, index)
}

func tableToMap(state *lua.State, index int) map[string]any {
	output := map[string]any{}
	if state.TypeOf(index) != lua.TypeTable {
		return output
	}

	index = state.AbsIndex(index)
	state.PushNil()
	for state.Next(index) {
		if state.TypeOf(-2) == lua.TypeString {
			key, _ := state.ToString(-2)
			output[key] = luaToGo(state, -1)
		}
		state.Pop(1)
	}
	return output
}

func luaToGo(state *lua.State, index int) any {
	switch state.TypeOf(index) {
	case lua.TypeString:
		value, _ := state.ToString(index)
		return value
	case lua.TypeNumber:
		value, _ := state.ToNumber(index)
		return normalizeNumber(value)
	case lua.TypeBoolean:
		return state.ToBoolean(index)
	case lua.TypeTable:
		return tableToGo(state, index)
	case lua.TypeUserData:
		return state.ToUserData(index)
	default:
		return nil
	}
}

func tableToGo(state *lua.State, index int) any {
	if state.TypeOf(index) != lua.TypeTable {
		return nil
	}

	index = state.AbsIndex(index)
	isArray := true
	maxIndex := 0
	count := 0
	state.PushNil()
	for state.Next(index) {
		if isArray {
			if state.TypeOf(-2) != lua.TypeNumber {
				isArray = false
			} else if idx, ok := state.ToInteger(-2); ok && idx > 0 {
				count++
				if idx > maxIndex {
					maxIndex = idx
				}
			} else {
				isArray = false
			}
		}
		state.Pop(1)
	}

	if isArray && count > 0 && maxIndex == count {
		result := make([]any, 0, maxIndex)
		for i := 1; i <= maxIndex; i++ {
			state.RawGetInt(index, i)
			result = append(result, luaToGo(state, -1))
			state.Pop(1)
		}
		return result
	}

	return tableToMap(state, index)
}

func normalizeNumber(value float64) any {
	if math.Mod(value, 1) == 0 {
		return int(value)
	}
	return value
}
