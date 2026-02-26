package authz

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

// Action identifies what the caller wants to do.
type Action string

const (
	// ActionUnspecified represents an invalid or missing action.
	ActionUnspecified Action = ""
	// ActionRead represents read-only access.
	ActionRead Action = "read"
	// ActionManage represents governance/administrative mutation.
	ActionManage Action = "manage"
	// ActionMutate represents standard mutable operations.
	ActionMutate Action = "mutate"
	// ActionTransferOwnership represents ownership transfer operations.
	ActionTransferOwnership Action = "transfer_ownership"
)

// Resource identifies the campaign-scoped entity class a decision applies to.
type Resource string

const (
	// ResourceUnspecified represents an invalid or missing resource.
	ResourceUnspecified Resource = ""
	// ResourceCampaign is campaign metadata and lifecycle.
	ResourceCampaign Resource = "campaign"
	// ResourceParticipant is campaign participant records.
	ResourceParticipant Resource = "participant"
	// ResourceInvite is campaign invite records.
	ResourceInvite Resource = "invite"
	// ResourceSession is campaign session lifecycle and controls.
	ResourceSession Resource = "session"
	// ResourceCharacter is campaign character metadata and ownership.
	ResourceCharacter Resource = "character"
)

// Capability identifies a policy row via (action, resource).
type Capability struct {
	Action   Action
	Resource Resource
}

// Label returns a stable machine-readable capability label.
func (c Capability) Label() string {
	action := strings.TrimSpace(string(c.Action))
	resource := strings.TrimSpace(string(c.Resource))
	if action == "" || resource == "" {
		return "unknown"
	}
	return action + "_" + resource
}

// Valid reports whether the capability has both action and resource set.
func (c Capability) Valid() bool {
	return strings.TrimSpace(string(c.Action)) != "" && strings.TrimSpace(string(c.Resource)) != ""
}

// Canonical capabilities used by campaign policy checks.
var (
	CapabilityReadCampaign               = Capability{Action: ActionRead, Resource: ResourceCampaign}
	CapabilityReadParticipants           = Capability{Action: ActionRead, Resource: ResourceParticipant}
	CapabilityReadCharacters             = Capability{Action: ActionRead, Resource: ResourceCharacter}
	CapabilityReadSessions               = Capability{Action: ActionRead, Resource: ResourceSession}
	CapabilityReadInvites                = Capability{Action: ActionRead, Resource: ResourceInvite}
	CapabilityManageCampaign             = Capability{Action: ActionManage, Resource: ResourceCampaign}
	CapabilityManageParticipants         = Capability{Action: ActionManage, Resource: ResourceParticipant}
	CapabilityManageInvites              = Capability{Action: ActionManage, Resource: ResourceInvite}
	CapabilityManageSessions             = Capability{Action: ActionManage, Resource: ResourceSession}
	CapabilityMutateCharacters           = Capability{Action: ActionMutate, Resource: ResourceCharacter}
	CapabilityManageCharacters           = Capability{Action: ActionManage, Resource: ResourceCharacter}
	CapabilityTransferCharacterOwnership = Capability{Action: ActionTransferOwnership, Resource: ResourceCharacter}
)

// PolicyDecision reports if authorization is allowed and why.
type PolicyDecision struct {
	Allowed    bool
	ReasonCode string
}

// RolePolicyRow is one canonical row in the role/action/resource matrix.
type RolePolicyRow struct {
	Role     participant.CampaignAccess
	Action   Action
	Resource Resource
}

const (
	// ReasonAllowAccessLevel indicates the role/action/resource table allows access.
	ReasonAllowAccessLevel = "AUTHZ_ALLOW_ACCESS_LEVEL"
	// ReasonAllowAdminOverride indicates platform-admin override was used.
	ReasonAllowAdminOverride = "AUTHZ_ALLOW_ADMIN_OVERRIDE"
	// ReasonAllowResourceOwner indicates member-level character mutation passed owner check.
	ReasonAllowResourceOwner = "AUTHZ_ALLOW_RESOURCE_OWNER"

	// ReasonDenyAccessLevelRequired indicates role/action/resource is not allowed.
	ReasonDenyAccessLevelRequired = "AUTHZ_DENY_ACCESS_LEVEL_REQUIRED"
	// ReasonDenyNotResourceOwner indicates member-level character mutation failed owner check.
	ReasonDenyNotResourceOwner = "AUTHZ_DENY_NOT_RESOURCE_OWNER"
	// ReasonDenyOverrideReasonRequired indicates admin override omitted required reason text.
	ReasonDenyOverrideReasonRequired = "AUTHZ_DENY_OVERRIDE_REASON_REQUIRED"
	// ReasonDenyTargetIsOwner indicates a manager attempted to mutate an owner target.
	ReasonDenyTargetIsOwner = "AUTHZ_DENY_TARGET_IS_OWNER"
	// ReasonDenyLastOwnerGuard indicates operation would remove/demote final owner.
	ReasonDenyLastOwnerGuard = "AUTHZ_DENY_LAST_OWNER_GUARD"
	// ReasonDenyManagerOwnerMutationForbidden indicates manager attempted owner assignment.
	ReasonDenyManagerOwnerMutationForbidden = "AUTHZ_DENY_MANAGER_OWNER_MUTATION_FORBIDDEN"

	// ReasonDenyMissingIdentity indicates no participant-id/user-id identity was provided.
	ReasonDenyMissingIdentity = "AUTHZ_DENY_MISSING_IDENTITY"
	// ReasonDenyActorNotFound indicates identity did not resolve to campaign participant.
	ReasonDenyActorNotFound = "AUTHZ_DENY_ACTOR_NOT_FOUND"

	// ReasonErrorDependencyUnavailable indicates required dependency was missing.
	ReasonErrorDependencyUnavailable = "AUTHZ_ERROR_DEPENDENCY_UNAVAILABLE"
	// ReasonErrorActorLoad indicates participant resolution failed with internal error.
	ReasonErrorActorLoad = "AUTHZ_ERROR_ACTOR_LOAD"
	// ReasonErrorOwnerResolution indicates character-owner resolution failed.
	ReasonErrorOwnerResolution = "AUTHZ_ERROR_OWNER_RESOLUTION"
)

var rolePolicyTable = []RolePolicyRow{
	{Role: participant.CampaignAccessOwner, Action: ActionRead, Resource: ResourceCampaign},
	{Role: participant.CampaignAccessManager, Action: ActionRead, Resource: ResourceCampaign},
	{Role: participant.CampaignAccessMember, Action: ActionRead, Resource: ResourceCampaign},

	{Role: participant.CampaignAccessOwner, Action: ActionRead, Resource: ResourceParticipant},
	{Role: participant.CampaignAccessManager, Action: ActionRead, Resource: ResourceParticipant},
	{Role: participant.CampaignAccessMember, Action: ActionRead, Resource: ResourceParticipant},

	{Role: participant.CampaignAccessOwner, Action: ActionRead, Resource: ResourceCharacter},
	{Role: participant.CampaignAccessManager, Action: ActionRead, Resource: ResourceCharacter},
	{Role: participant.CampaignAccessMember, Action: ActionRead, Resource: ResourceCharacter},

	{Role: participant.CampaignAccessOwner, Action: ActionRead, Resource: ResourceSession},
	{Role: participant.CampaignAccessManager, Action: ActionRead, Resource: ResourceSession},
	{Role: participant.CampaignAccessMember, Action: ActionRead, Resource: ResourceSession},

	{Role: participant.CampaignAccessOwner, Action: ActionRead, Resource: ResourceInvite},
	{Role: participant.CampaignAccessManager, Action: ActionRead, Resource: ResourceInvite},

	{Role: participant.CampaignAccessOwner, Action: ActionManage, Resource: ResourceCampaign},

	{Role: participant.CampaignAccessOwner, Action: ActionManage, Resource: ResourceParticipant},
	{Role: participant.CampaignAccessManager, Action: ActionManage, Resource: ResourceParticipant},

	{Role: participant.CampaignAccessOwner, Action: ActionManage, Resource: ResourceInvite},
	{Role: participant.CampaignAccessManager, Action: ActionManage, Resource: ResourceInvite},

	{Role: participant.CampaignAccessOwner, Action: ActionManage, Resource: ResourceSession},
	{Role: participant.CampaignAccessManager, Action: ActionManage, Resource: ResourceSession},

	{Role: participant.CampaignAccessOwner, Action: ActionMutate, Resource: ResourceCharacter},
	{Role: participant.CampaignAccessManager, Action: ActionMutate, Resource: ResourceCharacter},
	{Role: participant.CampaignAccessMember, Action: ActionMutate, Resource: ResourceCharacter},

	{Role: participant.CampaignAccessOwner, Action: ActionManage, Resource: ResourceCharacter},
	{Role: participant.CampaignAccessManager, Action: ActionManage, Resource: ResourceCharacter},

	{Role: participant.CampaignAccessOwner, Action: ActionTransferOwnership, Resource: ResourceCharacter},
}

// PolicyTable returns a copy of the canonical role/action/resource matrix.
func PolicyTable() []RolePolicyRow {
	return append([]RolePolicyRow(nil), rolePolicyTable...)
}

// CapabilityFromActionResource maps action/resource inputs to a known capability.
func CapabilityFromActionResource(action Action, resource Resource) (Capability, bool) {
	candidate := Capability{Action: action, Resource: resource}
	if !candidate.Valid() {
		return Capability{}, false
	}
	for _, row := range rolePolicyTable {
		if row.Action == candidate.Action && row.Resource == candidate.Resource {
			return candidate, true
		}
	}
	return Capability{}, false
}

// CanCampaignAccess evaluates role/action/resource authorization using the
// canonical matrix and returns a machine-readable decision.
func CanCampaignAccess(access participant.CampaignAccess, capability Capability) PolicyDecision {
	if !capability.Valid() {
		return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyAccessLevelRequired}
	}
	for _, row := range rolePolicyTable {
		if row.Action != capability.Action || row.Resource != capability.Resource {
			continue
		}
		if row.Role == access {
			return PolicyDecision{Allowed: true, ReasonCode: ReasonAllowAccessLevel}
		}
	}
	return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyAccessLevelRequired}
}

// CanCharacterMutation checks baseline role access and member ownership guard for
// character mutations.
func CanCharacterMutation(access participant.CampaignAccess, actorParticipantID, ownerParticipantID string) PolicyDecision {
	decision := CanCampaignAccess(access, CapabilityMutateCharacters)
	if !decision.Allowed {
		return decision
	}
	if access != participant.CampaignAccessMember {
		return decision
	}
	actorParticipantID = strings.TrimSpace(actorParticipantID)
	ownerParticipantID = strings.TrimSpace(ownerParticipantID)
	if actorParticipantID == "" || ownerParticipantID == "" || actorParticipantID != ownerParticipantID {
		return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyNotResourceOwner}
	}
	return PolicyDecision{Allowed: true, ReasonCode: ReasonAllowResourceOwner}
}

// CanParticipantAccessChange evaluates role-based participant access updates and
// enforces manager/owner governance invariants.
func CanParticipantAccessChange(
	actorAccess participant.CampaignAccess,
	targetAccess participant.CampaignAccess,
	requestedAccess participant.CampaignAccess,
	ownerCount int,
) PolicyDecision {
	decision := CanCampaignAccess(actorAccess, CapabilityManageParticipants)
	if !decision.Allowed {
		return decision
	}
	if requestedAccess == participant.CampaignAccessUnspecified {
		return decision
	}
	if actorAccess == participant.CampaignAccessManager {
		if targetAccess == participant.CampaignAccessOwner {
			return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyTargetIsOwner}
		}
		if requestedAccess == participant.CampaignAccessOwner {
			return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyManagerOwnerMutationForbidden}
		}
	}
	if targetAccess == participant.CampaignAccessOwner && requestedAccess != participant.CampaignAccessOwner && ownerCount <= 1 {
		return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyLastOwnerGuard}
	}
	return decision
}

// CanParticipantRemoval evaluates role-based participant removal and enforces
// manager/owner governance invariants.
func CanParticipantRemoval(
	actorAccess participant.CampaignAccess,
	targetAccess participant.CampaignAccess,
	ownerCount int,
) PolicyDecision {
	decision := CanCampaignAccess(actorAccess, CapabilityManageParticipants)
	if !decision.Allowed {
		return decision
	}
	if actorAccess == participant.CampaignAccessManager && targetAccess == participant.CampaignAccessOwner {
		return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyTargetIsOwner}
	}
	if targetAccess == participant.CampaignAccessOwner && ownerCount <= 1 {
		return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyLastOwnerGuard}
	}
	return decision
}
