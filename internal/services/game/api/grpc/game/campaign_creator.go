package game

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
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
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type campaignApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
	aiClient    aiv1.AgentServiceClient
}

func newCampaignApplication(service *CampaignService) campaignApplication {
	app := campaignApplication{
		stores:      service.stores,
		clock:       service.clock,
		idGenerator: service.idGenerator,
		aiClient:    service.aiClient,
	}
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

func (c campaignApplication) SetCampaignAIBinding(ctx context.Context, campaignID, aiAgentID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.CampaignRecord{}, err
	}

	ownerActor, err := requireCampaignOwner(ctx, c.stores, campaignRecord)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := c.validateAIBindingAgent(ctx, campaignID, aiAgentID, ownerActor.UserID); err != nil {
		return storage.CampaignRecord{}, err
	}

	if c.stores.Domain == nil {
		return storage.CampaignRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payloadJSON, err := json.Marshal(campaign.AIBindPayload{AIAgentID: strings.TrimSpace(aiAgentID)})
	if err != nil {
		return storage.CampaignRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}
	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
		c.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCampaignAIBind,
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
	if err := rotateCampaignAIAuthEpoch(ctx, c.stores, campaignID, aiAuthRotateReasonCampaignAIBound, actorID, actorType); err != nil {
		return storage.CampaignRecord{}, err
	}

	updated, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, status.Errorf(codes.Internal, "load campaign: %v", err)
	}
	return updated, nil
}

func (c campaignApplication) ClearCampaignAIBinding(ctx context.Context, campaignID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.CampaignRecord{}, err
	}
	if _, err := requireCampaignOwner(ctx, c.stores, campaignRecord); err != nil {
		return storage.CampaignRecord{}, err
	}
	if strings.TrimSpace(campaignRecord.AIAgentID) == "" {
		return campaignRecord, nil
	}

	actorID, actorType := resolveCommandActor(ctx)
	return clearCampaignAIBindingByCommand(
		ctx,
		c.stores,
		campaignID,
		actorID,
		actorType,
		grpcmeta.RequestIDFromContext(ctx),
		grpcmeta.InvocationIDFromContext(ctx),
	)
}

func (c campaignApplication) validateAIBindingAgent(ctx context.Context, campaignID, aiAgentID, ownerUserID string) error {
	if c.aiClient == nil {
		return status.Error(codes.Internal, "ai agent client is not configured")
	}

	callCtx := grpcauthctx.WithUserID(ctx, ownerUserID)
	_, err := c.aiClient.ValidateCampaignAgentBinding(callCtx, &aiv1.ValidateCampaignAgentBindingRequest{
		AgentId:    strings.TrimSpace(aiAgentID),
		CampaignId: campaignID,
	})
	if err != nil {
		return err
	}
	return nil
}

func requireCampaignOwner(ctx context.Context, stores Stores, campaignRecord storage.CampaignRecord) (storage.ParticipantRecord, error) {
	actor, err := requirePolicyActor(ctx, stores, domainauthz.CapabilityManageCampaign, campaignRecord)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	if actor.CampaignAccess != participant.CampaignAccessOwner {
		return storage.ParticipantRecord{}, status.Error(codes.PermissionDenied, "owner permission is required")
	}
	if strings.TrimSpace(actor.UserID) == "" {
		return storage.ParticipantRecord{}, status.Error(codes.PermissionDenied, "owner user identity is required")
	}
	return actor, nil
}

func clearCampaignAIBindingByCommand(
	ctx context.Context,
	stores Stores,
	campaignID string,
	actorID string,
	actorType command.ActorType,
	requestID string,
	invocationID string,
) (storage.CampaignRecord, error) {
	if stores.Domain == nil {
		return storage.CampaignRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}

	payloadJSON, err := json.Marshal(campaign.AIUnbindPayload{})
	if err != nil {
		return storage.CampaignRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		stores.Domain,
		stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeCampaignAIUnbind,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "campaign",
			EntityID:     campaignID,
			PayloadJSON:  payloadJSON,
		}),
		domainCommandApplyOptions{},
	)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := rotateCampaignAIAuthEpoch(ctx, stores, campaignID, aiAuthRotateReasonCampaignAIUnbound, actorID, actorType); err != nil {
		return storage.CampaignRecord{}, err
	}

	updated, err := stores.Campaign.Get(ctx, campaignID)
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
