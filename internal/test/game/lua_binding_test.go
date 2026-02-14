//go:build scenario

package game

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"

	"github.com/Shopify/go-lua"
)

const (
	scenarioTypeName = "scenario"
	gmActionTypeName = "gm_action"
)

type Scenario struct {
	Name  string
	Steps []Step
}

type Step struct {
	Kind string
	Args map[string]any
}

type gmAction struct {
	scenario  *Scenario
	stepIndex int
}

func loadScenarioFromFile(path string) (*Scenario, error) {
	state := lua.NewState()
	lua.OpenLibraries(state)

	registerLuaTypes(state)

	if err := lua.LoadFile(state, path, ""); err != nil {
		return nil, fmt.Errorf("load lua: %w", err)
	}
	if err := state.ProtectedCall(0, 1, 0); err != nil {
		return nil, fmt.Errorf("run lua: %w", err)
	}

	if state.TypeOf(-1) != lua.TypeUserData {
		state.Pop(1)
		return nil, fmt.Errorf("scenario script must return Scenario")
	}
	ud := state.ToUserData(-1)
	state.Pop(1)
	scenario, ok := ud.(*Scenario)
	if !ok || scenario == nil {
		return nil, fmt.Errorf("scenario script returned invalid Scenario")
	}
	if strings.TrimSpace(scenario.Name) == "" {
		scenario.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return scenario, nil
}

func registerLuaTypes(state *lua.State) {
	registerScenarioType(state)
	registerGMActionType(state)
	registerScenarioConstructor(state)
	registerModifierHelpers(state)
}

func registerScenarioType(state *lua.State) {
	lua.NewMetaTable(state, scenarioTypeName)
	state.NewTable()
	lua.SetFunctions(state, scenarioMethods, 0)
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
	{Name: "campaign", Function: scenarioCampaign},
	{Name: "start_session", Function: scenarioStartSession},
	{Name: "end_session", Function: scenarioEndSession},
	{Name: "pc", Function: scenarioPC},
	{Name: "npc", Function: scenarioNPC},
	{Name: "prefab", Function: scenarioPrefab},
	{Name: "adversary", Function: scenarioAdversary},
	{Name: "gm_fear", Function: scenarioGMFear},
	{Name: "reaction", Function: scenarioReaction},
	{Name: "attack", Function: scenarioAttack},
	{Name: "multi_attack", Function: scenarioMultiAttack},
	{Name: "combined_damage", Function: scenarioCombinedDamage},
	{Name: "adversary_attack", Function: scenarioAdversaryAttack},
	{Name: "apply_condition", Function: scenarioApplyCondition},
	{Name: "gm_spend_fear", Function: scenarioGMSpendFear},
	{Name: "group_action", Function: scenarioGroupAction},
	{Name: "tag_team", Function: scenarioTagTeam},
	{Name: "rest", Function: scenarioRest},
	{Name: "downtime_move", Function: scenarioDowntimeMove},
	{Name: "death_move", Function: scenarioDeathMove},
	{Name: "blaze_of_glory", Function: scenarioBlazeOfGlory},
	{Name: "swap_loadout", Function: scenarioSwapLoadout},
	{Name: "countdown_create", Function: scenarioCountdownCreate},
	{Name: "countdown_update", Function: scenarioCountdownUpdate},
	{Name: "countdown_delete", Function: scenarioCountdownDelete},
	{Name: "action_roll", Function: scenarioActionRoll},
	{Name: "reaction_roll", Function: scenarioReactionRoll},
	{Name: "damage_roll", Function: scenarioDamageRoll},
	{Name: "adversary_attack_roll", Function: scenarioAdversaryAttackRoll},
	{Name: "apply_roll_outcome", Function: scenarioApplyRollOutcome},
	{Name: "apply_attack_outcome", Function: scenarioApplyAttackOutcome},
	{Name: "apply_adversary_attack_outcome", Function: scenarioApplyAdversaryAttackOutcome},
	{Name: "apply_reaction_outcome", Function: scenarioApplyReactionOutcome},
	{Name: "mitigate_damage", Function: scenarioMitigateDamage},
}

func scenarioCampaign(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "campaign", data)
	return 0
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
	scenario := checkScenario(state)
	name := lua.CheckString(state, 2)
	opts := optionalTable(state, 3)
	data := map[string]any{"name": name}
	for key, value := range opts {
		data[key] = value
	}
	appendStep(scenario, "adversary", data)
	return 0
}

func scenarioGMFear(state *lua.State) int {
	scenario := checkScenario(state)
	value := int(lua.CheckNumber(state, 2))
	appendStep(scenario, "gm_fear", map[string]any{"value": value})
	return 0
}

func scenarioReaction(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "reaction", data)
	return 0
}

func scenarioAttack(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "attack", data)
	return 0
}

func scenarioMultiAttack(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "multi_attack", data)
	return 0
}

func scenarioCombinedDamage(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "combined_damage", data)
	return 0
}

func scenarioAdversaryAttack(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "adversary_attack", data)
	return 0
}

func scenarioApplyCondition(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "apply_condition", data)
	return 0
}

func scenarioGroupAction(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "group_action", data)
	return 0
}

func scenarioTagTeam(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "tag_team", data)
	return 0
}

func scenarioRest(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "rest", data)
	return 0
}

func scenarioDowntimeMove(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "downtime_move", data)
	return 0
}

func scenarioDeathMove(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "death_move", data)
	return 0
}

func scenarioBlazeOfGlory(state *lua.State) int {
	scenario := checkScenario(state)
	name := lua.CheckString(state, 2)
	appendStep(scenario, "blaze_of_glory", map[string]any{"target": name})
	return 0
}

func scenarioGMSpendFear(state *lua.State) int {
	scenario := checkScenario(state)
	amount := int(lua.CheckNumber(state, 2))
	stepIndex := appendStep(scenario, "gm_spend_fear", map[string]any{"amount": amount})
	state.PushUserData(&gmAction{scenario: scenario, stepIndex: stepIndex})
	lua.SetMetaTableNamed(state, gmActionTypeName)
	return 1
}

func scenarioSwapLoadout(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "swap_loadout", data)
	return 0
}

func scenarioCountdownCreate(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "countdown_create", data)
	return 0
}

func scenarioCountdownUpdate(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "countdown_update", data)
	return 0
}

func scenarioCountdownDelete(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "countdown_delete", data)
	return 0
}

func scenarioActionRoll(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "action_roll", data)
	return 0
}

func scenarioReactionRoll(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "reaction_roll", data)
	return 0
}

func scenarioDamageRoll(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "damage_roll", data)
	return 0
}

func scenarioAdversaryAttackRoll(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "adversary_attack_roll", data)
	return 0
}

func scenarioApplyRollOutcome(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "apply_roll_outcome", data)
	return 0
}

func scenarioApplyAttackOutcome(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "apply_attack_outcome", data)
	return 0
}

func scenarioApplyAdversaryAttackOutcome(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "apply_adversary_attack_outcome", data)
	return 0
}

func scenarioApplyReactionOutcome(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "apply_reaction_outcome", data)
	return 0
}

func scenarioMitigateDamage(state *lua.State) int {
	scenario := checkScenario(state)
	lua.CheckType(state, 2, lua.TypeTable)
	data := tableToMap(state, 2)
	appendStep(scenario, "mitigate_damage", data)
	return 0
}

var gmActionMethods = []lua.RegistryFunction{
	{Name: "spotlight", Function: gmActionSpotlight},
}

func gmActionSpotlight(state *lua.State) int {
	ud := lua.CheckUserData(state, 1, gmActionTypeName)
	action, ok := ud.(*gmAction)
	if !ok || action == nil {
		lua.Errorf(state, "invalid gm action")
		return 0
	}
	name := lua.CheckString(state, 2)
	if action.stepIndex < 0 || action.stepIndex >= len(action.scenario.Steps) {
		lua.Errorf(state, "gm action is out of range")
		return 0
	}
	step := &action.scenario.Steps[action.stepIndex]
	if step.Args == nil {
		step.Args = map[string]any{}
	}
	step.Args["target"] = name
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

func appendStep(scenario *Scenario, kind string, data map[string]any) int {
	if scenario == nil {
		return -1
	}
	if data == nil {
		data = map[string]any{}
	}
	scenario.Steps = append(scenario.Steps, Step{Kind: kind, Args: data})
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
