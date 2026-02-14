package daggerheart

import (
	"fmt"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func daggerheartCountdownKindFromProto(kind pb.DaggerheartCountdownKind) (string, error) {
	switch kind {
	case pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS:
		return daggerheart.CountdownKindProgress, nil
	case pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_CONSEQUENCE:
		return daggerheart.CountdownKindConsequence, nil
	case pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_UNSPECIFIED:
		return "", fmt.Errorf("countdown kind is required")
	default:
		return "", fmt.Errorf("countdown kind %v is invalid", kind)
	}
}

func daggerheartCountdownKindToProto(kind string) pb.DaggerheartCountdownKind {
	switch kind {
	case daggerheart.CountdownKindProgress:
		return pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS
	case daggerheart.CountdownKindConsequence:
		return pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_CONSEQUENCE
	default:
		return pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_UNSPECIFIED
	}
}

func daggerheartCountdownDirectionFromProto(direction pb.DaggerheartCountdownDirection) (string, error) {
	switch direction {
	case pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE:
		return daggerheart.CountdownDirectionIncrease, nil
	case pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_DECREASE:
		return daggerheart.CountdownDirectionDecrease, nil
	case pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_UNSPECIFIED:
		return "", fmt.Errorf("countdown direction is required")
	default:
		return "", fmt.Errorf("countdown direction %v is invalid", direction)
	}
}

func daggerheartCountdownDirectionToProto(direction string) pb.DaggerheartCountdownDirection {
	switch direction {
	case daggerheart.CountdownDirectionIncrease:
		return pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE
	case daggerheart.CountdownDirectionDecrease:
		return pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_DECREASE
	default:
		return pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_UNSPECIFIED
	}
}

func daggerheartCountdownToProto(countdown storage.DaggerheartCountdown) *pb.DaggerheartCountdown {
	return &pb.DaggerheartCountdown{
		CountdownId: countdown.CountdownID,
		Name:        countdown.Name,
		Kind:        daggerheartCountdownKindToProto(countdown.Kind),
		Current:     int32(countdown.Current),
		Max:         int32(countdown.Max),
		Direction:   daggerheartCountdownDirectionToProto(countdown.Direction),
		Looping:     countdown.Looping,
	}
}
