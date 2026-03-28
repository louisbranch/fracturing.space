package authz

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
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
// Accessor functions prevent accidental reassignment of policy constants.

// CapabilityReadCampaign returns the campaign read capability.
func CapabilityReadCampaign() Capability {
	return Capability{Action: ActionRead, Resource: ResourceCampaign}
}

// CapabilityReadInvites returns the invite read capability.
func CapabilityReadInvites() Capability {
	return Capability{Action: ActionRead, Resource: ResourceInvite}
}

// CapabilityManageCampaign returns the campaign manage capability.
func CapabilityManageCampaign() Capability {
	return Capability{Action: ActionManage, Resource: ResourceCampaign}
}

// CapabilityManageParticipants returns the participant manage capability.
func CapabilityManageParticipants() Capability {
	return Capability{Action: ActionManage, Resource: ResourceParticipant}
}

// CapabilityManageInvites returns the invite manage capability.
func CapabilityManageInvites() Capability {
	return Capability{Action: ActionManage, Resource: ResourceInvite}
}

// CapabilityManageSessions returns the session manage capability.
func CapabilityManageSessions() Capability {
	return Capability{Action: ActionManage, Resource: ResourceSession}
}

// CapabilityMutateCharacters returns the character mutate capability.
func CapabilityMutateCharacters() Capability {
	return Capability{Action: ActionMutate, Resource: ResourceCharacter}
}

// CapabilityManageCharacters returns the character manage capability.
func CapabilityManageCharacters() Capability {
	return Capability{Action: ActionManage, Resource: ResourceCharacter}
}

// CapabilityTransferCharacterOwnership returns the character ownership transfer capability.
func CapabilityTransferCharacterOwnership() Capability {
	return Capability{Action: ActionTransferOwnership, Resource: ResourceCharacter}
}

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
	// ReasonDenyTargetIsAIParticipant indicates participant removal target is AI-controlled.
	ReasonDenyTargetIsAIParticipant = "AUTHZ_DENY_TARGET_IS_AI_PARTICIPANT"
	// ReasonDenyTargetOwnsActiveCharacters indicates participant removal target still owns active characters.
	ReasonDenyTargetOwnsActiveCharacters = "AUTHZ_DENY_TARGET_OWNS_ACTIVE_CHARACTERS"
	// ReasonDenyTargetControlsActiveCharacters indicates participant removal target still controls active characters.
	ReasonDenyTargetControlsActiveCharacters = "AUTHZ_DENY_TARGET_CONTROLS_ACTIVE_CHARACTERS"

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

// policyEntry declares which roles are allowed for a specific capability.
// Roles not listed are implicitly denied.
type policyEntry struct {
	Action       Action
	Resource     Resource
	AllowedRoles []participant.CampaignAccess
}

// allActions enumerates every recognized Action value (excluding Unspecified).
var allActions = []Action{
	ActionRead,
	ActionManage,
	ActionMutate,
	ActionTransferOwnership,
}

// allResources enumerates every recognized Resource value (excluding Unspecified).
var allResources = []Resource{
	ResourceCampaign,
	ResourceParticipant,
	ResourceInvite,
	ResourceSession,
	ResourceCharacter,
}

// allRoles enumerates every recognized campaign role (excluding Unspecified).
var allRoles = []participant.CampaignAccess{
	participant.CampaignAccessOwner,
	participant.CampaignAccessManager,
	participant.CampaignAccessMember,
}

// policyMatrix is the canonical role/action/resource authorization matrix.
// Each entry declares which roles are allowed for a specific (action, resource)
// capability. The exhaustiveness test validates that every recognized capability
// has exactly one entry and every Capability*() accessor maps to an entry.
//
// The matrix evaluates (actor role x target access x operation) tuples. Some
// intersections are intentionally undefined and fall through to deny-by-default.
// Admin override (ReasonAllowAdminOverride) bypasses the matrix entirely.
var policyMatrix = []policyEntry{
	// Read capabilities — broad access.
	{ActionRead, ResourceCampaign, roles(participant.CampaignAccessOwner, participant.CampaignAccessManager, participant.CampaignAccessMember)},
	{ActionRead, ResourceParticipant, roles(participant.CampaignAccessOwner, participant.CampaignAccessManager, participant.CampaignAccessMember)},
	{ActionRead, ResourceCharacter, roles(participant.CampaignAccessOwner, participant.CampaignAccessManager, participant.CampaignAccessMember)},
	{ActionRead, ResourceSession, roles(participant.CampaignAccessOwner, participant.CampaignAccessManager, participant.CampaignAccessMember)},
	{ActionRead, ResourceInvite, roles(participant.CampaignAccessOwner, participant.CampaignAccessManager)},

	// Manage capabilities — governance and administrative mutation.
	{ActionManage, ResourceCampaign, roles(participant.CampaignAccessOwner, participant.CampaignAccessManager)},
	{ActionManage, ResourceParticipant, roles(participant.CampaignAccessOwner, participant.CampaignAccessManager)},
	{ActionManage, ResourceInvite, roles(participant.CampaignAccessOwner, participant.CampaignAccessManager)},
	{ActionManage, ResourceSession, roles(participant.CampaignAccessOwner, participant.CampaignAccessManager)},
	{ActionManage, ResourceCharacter, roles(participant.CampaignAccessOwner, participant.CampaignAccessManager)},

	// Mutate capabilities — standard mutable operations.
	{ActionMutate, ResourceCharacter, roles(participant.CampaignAccessOwner, participant.CampaignAccessManager, participant.CampaignAccessMember)},

	// Transfer capabilities — ownership transfers.
	{ActionTransferOwnership, ResourceCharacter, roles(participant.CampaignAccessOwner)},
}

// roles is a convenience constructor for AllowedRoles slices.
func roles(rr ...participant.CampaignAccess) []participant.CampaignAccess { return rr }

// policyIndex is the lookup structure built from policyMatrix.
// Key is "action:resource", value is the set of allowed roles.
type policyIndex map[string]map[participant.CampaignAccess]bool

func buildPolicyIndex() policyIndex {
	idx := make(policyIndex, len(policyMatrix))
	for _, entry := range policyMatrix {
		key := string(entry.Action) + ":" + string(entry.Resource)
		allowed := make(map[participant.CampaignAccess]bool, len(entry.AllowedRoles))
		for _, role := range entry.AllowedRoles {
			allowed[role] = true
		}
		idx[key] = allowed
	}
	return idx
}

var matrixIndex = buildPolicyIndex()

// PolicyTable returns the canonical role/action/resource matrix expanded to
// individual rows for inspection and backward-compatible iteration.
func PolicyTable() []RolePolicyRow {
	var rows []RolePolicyRow
	for _, entry := range policyMatrix {
		for _, role := range entry.AllowedRoles {
			rows = append(rows, RolePolicyRow{
				Role:     role,
				Action:   entry.Action,
				Resource: entry.Resource,
			})
		}
	}
	return rows
}

// CapabilityFromActionResource maps action/resource inputs to a known capability.
func CapabilityFromActionResource(action Action, resource Resource) (Capability, bool) {
	candidate := Capability{Action: action, Resource: resource}
	if !candidate.Valid() {
		return Capability{}, false
	}
	key := string(action) + ":" + string(resource)
	if _, ok := matrixIndex[key]; ok {
		return candidate, true
	}
	return Capability{}, false
}

// CanCampaignAccess evaluates role/action/resource authorization using the
// canonical matrix and returns a machine-readable decision.
func CanCampaignAccess(access participant.CampaignAccess, capability Capability) PolicyDecision {
	if !capability.Valid() {
		return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyAccessLevelRequired}
	}
	key := string(capability.Action) + ":" + string(capability.Resource)
	if allowed, ok := matrixIndex[key]; ok && allowed[access] {
		return PolicyDecision{Allowed: true, ReasonCode: ReasonAllowAccessLevel}
	}
	return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyAccessLevelRequired}
}

// CanCharacterMutation checks baseline role access and member ownership guard for
// character mutations.
func CanCharacterMutation(access participant.CampaignAccess, actorParticipantID, ownerParticipantID ids.ParticipantID) PolicyDecision {
	decision := CanCampaignAccess(access, CapabilityMutateCharacters())
	if !decision.Allowed {
		return decision
	}
	if access != participant.CampaignAccessMember {
		return decision
	}
	actor := ids.ParticipantID(strings.TrimSpace(actorParticipantID.String()))
	owner := ids.ParticipantID(strings.TrimSpace(ownerParticipantID.String()))
	if actor == "" || owner == "" || actor != owner {
		return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyNotResourceOwner}
	}
	return PolicyDecision{Allowed: true, ReasonCode: ReasonAllowResourceOwner}
}

// CanParticipantMutation evaluates baseline participant-governance mutation
// access and manager owner-target guard rails.
func CanParticipantMutation(actorAccess participant.CampaignAccess, targetAccess participant.CampaignAccess) PolicyDecision {
	decision := CanCampaignAccess(actorAccess, CapabilityManageParticipants())
	if !decision.Allowed {
		return decision
	}
	if actorAccess == participant.CampaignAccessManager && targetAccess == participant.CampaignAccessOwner {
		return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyTargetIsOwner}
	}
	return decision
}

// CanParticipantAccessChange evaluates role-based participant access updates and
// enforces manager/owner governance invariants.
func CanParticipantAccessChange(
	actorAccess participant.CampaignAccess,
	targetAccess participant.CampaignAccess,
	requestedAccess participant.CampaignAccess,
	ownerCount int,
) PolicyDecision {
	decision := CanParticipantMutation(actorAccess, targetAccess)
	if !decision.Allowed {
		return decision
	}
	if requestedAccess == participant.CampaignAccessUnspecified {
		return decision
	}
	if actorAccess == participant.CampaignAccessManager {
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
	targetController participant.Controller,
) PolicyDecision {
	decision := CanParticipantMutation(actorAccess, targetAccess)
	if !decision.Allowed {
		return decision
	}
	if targetController == participant.ControllerAI {
		return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyTargetIsAIParticipant}
	}
	if targetAccess == participant.CampaignAccessOwner && ownerCount <= 1 {
		return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyLastOwnerGuard}
	}
	return decision
}

// CanParticipantRemovalEligibility evaluates participant removal and enforces
// active character ownership/control guards after role-based invariants pass.
func CanParticipantRemovalEligibility(
	actorAccess participant.CampaignAccess,
	targetAccess participant.CampaignAccess,
	ownerCount int,
	targetController participant.Controller,
	targetOwnsActiveCharacters bool,
	targetControlsActiveCharacters bool,
) PolicyDecision {
	decision := CanParticipantRemoval(actorAccess, targetAccess, ownerCount, targetController)
	if !decision.Allowed {
		return decision
	}
	if targetOwnsActiveCharacters {
		return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyTargetOwnsActiveCharacters}
	}
	if targetControlsActiveCharacters {
		return PolicyDecision{Allowed: false, ReasonCode: ReasonDenyTargetControlsActiveCharacters}
	}
	return decision
}
