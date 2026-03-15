package gateway

import campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"

// AuthorizationDeps keeps authorization dependencies explicit.
type AuthorizationDeps struct {
	Client AuthorizationClient
}

// authorizationGateway maps unary authorization checks for mutation guards.
type authorizationGateway struct {
	authorization AuthorizationDeps
}

// batchAuthorizationGateway maps row-hydration authorization checks separately from unary guards.
type batchAuthorizationGateway struct {
	authorization AuthorizationDeps
}

// NewAuthorizationGateway builds the unary authorization adapter from explicit
// dependencies.
func NewAuthorizationGateway(deps AuthorizationDeps) campaignapp.AuthorizationGateway {
	if deps.Client == nil {
		return nil
	}
	return authorizationGateway{authorization: deps}
}

// NewBatchAuthorizationGateway builds the batch authorization adapter from
// explicit dependencies.
func NewBatchAuthorizationGateway(deps AuthorizationDeps) campaignapp.BatchAuthorizationGateway {
	if deps.Client == nil {
		return nil
	}
	return batchAuthorizationGateway{authorization: deps}
}
