package game

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListCampaignsPageSize = 10
	maxListCampaignsPageSize     = 10
)

// CampaignService implements the game.v1.CampaignService gRPC API.
type CampaignService struct {
	campaignv1.UnimplementedCampaignServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
	authClient  authv1.AuthServiceClient
}

// NewCampaignService creates a CampaignService with default dependencies.
func NewCampaignService(stores Stores) *CampaignService {
	return &CampaignService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}

// NewCampaignServiceWithAuth creates a CampaignService with an auth client.
func NewCampaignServiceWithAuth(stores Stores, authClient authv1.AuthServiceClient) *CampaignService {
	service := NewCampaignService(stores)
	service.authClient = authClient
	return service
}

// CreateCampaign creates a new campaign metadata record.
func (s *CampaignService) CreateCampaign(ctx context.Context, in *campaignv1.CreateCampaignRequest) (*campaignv1.CreateCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create campaign request is required")
	}

	system := in.GetSystem()
	if system == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "game system is required")
	}

	input := campaign.CreateCampaignInput{
		Name:         in.GetName(),
		Locale:       in.GetLocale(),
		System:       system,
		GmMode:       gmModeFromProto(in.GetGmMode()),
		Intent:       campaignIntentFromProto(in.GetIntent()),
		AccessPolicy: campaignAccessPolicyFromProto(in.GetAccessPolicy()),
		ThemePrompt:  in.GetThemePrompt(),
	}
	normalized, err := campaign.NormalizeCreateCampaignInput(input)
	if err != nil {
		return nil, handleDomainError(err)
	}

	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return nil, handleDomainError(apperrors.New(
			apperrors.CodeCampaignCreatorUserMissing,
			"creator user id is required",
		))
	}

	campaignID, err := s.idGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate campaign id: %v", err)
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
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
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
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := s.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	creatorID, err := s.idGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate participant id: %v", err)
	}

	creatorDisplayName := strings.TrimSpace(in.GetCreatorDisplayName())
	if creatorDisplayName == "" {
		if s.authClient == nil {
			return nil, status.Error(codes.Internal, "auth client is not configured")
		}
		userResponse, err := s.authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: userID})
		if err != nil {
			if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.NotFound {
				return nil, handleDomainError(apperrors.New(
					apperrors.CodeCampaignCreatorUserMissing,
					"creator user not found",
				))
			}
			return nil, status.Errorf(codes.Internal, "get auth user: %v", err)
		}
		if userResponse == nil || userResponse.GetUser() == nil {
			return nil, status.Error(codes.Internal, "auth user response is missing")
		}
		creatorDisplayName = strings.TrimSpace(userResponse.GetUser().GetDisplayName())
		if creatorDisplayName == "" {
			return nil, handleDomainError(apperrors.New(
				apperrors.CodeCampaignCreatorUserMissing,
				"creator user display name is required",
			))
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
		return nil, status.Errorf(codes.Internal, "encode participant payload: %v", err)
	}

	participantEvent, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
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
		return nil, status.Errorf(codes.Internal, "append participant event: %v", err)
	}

	if err := applier.Apply(ctx, participantEvent); err != nil {
		return nil, status.Errorf(codes.Internal, "apply participant event: %v", err)
	}

	ownerParticipant, err := s.stores.Participant.GetParticipant(ctx, campaignID, creatorID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load owner participant: %v", err)
	}

	created, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load campaign: %v", err)
	}

	return &campaignv1.CreateCampaignResponse{
		Campaign:         campaignToProto(created),
		OwnerParticipant: participantToProto(ownerParticipant),
	}, nil
}

// ListCampaigns returns a page of campaign metadata records.
func (s *CampaignService) ListCampaigns(ctx context.Context, in *campaignv1.ListCampaignsRequest) (*campaignv1.ListCampaignsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list campaigns request is required")
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListCampaignsPageSize,
		Max:     maxListCampaignsPageSize,
	})

	// TODO: Apply access policy/intent gates for campaign listing.

	page, err := s.stores.Campaign.List(ctx, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list campaigns: %v", err)
	}

	response := &campaignv1.ListCampaignsResponse{
		NextPageToken: page.NextPageToken,
	}
	if len(page.Campaigns) == 0 {
		return response, nil
	}

	response.Campaigns = make([]*campaignv1.Campaign, 0, len(page.Campaigns))
	for _, c := range page.Campaigns {
		response.Campaigns = append(response.Campaigns, campaignToProto(c))
	}

	return response, nil
}

// GetCampaign returns a campaign metadata record by ID.
func (s *CampaignService) GetCampaign(ctx context.Context, in *campaignv1.GetCampaignRequest) (*campaignv1.GetCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	// TODO: Apply access policy/intent gates for campaign read.
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}

	return &campaignv1.GetCampaignResponse{
		Campaign: campaignToProto(c),
	}, nil
}

// EndCampaign marks a campaign as completed.
func (s *CampaignService) EndCampaign(ctx context.Context, in *campaignv1.EndCampaignRequest) (*campaignv1.EndCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "end campaign request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	if err := ensureNoActiveSession(ctx, s.stores.Session, campaignID); err != nil {
		return nil, err
	}
	if _, err := campaign.TransitionCampaignStatus(c, campaign.CampaignStatusCompleted, s.now); err != nil {
		return nil, handleDomainError(err)
	}

	payload := event.CampaignUpdatedPayload{
		Fields: map[string]any{
			"status": campaignStatusToProto(campaign.CampaignStatusCompleted).String(),
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.now().UTC(),
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
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := s.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load campaign: %v", err)
	}

	return &campaignv1.EndCampaignResponse{Campaign: campaignToProto(updated)}, nil
}

// ArchiveCampaign archives a campaign.
func (s *CampaignService) ArchiveCampaign(ctx context.Context, in *campaignv1.ArchiveCampaignRequest) (*campaignv1.ArchiveCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "archive campaign request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	if err := ensureNoActiveSession(ctx, s.stores.Session, campaignID); err != nil {
		return nil, err
	}
	if _, err := campaign.TransitionCampaignStatus(c, campaign.CampaignStatusArchived, s.now); err != nil {
		return nil, handleDomainError(err)
	}

	payload := event.CampaignUpdatedPayload{
		Fields: map[string]any{
			"status": campaignStatusToProto(campaign.CampaignStatusArchived).String(),
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.now().UTC(),
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
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := s.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load campaign: %v", err)
	}

	return &campaignv1.ArchiveCampaignResponse{Campaign: campaignToProto(updated)}, nil
}

// RestoreCampaign restores an archived campaign to draft state.
func (s *CampaignService) RestoreCampaign(ctx context.Context, in *campaignv1.RestoreCampaignRequest) (*campaignv1.RestoreCampaignResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "restore campaign request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if _, err := campaign.TransitionCampaignStatus(c, campaign.CampaignStatusDraft, s.now); err != nil {
		return nil, handleDomainError(err)
	}

	payload := event.CampaignUpdatedPayload{
		Fields: map[string]any{
			"status": campaignStatusToProto(campaign.CampaignStatusDraft).String(),
		},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.now().UTC(),
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
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := s.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load campaign: %v", err)
	}

	return &campaignv1.RestoreCampaignResponse{Campaign: campaignToProto(updated)}, nil
}

func (s *CampaignService) now() time.Time {
	if s == nil || s.clock == nil {
		return time.Now()
	}
	return s.clock()
}

func ensureNoActiveSession(ctx context.Context, store storage.SessionStore, campaignID string) error {
	if store == nil {
		return status.Error(codes.Internal, "session store is not configured")
	}
	_, err := store.GetActiveSession(ctx, campaignID)
	if err == nil {
		return apperrors.HandleError(storage.ErrActiveSessionExists, apperrors.DefaultLocale)
	}
	if errors.Is(err, storage.ErrNotFound) {
		return nil
	}
	return status.Errorf(codes.Internal, "check active session: %v", err)
}

// handleDomainError converts domain errors to gRPC status using the structured error system.
// For domain errors (*apperrors.Error), it returns a properly formatted gRPC status with
// error details including ErrorInfo and LocalizedMessage.
// For non-domain errors, it falls back to an internal error.
//
// TODO: Extract locale from gRPC metadata (e.g., "accept-language" header) to enable
// proper i18n support. Currently hardcoded to DefaultLocale.
func handleDomainError(err error) error {
	return apperrors.HandleError(err, apperrors.DefaultLocale)
}
