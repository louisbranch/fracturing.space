package service

import (
	"context"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CreateParticipant creates a participant (GM or player) for a campaign.
func (s *CampaignService) CreateParticipant(ctx context.Context, in *campaignv1.CreateParticipantRequest) (*campaignv1.CreateParticipantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create participant request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}

	// Validate campaign exists
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	campaign, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "check campaign: %v", err)
	}

	participant, err := domain.CreateParticipant(domain.CreateParticipantInput{
		CampaignID:  campaignID,
		DisplayName: in.GetDisplayName(),
		Role:        participantRoleFromProto(in.GetRole()),
		Controller:  controllerFromProto(in.GetController()),
	}, s.clock, s.idGenerator)
	if err != nil {
		if errors.Is(err, domain.ErrEmptyDisplayName) || errors.Is(err, domain.ErrInvalidParticipantRole) || errors.Is(err, domain.ErrEmptyCampaignID) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "create participant: %v", err)
	}

	if err := s.stores.Participant.PutParticipant(ctx, participant); err != nil {
		return nil, status.Errorf(codes.Internal, "persist participant: %v", err)
	}

	// Increment player count if the participant is a player
	if participant.Role == domain.ParticipantRolePlayer {
		// TODO: Fix race condition - Get and Put are not atomic. If multiple players
		// register concurrently, the player count may be incorrect. Consider using
		// transactions or atomic increment operations if the storage layer supports them.
		campaign.PlayerCount++
		campaign.UpdatedAt = s.clock().UTC()
		if err := s.stores.Campaign.Put(ctx, campaign); err != nil {
			return nil, status.Errorf(codes.Internal, "update campaign player count: %v", err)
		}
	}

	response := &campaignv1.CreateParticipantResponse{
		Participant: &campaignv1.Participant{
			Id:          participant.ID,
			CampaignId:  participant.CampaignID,
			DisplayName: participant.DisplayName,
			Role:        participantRoleToProto(participant.Role),
			Controller:  controllerToProto(participant.Controller),
			CreatedAt:   timestamppb.New(participant.CreatedAt),
			UpdatedAt:   timestamppb.New(participant.UpdatedAt),
		},
	}

	return response, nil
}

// ListParticipants returns a page of participant records for a campaign.
func (s *CampaignService) ListParticipants(ctx context.Context, in *campaignv1.ListParticipantsRequest) (*campaignv1.ListParticipantsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list participants request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}

	// Validate campaign exists
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	_, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "check campaign: %v", err)
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

	response := &campaignv1.ListParticipantsResponse{
		NextPageToken: page.NextPageToken,
	}
	if len(page.Participants) == 0 {
		return response, nil
	}

	response.Participants = make([]*campaignv1.Participant, 0, len(page.Participants))
	for _, participant := range page.Participants {
		response.Participants = append(response.Participants, &campaignv1.Participant{
			Id:          participant.ID,
			CampaignId:  participant.CampaignID,
			DisplayName: participant.DisplayName,
			Role:        participantRoleToProto(participant.Role),
			Controller:  controllerToProto(participant.Controller),
			CreatedAt:   timestamppb.New(participant.CreatedAt),
			UpdatedAt:   timestamppb.New(participant.UpdatedAt),
		})
	}

	return response, nil
}

// participantRoleFromProto maps a protobuf participant role to the domain representation.
func participantRoleFromProto(role campaignv1.ParticipantRole) domain.ParticipantRole {
	switch role {
	case campaignv1.ParticipantRole_GM:
		return domain.ParticipantRoleGM
	case campaignv1.ParticipantRole_PLAYER:
		return domain.ParticipantRolePlayer
	default:
		return domain.ParticipantRoleUnspecified
	}
}

// participantRoleToProto maps a domain participant role to the protobuf representation.
func participantRoleToProto(role domain.ParticipantRole) campaignv1.ParticipantRole {
	switch role {
	case domain.ParticipantRoleGM:
		return campaignv1.ParticipantRole_GM
	case domain.ParticipantRolePlayer:
		return campaignv1.ParticipantRole_PLAYER
	default:
		return campaignv1.ParticipantRole_ROLE_UNSPECIFIED
	}
}

// controllerFromProto maps a protobuf controller to the domain representation.
func controllerFromProto(controller campaignv1.Controller) domain.Controller {
	switch controller {
	case campaignv1.Controller_CONTROLLER_HUMAN:
		return domain.ControllerHuman
	case campaignv1.Controller_CONTROLLER_AI:
		return domain.ControllerAI
	default:
		return domain.ControllerUnspecified
	}
}

// controllerToProto maps a domain controller to the protobuf representation.
func controllerToProto(controller domain.Controller) campaignv1.Controller {
	switch controller {
	case domain.ControllerHuman:
		return campaignv1.Controller_CONTROLLER_HUMAN
	case domain.ControllerAI:
		return campaignv1.Controller_CONTROLLER_AI
	default:
		return campaignv1.Controller_CONTROLLER_UNSPECIFIED
	}
}
