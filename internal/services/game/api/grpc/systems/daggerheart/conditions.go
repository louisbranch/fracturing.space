package daggerheart

import (
	"fmt"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func daggerheartConditionsFromProto(conditions []pb.DaggerheartCondition) ([]string, error) {
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

func daggerheartConditionsToProto(conditions []string) []pb.DaggerheartCondition {
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
