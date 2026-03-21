package conditiontransport

import (
	"fmt"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/mechanics"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

type ConditionStateView struct {
	ID            string
	Class         pb.DaggerheartConditionClass
	Standard      pb.DaggerheartCondition
	Code          string
	Label         string
	Source        string
	SourceID      string
	ClearTriggers []pb.DaggerheartConditionClearTrigger
}

func ConditionStatesFromProto(states []*pb.DaggerheartConditionState) ([]*ConditionStateView, error) {
	if len(states) == 0 {
		return nil, nil
	}
	result := make([]*ConditionStateView, 0, len(states))
	for _, state := range states {
		view, err := conditionStateFromProto(state)
		if err != nil {
			return nil, err
		}
		result = append(result, view)
	}
	return result, nil
}

func ConditionStateViewsToProto(states []*ConditionStateView) []*pb.DaggerheartConditionState {
	if len(states) == 0 {
		return nil
	}
	result := make([]*pb.DaggerheartConditionState, 0, len(states))
	for _, state := range states {
		if state == nil {
			continue
		}
		result = append(result, &pb.DaggerheartConditionState{
			Id:            state.ID,
			Class:         state.Class,
			Standard:      state.Standard,
			Code:          state.Code,
			Label:         state.Label,
			Source:        state.Source,
			SourceId:      state.SourceID,
			ClearTriggers: append([]pb.DaggerheartConditionClearTrigger(nil), state.ClearTriggers...),
		})
	}
	return result
}

func ConditionsToProto(conditions []string) []*pb.DaggerheartConditionState {
	return ConditionStateViewsToProto(ConditionsToViews(conditions, ""))
}

func ProjectionConditionStatesToProto(states []projectionstore.DaggerheartConditionState) []*pb.DaggerheartConditionState {
	return ConditionStateViewsToProto(ProjectionConditionStatesToViews(states))
}

func ProjectionConditionStatesToViews(states []projectionstore.DaggerheartConditionState) []*ConditionStateView {
	if len(states) == 0 {
		return nil
	}
	result := make([]*ConditionStateView, 0, len(states))
	for _, state := range states {
		result = append(result, &ConditionStateView{
			ID:            state.ID,
			Class:         conditionClassToProto(state.Class),
			Standard:      standardConditionProtoOrZero(state.Standard),
			Code:          state.Code,
			Label:         state.Label,
			Source:        state.Source,
			SourceID:      state.SourceID,
			ClearTriggers: projectionClearTriggersToProto(state.ClearTriggers),
		})
	}
	return result
}

func ProjectionConditionStatesToDomain(states []projectionstore.DaggerheartConditionState) []rules.ConditionState {
	if len(states) == 0 {
		return nil
	}
	result := make([]rules.ConditionState, 0, len(states))
	for _, state := range states {
		entry := rules.ConditionState{
			ID:       state.ID,
			Class:    rules.ConditionClass(strings.TrimSpace(state.Class)),
			Standard: strings.TrimSpace(state.Standard),
			Code:     strings.TrimSpace(state.Code),
			Label:    strings.TrimSpace(state.Label),
			Source:   strings.TrimSpace(state.Source),
			SourceID: strings.TrimSpace(state.SourceID),
		}
		for _, trigger := range state.ClearTriggers {
			entry.ClearTriggers = append(entry.ClearTriggers, rules.ConditionClearTrigger(strings.TrimSpace(trigger)))
		}
		result = append(result, entry)
	}
	return result
}

func DomainConditionStatesToViews(states []rules.ConditionState) []*ConditionStateView {
	if len(states) == 0 {
		return nil
	}
	result := make([]*ConditionStateView, 0, len(states))
	for _, state := range states {
		result = append(result, &ConditionStateView{
			ID:            state.ID,
			Class:         conditionClassToProto(string(state.Class)),
			Standard:      standardConditionProtoOrZero(state.Standard),
			Code:          state.Code,
			Label:         state.Label,
			Source:        state.Source,
			SourceID:      state.SourceID,
			ClearTriggers: domainClearTriggersToProto(state.ClearTriggers),
		})
	}
	return result
}

func ConditionStateViewsToDomain(states []*ConditionStateView) ([]rules.ConditionState, error) {
	if len(states) == 0 {
		return nil, nil
	}
	result := make([]rules.ConditionState, 0, len(states))
	for _, state := range states {
		if state == nil {
			continue
		}
		entry := rules.ConditionState{
			ID:       strings.TrimSpace(state.ID),
			Class:    conditionClassFromProto(state.Class),
			Code:     strings.TrimSpace(state.Code),
			Label:    strings.TrimSpace(state.Label),
			Source:   strings.TrimSpace(state.Source),
			SourceID: strings.TrimSpace(state.SourceID),
		}
		if entry.Class == rules.ConditionClassStandard {
			standard, err := standardConditionCode(state.Standard)
			if err != nil {
				return nil, err
			}
			entry.Standard = standard
			if entry.Code == "" {
				entry.Code = standard
			}
		}
		for _, trigger := range state.ClearTriggers {
			domainTrigger, err := clearTriggerFromProto(trigger)
			if err != nil {
				return nil, err
			}
			entry.ClearTriggers = append(entry.ClearTriggers, domainTrigger)
		}
		result = append(result, entry)
	}
	return rules.NormalizeConditionStates(result)
}

func ConditionsToViews(conditions []string, source string) []*ConditionStateView {
	if len(conditions) == 0 {
		return nil
	}
	result := make([]*ConditionStateView, 0, len(conditions))
	for _, condition := range conditions {
		standard, err := standardConditionProto(condition)
		if err != nil {
			result = append(result, &ConditionStateView{
				ID:     condition,
				Class:  pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_SPECIAL,
				Code:   condition,
				Label:  condition,
				Source: source,
			})
			continue
		}
		result = append(result, &ConditionStateView{
			ID:       condition,
			Class:    pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD,
			Standard: standard,
			Code:     condition,
			Label:    condition,
			Source:   source,
		})
	}
	return result
}

func ConditionStateViewsToCodes(states []*ConditionStateView) []string {
	if len(states) == 0 {
		return nil
	}
	result := make([]string, 0, len(states))
	for _, state := range states {
		if state == nil {
			continue
		}
		result = append(result, state.Code)
	}
	return result
}

func conditionStateFromProto(state *pb.DaggerheartConditionState) (*ConditionStateView, error) {
	if state == nil {
		return nil, fmt.Errorf("condition state is required")
	}
	id := strings.TrimSpace(state.GetId())
	if id == "" {
		return nil, fmt.Errorf("condition id is required")
	}
	class := state.GetClass()
	if class == pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_UNSPECIFIED {
		return nil, fmt.Errorf("condition class is required")
	}
	code := strings.TrimSpace(state.GetCode())
	label := strings.TrimSpace(state.GetLabel())
	switch class {
	case pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD:
		codeFromStandard, err := standardConditionCode(state.GetStandard())
		if err != nil {
			return nil, err
		}
		if code == "" {
			code = codeFromStandard
		}
		if label == "" {
			label = codeFromStandard
		}
	case pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_TAG, pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_SPECIAL:
		if code == "" {
			return nil, fmt.Errorf("condition code is required")
		}
		if label == "" {
			label = code
		}
	default:
		return nil, fmt.Errorf("condition class %v is invalid", class)
	}
	for _, trigger := range state.GetClearTriggers() {
		if trigger == pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_UNSPECIFIED {
			return nil, fmt.Errorf("condition clear trigger is required")
		}
	}
	return &ConditionStateView{
		ID:            id,
		Class:         class,
		Standard:      state.GetStandard(),
		Code:          code,
		Label:         label,
		Source:        strings.TrimSpace(state.GetSource()),
		SourceID:      strings.TrimSpace(state.GetSourceId()),
		ClearTriggers: append([]pb.DaggerheartConditionClearTrigger(nil), state.GetClearTriggers()...),
	}, nil
}

func standardConditionCode(condition pb.DaggerheartCondition) (string, error) {
	switch condition {
	case pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN:
		return rules.ConditionHidden, nil
	case pb.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED:
		return rules.ConditionRestrained, nil
	case pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE:
		return rules.ConditionVulnerable, nil
	case pb.DaggerheartCondition_DAGGERHEART_CONDITION_CLOAKED:
		return rules.ConditionCloaked, nil
	case pb.DaggerheartCondition_DAGGERHEART_CONDITION_UNSPECIFIED:
		return "", fmt.Errorf("condition standard is required")
	default:
		return "", fmt.Errorf("condition standard %v is invalid", condition)
	}
}

func standardConditionProto(condition string) (pb.DaggerheartCondition, error) {
	switch condition {
	case rules.ConditionHidden:
		return pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN, nil
	case rules.ConditionRestrained:
		return pb.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED, nil
	case rules.ConditionVulnerable:
		return pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE, nil
	case rules.ConditionCloaked:
		return pb.DaggerheartCondition_DAGGERHEART_CONDITION_CLOAKED, nil
	default:
		return pb.DaggerheartCondition_DAGGERHEART_CONDITION_UNSPECIFIED, fmt.Errorf("condition %q is invalid", condition)
	}
}

func standardConditionProtoOrZero(condition string) pb.DaggerheartCondition {
	value, err := standardConditionProto(strings.TrimSpace(condition))
	if err != nil {
		return pb.DaggerheartCondition_DAGGERHEART_CONDITION_UNSPECIFIED
	}
	return value
}

func conditionClassToProto(value string) pb.DaggerheartConditionClass {
	switch strings.TrimSpace(value) {
	case string(rules.ConditionClassStandard):
		return pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD
	case string(rules.ConditionClassTag):
		return pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_TAG
	case string(rules.ConditionClassSpecial):
		return pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_SPECIAL
	default:
		return pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_UNSPECIFIED
	}
}

func conditionClassFromProto(value pb.DaggerheartConditionClass) rules.ConditionClass {
	switch value {
	case pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD:
		return rules.ConditionClassStandard
	case pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_TAG:
		return rules.ConditionClassTag
	case pb.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_SPECIAL:
		return rules.ConditionClassSpecial
	default:
		return ""
	}
}

func clearTriggerFromProto(trigger pb.DaggerheartConditionClearTrigger) (rules.ConditionClearTrigger, error) {
	switch trigger {
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST:
		return rules.ConditionClearTriggerShortRest, nil
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST:
		return rules.ConditionClearTriggerLongRest, nil
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SESSION_END:
		return rules.ConditionClearTriggerSessionEnd, nil
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN:
		return rules.ConditionClearTriggerDamageTaken, nil
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_UNSPECIFIED:
		return "", fmt.Errorf("condition clear trigger is required")
	default:
		return "", fmt.Errorf("condition clear trigger %v is invalid", trigger)
	}
}

func projectionClearTriggersToProto(values []string) []pb.DaggerheartConditionClearTrigger {
	if len(values) == 0 {
		return nil
	}
	result := make([]pb.DaggerheartConditionClearTrigger, 0, len(values))
	for _, value := range values {
		result = append(result, domainClearTriggerToProto(rules.ConditionClearTrigger(strings.TrimSpace(value))))
	}
	return result
}

func domainClearTriggersToProto(values []rules.ConditionClearTrigger) []pb.DaggerheartConditionClearTrigger {
	if len(values) == 0 {
		return nil
	}
	result := make([]pb.DaggerheartConditionClearTrigger, 0, len(values))
	for _, value := range values {
		result = append(result, domainClearTriggerToProto(value))
	}
	return result
}

func domainClearTriggerToProto(value rules.ConditionClearTrigger) pb.DaggerheartConditionClearTrigger {
	switch value {
	case rules.ConditionClearTriggerShortRest:
		return pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST
	case rules.ConditionClearTriggerLongRest:
		return pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST
	case rules.ConditionClearTriggerSessionEnd:
		return pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SESSION_END
	case rules.ConditionClearTriggerDamageTaken:
		return pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN
	default:
		return pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_UNSPECIFIED
	}
}

func lifeStateFromProto(state pb.DaggerheartLifeState) (string, error) {
	switch state {
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED:
		return "", fmt.Errorf("life_state is required")
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE:
		return daggerheartstate.LifeStateAlive, nil
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS:
		return mechanics.LifeStateUnconscious, nil
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY:
		return mechanics.LifeStateBlazeOfGlory, nil
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD:
		return mechanics.LifeStateDead, nil
	default:
		return "", fmt.Errorf("life_state %v is invalid", state)
	}
}

// LifeStateToProto maps stored life-state strings into the public gRPC enum so
// root response shaping does not retain duplicate life-state helpers.
func LifeStateToProto(state string) pb.DaggerheartLifeState {
	switch state {
	case daggerheartstate.LifeStateAlive:
		return pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE
	case mechanics.LifeStateUnconscious:
		return pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS
	case mechanics.LifeStateBlazeOfGlory:
		return pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY
	case mechanics.LifeStateDead:
		return pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD
	default:
		return pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED
	}
}
