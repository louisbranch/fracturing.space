package game

import (
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit/events"
)

const (
	authzEventDecisionName                = events.AuthzDecision
	authzDecisionAllow                    = "allow"
	authzDecisionDeny                     = "deny"
	authzDecisionOverride                 = "override"
	authzPlatformRoleHeader               = grpcmeta.PlatformRoleHeader
	authzOverrideReasonHeader             = grpcmeta.AuthzOverrideReasonHeader
	authzPlatformRoleAdmin                = grpcmeta.PlatformRoleAdmin
	authzReasonAllowAdminOverride         = domainauthz.ReasonAllowAdminOverride
	authzReasonAllowAccessLevel           = domainauthz.ReasonAllowAccessLevel
	authzReasonAllowResourceOwner         = domainauthz.ReasonAllowResourceOwner
	authzReasonDenyAccessLevelRequired    = domainauthz.ReasonDenyAccessLevelRequired
	authzReasonDenyMissingIdentity        = domainauthz.ReasonDenyMissingIdentity
	authzReasonDenyActorNotFound          = domainauthz.ReasonDenyActorNotFound
	authzReasonDenyNotResourceOwner       = domainauthz.ReasonDenyNotResourceOwner
	authzReasonDenyOverrideReasonRequired = domainauthz.ReasonDenyOverrideReasonRequired
	authzReasonErrorDependencyUnavailable = domainauthz.ReasonErrorDependencyUnavailable
	authzReasonErrorActorLoad             = domainauthz.ReasonErrorActorLoad
	authzReasonErrorOwnerResolution       = domainauthz.ReasonErrorOwnerResolution
)
