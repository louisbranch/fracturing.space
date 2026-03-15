// Package authz provides authorization policy enforcement, evaluator, and
// telemetry shared across entity-scoped transport subpackages.
package authz

import (
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit/events"
)

const (
	EventDecisionName                = events.AuthzDecision
	DecisionAllow                    = "allow"
	DecisionDeny                     = "deny"
	DecisionOverride                 = "override"
	PlatformRoleHeader               = grpcmeta.PlatformRoleHeader
	OverrideReasonHeader             = grpcmeta.AuthzOverrideReasonHeader
	PlatformRoleAdmin                = grpcmeta.PlatformRoleAdmin
	ReasonAllowAdminOverride         = domainauthz.ReasonAllowAdminOverride
	ReasonAllowAccessLevel           = domainauthz.ReasonAllowAccessLevel
	ReasonAllowResourceOwner         = domainauthz.ReasonAllowResourceOwner
	ReasonDenyAccessLevelRequired    = domainauthz.ReasonDenyAccessLevelRequired
	ReasonDenyMissingIdentity        = domainauthz.ReasonDenyMissingIdentity
	ReasonDenyActorNotFound          = domainauthz.ReasonDenyActorNotFound
	ReasonDenyNotResourceOwner       = domainauthz.ReasonDenyNotResourceOwner
	ReasonDenyOverrideReasonRequired = domainauthz.ReasonDenyOverrideReasonRequired
	ReasonErrorDependencyUnavailable = domainauthz.ReasonErrorDependencyUnavailable
	ReasonErrorActorLoad             = domainauthz.ReasonErrorActorLoad
	ReasonErrorOwnerResolution       = domainauthz.ReasonErrorOwnerResolution
)
