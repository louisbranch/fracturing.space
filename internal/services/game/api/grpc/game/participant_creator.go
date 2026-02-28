package game

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
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
	if err := requirePolicy(ctx, c.stores, domainauthz.CapabilityManageParticipants, campaignRecord); err != nil {
		return storage.ParticipantRecord{}, err
	}

	userID := strings.TrimSpace(in.GetUserId())
	profile := loadSocialProfileSnapshot(ctx, c.stores.Social, userID)

	name := strings.TrimSpace(in.GetName())
	if name == "" {
		if profile.Name != "" {
			name = profile.Name
		} else if userID != "" {
			name = defaultUnknownParticipantName(campaignRecord.Locale)
		}
	}
	if name == "" {
		return storage.ParticipantRecord{}, apperrors.New(apperrors.CodeParticipantEmptyDisplayName, "name is required")
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
	avatarSetID := strings.TrimSpace(in.GetAvatarSetId())
	avatarAssetID := strings.TrimSpace(in.GetAvatarAssetId())
	if avatarSetID == "" && avatarAssetID == "" {
		avatarSetID = profile.AvatarSetID
		avatarAssetID = profile.AvatarAssetID
	}
	pronouns := sharedpronouns.FromProto(in.GetPronouns())
	if pronouns == "" {
		pronouns = profile.Pronouns
	}
	if pronouns == "" && userID != "" {
		pronouns = defaultUnknownParticipantPronouns()
	}

	applier := c.stores.Applier()
	if c.stores.Domain == nil {
		return storage.ParticipantRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := participant.JoinPayload{
		ParticipantID:  participantID,
		UserID:         userID,
		Name:           name,
		Role:           string(role),
		Controller:     string(controller),
		CampaignAccess: string(access),
		AvatarSetID:    avatarSetID,
		AvatarAssetID:  avatarAssetID,
		Pronouns:       pronouns,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.ParticipantRecord{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeParticipantJoin,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "participant",
			EntityID:     participantID,
			PayloadJSON:  payloadJSON,
		}),
		domainCommandApplyOptions{
			applyErr: domainApplyErrorWithCodePreserve("apply event"),
		},
	)
	if err != nil {
		return storage.ParticipantRecord{}, err
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
	policyActor, err := requirePolicyActor(ctx, c.stores, domainauthz.CapabilityManageParticipants, campaignRecord)
	if err != nil {
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
	targetAccessBefore := current.CampaignAccess
	if in.GetCampaignAccess() == campaignv1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED {
		decision := domainauthz.CanParticipantMutation(policyActor.CampaignAccess, targetAccessBefore)
		if !decision.Allowed {
			authErr := participantPolicyDecisionError(decision.ReasonCode)
			emitAuthzDecisionTelemetry(
				ctx,
				c.stores.Audit,
				campaignID,
				domainauthz.CapabilityManageParticipants,
				authzDecisionDeny,
				decision.ReasonCode,
				policyActor,
				authErr,
				map[string]any{
					"target_participant_id":  participantID,
					"target_campaign_access": strings.TrimSpace(string(targetAccessBefore)),
				},
			)
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
			emitAuthzDecisionTelemetry(
				ctx,
				c.stores.Audit,
				campaignID,
				domainauthz.CapabilityManageParticipants,
				authzDecisionDeny,
				decision.ReasonCode,
				policyActor,
				authErr,
				map[string]any{
					"target_participant_id":     participantID,
					"target_campaign_access":    strings.TrimSpace(string(targetAccessBefore)),
					"requested_campaign_access": strings.TrimSpace(string(access)),
				},
			)
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

	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
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
		domainCommandApplyOptions{
			applyErr: domainApplyErrorWithCodePreserve("apply event"),
		},
	)
	if err != nil {
		return storage.ParticipantRecord{}, err
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
	policyActor, err := requirePolicyActor(ctx, c.stores, domainauthz.CapabilityManageParticipants, campaignRecord)
	if err != nil {
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
	ownerCount, err := countCampaignOwners(ctx, c.stores.Participant, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	targetOwnsActiveCharacters, err := participantOwnsActiveCharacters(ctx, c.stores.Character, campaignID, participantID)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}
	decision := domainauthz.CanParticipantRemovalWithOwnedResources(
		policyActor.CampaignAccess,
		current.CampaignAccess,
		ownerCount,
		targetOwnsActiveCharacters,
	)
	if !decision.Allowed {
		authErr := participantPolicyDecisionError(decision.ReasonCode)
		emitAuthzDecisionTelemetry(
			ctx,
			c.stores.Audit,
			campaignID,
			domainauthz.CapabilityManageParticipants,
			authzDecisionDeny,
			decision.ReasonCode,
			policyActor,
			authErr,
			map[string]any{
				"target_participant_id":         participantID,
				"target_campaign_access":        strings.TrimSpace(string(current.CampaignAccess)),
				"target_owns_active_characters": targetOwnsActiveCharacters,
			},
		)
		return storage.ParticipantRecord{}, authErr
	}

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

	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		c.stores.Domain,
		applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeParticipantLeave,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "participant",
			EntityID:     participantID,
			PayloadJSON:  payloadJSON,
		}),
		domainCommandApplyOptions{
			applyErr: domainApplyErrorWithCodePreserve("apply event"),
		},
	)
	if err != nil {
		return storage.ParticipantRecord{}, err
	}

	return current, nil
}

// ensureParticipantHasNoOwnedCharacters scans projection-backed character state
// and returns an error if any active character is currently owned by participantID.
func ensureParticipantHasNoOwnedCharacters(ctx context.Context, characters storage.CharacterStore, campaignID, participantID string) error {
	ownsActiveCharacters, err := participantOwnsActiveCharacters(ctx, characters, campaignID, participantID)
	if err != nil {
		return err
	}
	if ownsActiveCharacters {
		return status.Error(codes.FailedPrecondition, "participant owns active characters; transfer ownership first")
	}
	return nil
}

// countCampaignOwners returns current owner-seat count for invariant checks.
func countCampaignOwners(ctx context.Context, participants storage.ParticipantStore, campaignID string) (int, error) {
	if participants == nil {
		return 0, status.Error(codes.Internal, "participant store is not configured")
	}
	records, err := participants.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return 0, status.Errorf(codes.Internal, "list participants: %v", err)
	}
	ownerCount := 0
	for _, record := range records {
		if record.CampaignAccess == participant.CampaignAccessOwner {
			ownerCount++
		}
	}
	return ownerCount, nil
}

// participantPolicyDecisionError maps policy reason codes to denial messages.
func participantPolicyDecisionError(reasonCode string) error {
	switch strings.TrimSpace(reasonCode) {
	case domainauthz.ReasonDenyManagerOwnerMutationForbidden:
		return status.Error(codes.PermissionDenied, "manager cannot assign owner access")
	case domainauthz.ReasonDenyTargetIsOwner:
		return status.Error(codes.PermissionDenied, "manager cannot mutate owner participant")
	case domainauthz.ReasonDenyLastOwnerGuard:
		return status.Error(codes.PermissionDenied, "cannot remove or demote final owner")
	case domainauthz.ReasonDenyTargetOwnsActiveCharacters:
		return status.Error(codes.FailedPrecondition, "participant owns active characters; transfer ownership first")
	default:
		return status.Error(codes.PermissionDenied, "participant lacks permission")
	}
}
