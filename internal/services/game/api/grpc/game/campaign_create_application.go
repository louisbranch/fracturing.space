package game

import (
	"context"
	"encoding/json"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c campaignApplication) CreateCampaign(ctx context.Context, in *campaignv1.CreateCampaignRequest) (storage.CampaignRecord, storage.ParticipantRecord, error) {
	gmMode := in.GetGmMode()
	switch gmMode {
	case campaignv1.GmMode_GM_MODE_UNSPECIFIED, campaignv1.GmMode_AI, campaignv1.GmMode_HUMAN, campaignv1.GmMode_HYBRID:
	default:
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "campaign gm mode is invalid")
	}

	input := campaign.CreateInput{
		Name:         in.GetName(),
		Locale:       platformi18n.LocaleString(in.GetLocale()),
		System:       campaign.GameSystem(in.GetSystem().String()),
		GmMode:       gmModeFromProto(gmMode),
		Intent:       campaignIntentFromProto(in.GetIntent()),
		AccessPolicy: campaignAccessPolicyFromProto(in.GetAccessPolicy()),
		ThemePrompt:  in.GetThemePrompt(),
	}
	if in.GetSystem() == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "game system is required")
	}

	normalized, err := campaign.NormalizeCreateInput(input)
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, err
	}

	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, apperrors.New(
			apperrors.CodeCampaignCreatorUserMissing,
			"creator user id is required",
		)
	}

	campaignID, err := c.idGenerator()
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "generate campaign id: %v", err)
	}

	creatorID, err := c.idGenerator()
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "generate participant id: %v", err)
	}

	campaignPayload := campaign.CreatePayload{
		Name:         normalized.Name,
		Locale:       normalized.Locale,
		GameSystem:   normalized.System.String(),
		GmMode:       gmModeToProto(normalized.GmMode).String(),
		Intent:       campaignIntentToProto(normalized.Intent).String(),
		AccessPolicy: campaignAccessPolicyToProto(normalized.AccessPolicy).String(),
		ThemePrompt:  normalized.ThemePrompt,
	}

	defaultLocale, ok := platformi18n.ParseLocale(normalized.Locale)
	if !ok {
		defaultLocale = platformi18n.DefaultLocale()
	}

	profile := loadSocialProfileSnapshot(ctx, c.stores.Social, userID)
	creatorDisplayName := strings.TrimSpace(profile.Name)
	if creatorDisplayName == "" {
		creatorDisplayName = defaultUnknownParticipantName(defaultLocale)
	}
	creatorPronouns := strings.TrimSpace(profile.Pronouns)
	if creatorPronouns == "" && userID != "" {
		creatorPronouns = defaultUnknownParticipantPronouns()
	}

	creatorRole := "GM"
	if normalized.GmMode == campaign.GmModeAI {
		creatorRole = "PLAYER"
	}

	participantPayloads := []participant.JoinPayload{
		{
			ParticipantID:  ids.ParticipantID(creatorID),
			UserID:         ids.UserID(userID),
			Name:           creatorDisplayName,
			Role:           creatorRole,
			Controller:     "HUMAN",
			CampaignAccess: "OWNER",
			AvatarSetID:    profile.AvatarSetID,
			AvatarAssetID:  profile.AvatarAssetID,
			Pronouns:       creatorPronouns,
		},
	}
	if normalized.GmMode == campaign.GmModeAI || normalized.GmMode == campaign.GmModeHybrid {
		aiParticipantID, err := c.idGenerator()
		if err != nil {
			return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "generate ai participant id: %v", err)
		}
		participantPayloads = append(participantPayloads, participant.JoinPayload{
			ParticipantID:  ids.ParticipantID(aiParticipantID),
			UserID:         "",
			Name:           defaultAIParticipantName(defaultLocale),
			Role:           "GM",
			Controller:     "AI",
			CampaignAccess: "MEMBER",
			Pronouns:       defaultAIParticipantPronouns(),
		})
	}

	workflowPayloadJSON, err := json.Marshal(campaign.CreateWithParticipantsPayload{
		Campaign:     campaignPayload,
		Participants: participantPayloads,
	})
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "encode create workflow payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Write,
		c.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCampaignCreateWithParticipants,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "campaign",
			EntityID:     campaignID,
			PayloadJSON:  workflowPayloadJSON,
		}),
		domainwrite.Options{},
	)
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, err
	}

	ownerParticipant, err := c.stores.Participant.GetParticipant(ctx, campaignID, creatorID)
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "load owner participant: %v", err)
	}

	created, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "load campaign: %v", err)
	}

	return created, ownerParticipant, nil
}
