package countdowntransport

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func campaignSupportsDaggerheart(record storage.CampaignRecord) bool {
	systemID, ok := systembridge.NormalizeSystemID(record.System.String())
	return ok && systemID == systembridge.SystemIDDaggerheart
}

func requireDaggerheartSystem(record storage.CampaignRecord, unsupportedMessage string) error {
	if campaignSupportsDaggerheart(record) {
		return nil
	}
	return status.Error(codes.FailedPrecondition, unsupportedMessage)
}

func ensureNoOpenSessionGate(ctx context.Context, store SessionGateStore, campaignID, sessionID string) error {
	if store == nil || strings.TrimSpace(campaignID) == "" || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	gate, err := store.GetOpenSessionGate(ctx, campaignID, sessionID)
	if err == nil {
		return status.Errorf(codes.FailedPrecondition, "session gate is open: %s", gate.GateID)
	}
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	return grpcerror.Internal("load session gate", err)
}

func handleDomainError(err error) error {
	return grpcerror.HandleDomainError(err)
}
