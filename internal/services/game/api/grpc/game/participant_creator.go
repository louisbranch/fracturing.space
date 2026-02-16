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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
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

func (c participantApplication) CreateParticipant(ctx context.Context, campaignID string, in *campaignv1.CreateParticipantRequest) (storage.ParticipantRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.ParticipantRecord{}, err
	}
	if err := requirePolicy(ctx, c.stores, policyActionManageParticipants, campaignRecord); err != nil {
		return storage.ParticipantRecord{}, err
	}

	displayName := strings.TrimSpace(in.GetDisplayName())
	if displayName == "" {
		return storage.ParticipantRecord{}, apperrors.New(apperrors.CodeParticipantEmptyDisplayName, "display name is required")
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

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	applier := c.stores.Applier()
	if c.stores.Domain == nil {
		return storage.ParticipantRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := participant.JoinPayload{
		ParticipantID:  participantID,
		UserID:         strings.TrimSpace(in.GetUserId()),
		DisplayName:    displayName,
		Role:           string(role),
		Controller:     string(controller),
		CampaignAccess: string(access),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	result, err := c.stores.Domain.Execute(ctx, command.Command{
		CampaignID:   campaignID,
		Type:         command.Type("participant.join"),
		ActorType:    actorType,
		ActorID:      actorID,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		EntityType:   "participant",
		EntityID:     participantID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "execute domain command: %v", err)
	}
	if len(result.Decision.Rejections) > 0 {
		return storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, result.Decision.Rejections[0].Message)
	}
	for _, evt := range result.Decision.Events {
		if err := applier.Apply(ctx, evt); err != nil {
			if apperrors.GetCode(err) != apperrors.CodeUnknown {
				return storage.ParticipantRecord{}, err
			}
			return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "apply event: %v", err)
		}
	}

	created, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "load participant: %v", err)
	}

	return created, nil
}

func (c participantApplication) UpdateParticipant(ctx context.Context, campaignID string, in *campaignv1.UpdateParticipantRequest) (storage.ParticipantRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.ParticipantRecord{}, err
	}
	if err := requirePolicy(ctx, c.stores, policyActionManageParticipants, campaignRecord); err != nil {
		return storage.ParticipantRecord{}, err
	}

	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		return storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "participant id is required")
	}

	current, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}

	fields := make(map[string]any)
	if displayName := in.GetDisplayName(); displayName != nil {
		trimmed := strings.TrimSpace(displayName.GetValue())
		if trimmed == "" {
			return storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "display_name must not be empty")
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
		if role == participant.RoleUnspecified {
			return storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "role is invalid")
		}
		current.Role = role
		fields["role"] = in.GetRole().String()
	}
	if in.GetController() != campaignv1.Controller_CONTROLLER_UNSPECIFIED {
		controller := controllerFromProto(in.GetController())
		if controller == participant.ControllerUnspecified {
			return storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "controller is invalid")
		}
		current.Controller = controller
		fields["controller"] = in.GetController().String()
	}
	if in.GetCampaignAccess() != campaignv1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED {
		access := campaignAccessFromProto(in.GetCampaignAccess())
		if access == participant.CampaignAccessUnspecified {
			return storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "campaign_access is invalid")
		}
		current.CampaignAccess = access
		fields["campaign_access"] = in.GetCampaignAccess().String()
	}
	if len(fields) == 0 {
		return storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "at least one field must be provided")
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	applier := c.stores.Applier()
	if c.stores.Domain == nil {
		return storage.ParticipantRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payloadFields := make(map[string]string, len(fields))
	for key, value := range fields {
		stringValue, ok := value.(string)
		if !ok {
			return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "participant update field %s must be string", key)
		}
		payloadFields[key] = stringValue
	}
	payload := participant.UpdatePayload{
		ParticipantID: participantID,
		Fields:        payloadFields,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	result, err := c.stores.Domain.Execute(ctx, command.Command{
		CampaignID:   campaignID,
		Type:         command.Type("participant.update"),
		ActorType:    actorType,
		ActorID:      actorID,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		EntityType:   "participant",
		EntityID:     participantID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "execute domain command: %v", err)
	}
	if len(result.Decision.Rejections) > 0 {
		return storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, result.Decision.Rejections[0].Message)
	}
	for _, evt := range result.Decision.Events {
		if err := applier.Apply(ctx, evt); err != nil {
			if apperrors.GetCode(err) != apperrors.CodeUnknown {
				return storage.ParticipantRecord{}, err
			}
			return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "apply event: %v", err)
		}
	}

	updated, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "load participant: %v", err)
	}

	return updated, nil
}

func (c participantApplication) DeleteParticipant(ctx context.Context, campaignID string, in *campaignv1.DeleteParticipantRequest) (storage.ParticipantRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.ParticipantRecord{}, err
	}

	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		return storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "participant id is required")
	}

	current, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	reason := strings.TrimSpace(in.GetReason())
	applier := c.stores.Applier()
	if c.stores.Domain == nil {
		return storage.ParticipantRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := participant.LeavePayload{
		ParticipantID: participantID,
		Reason:        reason,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	result, err := c.stores.Domain.Execute(ctx, command.Command{
		CampaignID:   campaignID,
		Type:         command.Type("participant.leave"),
		ActorType:    actorType,
		ActorID:      actorID,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		EntityType:   "participant",
		EntityID:     participantID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "execute domain command: %v", err)
	}
	if len(result.Decision.Rejections) > 0 {
		return storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, result.Decision.Rejections[0].Message)
	}
	for _, evt := range result.Decision.Events {
		if err := applier.Apply(ctx, evt); err != nil {
			if apperrors.GetCode(err) != apperrors.CodeUnknown {
				return storage.ParticipantRecord{}, err
			}
			return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "apply event: %v", err)
		}
	}

	return current, nil
}
