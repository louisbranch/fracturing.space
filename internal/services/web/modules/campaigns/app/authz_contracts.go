package app

import "context"

// AuthorizationDecision captures one authorization check result.
type AuthorizationDecision struct {
	CheckID             string
	Evaluated           bool
	Allowed             bool
	ReasonCode          string
	ActorCampaignAccess string
}

// ParticipantGovernanceOperation scopes participant authz intent for policy evaluation.
type ParticipantGovernanceOperation string

const (
	ParticipantGovernanceOperationUnspecified  ParticipantGovernanceOperation = ""
	ParticipantGovernanceOperationMutate       ParticipantGovernanceOperation = "mutate"
	ParticipantGovernanceOperationAccessChange ParticipantGovernanceOperation = "access_change"
	ParticipantGovernanceOperationRemove       ParticipantGovernanceOperation = "remove"
)

// AuthorizationTarget scopes a check to a specific resource instance.
type AuthorizationTarget struct {
	ResourceID              string
	OwnerParticipantID      string
	TargetParticipantID     string
	TargetCampaignAccess    string
	RequestedCampaignAccess string
	ParticipantOperation    ParticipantGovernanceOperation
}

// AuthorizationCheck describes one authz request in batch evaluation.
type AuthorizationCheck struct {
	CheckID  string
	Action   AuthorizationAction
	Resource AuthorizationResource
	Target   *AuthorizationTarget
}

// AuthorizationGateway performs unary campaign action checks.
type AuthorizationGateway interface {
	CanCampaignAction(context.Context, string, AuthorizationAction, AuthorizationResource, *AuthorizationTarget) (AuthorizationDecision, error)
}

// BatchAuthorizationGateway performs batched campaign action checks.
type BatchAuthorizationGateway interface {
	BatchCanCampaignAction(context.Context, string, []AuthorizationCheck) ([]AuthorizationDecision, error)
}

// AuthzGateway combines unary and batch authorization checks.
type AuthzGateway interface {
	AuthorizationGateway
	BatchAuthorizationGateway
}

// AuthorizationAction defines an action dimension for campaign authz checks.
type AuthorizationAction string

// AuthorizationResource defines a resource dimension for campaign authz checks.
type AuthorizationResource string

const (
	AuthorizationActionManage AuthorizationAction = "manage"
	AuthorizationActionMutate AuthorizationAction = "mutate"

	AuthorizationResourceSession     AuthorizationResource = "session"
	AuthorizationResourceCampaign    AuthorizationResource = "campaign"
	AuthorizationResourceParticipant AuthorizationResource = "participant"
	AuthorizationResourceCharacter   AuthorizationResource = "character"
	AuthorizationResourceInvite      AuthorizationResource = "invite"
)
