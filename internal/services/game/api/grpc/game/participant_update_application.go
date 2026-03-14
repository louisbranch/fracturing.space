package game

import (
	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (c participantApplication) UpdateParticipant(ctx context.Context, campaignID string, in *campaignv1.UpdateParticipantRequest) (storage.ParticipantRecord, error) {
	campaignRecord, err := c.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.ParticipantRecord{}, err
	}
	policyActor, err := requirePolicyActor(ctx, c.stores, domainauthz.CapabilityManageParticipants, campaignRecord)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}

	participantID, err := validate.RequiredID(in.GetParticipantId(), "participant id")
	if err != nil {
		return storage.ParticipantRecord{}, err
	}

	current, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	targetAccessBefore := current.CampaignAccess
	if in.GetCampaignAccess() == campaignv1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED {
		decision := domainauthz.CanParticipantMutation(policyActor.CampaignAccess, targetAccessBefore)
		if !decision.Allowed {
			authErr := participantPolicyDecisionError(decision.ReasonCode)
			emitAuthzDecisionTelemetry(ctx, authzDecisionEvent{
				Store:      c.stores.Audit,
				CampaignID: campaignID,
				Capability: domainauthz.CapabilityManageParticipants,
				Decision:   authzDecisionDeny,
				ReasonCode: decision.ReasonCode,
				Actor:      policyActor,
				Err:        authErr,
				ExtraAttributes: map[string]any{
					"target_participant_id":  participantID,
					"target_campaign_access": strings.TrimSpace(string(targetAccessBefore)),
				},
			})
			return storage.ParticipantRecord{}, authErr
		}
	}

	fields := make(map[string]any)
	if name := in.GetName(); name != nil {
		trimmed := strings.TrimSpace(name.GetValue())
		if trimmed == "" {
			return storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "name must not be empty")
		}
		current.Name = trimmed
		fields["name"] = trimmed
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
		ownerCount, err := countCampaignOwners(ctx, c.stores.Participant, campaignID)
		if err != nil {
			return storage.ParticipantRecord{}, err
		}
		decision := domainauthz.CanParticipantAccessChange(policyActor.CampaignAccess, targetAccessBefore, access, ownerCount)
		if !decision.Allowed {
			authErr := participantPolicyDecisionError(decision.ReasonCode)
			emitAuthzDecisionTelemetry(ctx, authzDecisionEvent{
				Store:      c.stores.Audit,
				CampaignID: campaignID,
				Capability: domainauthz.CapabilityManageParticipants,
				Decision:   authzDecisionDeny,
				ReasonCode: decision.ReasonCode,
				Actor:      policyActor,
				Err:        authErr,
				ExtraAttributes: map[string]any{
					"target_participant_id":     participantID,
					"target_campaign_access":    strings.TrimSpace(string(targetAccessBefore)),
					"requested_campaign_access": strings.TrimSpace(string(access)),
				},
			})
			return storage.ParticipantRecord{}, authErr
		}
		current.CampaignAccess = access
		fields["campaign_access"] = in.GetCampaignAccess().String()
	}
	if avatarSetID := in.GetAvatarSetId(); avatarSetID != nil {
		trimmed := strings.TrimSpace(avatarSetID.GetValue())
		current.AvatarSetID = trimmed
		fields["avatar_set_id"] = trimmed
	}
	if avatarAssetID := in.GetAvatarAssetId(); avatarAssetID != nil {
		trimmed := strings.TrimSpace(avatarAssetID.GetValue())
		current.AvatarAssetID = trimmed
		fields["avatar_asset_id"] = trimmed
	}
	if pronouns := in.GetPronouns(); pronouns != nil {
		current.Pronouns = sharedpronouns.FromProto(pronouns)
		fields["pronouns"] = current.Pronouns
	}
	if len(fields) == 0 {
		return storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "at least one field must be provided")
	}

	applier := c.stores.Applier()
	payloadFields := make(map[string]string, len(fields))
	for key, value := range fields {
		stringValue, ok := value.(string)
		if !ok {
			return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "participant update field %s must be string", key)
		}
		payloadFields[key] = stringValue
	}
	payload := participant.UpdatePayload{
		ParticipantID: ids.ParticipantID(participantID),
		Fields:        payloadFields,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.ParticipantRecord{}, grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Write,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeParticipantUpdate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "participant",
			EntityID:     participantID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: domainApplyErrorWithCodePreserve("apply event"),
		},
	)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}

	updated, err := c.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, grpcerror.Internal("load participant", err)
	}

	if shouldClearCampaignAIBindingOnAccessChange(targetAccessBefore, updated.CampaignAccess) {
		campaignRecord, campaignErr := c.stores.Campaign.Get(ctx, campaignID)
		if campaignErr != nil {
			return storage.ParticipantRecord{}, campaignErr
		}
		if strings.TrimSpace(campaignRecord.AIAgentID) != "" {
			if _, clearErr := clearCampaignAIBindingByCommand(
				ctx,
				c.stores,
				campaignID,
				actorID,
				actorType,
				grpcmeta.RequestIDFromContext(ctx),
				grpcmeta.InvocationIDFromContext(ctx),
			); clearErr != nil {
				return storage.ParticipantRecord{}, clearErr
			}
		}
	}

	return updated, nil
}
