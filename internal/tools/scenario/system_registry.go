package scenario

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/Shopify/go-lua"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
)

type systemStepRunner func(r *Runner, ctx context.Context, state *scenarioState, step Step) error

type systemCharacterReadinessRunner func(r *Runner, ctx context.Context, state *scenarioState, characterID string) error

type systemCharacterNeedsReadinessRunner func(r *Runner, ctx context.Context, state *scenarioState, characterID string) (bool, error)

type scenarioSystemRegistration struct {
	id                       string
	gameSystem               commonv1.GameSystem
	dslMethods               []lua.RegistryFunction
	stepKinds                map[string]struct{}
	runStep                  systemStepRunner
	ensureCharacterReadiness systemCharacterReadinessRunner
	characterNeedsReadiness  systemCharacterNeedsReadinessRunner
}

var scenarioSystemRegistry = map[string]scenarioSystemRegistration{
	"DAGGERHEART": {
		id:         "DAGGERHEART",
		gameSystem: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		dslMethods: daggerheartSystemMethods,
		stepKinds:  daggerheartSystemStepKinds,
	},
}

var daggerheartSystemMethods = []lua.RegistryFunction{
	{Name: "adversary", Function: scenarioAdversary},
	{Name: "gm_fear", Function: scenarioGMFear},
	{Name: "reaction", Function: scenarioReaction},
	{Name: "group_reaction", Function: scenarioGroupReaction},
	{Name: "attack", Function: scenarioAttack},
	{Name: "multi_attack", Function: scenarioMultiAttack},
	{Name: "combined_damage", Function: scenarioCombinedDamage},
	{Name: "adversary_attack", Function: scenarioAdversaryAttack},
	{Name: "adversary_reaction", Function: scenarioAdversaryReaction},
	{Name: "adversary_update", Function: scenarioAdversaryUpdate},
	{Name: "apply_condition", Function: scenarioApplyCondition},
	{Name: "gm_spend_fear", Function: scenarioGMSpendFear},
	{Name: "group_action", Function: scenarioGroupAction},
	{Name: "tag_team", Function: scenarioTagTeam},
	{Name: "temporary_armor", Function: scenarioTemporaryArmor},
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

var daggerheartSystemStepKinds = map[string]struct{}{
	"adversary":                      {},
	"gm_fear":                        {},
	"reaction":                       {},
	"group_reaction":                 {},
	"gm_spend_fear":                  {},
	"apply_condition":                {},
	"group_action":                   {},
	"tag_team":                       {},
	"temporary_armor":                {},
	"rest":                           {},
	"downtime_move":                  {},
	"death_move":                     {},
	"blaze_of_glory":                 {},
	"attack":                         {},
	"multi_attack":                   {},
	"combined_damage":                {},
	"adversary_attack":               {},
	"adversary_reaction":             {},
	"adversary_update":               {},
	"swap_loadout":                   {},
	"countdown_create":               {},
	"countdown_update":               {},
	"countdown_delete":               {},
	"action_roll":                    {},
	"reaction_roll":                  {},
	"damage_roll":                    {},
	"adversary_attack_roll":          {},
	"apply_roll_outcome":             {},
	"apply_attack_outcome":           {},
	"apply_adversary_attack_outcome": {},
	"apply_reaction_outcome":         {},
	"mitigate_damage":                {},
}

var (
	scenarioSystemRegistryOnce       sync.Once
	scenarioSystemIDsSorted          []string
	scenarioSystemByID               map[string]scenarioSystemRegistration
	scenarioSystemByGameSystem       map[commonv1.GameSystem]scenarioSystemRegistration
	scenarioSystemHandleMethods      []lua.RegistryFunction
	scenarioSystemStepKindsToSystems map[string][]string
	scenarioSystemKnownStepKinds     map[string]struct{}
)

var sharedScenarioStepKinds = map[string]struct{}{}

func indexScenarioSystemRegistry() {
	scenarioSystemRegistryOnce.Do(func() {
		normalizedRegistry := make(map[string]scenarioSystemRegistration, len(scenarioSystemRegistry))
		ids := make([]string, 0, len(scenarioSystemRegistry))
		for id, registration := range scenarioSystemRegistry {
			normalizedID := strings.ToUpper(strings.TrimSpace(id))
			if normalizedID == "" {
				continue
			}
			if _, exists := normalizedRegistry[normalizedID]; exists {
				panic(fmt.Sprintf("invalid scenario system registry: duplicate key %q after normalization", normalizedID))
			}
			registration = bindScenarioSystemRuntimeHooks(registration)
			normalizedRegistry[normalizedID] = registration
			ids = append(ids, normalizedID)
		}
		if err := validateScenarioSystemRegistry(normalizedRegistry); err != nil {
			panic(fmt.Sprintf("invalid scenario system registry: %v", err))
		}
		sort.Strings(ids)
		scenarioSystemIDsSorted = ids

		scenarioSystemByID = make(map[string]scenarioSystemRegistration, len(ids))
		scenarioSystemByGameSystem = make(map[commonv1.GameSystem]scenarioSystemRegistration, len(ids))
		scenarioSystemStepKindsToSystems = map[string][]string{}
		scenarioSystemKnownStepKinds = map[string]struct{}{}
		methodNames := map[string]struct{}{}

		for _, id := range ids {
			registration := normalizedRegistry[id]
			scenarioSystemByID[registration.id] = registration
			scenarioSystemByGameSystem[registration.gameSystem] = registration

			for _, method := range registration.dslMethods {
				if strings.TrimSpace(method.Name) == "" {
					continue
				}
				// System handles share a metatable, so keep one method entry per name.
				if _, exists := methodNames[method.Name]; exists {
					continue
				}
				methodNames[method.Name] = struct{}{}
				scenarioSystemHandleMethods = append(scenarioSystemHandleMethods, method)
			}
			for stepKind := range registration.stepKinds {
				if strings.TrimSpace(stepKind) == "" {
					continue
				}
				scenarioSystemKnownStepKinds[stepKind] = struct{}{}
				scenarioSystemStepKindsToSystems[stepKind] = append(scenarioSystemStepKindsToSystems[stepKind], registration.id)
			}
		}
		for stepKind := range scenarioSystemStepKindsToSystems {
			sort.Strings(scenarioSystemStepKindsToSystems[stepKind])
		}
	})
}

func registeredScenarioSystemIDs() []string {
	indexScenarioSystemRegistry()
	return append([]string(nil), scenarioSystemIDsSorted...)
}

func registeredScenarioSystemMethods() []lua.RegistryFunction {
	indexScenarioSystemRegistry()
	return append([]lua.RegistryFunction(nil), scenarioSystemHandleMethods...)
}

func scenarioSystemForID(systemID string) (scenarioSystemRegistration, bool) {
	indexScenarioSystemRegistry()
	normalized := strings.ToUpper(strings.TrimSpace(systemID))
	registration, ok := scenarioSystemByID[normalized]
	return registration, ok
}

func scenarioSystemForGameSystem(system commonv1.GameSystem) (scenarioSystemRegistration, bool) {
	indexScenarioSystemRegistry()
	registration, ok := scenarioSystemByGameSystem[system]
	return registration, ok
}

func isKnownScenarioSystemStepKind(stepKind string) bool {
	indexScenarioSystemRegistry()
	_, ok := scenarioSystemKnownStepKinds[stepKind]
	return ok
}

func registeredSystemsForStepKind(stepKind string) []string {
	indexScenarioSystemRegistry()
	systems := scenarioSystemStepKindsToSystems[stepKind]
	return append([]string(nil), systems...)
}

func registeredScenarioSystemIDForGameSystem(system commonv1.GameSystem) (string, error) {
	registration, ok := scenarioSystemForGameSystem(system)
	if !ok {
		return "", unsupportedScenarioSystemError(strings.TrimPrefix(system.String(), "GAME_SYSTEM_"))
	}
	return registration.id, nil
}

func registeredScenarioSystemIDForValue(value string) (string, commonv1.GameSystem, error) {
	gameSystem, err := parseGameSystem(value)
	if err != nil {
		return "", commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, err
	}
	systemID, err := registeredScenarioSystemIDForGameSystem(gameSystem)
	if err != nil {
		return "", commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, err
	}
	return systemID, gameSystem, nil
}

func scenarioSystemForState(state *scenarioState) (scenarioSystemRegistration, bool, error) {
	if state == nil || state.campaignSystem == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return scenarioSystemRegistration{}, false, nil
	}
	registration, ok := scenarioSystemForGameSystem(state.campaignSystem)
	if !ok {
		return scenarioSystemRegistration{}, false, unsupportedScenarioSystemError(strings.TrimPrefix(state.campaignSystem.String(), "GAME_SYSTEM_"))
	}
	return registration, true, nil
}

func unsupportedScenarioSystemError(value string) error {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	registered := registeredScenarioSystemIDs()
	if len(registered) == 0 {
		return fmt.Errorf("unsupported scenario system %q", normalized)
	}
	return fmt.Errorf("unsupported scenario system %q (registered systems: %s)", normalized, strings.Join(registered, ", "))
}

func bindScenarioSystemRuntimeHooks(registration scenarioSystemRegistration) scenarioSystemRegistration {
	switch registration.id {
	case "DAGGERHEART":
		registration.runStep = (*Runner).runDaggerheartStep
		registration.ensureCharacterReadiness = (*Runner).ensureDaggerheartCharacterReadiness
		registration.characterNeedsReadiness = (*Runner).daggerheartCharacterNeedsReadiness
	}
	return registration
}

func validateScenarioSystemRegistry(registry map[string]scenarioSystemRegistration) error {
	if len(registry) == 0 {
		return fmt.Errorf("scenario system registry is empty")
	}

	seenIDs := make(map[string]struct{}, len(registry))
	seenGameSystems := make(map[commonv1.GameSystem]string, len(registry))
	seenMethodOwners := map[string]string{}
	seenStepKindOwners := map[string]string{}

	for key, registration := range registry {
		normalizedKey := strings.ToUpper(strings.TrimSpace(key))
		if normalizedKey == "" {
			return fmt.Errorf("scenario system registry contains empty key")
		}

		normalizedID := strings.ToUpper(strings.TrimSpace(registration.id))
		if normalizedID == "" {
			return fmt.Errorf("scenario system %q has empty id", normalizedKey)
		}
		if normalizedID != normalizedKey {
			return fmt.Errorf("scenario system key %q must match id %q", normalizedKey, normalizedID)
		}
		if _, exists := seenIDs[normalizedID]; exists {
			return fmt.Errorf("scenario system id %q is duplicated", normalizedID)
		}
		seenIDs[normalizedID] = struct{}{}

		if registration.gameSystem == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
			return fmt.Errorf("scenario system %q must not use GAME_SYSTEM_UNSPECIFIED", normalizedID)
		}
		if _, exists := commonv1.GameSystem_name[int32(registration.gameSystem)]; !exists {
			return fmt.Errorf("scenario system %q uses unknown game system value %d", normalizedID, int32(registration.gameSystem))
		}
		if owner, exists := seenGameSystems[registration.gameSystem]; exists && owner != normalizedID {
			return fmt.Errorf("game system %q is registered by multiple scenario systems (%s, %s)", registration.gameSystem.String(), owner, normalizedID)
		}
		seenGameSystems[registration.gameSystem] = normalizedID

		if len(registration.dslMethods) == 0 {
			return fmt.Errorf("scenario system %q must define at least one DSL method", normalizedID)
		}
		for _, method := range registration.dslMethods {
			name := strings.TrimSpace(method.Name)
			if name == "" {
				return fmt.Errorf("scenario system %q has an empty DSL method name", normalizedID)
			}
			if method.Function == nil {
				return fmt.Errorf("scenario system %q has nil DSL method function for %q", normalizedID, name)
			}
			if owner, exists := seenMethodOwners[name]; exists {
				if owner == normalizedID {
					return fmt.Errorf("scenario DSL method %q is duplicated in system %s", name, normalizedID)
				}
				return fmt.Errorf("scenario DSL method %q is registered by multiple systems (%s, %s)", name, owner, normalizedID)
			}
			seenMethodOwners[name] = normalizedID
		}

		if registration.runStep == nil {
			return fmt.Errorf("scenario system %q must define a step runner", normalizedID)
		}
		if len(registration.stepKinds) == 0 {
			return fmt.Errorf("scenario system %q must define at least one step kind", normalizedID)
		}
		for stepKind := range registration.stepKinds {
			normalizedStepKind := strings.TrimSpace(stepKind)
			if normalizedStepKind == "" {
				return fmt.Errorf("scenario system %q has empty step kind", normalizedID)
			}
			if owner, exists := seenStepKindOwners[normalizedStepKind]; exists && owner != normalizedID {
				if _, allowed := sharedScenarioStepKinds[normalizedStepKind]; !allowed {
					return fmt.Errorf("scenario step kind %q is registered by multiple systems (%s, %s)", normalizedStepKind, owner, normalizedID)
				}
			}
			seenStepKindOwners[normalizedStepKind] = normalizedID
		}
	}

	return nil
}
