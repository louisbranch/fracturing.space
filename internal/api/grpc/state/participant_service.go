package state

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/state/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/id"
	"github.com/louisbranch/fracturing.space/internal/state/campaign"
	"github.com/louisbranch/fracturing.space/internal/state/event"
	"github.com/louisbranch/fracturing.space/internal/state/participant"
	"github.com/louisbranch/fracturing.space/internal/state/projection"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListParticipantsPageSize = 10
	maxListParticipantsPageSize     = 10
)

// ParticipantService implements the state.v1.ParticipantService gRPC API.
type ParticipantService struct {
	statev1.UnimplementedParticipantServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

// NewParticipantService creates a ParticipantService with default dependencies.
func NewParticipantService(stores Stores) *ParticipantService {
	return &ParticipantService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}

// CreateParticipant creates a participant (GM or player) for a campaign.
func (s *ParticipantService) CreateParticipant(ctx context.Context, in *statev1.CreateParticipantRequest) (*statev1.CreateParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create participant request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}

	input := participant.CreateParticipantInput{
		CampaignID:  campaignID,
		DisplayName: in.GetDisplayName(),
		Role:        participantRoleFromProto(in.GetRole()),
		Controller:  controllerFromProto(in.GetController()),
	}
	normalized, err := participant.NormalizeCreateParticipantInput(input)
	if err != nil {
		return nil, handleDomainError(err)
	}

	participantID, err := s.idGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate participant id: %v", err)
	}

	roleLabel := ""
	switch normalized.Role {
	case participant.ParticipantRoleGM:
		roleLabel = "GM"
	case participant.ParticipantRolePlayer:
		roleLabel = "PLAYER"
	}
	controllerLabel := ""
	switch normalized.Controller {
	case participant.ControllerHuman:
		controllerLabel = "HUMAN"
	case participant.ControllerAI:
		controllerLabel = "AI"
	}

	payload := event.ParticipantJoinedPayload{
		ParticipantID: participantID,
		DisplayName:   normalized.DisplayName,
		Role:          roleLabel,
		Controller:    controllerLabel,
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
		Type:         event.TypeParticipantJoined,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "participant",
		EntityID:     participantID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := projection.Applier{Campaign: s.stores.Campaign, Participant: s.stores.Participant}
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	created, err := s.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load participant: %v", err)
	}

	return &statev1.CreateParticipantResponse{
		Participant: participantToProto(created),
	}, nil
}

// UpdateParticipant updates a participant.
func (s *ParticipantService) UpdateParticipant(ctx context.Context, in *statev1.UpdateParticipantRequest) (*statev1.UpdateParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update participant request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}

	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		return nil, status.Error(codes.InvalidArgument, "participant id is required")
	}

	current, err := s.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	fields := make(map[string]any)
	if displayName := in.GetDisplayName(); displayName != nil {
		trimmed := strings.TrimSpace(displayName.GetValue())
		if trimmed == "" {
			return nil, status.Error(codes.InvalidArgument, "display_name must not be empty")
		}
		current.DisplayName = trimmed
		fields["display_name"] = trimmed
	}
	if in.GetRole() != statev1.ParticipantRole_ROLE_UNSPECIFIED {
		role := participantRoleFromProto(in.GetRole())
		if role == participant.ParticipantRoleUnspecified {
			return nil, status.Error(codes.InvalidArgument, "role is invalid")
		}
		current.Role = role
		fields["role"] = in.GetRole().String()
	}
	if in.GetController() != statev1.Controller_CONTROLLER_UNSPECIFIED {
		controller := controllerFromProto(in.GetController())
		if controller == participant.ControllerUnspecified {
			return nil, status.Error(codes.InvalidArgument, "controller is invalid")
		}
		current.Controller = controller
		fields["controller"] = in.GetController().String()
	}
	if len(fields) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one field must be provided")
	}

	payload := event.ParticipantUpdatedPayload{
		ParticipantID: participantID,
		Fields:        fields,
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
		Type:         event.TypeParticipantUpdated,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "participant",
		EntityID:     participantID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := projection.Applier{Campaign: s.stores.Campaign, Participant: s.stores.Participant}
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := s.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load participant: %v", err)
	}

	return &statev1.UpdateParticipantResponse{Participant: participantToProto(updated)}, nil
}

// DeleteParticipant deletes a participant.
func (s *ParticipantService) DeleteParticipant(ctx context.Context, in *statev1.DeleteParticipantRequest) (*statev1.DeleteParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete participant request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}

	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		return nil, status.Error(codes.InvalidArgument, "participant id is required")
	}

	current, err := s.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	payload := event.ParticipantLeftPayload{
		ParticipantID: participantID,
		Reason:        strings.TrimSpace(in.GetReason()),
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
		Type:         event.TypeParticipantLeft,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "participant",
		EntityID:     participantID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := projection.Applier{Campaign: s.stores.Campaign, Participant: s.stores.Participant}
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	return &statev1.DeleteParticipantResponse{Participant: participantToProto(current)}, nil
}

// ListParticipants returns a page of participant records for a campaign.
func (s *ParticipantService) ListParticipants(ctx context.Context, in *statev1.ListParticipantsRequest) (*statev1.ListParticipantsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list participants request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = defaultListParticipantsPageSize
	}
	if pageSize > maxListParticipantsPageSize {
		pageSize = maxListParticipantsPageSize
	}

	page, err := s.stores.Participant.ListParticipants(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list participants: %v", err)
	}

	response := &statev1.ListParticipantsResponse{
		NextPageToken: page.NextPageToken,
	}
	if len(page.Participants) == 0 {
		return response, nil
	}

	response.Participants = make([]*statev1.Participant, 0, len(page.Participants))
	for _, p := range page.Participants {
		response.Participants = append(response.Participants, participantToProto(p))
	}

	return response, nil
}

// GetParticipant returns a participant by campaign ID and participant ID.
func (s *ParticipantService) GetParticipant(ctx context.Context, in *statev1.GetParticipantRequest) (*statev1.GetParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get participant request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		return nil, status.Error(codes.InvalidArgument, "participant id is required")
	}

	c, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}

	p, err := s.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return nil, handleDomainError(err)
	}

	return &statev1.GetParticipantResponse{
		Participant: participantToProto(p),
	}, nil
}
