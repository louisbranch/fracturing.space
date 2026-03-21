package service

import (
	"context"
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// InvokeAuthorizationResult captures the outcome of one invoke authorization
// check so callers do not juggle positional booleans.
type InvokeAuthorizationResult struct {
	Authorized      bool
	SharedAccess    bool
	AccessRequestID string
}

// AccessibleAgentResolver centralizes owner/shared invoke visibility so agent
// lookup and invocation share one access policy source.
type AccessibleAgentResolver struct {
	agentStore         storage.AgentStore
	accessRequestStore storage.AccessRequestStore
}

// NewAccessibleAgentResolver builds an accessible agent resolver from stores.
func NewAccessibleAgentResolver(agentStore storage.AgentStore, accessRequestStore storage.AccessRequestStore) *AccessibleAgentResolver {
	return &AccessibleAgentResolver{
		agentStore:         agentStore,
		accessRequestStore: accessRequestStore,
	}
}

// IsAuthorizedToInvokeAgent checks whether the caller is authorized to invoke
// the given agent, either as owner or through approved shared access.
func (r *AccessibleAgentResolver) IsAuthorizedToInvokeAgent(ctx context.Context, callerUserID string, agentRecord agent.Agent) (InvokeAuthorizationResult, error) {
	if agentRecord.OwnerUserID == "" {
		return InvokeAuthorizationResult{}, Errorf(ErrKindFailedPrecondition, "agent owner is unavailable")
	}
	if callerUserID == "" {
		return InvokeAuthorizationResult{}, nil
	}
	if callerUserID == agentRecord.OwnerUserID {
		return InvokeAuthorizationResult{Authorized: true}, nil
	}
	if r.accessRequestStore == nil {
		return InvokeAuthorizationResult{}, nil
	}
	rec, err := r.accessRequestStore.GetApprovedInvokeAccessByRequesterForAgent(
		ctx,
		callerUserID,
		agentRecord.OwnerUserID,
		agentRecord.ID,
	)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return InvokeAuthorizationResult{}, nil
		}
		return InvokeAuthorizationResult{}, Wrapf(ErrKindInternal, err, "get approved invoke access request")
	}
	return InvokeAuthorizationResult{
		Authorized:      true,
		SharedAccess:    true,
		AccessRequestID: rec.ID,
	}, nil
}

// ListAccessibleAgents returns a page of agents the user can invoke (owned +
// shared via approved invoke access) using a single paginated store query.
func (r *AccessibleAgentResolver) ListAccessibleAgents(ctx context.Context, userID string, pageSize int, pageToken string) (agent.Page, error) {
	page, err := r.agentStore.ListAccessibleAgents(ctx, userID, pageSize, pageToken)
	if err != nil {
		return agent.Page{}, Wrapf(ErrKindInternal, err, "list accessible agents")
	}
	return page, nil
}
