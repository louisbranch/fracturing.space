package game

import (
	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
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
	if err := requirePolicy(ctx, c.stores, domainauthz.CapabilityManageParticipants, campaignRecord); err != nil {
		return storage.ParticipantRecord{}, err
	}

	userID := strings.TrimSpace(in.GetUserId())
	profile := loadSocialProfileSnapshot(ctx, c.stores.Social, userID)

	name := strings.TrimSpace(in.GetName())
	if name == "" {
		if profile.Name != "" {
			name = profile.Name
		} else if userID != "" {
			name = defaultUnknownParticipantName(campaignRecord.Locale)
		}
	}
	if name == "" {
		return storage.ParticipantRecord{}, apperrors.New(apperrors.CodeParticipantEmptyDisplayName, "name is required")
	}
	role := participantRoleFromProto(in.GetRole())
	if role == participant.RoleUnspecified {
		return storage.ParticipantRecord{}, apperrors.New(apperrors.CodeParticipantInvalidRole, "participant role is required")
	}
	controller := controllerFromProto(in.GetController())
	if controller == participant.ControllerUnspecified {
		controller = participant.ControllerHuman
	}
	access := participant.CampaignAccessMember

	participantID, err := c.idGenerator()
	if err != nil {
		return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "generate participant id: %v", err)
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
		pronouns = defaultUnknownParticipantPronouns()
	}

	applier := c.stores.Applier()
	payload := participant.JoinPayload{
		ParticipantID:  participantID,
		UserID:         userID,
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
		return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeParticipantJoin,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "participant",
			EntityID:     participantID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: domainApplyErrorWithCodePreserve("apply event"),
		},
	)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}

	created, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "load participant: %v", err)
	}

	return created, nil
}
