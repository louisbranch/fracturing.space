package game

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type campaignApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

func newCampaignApplication(service *CampaignService) campaignApplication {
	app := campaignApplication{stores: service.stores, clock: service.clock, idGenerator: service.idGenerator}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}

func (c campaignApplication) CreateCampaign(ctx context.Context, in *campaignv1.CreateCampaignRequest) (storage.CampaignRecord, storage.ParticipantRecord, error) {
	input := campaign.CreateInput{
		Name:         in.GetName(),
		Locale:       in.GetLocale(),
		System:       in.GetSystem(),
		GmMode:       gmModeFromProto(in.GetGmMode()),
		Intent:       campaignIntentFromProto(in.GetIntent()),
		AccessPolicy: campaignAccessPolicyFromProto(in.GetAccessPolicy()),
		ThemePrompt:  in.GetThemePrompt(),
	}
	if input.System == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
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

	applier := c.stores.Applier()
	if c.stores.Domain == nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := campaign.CreatePayload{
		Name:         normalized.Name,
		Locale:       platformi18n.LocaleString(normalized.Locale),
		GameSystem:   normalized.System.String(),
		GmMode:       gmModeToProto(normalized.GmMode).String(),
		Intent:       campaignIntentToProto(normalized.Intent).String(),
		AccessPolicy: campaignAccessPolicyToProto(normalized.AccessPolicy).String(),
		ThemePrompt:  normalized.ThemePrompt,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCampaignCreate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "campaign",
			EntityID:     campaignID,
			PayloadJSON:  payloadJSON,
		}),
		domainCommandApplyOptions{},
	)
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, err
	}

	creatorID, err := c.idGenerator()
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "generate participant id: %v", err)
	}

	profile := loadSocialProfileSnapshot(ctx, c.stores.Social, userID)
	creatorDisplayName := strings.TrimSpace(profile.Name)
	if creatorDisplayName == "" {
		creatorDisplayName = defaultUnknownParticipantName(normalized.Locale)
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
			ParticipantID:  creatorID,
			UserID:         userID,
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
			ParticipantID:  aiParticipantID,
			UserID:         "",
			Name:           defaultAIParticipantName(normalized.Locale),
			Role:           "GM",
			Controller:     "AI",
			CampaignAccess: "MEMBER",
			Pronouns:       defaultAIParticipantPronouns(),
		})
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	for _, participantPayload := range participantPayloads {
		participantPayloadJSON, err := json.Marshal(participantPayload)
		if err != nil {
			return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "encode participant payload: %v", err)
		}

		_, err = executeAndApplyDomainCommand(
			ctx,
			c.stores.Domain,
			applier,
			commandbuild.Core(commandbuild.CoreInput{
				CampaignID:   campaignID,
				Type:         commandTypeParticipantJoin,
				ActorType:    command.ActorTypeSystem,
				ActorID:      "",
				RequestID:    requestID,
				InvocationID: invocationID,
				EntityType:   "participant",
				EntityID:     participantPayload.ParticipantID,
				PayloadJSON:  participantPayloadJSON,
			}),
			domainCommandApplyOptions{
				applyErrMessage: "apply participant event",
			},
		)
		if err != nil {
			return storage.CampaignRecord{}, storage.ParticipantRecord{}, err
		}
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

func (c campaignApplication) EndCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := requirePolicy(ctx, c.stores, domainauthz.CapabilityManageCampaign, campaignRecord); err != nil {
		return storage.CampaignRecord{}, err
	}

	if err := ensureNoActiveSession(ctx, c.stores.Session, campaignID); err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := validateCampaignStatusTransition(campaignRecord, campaign.StatusCompleted); err != nil {
		return storage.CampaignRecord{}, err
	}
	if c.stores.Domain == nil {
		return storage.CampaignRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
		c.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCampaignEnd,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "campaign",
			EntityID:     campaignID,
		}),
		domainCommandApplyOptions{},
	)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	updated, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, status.Errorf(codes.Internal, "load campaign: %v", err)
	}

	return updated, nil
}

func (c campaignApplication) ArchiveCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := requirePolicy(ctx, c.stores, domainauthz.CapabilityManageCampaign, campaignRecord); err != nil {
		return storage.CampaignRecord{}, err
	}

	if err := ensureNoActiveSession(ctx, c.stores.Session, campaignID); err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := validateCampaignStatusTransition(campaignRecord, campaign.StatusArchived); err != nil {
		return storage.CampaignRecord{}, err
	}
	if c.stores.Domain == nil {
		return storage.CampaignRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
		c.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCampaignArchive,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "campaign",
			EntityID:     campaignID,
		}),
		domainCommandApplyOptions{},
	)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	updated, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, status.Errorf(codes.Internal, "load campaign: %v", err)
	}

	return updated, nil
}

func (c campaignApplication) RestoreCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := requirePolicy(ctx, c.stores, domainauthz.CapabilityManageCampaign, campaignRecord); err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := validateCampaignStatusTransition(campaignRecord, campaign.StatusDraft); err != nil {
		return storage.CampaignRecord{}, err
	}
	if c.stores.Domain == nil {
		return storage.CampaignRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
		c.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCampaignRestore,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "campaign",
			EntityID:     campaignID,
		}),
		domainCommandApplyOptions{},
	)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	updated, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, status.Errorf(codes.Internal, "load campaign: %v", err)
	}

	return updated, nil
}

func (c campaignApplication) SetCampaignCover(ctx context.Context, campaignID, coverAssetID, coverSetID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := requirePolicy(ctx, c.stores, domainauthz.CapabilityManageCampaign, campaignRecord); err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.CampaignRecord{}, err
	}

	if c.stores.Domain == nil {
		return storage.CampaignRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	actorID, actorType := resolveCommandActor(ctx)

	fields := map[string]string{"cover_asset_id": coverAssetID}
	if strings.TrimSpace(coverSetID) != "" {
		fields["cover_set_id"] = strings.TrimSpace(coverSetID)
	}
	payload := campaign.UpdatePayload{Fields: fields}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.CampaignRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
		c.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCampaignUpdate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "campaign",
			EntityID:     campaignID,
			PayloadJSON:  payloadJSON,
		}),
		domainCommandApplyOptions{},
	)
	if err != nil {
		return storage.CampaignRecord{}, err
	}

	updated, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, status.Errorf(codes.Internal, "load campaign: %v", err)
	}
	return updated, nil
}

// validateCampaignStatusTransition ensures the target status is allowed from the current state.
func validateCampaignStatusTransition(record storage.CampaignRecord, target campaign.Status) error {
	if campaign.IsStatusTransitionAllowed(record.Status, target) {
		return nil
	}
	fromStatus := campaignStatusLabel(record.Status)
	toStatus := campaignStatusLabel(target)
	return apperrors.WithMetadata(
		apperrors.CodeCampaignInvalidStatusTransition,
		fmt.Sprintf("campaign status transition not allowed: %s -> %s", fromStatus, toStatus),
		map[string]string{"FromStatus": fromStatus, "ToStatus": toStatus},
	)
}

// campaignStatusLabel returns a stable label for campaign status errors.
func campaignStatusLabel(status campaign.Status) string {
	switch status {
	case campaign.StatusDraft:
		return "DRAFT"
	case campaign.StatusActive:
		return "ACTIVE"
	case campaign.StatusCompleted:
		return "COMPLETED"
	case campaign.StatusArchived:
		return "ARCHIVED"
	default:
		return "UNSPECIFIED"
	}
}
