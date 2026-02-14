package daggerheart

import (
	"fmt"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

func daggerheartLifeStateFromProto(state pb.DaggerheartLifeState) (string, error) {
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

func daggerheartLifeStateToProto(state string) pb.DaggerheartLifeState {
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

func daggerheartDeathMoveFromProto(move pb.DaggerheartDeathMove) (string, error) {
	switch move {
	case pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED:
		return "", fmt.Errorf("death move is required")
	case pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_BLAZE_OF_GLORY:
		return daggerheart.DeathMoveBlazeOfGlory, nil
	case pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH:
		return daggerheart.DeathMoveAvoidDeath, nil
	case pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL:
		return daggerheart.DeathMoveRiskItAll, nil
	default:
		return "", fmt.Errorf("death move %v is invalid", move)
	}
}

func daggerheartDeathMoveToProto(move string) pb.DaggerheartDeathMove {
	switch move {
	case daggerheart.DeathMoveBlazeOfGlory:
		return pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_BLAZE_OF_GLORY
	case daggerheart.DeathMoveAvoidDeath:
		return pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_AVOID_DEATH
	case daggerheart.DeathMoveRiskItAll:
		return pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_RISK_IT_ALL
	default:
		return pb.DaggerheartDeathMove_DAGGERHEART_DEATH_MOVE_UNSPECIFIED
	}
}
