package game

import (
	"context"
	"encoding/json"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a inviteApplication) CreateInvite(ctx context.Context, campaignID string, in *campaignv1.CreateInviteRequest) (storage.InviteRecord, error) {
	participantID, err := validate.RequiredID(in.GetParticipantId(), "participant id")
	if err != nil {
		return storage.InviteRecord{}, err
	}
	recipientUserID := strings.TrimSpace(in.GetRecipientUserId())

	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.InviteRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.InviteRecord{}, err
	}
	actor, err := requirePolicyActor(ctx, a.auth, domainauthz.CapabilityManageInvites, campaignRecord)
	if err != nil {
		return storage.InviteRecord{}, err
	}
	if _, err := a.stores.Participant.GetParticipant(ctx, campaignID, participantID); err != nil {
		return storage.InviteRecord{}, err
	}
	if recipientUserID != "" {
		if a.authClient == nil {
			return storage.InviteRecord{}, status.Error(codes.Internal, "auth client is not configured")
		}
		userResponse, err := a.authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: recipientUserID})
		if err != nil {
			if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.NotFound {
				return storage.InviteRecord{}, apperrors.New(
					apperrors.CodeInviteRecipientUserMissing,
					"invite recipient user not found",
				)
			}
			return storage.InviteRecord{}, grpcerror.Internal("get auth user", err)
		}
		if userResponse == nil || userResponse.GetUser() == nil {
			return storage.InviteRecord{}, status.Error(codes.Internal, "auth user response is missing")
		}
		if err := a.ensureCreateInviteRecipientAvailable(ctx, campaignID, recipientUserID); err != nil {
			return storage.InviteRecord{}, err
		}
	}

	inviteID, err := a.idGenerator()
	if err != nil {
		return storage.InviteRecord{}, grpcerror.Internal("generate invite id", err)
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	payload := invite.CreatePayload{
		InviteID:               ids.InviteID(inviteID),
		ParticipantID:          ids.ParticipantID(participantID),
		RecipientUserID:        ids.UserID(recipientUserID),
		CreatedByParticipantID: ids.ParticipantID(strings.TrimSpace(actor.ID)),
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
			CampaignID:   campaignID,
			Type:         commandTypeInviteCreate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "invite",
			EntityID:     inviteID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: domainApplyErrorWithCodePreserve("apply invite event"),
		},
	)
	if err != nil {
		return storage.InviteRecord{}, err
	}

	inv, err := a.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return storage.InviteRecord{}, grpcerror.Internal("load invite", err)
	}

	return inv, nil
}

func (a inviteApplication) ensureCreateInviteRecipientAvailable(ctx context.Context, campaignID, recipientUserID string) error {
	campaignIDs, err := a.stores.Participant.ListCampaignIDsByUser(ctx, recipientUserID)
	if err != nil {
		return grpcerror.Internal("list recipient participant campaigns", err)
	}
	for _, existingCampaignID := range campaignIDs {
		if existingCampaignID != campaignID {
			continue
		}
		return apperrors.WithMetadata(
			apperrors.CodeParticipantUserAlreadyClaimed,
			"participant user already claimed",
			map[string]string{
				"CampaignID": campaignID,
				"UserID":     recipientUserID,
			},
		)
	}

	page, err := a.stores.Invite.ListInvites(ctx, campaignID, recipientUserID, invite.StatusPending, 1, "")
	if err != nil {
		return grpcerror.Internal("list recipient pending invites", err)
	}
	if len(page.Invites) == 0 {
		return nil
	}

	return apperrors.WithMetadata(
		apperrors.CodeInviteRecipientAlreadyInvited,
		"invite recipient already has a pending invite",
		map[string]string{
			"CampaignID": campaignID,
			"InviteID":   page.Invites[0].ID,
			"UserID":     recipientUserID,
		},
	)
}
