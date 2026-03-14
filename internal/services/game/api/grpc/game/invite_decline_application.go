package game

import (
	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a inviteApplication) DeclineInvite(ctx context.Context, in *campaignv1.DeclineInviteRequest) (storage.InviteRecord, error) {
	inviteID, err := validate.RequiredID(in.GetInviteId(), "invite id")
	if err != nil {
		return storage.InviteRecord{}, err
	}
	userID, err := validate.RequiredID(grpcmeta.UserIDFromContext(ctx), "user id")
	if err != nil {
		return storage.InviteRecord{}, err
	}

	inv, err := a.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return storage.InviteRecord{}, err
	}
	recipientUserID := strings.TrimSpace(inv.RecipientUserID)
	if recipientUserID == "" {
		return storage.InviteRecord{}, status.Error(codes.FailedPrecondition, "invite is not targeted")
	}
	if recipientUserID != userID {
		return storage.InviteRecord{}, status.Error(codes.PermissionDenied, "invite recipient does not match")
	}

	campaignRecord, err := a.stores.Campaign.Get(ctx, inv.CampaignID)
	if err != nil {
		return storage.InviteRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.InviteRecord{}, err
	}

	switch inv.Status {
	case invite.StatusClaimed:
		return storage.InviteRecord{}, status.Error(codes.FailedPrecondition, "invite already claimed")
	case invite.StatusDeclined:
		return storage.InviteRecord{}, status.Error(codes.FailedPrecondition, "invite already declined")
	case invite.StatusRevoked:
		return storage.InviteRecord{}, status.Error(codes.FailedPrecondition, "invite already revoked")
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	payload := invite.DeclinePayload{
		InviteID: ids.InviteID(inv.ID),
		UserID:   ids.UserID(userID),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.InviteRecord{}, grpcerror.Internal("encode invite payload", err)
	}
	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   inv.CampaignID,
			Type:         commandTypeInviteDecline,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "invite",
			EntityID:     inv.ID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: domainApplyErrorWithCodePreserve("apply invite event"),
		},
	)
	if err != nil {
		return storage.InviteRecord{}, err
	}

	updated, err := a.stores.Invite.GetInvite(ctx, inv.ID)
	if err != nil {
		return storage.InviteRecord{}, grpcerror.Internal("load invite", err)
	}
	a.notifyInviteDeclined(ctx, updated, userID)

	return updated, nil
}
