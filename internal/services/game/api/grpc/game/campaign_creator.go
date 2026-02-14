package game

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
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

func (c campaignApplication) CreateCampaign(ctx context.Context, in *campaignv1.CreateCampaignRequest) (campaign.Campaign, participant.Participant, error) {
	input := campaign.CreateCampaignInput{
		Name:         in.GetName(),
		Locale:       in.GetLocale(),
		System:       in.GetSystem(),
		GmMode:       gmModeFromProto(in.GetGmMode()),
		Intent:       campaignIntentFromProto(in.GetIntent()),
		AccessPolicy: campaignAccessPolicyFromProto(in.GetAccessPolicy()),
		ThemePrompt:  in.GetThemePrompt(),
	}
	if input.System == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return campaign.Campaign{}, participant.Participant{}, status.Error(codes.InvalidArgument, "game system is required")
	}

	normalized, err := campaign.NormalizeCreateCampaignInput(input)
	if err != nil {
		return campaign.Campaign{}, participant.Participant{}, err
	}

	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return campaign.Campaign{}, participant.Participant{}, apperrors.New(
			apperrors.CodeCampaignCreatorUserMissing,
			"creator user id is required",
		)
	}

	campaignID, err := c.idGenerator()
	if err != nil {
		return campaign.Campaign{}, participant.Participant{}, status.Errorf(codes.Internal, "generate campaign id: %v", err)
	}

	payload := event.CampaignCreatedPayload{
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
		return campaign.Campaign{}, participant.Participant{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := c.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    c.clock().UTC(),
		Type:         event.TypeCampaignCreated,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "campaign",
		EntityID:     campaignID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return campaign.Campaign{}, participant.Participant{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := c.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return campaign.Campaign{}, participant.Participant{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	creatorID, err := c.idGenerator()
	if err != nil {
		return campaign.Campaign{}, participant.Participant{}, status.Errorf(codes.Internal, "generate participant id: %v", err)
	}

	creatorDisplayName := strings.TrimSpace(in.GetCreatorDisplayName())
	if creatorDisplayName == "" {
		if c.authClient == nil {
			return campaign.Campaign{}, participant.Participant{}, status.Error(codes.Internal, "auth client is not configured")
		}
		userResponse, err := c.authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: userID})
		if err != nil {
			if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.NotFound {
				return campaign.Campaign{}, participant.Participant{}, apperrors.New(
					apperrors.CodeCampaignCreatorUserMissing,
					"creator user not found",
				)
			}
			return campaign.Campaign{}, participant.Participant{}, status.Errorf(codes.Internal, "get auth user: %v", err)
		}
		if userResponse == nil || userResponse.GetUser() == nil {
			return campaign.Campaign{}, participant.Participant{}, status.Error(codes.Internal, "auth user response is missing")
		}
		creatorDisplayName = strings.TrimSpace(userResponse.GetUser().GetDisplayName())
		if creatorDisplayName == "" {
			return campaign.Campaign{}, participant.Participant{}, apperrors.New(
				apperrors.CodeCampaignCreatorUserMissing,
				"creator user display name is required",
			)
		}
	}

	participantPayload := event.ParticipantJoinedPayload{
		ParticipantID:  creatorID,
		UserID:         userID,
		DisplayName:    creatorDisplayName,
		Role:           "GM",
		Controller:     "HUMAN",
		CampaignAccess: "OWNER",
	}
	participantPayloadJSON, err := json.Marshal(participantPayload)
	if err != nil {
		return campaign.Campaign{}, participant.Participant{}, status.Errorf(codes.Internal, "encode participant payload: %v", err)
	}

	participantEvent, err := c.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    c.clock().UTC(),
		Type:         event.TypeParticipantJoined,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    event.ActorTypeSystem,
		ActorID:      "",
		EntityType:   "participant",
		EntityID:     creatorID,
		PayloadJSON:  participantPayloadJSON,
	})
	if err != nil {
		return campaign.Campaign{}, participant.Participant{}, status.Errorf(codes.Internal, "append participant event: %v", err)
	}

	if err := applier.Apply(ctx, participantEvent); err != nil {
		return campaign.Campaign{}, participant.Participant{}, status.Errorf(codes.Internal, "apply participant event: %v", err)
	}

	ownerParticipant, err := c.stores.Participant.GetParticipant(ctx, campaignID, creatorID)
	if err != nil {
		return campaign.Campaign{}, participant.Participant{}, status.Errorf(codes.Internal, "load owner participant: %v", err)
	}

	created, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return campaign.Campaign{}, participant.Participant{}, status.Errorf(codes.Internal, "load campaign: %v", err)
	}

	return created, ownerParticipant, nil
}

func (c campaignApplication) EndCampaign(ctx context.Context, campaignID string) (campaign.Campaign, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return campaign.Campaign{}, err
	}

	if err := ensureNoActiveSession(ctx, c.stores.Session, campaignID); err != nil {
		return campaign.Campaign{}, err
	}
	if _, err := campaign.TransitionCampaignStatus(campaignRecord, campaign.CampaignStatusCompleted, c.clock); err != nil {
		return campaign.Campaign{}, err
	}

	payload := event.CampaignUpdatedPayload{
		Fields: map[string]any{
			"status": campaignStatusToProto(campaign.CampaignStatusCompleted).String(),
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return campaign.Campaign{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := c.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    c.clock().UTC(),
		Type:         event.TypeCampaignUpdated,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "campaign",
		EntityID:     campaignID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return campaign.Campaign{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := c.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return campaign.Campaign{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return campaign.Campaign{}, status.Errorf(codes.Internal, "load campaign: %v", err)
	}

	return updated, nil
}

func (c campaignApplication) ArchiveCampaign(ctx context.Context, campaignID string) (campaign.Campaign, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return campaign.Campaign{}, err
	}

	if err := ensureNoActiveSession(ctx, c.stores.Session, campaignID); err != nil {
		return campaign.Campaign{}, err
	}
	if _, err := campaign.TransitionCampaignStatus(campaignRecord, campaign.CampaignStatusArchived, c.clock); err != nil {
		return campaign.Campaign{}, err
	}

	payload := event.CampaignUpdatedPayload{
		Fields: map[string]any{
			"status": campaignStatusToProto(campaign.CampaignStatusArchived).String(),
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return campaign.Campaign{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := c.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    c.clock().UTC(),
		Type:         event.TypeCampaignUpdated,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "campaign",
		EntityID:     campaignID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return campaign.Campaign{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := c.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return campaign.Campaign{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return campaign.Campaign{}, status.Errorf(codes.Internal, "load campaign: %v", err)
	}

	return updated, nil
}

func (c campaignApplication) RestoreCampaign(ctx context.Context, campaignID string) (campaign.Campaign, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return campaign.Campaign{}, err
	}
	if _, err := campaign.TransitionCampaignStatus(campaignRecord, campaign.CampaignStatusDraft, c.clock); err != nil {
		return campaign.Campaign{}, err
	}

	payload := event.CampaignUpdatedPayload{
		Fields: map[string]any{
			"status": campaignStatusToProto(campaign.CampaignStatusDraft).String(),
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return campaign.Campaign{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := c.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    c.clock().UTC(),
		Type:         event.TypeCampaignUpdated,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "campaign",
		EntityID:     campaignID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return campaign.Campaign{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := c.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return campaign.Campaign{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return campaign.Campaign{}, status.Errorf(codes.Internal, "load campaign: %v", err)
	}

	return updated, nil
}
