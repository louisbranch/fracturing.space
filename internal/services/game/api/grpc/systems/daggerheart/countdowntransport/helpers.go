package countdowntransport

import (
	"fmt"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

func countdownKindFromProto(kind pb.DaggerheartCountdownKind) (string, error) {
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

func countdownDirectionFromProto(direction pb.DaggerheartCountdownDirection) (string, error) {
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

func countdownKindToProto(kind string) pb.DaggerheartCountdownKind {
	switch kind {
	case daggerheart.CountdownKindProgress:
		return pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS
	case daggerheart.CountdownKindConsequence:
		return pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_CONSEQUENCE
	default:
		return pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_UNSPECIFIED
	}
}

func countdownDirectionToProto(direction string) pb.DaggerheartCountdownDirection {
	switch direction {
	case daggerheart.CountdownDirectionIncrease:
		return pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE
	case daggerheart.CountdownDirectionDecrease:
		return pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_DECREASE
	default:
		return pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_UNSPECIFIED
	}
}

func countdownFromStorage(countdown projectionstore.DaggerheartCountdown) daggerheart.Countdown {
	return daggerheart.Countdown{
		CampaignID: countdown.CampaignID,
		ID:         countdown.CountdownID,
		Name:       countdown.Name,
		Kind:       countdown.Kind,
		Current:    countdown.Current,
		Max:        countdown.Max,
		Direction:  countdown.Direction,
		Looping:    countdown.Looping,
	}
}

// CountdownToProto maps stored countdown state into the public gRPC response
// shape so root wrappers do not retain duplicate countdown transport helpers.
func CountdownToProto(countdown projectionstore.DaggerheartCountdown) *pb.DaggerheartCountdown {
	return &pb.DaggerheartCountdown{
		CountdownId: countdown.CountdownID,
		Name:        countdown.Name,
		Kind:        countdownKindToProto(countdown.Kind),
		Current:     int32(countdown.Current),
		Max:         int32(countdown.Max),
		Direction:   countdownDirectionToProto(countdown.Direction),
		Looping:     countdown.Looping,
	}
}
