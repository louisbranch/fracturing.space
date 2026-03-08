package game

import (
	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a inviteApplication) RevokeInvite(ctx context.Context, in *campaignv1.RevokeInviteRequest) (storage.InviteRecord, error) {
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return storage.InviteRecord{}, status.Error(codes.InvalidArgument, "invite id is required")
	}

	inv, err := a.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return storage.InviteRecord{}, err
	}
	campaignRecord, err := a.stores.Campaign.Get(ctx, inv.CampaignID)
	if err != nil {
		return storage.InviteRecord{}, err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageInvites, campaignRecord); err != nil {
		return storage.InviteRecord{}, err
	}
	if inv.Status == invite.StatusRevoked {
		return storage.InviteRecord{}, status.Error(codes.FailedPrecondition, "invite already revoked")
	}
	if inv.Status == invite.StatusClaimed {
		return storage.InviteRecord{}, status.Error(codes.FailedPrecondition, "invite already claimed")
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	applier := a.stores.Applier()
	payload := invite.RevokePayload{InviteID: ids.InviteID(inv.ID)}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.InviteRecord{}, status.Errorf(codes.Internal, "encode invite payload: %v", err)
	}
	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   inv.CampaignID,
			Type:         commandTypeInviteRevoke,
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
		return storage.InviteRecord{}, status.Errorf(codes.Internal, "load invite: %v", err)
	}

	return updated, nil
}
