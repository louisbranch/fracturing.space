package participanttransport

import (
	"context"
	"encoding/json"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BindParticipant binds a user to an unoccupied participant seat. The caller
// (invite service) is trusted to have verified authorization. This handler
// enforces participant-level preconditions and the one-user-per-campaign claim
// index constraint.
func (c participantApplication) BindParticipant(ctx context.Context, campaignID, participantID, userID string) (storage.ParticipantRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.ParticipantRecord{}, err
	}

	// Claim index: reject if user already holds a different seat in this campaign.
	if c.claimIndex != nil {
		claim, err := c.claimIndex.GetParticipantClaim(ctx, campaignID, userID)
		if lookupErr := grpcerror.OptionalLookupErrorContext(ctx, err, "load participant claim"); lookupErr != nil {
			return storage.ParticipantRecord{}, lookupErr
		}
		if err == nil && claim.ParticipantID != participantID {
			return storage.ParticipantRecord{}, apperrors.WithMetadata(
				apperrors.CodeParticipantUserAlreadyClaimed,
				"participant user already claimed",
				map[string]string{
					"CampaignID": campaignID,
					"UserID":     userID,
				},
			)
		}
		// Idempotent: same user already bound to this seat — return current state.
		if err == nil && claim.ParticipantID == participantID {
			current, getErr := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
			if getErr != nil {
				return storage.ParticipantRecord{}, grpcerror.Internal("load participant", getErr)
			}
			if strings.TrimSpace(current.UserID) == userID {
				return current, nil
			}
		}
	}

	// Replay authoritative participant state to detect binding conflicts the
	// projection may not yet reflect.
	if c.eventStore != nil {
		seatState, err := replayParticipantState(ctx, c.eventStore, campaignID, participantID)
		if err != nil {
			return storage.ParticipantRecord{}, grpcerror.Internal("load participant state", err)
		}
		if seatState.Joined && !seatState.Left && strings.TrimSpace(seatState.UserID.String()) != "" {
			if seatState.UserID.String() == userID {
				// Idempotent: already bound to this user.
				current, getErr := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
				if getErr != nil {
					return storage.ParticipantRecord{}, grpcerror.Internal("load participant", getErr)
				}
				return current, nil
			}
			return storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "participant already claimed")
		}
	}

	payload := participant.BindPayload{
		ParticipantID: ids.ParticipantID(participantID),
		UserID:        ids.UserID(userID),
	}
	payloadJSON, _ := json.Marshal(payload)

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	actorID, actorType := handler.ResolveCommandActor(ctx)

	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		c.write,
		c.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         handler.CommandTypeParticipantBind,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "participant",
			EntityID:     participantID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: handler.ApplyErrorWithCodePreserve("apply participant bind event"),
		},
	)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}

	// Best-effort: hydrate the seat with the user's social profile so the
	// participant immediately displays the claimer's name, pronouns, and avatar.
	handler.ApplyParticipantProfileSnapshot(
		ctx,
		c.write,
		c.applier,
		c.stores.Participant,
		c.stores.Character,
		c.stores.Social,
		campaignID,
		participantID,
		userID,
		requestID,
		invocationID,
		actorID,
		actorType,
	)

	updated, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, grpcerror.Internal("load participant", err)
	}

	return updated, nil
}
