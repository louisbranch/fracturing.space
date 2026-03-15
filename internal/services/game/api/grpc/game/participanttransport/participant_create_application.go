package participanttransport

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"

	"context"
	"encoding/json"
	"strings"

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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c participantApplication) CreateParticipant(ctx context.Context, campaignID string, in *campaignv1.CreateParticipantRequest) (storage.ParticipantRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.ParticipantRecord{}, err
	}
	policyActor, err := authz.RequirePolicyActor(ctx, c.auth, domainauthz.CapabilityManageParticipants(), campaignRecord)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}

	userID := strings.TrimSpace(in.GetUserId())
	profile := handler.LoadSocialProfileSnapshot(ctx, c.stores.Social, userID)

	name := strings.TrimSpace(in.GetName())
	if name == "" {
		if profile.Name != "" {
			name = profile.Name
		} else if userID != "" {
			name, err = handler.AuthUsername(
				ctx,
				c.authClient,
				userID,
				status.Error(codes.InvalidArgument, "participant user not found"),
			)
			if err != nil {
				return storage.ParticipantRecord{}, err
			}
		}
	}
	if name == "" {
		return storage.ParticipantRecord{}, apperrors.New(apperrors.CodeParticipantEmptyDisplayName, "name is required")
	}
	if err := validate.MaxLength(name, "name", validate.MaxNameLen); err != nil {
		return storage.ParticipantRecord{}, err
	}
	role := RoleFromProto(in.GetRole())
	if role == participant.RoleUnspecified {
		return storage.ParticipantRecord{}, apperrors.New(apperrors.CodeParticipantInvalidRole, "participant role is required")
	}
	controller := ControllerFromProto(in.GetController())
	if controller == participant.ControllerUnspecified {
		controller = participant.ControllerHuman
	}
	if disallowsHumanGMForCampaignGMMode(campaignRecord.GmMode, role, controller) {
		return storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "ai gm campaigns cannot create human gm participants")
	}
	access := CampaignAccessFromProto(in.GetCampaignAccess())
	if access == participant.CampaignAccessUnspecified {
		access = participant.CampaignAccessMember
	} else {
		ownerCount, err := authz.CountCampaignOwners(ctx, c.stores.Participant, campaignID)
		if err != nil {
			return storage.ParticipantRecord{}, err
		}
		decision := domainauthz.CanParticipantAccessChange(
			policyActor.CampaignAccess,
			participant.CampaignAccessUnspecified,
			access,
			ownerCount,
		)
		if !decision.Allowed {
			return storage.ParticipantRecord{}, participantPolicyDecisionError(decision.ReasonCode)
		}
	}

	participantID, err := c.idGenerator()
	if err != nil {
		return storage.ParticipantRecord{}, grpcerror.Internal("generate participant id", err)
	}
	avatarSetID := strings.TrimSpace(in.GetAvatarSetId())
	avatarAssetID := strings.TrimSpace(in.GetAvatarAssetId())
	if avatarSetID == "" && avatarAssetID == "" {
		avatarSetID = profile.AvatarSetID
		avatarAssetID = profile.AvatarAssetID
	}
	pronouns := sharedpronouns.FromProto(in.GetPronouns())
	if pronouns == "" {
		pronouns = profile.Pronouns
	}
	if pronouns == "" && userID != "" {
		pronouns = handler.DefaultUnknownParticipantPronouns()
	}

	payload := participant.JoinPayload{
		ParticipantID:  ids.ParticipantID(participantID),
		UserID:         ids.UserID(userID),
		Name:           name,
		Role:           string(role),
		Controller:     string(controller),
		CampaignAccess: string(access),
		AvatarSetID:    avatarSetID,
		AvatarAssetID:  avatarAssetID,
		Pronouns:       pronouns,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.ParticipantRecord{}, grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := handler.ResolveCommandActor(ctx)
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		c.write,
		c.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         handler.CommandTypeParticipantJoin,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "participant",
			EntityID:     participantID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: handler.ApplyErrorWithCodePreserve("apply event"),
		},
	)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}

	created, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, grpcerror.Internal("load participant", err)
	}

	return created, nil
}
