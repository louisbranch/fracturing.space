package conditiontransport

import (
	"fmt"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
)

func ConditionsFromProto(conditions []pb.DaggerheartCondition) ([]string, error) {
	if len(conditions) == 0 {
		return nil, nil
	}
	result := make([]string, 0, len(conditions))
	for _, condition := range conditions {
		switch condition {
		case pb.DaggerheartCondition_DAGGERHEART_CONDITION_UNSPECIFIED:
			return nil, fmt.Errorf("condition is required")
		case pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN:
			result = append(result, daggerheart.ConditionHidden)
		case pb.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED:
			result = append(result, daggerheart.ConditionRestrained)
		case pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE:
			result = append(result, daggerheart.ConditionVulnerable)
		default:
			return nil, fmt.Errorf("condition %v is invalid", condition)
		}
	}
	return result, nil
}

func ConditionsToProto(conditions []string) []pb.DaggerheartCondition {
	if len(conditions) == 0 {
		return nil
	}
	result := make([]pb.DaggerheartCondition, 0, len(conditions))
	for _, condition := range conditions {
		switch condition {
		case daggerheart.ConditionHidden:
			result = append(result, pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)
		case daggerheart.ConditionRestrained:
			result = append(result, pb.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED)
		case daggerheart.ConditionVulnerable:
			result = append(result, pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)
		}
	}
	return result
}

func lifeStateFromProto(state pb.DaggerheartLifeState) (string, error) {
	switch state {
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED:
		return "", fmt.Errorf("life_state is required")
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE:
		return daggerheart.LifeStateAlive, nil
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS:
		return daggerheart.LifeStateUnconscious, nil
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY:
		return daggerheart.LifeStateBlazeOfGlory, nil
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD:
		return daggerheart.LifeStateDead, nil
	default:
		return "", fmt.Errorf("life_state %v is invalid", state)
	}
}

// LifeStateToProto maps stored life-state strings into the public gRPC enum so
// root response shaping does not retain duplicate life-state helpers.
func LifeStateToProto(state string) pb.DaggerheartLifeState {
	switch state {
	case daggerheart.LifeStateAlive:
		return pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE
	case daggerheart.LifeStateUnconscious:
		return pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS
	case daggerheart.LifeStateBlazeOfGlory:
		return pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY
	case daggerheart.LifeStateDead:
		return pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD
	default:
		return pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED
	}
}
