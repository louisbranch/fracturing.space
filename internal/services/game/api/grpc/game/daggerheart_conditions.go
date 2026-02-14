package game

import (
	"fmt"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func daggerheartConditionsFromProto(conditions []daggerheartv1.DaggerheartCondition) ([]string, error) {
	if len(conditions) == 0 {
		return []string{}, nil
	}

	result := make([]string, 0, len(conditions))
	for _, condition := range conditions {
		switch condition {
		case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_UNSPECIFIED:
			return nil, fmt.Errorf("condition is required")
		case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN:
			result = append(result, daggerheart.ConditionHidden)
		case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED:
			result = append(result, daggerheart.ConditionRestrained)
		case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE:
			result = append(result, daggerheart.ConditionVulnerable)
		default:
			return nil, fmt.Errorf("condition %v is invalid", condition)
		}
	}
	return result, nil
}

func daggerheartConditionsToProto(conditions []string) []daggerheartv1.DaggerheartCondition {
	if len(conditions) == 0 {
		return nil
	}
	result := make([]daggerheartv1.DaggerheartCondition, 0, len(conditions))
	for _, condition := range conditions {
		switch condition {
		case daggerheart.ConditionHidden:
			result = append(result, daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)
		case daggerheart.ConditionRestrained:
			result = append(result, daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED)
		case daggerheart.ConditionVulnerable:
			result = append(result, daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)
		}
	}
	return result
}

func daggerheartLifeStateFromProto(state daggerheartv1.DaggerheartLifeState) (string, error) {
	switch state {
	case daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED:
		return "", fmt.Errorf("life_state is required")
	case daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE:
		return daggerheart.LifeStateAlive, nil
	case daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS:
		return daggerheart.LifeStateUnconscious, nil
	case daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY:
		return daggerheart.LifeStateBlazeOfGlory, nil
	case daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD:
		return daggerheart.LifeStateDead, nil
	default:
		return "", fmt.Errorf("life_state %v is invalid", state)
	}
}

func daggerheartLifeStateToProto(state string) daggerheartv1.DaggerheartLifeState {
	switch state {
	case daggerheart.LifeStateAlive:
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE
	case daggerheart.LifeStateUnconscious:
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS
	case daggerheart.LifeStateBlazeOfGlory:
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY
	case daggerheart.LifeStateDead:
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD
	default:
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED
	}
}
