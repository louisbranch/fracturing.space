package game

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
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
	authClient  authv1.AuthServiceClient
}

func newCampaignApplication(service *CampaignService) campaignApplication {
	app := campaignApplication{stores: service.stores, clock: service.clock, idGenerator: service.idGenerator, authClient: service.authClient}
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

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
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

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
		applier,
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("campaign.create"),
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "campaign",
			EntityID:     campaignID,
			PayloadJSON:  payloadJSON,
		},
		domainCommandApplyOptions{},
	)
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, err
	}

	creatorID, err := c.idGenerator()
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "generate participant id: %v", err)
	}

	creatorDisplayName := strings.TrimSpace(in.GetCreatorDisplayName())
	if creatorDisplayName == "" {
		if c.authClient == nil {
			return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Error(codes.Internal, "auth client is not configured")
		}
		userResponse, err := c.authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: userID})
		if err != nil {
			if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.NotFound {
				return storage.CampaignRecord{}, storage.ParticipantRecord{}, apperrors.New(
					apperrors.CodeCampaignCreatorUserMissing,
					"creator user not found",
				)
			}
			return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "get auth user: %v", err)
		}
		if userResponse == nil || userResponse.GetUser() == nil {
			return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Error(codes.Internal, "auth user response is missing")
		}
		creatorDisplayName = strings.TrimSpace(userResponse.GetUser().GetEmail())
		if creatorDisplayName == "" {
			return storage.CampaignRecord{}, storage.ParticipantRecord{}, apperrors.New(
				apperrors.CodeCampaignCreatorUserMissing,
				"creator user display name is required",
			)
		}
	}

	participantPayload := participant.JoinPayload{
		ParticipantID:  creatorID,
		UserID:         userID,
		Name:           creatorDisplayName,
		Role:           "GM",
		Controller:     "HUMAN",
		CampaignAccess: "OWNER",
	}
	participantPayloadJSON, err := json.Marshal(participantPayload)
	if err != nil {
		return storage.CampaignRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "encode participant payload: %v", err)
	}

	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
		applier,
		command.Command{
			CampaignID:   campaignID,
			Type:         command.Type("participant.join"),
			ActorType:    command.ActorTypeSystem,
			ActorID:      "",
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "participant",
			EntityID:     creatorID,
			PayloadJSON:  participantPayloadJSON,
		},
		domainCommandApplyOptions{
			applyErrMessage: "apply participant event",
		},
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

func (c campaignApplication) EndCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}

	if err := ensureNoActiveSession(ctx, c.stores.Session, campaignID); err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := validateCampaignStatusTransition(campaignRecord, campaign.StatusCompleted); err != nil {
		return storage.CampaignRecord{}, err
	}
	if c.stores.Domain != nil {
		actorID := grpcmeta.ParticipantIDFromContext(ctx)
		actorType := command.ActorTypeSystem
		if actorID != "" {
			actorType = command.ActorTypeParticipant
		}
		_, err = executeAndApplyDomainCommand(
			ctx,
			c.stores.Domain,
			c.stores.Applier(),
			command.Command{
				CampaignID:   campaignID,
				Type:         command.Type("campaign.end"),
				ActorType:    actorType,
				ActorID:      actorID,
				RequestID:    grpcmeta.RequestIDFromContext(ctx),
				InvocationID: grpcmeta.InvocationIDFromContext(ctx),
				EntityType:   "campaign",
				EntityID:     campaignID,
			},
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

	return storage.CampaignRecord{}, status.Error(codes.Internal, "domain engine is not configured")
}

func (c campaignApplication) ArchiveCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}

	if err := ensureNoActiveSession(ctx, c.stores.Session, campaignID); err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := validateCampaignStatusTransition(campaignRecord, campaign.StatusArchived); err != nil {
		return storage.CampaignRecord{}, err
	}
	if c.stores.Domain != nil {
		actorID := grpcmeta.ParticipantIDFromContext(ctx)
		actorType := command.ActorTypeSystem
		if actorID != "" {
			actorType = command.ActorTypeParticipant
		}
		_, err = executeAndApplyDomainCommand(
			ctx,
			c.stores.Domain,
			c.stores.Applier(),
			command.Command{
				CampaignID:   campaignID,
				Type:         command.Type("campaign.archive"),
				ActorType:    actorType,
				ActorID:      actorID,
				RequestID:    grpcmeta.RequestIDFromContext(ctx),
				InvocationID: grpcmeta.InvocationIDFromContext(ctx),
				EntityType:   "campaign",
				EntityID:     campaignID,
			},
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

	return storage.CampaignRecord{}, status.Error(codes.Internal, "domain engine is not configured")
}

func (c campaignApplication) RestoreCampaign(ctx context.Context, campaignID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := validateCampaignStatusTransition(campaignRecord, campaign.StatusDraft); err != nil {
		return storage.CampaignRecord{}, err
	}
	if c.stores.Domain != nil {
		actorID := grpcmeta.ParticipantIDFromContext(ctx)
		actorType := command.ActorTypeSystem
		if actorID != "" {
			actorType = command.ActorTypeParticipant
		}
		_, err = executeAndApplyDomainCommand(
			ctx,
			c.stores.Domain,
			c.stores.Applier(),
			command.Command{
				CampaignID:   campaignID,
				Type:         command.Type("campaign.restore"),
				ActorType:    actorType,
				ActorID:      actorID,
				RequestID:    grpcmeta.RequestIDFromContext(ctx),
				InvocationID: grpcmeta.InvocationIDFromContext(ctx),
				EntityType:   "campaign",
				EntityID:     campaignID,
			},
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

	return storage.CampaignRecord{}, status.Error(codes.Internal, "domain engine is not configured")
}

func (c campaignApplication) SetCampaignCover(ctx context.Context, campaignID, coverAssetID, coverSetID string) (storage.CampaignRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.CampaignRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.CampaignRecord{}, err
	}

	if c.stores.Domain != nil {
		actorID := grpcmeta.ParticipantIDFromContext(ctx)
		actorType := command.ActorTypeSystem
		if actorID != "" {
			actorType = command.ActorTypeParticipant
		}

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
			command.Command{
				CampaignID:   campaignID,
				Type:         command.Type("campaign.update"),
				ActorType:    actorType,
				ActorID:      actorID,
				RequestID:    grpcmeta.RequestIDFromContext(ctx),
				InvocationID: grpcmeta.InvocationIDFromContext(ctx),
				EntityType:   "campaign",
				EntityID:     campaignID,
				PayloadJSON:  payloadJSON,
			},
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

	return storage.CampaignRecord{}, status.Error(codes.Internal, "domain engine is not configured")
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
