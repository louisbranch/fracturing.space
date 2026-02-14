package game

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/policy"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type participantApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
}

func newParticipantApplication(service *ParticipantService) participantApplication {
	app := participantApplication{stores: service.stores, clock: service.clock, idGenerator: service.idGenerator}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}

func (c participantApplication) CreateParticipant(ctx context.Context, campaignID string, in *campaignv1.CreateParticipantRequest) (participant.Participant, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return participant.Participant{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return participant.Participant{}, err
	}
	if err := requirePolicy(ctx, c.stores, policy.ActionManageParticipants, campaignRecord); err != nil {
		return participant.Participant{}, err
	}

	input := participant.CreateParticipantInput{
		CampaignID:     campaignID,
		UserID:         in.GetUserId(),
		DisplayName:    in.GetDisplayName(),
		Role:           participantRoleFromProto(in.GetRole()),
		Controller:     controllerFromProto(in.GetController()),
		CampaignAccess: participant.CampaignAccessMember,
	}
	normalized, err := participant.NormalizeCreateParticipantInput(input)
	if err != nil {
		return participant.Participant{}, err
	}

	participantID, err := c.idGenerator()
	if err != nil {
		return participant.Participant{}, status.Errorf(codes.Internal, "generate participant id: %v", err)
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
	accessLabel := "MEMBER"
	switch normalized.CampaignAccess {
	case participant.CampaignAccessManager:
		accessLabel = "MANAGER"
	case participant.CampaignAccessOwner:
		accessLabel = "OWNER"
	}

	payload := event.ParticipantJoinedPayload{
		ParticipantID:  participantID,
		UserID:         normalized.UserID,
		DisplayName:    normalized.DisplayName,
		Role:           roleLabel,
		Controller:     controllerLabel,
		CampaignAccess: accessLabel,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return participant.Participant{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := c.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    c.clock().UTC(),
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
		return participant.Participant{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := c.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return participant.Participant{}, err
		}
		return participant.Participant{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	created, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return participant.Participant{}, status.Errorf(codes.Internal, "load participant: %v", err)
	}

	return created, nil
}

func (c participantApplication) UpdateParticipant(ctx context.Context, campaignID string, in *campaignv1.UpdateParticipantRequest) (participant.Participant, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return participant.Participant{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return participant.Participant{}, err
	}
	if err := requirePolicy(ctx, c.stores, policy.ActionManageParticipants, campaignRecord); err != nil {
		return participant.Participant{}, err
	}

	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		return participant.Participant{}, status.Error(codes.InvalidArgument, "participant id is required")
	}

	current, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return participant.Participant{}, err
	}

	fields := make(map[string]any)
	if displayName := in.GetDisplayName(); displayName != nil {
		trimmed := strings.TrimSpace(displayName.GetValue())
		if trimmed == "" {
			return participant.Participant{}, status.Error(codes.InvalidArgument, "display_name must not be empty")
		}
		current.DisplayName = trimmed
		fields["display_name"] = trimmed
	}
	if userID := in.GetUserId(); userID != nil {
		trimmed := strings.TrimSpace(userID.GetValue())
		current.UserID = trimmed
		fields["user_id"] = trimmed
	}
	if in.GetRole() != campaignv1.ParticipantRole_ROLE_UNSPECIFIED {
		role := participantRoleFromProto(in.GetRole())
		if role == participant.ParticipantRoleUnspecified {
			return participant.Participant{}, status.Error(codes.InvalidArgument, "role is invalid")
		}
		current.Role = role
		fields["role"] = in.GetRole().String()
	}
	if in.GetController() != campaignv1.Controller_CONTROLLER_UNSPECIFIED {
		controller := controllerFromProto(in.GetController())
		if controller == participant.ControllerUnspecified {
			return participant.Participant{}, status.Error(codes.InvalidArgument, "controller is invalid")
		}
		current.Controller = controller
		fields["controller"] = in.GetController().String()
	}
	if in.GetCampaignAccess() != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED {
		access := campaignAccessFromProto(in.GetCampaignAccess())
		if access == participant.CampaignAccessUnspecified {
			return participant.Participant{}, status.Error(codes.InvalidArgument, "campaign_access is invalid")
		}
		current.CampaignAccess = access
		fields["campaign_access"] = in.GetCampaignAccess().String()
	}
	if len(fields) == 0 {
		return participant.Participant{}, status.Error(codes.InvalidArgument, "at least one field must be provided")
	}

	payload := event.ParticipantUpdatedPayload{
		ParticipantID: participantID,
		Fields:        fields,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return participant.Participant{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := c.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    c.clock().UTC(),
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
		return participant.Participant{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := c.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return participant.Participant{}, err
		}
		return participant.Participant{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return participant.Participant{}, status.Errorf(codes.Internal, "load participant: %v", err)
	}

	return updated, nil
}

func (c participantApplication) DeleteParticipant(ctx context.Context, campaignID string, in *campaignv1.DeleteParticipantRequest) (participant.Participant, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return participant.Participant{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return participant.Participant{}, err
	}

	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		return participant.Participant{}, status.Error(codes.InvalidArgument, "participant id is required")
	}

	current, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return participant.Participant{}, err
	}

	payload := event.ParticipantLeftPayload{
		ParticipantID: participantID,
		Reason:        strings.TrimSpace(in.GetReason()),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return participant.Participant{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := c.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    c.clock().UTC(),
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
		return participant.Participant{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := c.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return participant.Participant{}, err
		}
		return participant.Participant{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	return current, nil
}
