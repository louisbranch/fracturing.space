package game

import (
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/telemetry"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// policyAction identifies a campaign management action requiring access checks.
type policyAction int

const (
	// policyActionManageParticipants allows managing participants.
	policyActionManageParticipants policyAction = iota + 1
	// policyActionManageInvites allows managing invites.
	policyActionManageInvites
	// policyActionManageSessions allows session lifecycle and controls.
	policyActionManageSessions
	// policyActionManageCharacters allows character write operations.
	policyActionManageCharacters
	// policyActionManageCampaign allows campaign governance mutations.
	policyActionManageCampaign
	// policyActionReadCampaign allows campaign-scoped read operations.
	policyActionReadCampaign
)

const (
	authzEventDecisionName                = "telemetry.authz.decision"
	authzDecisionAllow                    = "allow"
	authzDecisionDeny                     = "deny"
	authzDecisionOverride                 = "override"
	authzPlatformRoleHeader               = grpcmeta.PlatformRoleHeader
	authzOverrideReasonHeader             = grpcmeta.AuthzOverrideReasonHeader
	authzPlatformRoleAdmin                = grpcmeta.PlatformRoleAdmin
	authzReasonAllowAdminOverride         = "AUTHZ_ALLOW_ADMIN_OVERRIDE"
	authzReasonAllowAccessLevel           = "AUTHZ_ALLOW_ACCESS_LEVEL"
	authzReasonAllowResourceOwner         = "AUTHZ_ALLOW_RESOURCE_OWNER"
	authzReasonDenyAccessLevelRequired    = "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED"
	authzReasonDenyMissingIdentity        = "AUTHZ_DENY_MISSING_IDENTITY"
	authzReasonDenyActorNotFound          = "AUTHZ_DENY_ACTOR_NOT_FOUND"
	authzReasonDenyNotResourceOwner       = "AUTHZ_DENY_NOT_RESOURCE_OWNER"
	authzReasonDenyOverrideReasonRequired = "AUTHZ_DENY_OVERRIDE_REASON_REQUIRED"
	authzReasonErrorDependencyUnavailable = "AUTHZ_ERROR_DEPENDENCY_UNAVAILABLE"
	authzReasonErrorActorLoad             = "AUTHZ_ERROR_ACTOR_LOAD"
	authzReasonErrorOwnerResolution       = "AUTHZ_ERROR_OWNER_RESOLUTION"
)

// requirePolicy ensures the participant has access for the requested action.
func requirePolicy(ctx context.Context, stores Stores, action policyAction, campaignRecord storage.CampaignRecord) error {
	_, err := requirePolicyActor(ctx, stores, action, campaignRecord)
	return err
}

// requireReadPolicy ensures the actor can access campaign-scoped reads.
func requireReadPolicy(ctx context.Context, stores Stores, campaignRecord storage.CampaignRecord) error {
	return requirePolicy(ctx, stores, policyActionReadCampaign, campaignRecord)
}

// requirePolicyActor ensures access and returns the resolved participant actor.
func requirePolicyActor(ctx context.Context, stores Stores, action policyAction, campaignRecord storage.CampaignRecord) (storage.ParticipantRecord, error) {
	actor, reasonCode, err := authorizePolicyActor(ctx, stores, action, campaignRecord)
	if err != nil {
		emitAuthzDecisionTelemetry(ctx, stores.Telemetry, campaignRecord.ID, action, authzDecisionDeny, reasonCode, actor, err, nil)
		return storage.ParticipantRecord{}, err
	}
	emitAuthzDecisionTelemetry(
		ctx,
		stores.Telemetry,
		campaignRecord.ID,
		action,
		authzDecisionForReason(reasonCode),
		reasonCode,
		actor,
		nil,
		authzExtraAttributesForReason(ctx, reasonCode),
	)
	return actor, nil
}

// requireCharacterMutationPolicy enforces role policy and owner-only mutation for members.
func requireCharacterMutationPolicy(
	ctx context.Context,
	stores Stores,
	campaignRecord storage.CampaignRecord,
	characterID string,
) (storage.ParticipantRecord, error) {
	actor, reasonCode, err := authorizePolicyActor(ctx, stores, policyActionManageCharacters, campaignRecord)
	characterAttributes := map[string]any{
		"character_id": strings.TrimSpace(characterID),
	}
	if err != nil {
		emitAuthzDecisionTelemetry(
			ctx,
			stores.Telemetry,
			campaignRecord.ID,
			policyActionManageCharacters,
			authzDecisionDeny,
			reasonCode,
			actor,
			err,
			characterAttributes,
		)
		return storage.ParticipantRecord{}, err
	}
	decision := authzDecisionForReason(reasonCode)
	overrideAttributes := mergeAuthzAttributes(characterAttributes, authzExtraAttributesForReason(ctx, reasonCode))
	if decision == authzDecisionOverride {
		emitAuthzDecisionTelemetry(
			ctx,
			stores.Telemetry,
			campaignRecord.ID,
			policyActionManageCharacters,
			decision,
			reasonCode,
			actor,
			nil,
			overrideAttributes,
		)
		return actor, nil
	}
	if actor.CampaignAccess != participant.CampaignAccessMember {
		emitAuthzDecisionTelemetry(
			ctx,
			stores.Telemetry,
			campaignRecord.ID,
			policyActionManageCharacters,
			authzDecisionAllow,
			reasonCode,
			actor,
			nil,
			characterAttributes,
		)
		return actor, nil
	}
	ownerParticipantID, err := resolveCharacterMutationOwnerParticipantID(ctx, stores, campaignRecord.ID, characterID)
	if err != nil {
		emitAuthzDecisionTelemetry(ctx, stores.Telemetry, campaignRecord.ID, policyActionManageCharacters, authzDecisionDeny, authzReasonErrorOwnerResolution, actor, err, characterAttributes)
		return storage.ParticipantRecord{}, err
	}
	if ownerParticipantID == "" || ownerParticipantID != actor.ID {
		err := status.Error(codes.PermissionDenied, "participant lacks permission")
		emitAuthzDecisionTelemetry(ctx, stores.Telemetry, campaignRecord.ID, policyActionManageCharacters, authzDecisionDeny, authzReasonDenyNotResourceOwner, actor, err, map[string]any{
			"character_id":         characterAttributes["character_id"],
			"owner_participant_id": ownerParticipantID,
		})
		return storage.ParticipantRecord{}, err
	}
	emitAuthzDecisionTelemetry(ctx, stores.Telemetry, campaignRecord.ID, policyActionManageCharacters, authzDecisionAllow, authzReasonAllowResourceOwner, actor, nil, characterAttributes)
	return actor, nil
}

func authorizePolicyActor(ctx context.Context, stores Stores, action policyAction, campaignRecord storage.CampaignRecord) (storage.ParticipantRecord, string, error) {
	if overrideReason, overrideRequested := adminOverrideFromContext(ctx); overrideRequested {
		if overrideReason == "" {
			return storage.ParticipantRecord{}, authzReasonDenyOverrideReasonRequired, status.Error(codes.PermissionDenied, "admin override reason is required")
		}
		return storage.ParticipantRecord{
			ID:     strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx)),
			UserID: strings.TrimSpace(grpcmeta.UserIDFromContext(ctx)),
		}, authzReasonAllowAdminOverride, nil
	}

	if stores.Participant == nil {
		return storage.ParticipantRecord{}, authzReasonErrorDependencyUnavailable, status.Error(codes.Internal, "participant store is not configured")
	}
	actor, reasonCode, err := resolvePolicyActor(ctx, stores.Participant, campaignRecord.ID)
	if err != nil {
		return storage.ParticipantRecord{}, reasonCode, err
	}
	if !canPerformPolicyAction(action, actor.CampaignAccess) {
		return storage.ParticipantRecord{}, authzReasonDenyAccessLevelRequired, status.Error(codes.PermissionDenied, "participant lacks permission")
	}
	return actor, authzReasonAllowAccessLevel, nil
}

func resolvePolicyActor(ctx context.Context, participants storage.ParticipantStore, campaignID string) (storage.ParticipantRecord, string, error) {
	actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if actorID != "" {
		actor, err := participants.GetParticipant(ctx, campaignID, actorID)
		if err != nil {
			if err == storage.ErrNotFound {
				return storage.ParticipantRecord{}, authzReasonDenyActorNotFound, status.Error(codes.PermissionDenied, "participant lacks permission")
			}
			return storage.ParticipantRecord{}, authzReasonErrorActorLoad, status.Errorf(codes.Internal, "load participant: %v", err)
		}
		return actor, authzReasonAllowAccessLevel, nil
	}

	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return storage.ParticipantRecord{}, authzReasonDenyMissingIdentity, status.Error(codes.PermissionDenied, "missing participant identity")
	}

	campaignParticipants, err := participants.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return storage.ParticipantRecord{}, authzReasonErrorActorLoad, status.Errorf(codes.Internal, "load participants: %v", err)
	}
	for _, participantRecord := range campaignParticipants {
		if strings.TrimSpace(participantRecord.UserID) == userID {
			return participantRecord, authzReasonAllowAccessLevel, nil
		}
	}
	return storage.ParticipantRecord{}, authzReasonDenyActorNotFound, status.Error(codes.PermissionDenied, "participant lacks permission")
}

// canPerformPolicyAction enforces the v0 access model for management actions.
func canPerformPolicyAction(action policyAction, access participant.CampaignAccess) bool {
	switch action {
	case policyActionManageParticipants, policyActionManageInvites, policyActionManageSessions:
		return access == participant.CampaignAccessOwner || access == participant.CampaignAccessManager
	case policyActionManageCharacters:
		return access == participant.CampaignAccessOwner || access == participant.CampaignAccessManager || access == participant.CampaignAccessMember
	case policyActionManageCampaign:
		return access == participant.CampaignAccessOwner
	case policyActionReadCampaign:
		return access == participant.CampaignAccessOwner ||
			access == participant.CampaignAccessManager ||
			access == participant.CampaignAccessMember
	default:
		return false
	}
}

// characterOwnershipState tracks the current owner and deletion state of a character
// as resolved from the event journal.
type characterOwnershipState struct {
	ownerParticipantID string
	deleted            bool
}

// replayCharacterOwnership replays the event journal for a campaign and returns
// a map from character ID to its ownership state. Participant-deletion guards
// consume this map to avoid duplicating pagination and event-matching logic.
func replayCharacterOwnership(ctx context.Context, events storage.EventStore, campaignID string) (map[string]characterOwnershipState, error) {
	if events == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	ownership := make(map[string]characterOwnershipState)

	const pageSize = 200
	afterSeq := uint64(0)
	for {
		page, err := events.ListEvents(ctx, campaignID, afterSeq, pageSize)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load events: %v", err)
		}
		if len(page) == 0 {
			break
		}

		for _, evt := range page {
			if evt.Seq > afterSeq {
				afterSeq = evt.Seq
			}
			characterID := strings.TrimSpace(evt.EntityID)
			if characterID == "" {
				continue
			}

			switch evt.Type {
			case eventTypeCharacterCreated:
				ownerParticipantID := ""
				if len(evt.PayloadJSON) > 0 {
					var payload character.CreatePayload
					if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
						return nil, status.Errorf(codes.Internal, "decode character.created payload: %v", err)
					}
					ownerParticipantID = strings.TrimSpace(payload.OwnerParticipantID)
				}
				if ownerParticipantID == "" {
					ownerParticipantID = strings.TrimSpace(evt.ActorID)
				}
				ownership[characterID] = characterOwnershipState{
					ownerParticipantID: ownerParticipantID,
				}
			case eventTypeCharacterUpdated:
				if len(evt.PayloadJSON) == 0 {
					continue
				}
				var payload character.UpdatePayload
				if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
					return nil, status.Errorf(codes.Internal, "decode character.updated payload: %v", err)
				}
				if updatedOwnerParticipantID, ok := payload.Fields["owner_participant_id"]; ok {
					state := ownership[characterID]
					state.ownerParticipantID = strings.TrimSpace(updatedOwnerParticipantID)
					ownership[characterID] = state
				}
			case eventTypeCharacterDeleted:
				state := ownership[characterID]
				state.deleted = true
				ownership[characterID] = state
			}
		}

		if len(page) < pageSize {
			break
		}
	}

	return ownership, nil
}

// resolveCharacterMutationOwnerParticipantID resolves the owner participant for
// member-only character mutation checks.
//
// The lookup prefers event-backed ownership when an event store is available,
// because ownership and controller can diverge. When event replay is not
// configured, it falls back to the character projection participant id.
func resolveCharacterMutationOwnerParticipantID(
	ctx context.Context,
	stores Stores,
	campaignID string,
	characterID string,
) (string, error) {
	characterID = strings.TrimSpace(characterID)
	if characterID == "" {
		return "", nil
	}

	if stores.Event != nil {
		ownership, err := replayCharacterOwnership(ctx, stores.Event, campaignID)
		if err != nil {
			return "", err
		}
		state, ok := ownership[characterID]
		if !ok {
			return "", nil
		}
		return strings.TrimSpace(state.ownerParticipantID), nil
	}

	if stores.Character == nil {
		return "", status.Error(codes.Internal, "character owner store is not configured")
	}
	characterRecord, err := stores.Character.GetCharacter(ctx, campaignID, characterID)
	if err != nil {
		if err == storage.ErrNotFound {
			return "", nil
		}
		return "", status.Errorf(codes.Internal, "load character owner: %v", err)
	}
	return strings.TrimSpace(characterRecord.ParticipantID), nil
}

func adminOverrideFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", false
	}
	role := strings.ToUpper(strings.TrimSpace(grpcmeta.FirstMetadataValue(md, authzPlatformRoleHeader)))
	if role != authzPlatformRoleAdmin {
		return "", false
	}
	reason := strings.TrimSpace(grpcmeta.FirstMetadataValue(md, authzOverrideReasonHeader))
	return reason, true
}

func authzDecisionForReason(reasonCode string) string {
	if reasonCode == authzReasonAllowAdminOverride {
		return authzDecisionOverride
	}
	return authzDecisionAllow
}

func authzExtraAttributesForReason(ctx context.Context, reasonCode string) map[string]any {
	if reasonCode != authzReasonAllowAdminOverride {
		return nil
	}
	reason, requested := adminOverrideFromContext(ctx)
	if !requested || reason == "" {
		return nil
	}
	return map[string]any{
		"override_reason": reason,
	}
}

func mergeAuthzAttributes(attributes ...map[string]any) map[string]any {
	var merged map[string]any
	for _, attrs := range attributes {
		if len(attrs) == 0 {
			continue
		}
		if merged == nil {
			merged = make(map[string]any, len(attrs))
		}
		for key, value := range attrs {
			merged[key] = value
		}
	}
	return merged
}

func policyActionLabel(action policyAction) string {
	switch action {
	case policyActionManageParticipants:
		return "manage_participants"
	case policyActionManageInvites:
		return "manage_invites"
	case policyActionManageSessions:
		return "manage_sessions"
	case policyActionManageCharacters:
		return "manage_characters"
	case policyActionManageCampaign:
		return "manage_campaign"
	case policyActionReadCampaign:
		return "read_campaign"
	default:
		return "unknown"
	}
}

func emitAuthzDecisionTelemetry(
	ctx context.Context,
	store storage.TelemetryStore,
	campaignID string,
	action policyAction,
	decision string,
	reasonCode string,
	actor storage.ParticipantRecord,
	authErr error,
	extraAttributes map[string]any,
) {
	severity := telemetry.SeverityInfo
	code := codes.OK
	if authErr != nil {
		severity = telemetry.SeverityWarn
		if st, ok := status.FromError(authErr); ok {
			code = st.Code()
		}
		if code == codes.Internal {
			severity = telemetry.SeverityError
		}
	}

	actorID := strings.TrimSpace(actor.ID)
	if actorID == "" {
		actorID = strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	}
	actorType := "system"
	if actorID != "" {
		actorType = "participant"
	}

	var traceID, spanID string
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		traceID = sc.TraceID().String()
		spanID = sc.SpanID().String()
	}

	attributes := map[string]any{
		"decision":      decision,
		"reason_code":   reasonCode,
		"policy_action": policyActionLabel(action),
		"grpc_code":     code.String(),
	}
	if access := strings.TrimSpace(string(actor.CampaignAccess)); access != "" {
		attributes["campaign_access"] = access
	}
	if userID := strings.TrimSpace(actor.UserID); userID != "" {
		attributes["actor_user_id"] = userID
	}
	for key, value := range extraAttributes {
		attributes[key] = value
	}

	emitter := telemetry.NewEmitter(store)
	if err := emitter.Emit(ctx, storage.TelemetryEvent{
		EventName:    authzEventDecisionName,
		Severity:     string(severity),
		CampaignID:   strings.TrimSpace(campaignID),
		ActorType:    actorType,
		ActorID:      actorID,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		TraceID:      traceID,
		SpanID:       spanID,
		Attributes:   attributes,
	}); err != nil {
		log.Printf("telemetry emit %s: %v", authzEventDecisionName, err)
	}
}
