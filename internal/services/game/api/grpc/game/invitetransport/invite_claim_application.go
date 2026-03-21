package invitetransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/inviteclaimworkflow"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a inviteApplication) ClaimInvite(ctx context.Context, campaignID string, in *campaignv1.ClaimInviteRequest) (storage.InviteRecord, storage.ParticipantRecord, error) {
	inviteID, err := validate.RequiredID(in.GetInviteId(), "invite id")
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	if _, err := validate.RequiredID(in.GetJoinGrant(), "join grant"); err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	userID, err := validate.RequiredID(grpcmeta.UserIDFromContext(ctx), "user id")
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}

	inv, err := a.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	if inv.CampaignID != campaignID {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "invite campaign does not match")
	}
	if recipient := strings.TrimSpace(inv.RecipientUserID); recipient != "" && recipient != userID {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.PermissionDenied, "invite recipient does not match")
	}
	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}

	claims, err := a.joinGrantVerifier.Validate(in.GetJoinGrant(), joingrant.Expectation{
		CampaignID: campaignID,
		InviteID:   inv.ID,
		UserID:     userID,
	})
	if err != nil {
		if errors.Is(err, joingrant.ErrVerifierNotConfigured) {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("join grant validation is not configured", err)
		}
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, err
		}
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("validate join grant", err)
	}
	if a.stores.ClaimIndex != nil {
		claim, err := a.stores.ClaimIndex.GetParticipantClaim(ctx, campaignID, userID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load participant claim", err)
		}
		if err == nil && claim.ParticipantID != inv.ParticipantID {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, apperrors.WithMetadata(
				apperrors.CodeParticipantUserAlreadyClaimed,
				"participant user already claimed",
				map[string]string{
					"CampaignID": campaignID,
					"UserID":     userID,
				},
			)
		}
	}
	claimEvent, err := findInviteClaimByJTI(ctx, a.stores.Event, campaignID, claims.JWTID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("lookup join grant", err)
	}
	if claimEvent != nil {
		var payload invite.ClaimPayload
		if err := json.Unmarshal(claimEvent.PayloadJSON, &payload); err != nil {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("decode prior claim", err)
		}
		if payload.InviteID != ids.InviteID(inv.ID) || payload.UserID != ids.UserID(userID) {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, apperrors.New(apperrors.CodeInviteJoinGrantUsed, "join grant already used")
		}
		updatedInvite, err := a.stores.Invite.GetInvite(ctx, inv.ID)
		if err != nil {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load invite", err)
		}
		updatedParticipant, err := a.stores.Participant.GetParticipant(ctx, campaignID, inv.ParticipantID)
		if err != nil {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load participant", err)
		}
		return updatedInvite, updatedParticipant, nil
	}
	if inv.Status == invite.StatusClaimed {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "invite already claimed")
	}
	if inv.Status == invite.StatusDeclined {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "invite already declined")
	}
	if inv.Status == invite.StatusRevoked {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "invite already revoked")
	}

	seat, err := a.stores.Participant.GetParticipant(ctx, campaignID, inv.ParticipantID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	inviteState, err := loadInviteReplayState(ctx, a.stores.Event, campaignID, inv.ID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load invite state", err)
	}
	if !inviteState.Created {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.NotFound, "invite not found")
	}
	if inviteState.Status == invite.StatusClaimed {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "invite already claimed")
	}
	if inviteState.Status == invite.StatusDeclined {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "invite already declined")
	}
	if inviteState.Status == invite.StatusRevoked {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "invite already revoked")
	}

	participantStates, err := loadCampaignParticipantReplayStates(ctx, a.stores.Event, campaignID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load participant bindings", err)
	}
	if claimedParticipantID, ok := findClaimedParticipantForUser(participantStates, userID); ok && claimedParticipantID != inv.ParticipantID {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, apperrors.WithMetadata(
			apperrors.CodeParticipantUserAlreadyClaimed,
			"participant user already claimed",
			map[string]string{
				"CampaignID": campaignID,
				"UserID":     userID,
			},
		)
	}

	seatState, err := loadParticipantReplayState(ctx, a.stores.Event, campaignID, inv.ParticipantID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load participant state", err)
	}
	if participantStateHasActiveUserBinding(seatState) {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "participant already claimed")
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	payloadJSON, err := json.Marshal(inviteclaimworkflow.ClaimBindPayload{
		InviteID:      ids.InviteID(inv.ID),
		ParticipantID: ids.ParticipantID(seat.ID),
		UserID:        ids.UserID(userID),
		JWTID:         claims.JWTID,
	})
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("encode invite claim workflow payload", err)
	}

	actorID, actorType := handler.ResolveCommandActor(ctx)
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         handler.CommandTypeInviteClaimBind,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "invite",
			EntityID:     inv.ID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: handler.ApplyErrorWithCodePreserve("apply invite claim workflow event"),
		},
	)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}

	updatedInvite, err := a.stores.Invite.GetInvite(ctx, inv.ID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load invite", err)
	}
	applyParticipantProfileSnapshot(
		ctx,
		a.write,
		a.applier,
		a.stores.Participant,
		a.stores.Character,
		a.stores.Social,
		campaignID,
		seat.ID,
		userID,
		requestID,
		invocationID,
		actorID,
		actorType,
	)
	updatedParticipant, err := a.stores.Participant.GetParticipant(ctx, campaignID, seat.ID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load participant", err)
	}

	return updatedInvite, updatedParticipant, nil
}
