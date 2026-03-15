package workfloweffects

import (
	"context"
	"errors"
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AdvanceBreathCountdown creates the breath countdown on demand and advances it
// according to the roll outcome.
func (h *Handler) AdvanceBreathCountdown(ctx context.Context, campaignID, sessionID, countdownID string, failed bool) error {
	countdownID = strings.TrimSpace(countdownID)
	if countdownID == "" {
		return nil
	}
	if h == nil || h.deps.Daggerheart == nil || h.deps.CreateCountdown == nil || h.deps.UpdateCountdown == nil {
		return status.Error(codes.Internal, "workflow effects dependencies are not configured")
	}

	if _, err := h.deps.Daggerheart.GetDaggerheartCountdown(ctx, campaignID, countdownID); err != nil {
		if !errors.Is(err, storage.ErrNotFound) {
			return grpcerror.HandleDomainError(err)
		}
		createErr := h.deps.CreateCountdown(ctx, &pb.DaggerheartCreateCountdownRequest{
			CampaignId:  campaignID,
			SessionId:   sessionID,
			CountdownId: countdownID,
			Name:        daggerheart.BreathCountdownName,
			Kind:        pb.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_CONSEQUENCE,
			Current:     daggerheart.BreathCountdownInitial,
			Max:         daggerheart.BreathCountdownMax,
			Direction:   pb.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
			Looping:     false,
		})
		if createErr != nil && status.Code(createErr) != codes.FailedPrecondition {
			return createErr
		}
	}

	advance := daggerheart.ResolveBreathCountdownAdvance(failed)
	return h.deps.UpdateCountdown(ctx, &pb.DaggerheartUpdateCountdownRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CountdownId: countdownID,
		Delta:       int32(advance.Delta),
		Reason:      advance.Reason,
	})
}
