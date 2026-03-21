package ai

import (
	"context"
	"errors"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// accessibleAgentResolver centralizes owner/shared invoke visibility so agent
// lookup and invocation share one access policy source.
type accessibleAgentResolver struct {
	agentStore         storage.AgentStore
	accessRequestStore storage.AccessRequestStore
}

func newAccessibleAgentResolver(agentStore storage.AgentStore, accessRequestStore storage.AccessRequestStore) accessibleAgentResolver {
	return accessibleAgentResolver{
		agentStore:         agentStore,
		accessRequestStore: accessRequestStore,
	}
}

func (r accessibleAgentResolver) isAuthorizedToInvokeAgent(ctx context.Context, callerUserID string, agentRecord storage.AgentRecord) (bool, bool, string, error) {
	ownerUserID := strings.TrimSpace(agentRecord.OwnerUserID)
	if ownerUserID == "" {
		return false, false, "", status.Error(codes.FailedPrecondition, "agent owner is unavailable")
	}
	callerUserID = strings.TrimSpace(callerUserID)
	if callerUserID == "" {
		return false, false, "", nil
	}
	if callerUserID == ownerUserID {
		return true, false, "", nil
	}
	if r.accessRequestStore == nil {
		return false, false, "", nil
	}
	rec, err := r.accessRequestStore.GetApprovedInvokeAccessByRequesterForAgent(
		ctx,
		callerUserID,
		ownerUserID,
		strings.TrimSpace(agentRecord.ID),
	)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return false, false, "", nil
		}
		return false, false, "", status.Errorf(codes.Internal, "get approved invoke access request: %v", err)
	}
	return true, true, strings.TrimSpace(rec.ID), nil
}

func (r accessibleAgentResolver) collectAccessibleAgents(ctx context.Context, userID string) ([]storage.AgentRecord, error) {
	accessibleByID := make(map[string]storage.AgentRecord)
	pageToken := ""
	for {
		page, err := r.agentStore.ListAgentsByOwner(ctx, userID, maxPageSize, pageToken)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list agents: %v", err)
		}
		for _, rec := range page.Agents {
			if strings.TrimSpace(rec.ID) == "" {
				continue
			}
			accessibleByID[rec.ID] = rec
		}

		nextPageToken := strings.TrimSpace(page.NextPageToken)
		if nextPageToken == "" || nextPageToken == pageToken {
			break
		}
		pageToken = nextPageToken
	}

	if r.accessRequestStore == nil {
		return mapValues(accessibleByID), nil
	}

	pageToken = ""
	for {
		page, err := r.accessRequestStore.ListApprovedInvokeAccessRequestsByRequester(ctx, userID, maxPageSize, pageToken)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list approved invoke access requests: %v", err)
		}
		for _, rec := range page.AccessRequests {
			agentID := strings.TrimSpace(rec.AgentID)
			if agentID == "" {
				continue
			}
			if _, exists := accessibleByID[agentID]; exists {
				continue
			}
			agentRecord, err := r.agentStore.GetAgent(ctx, agentID)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					continue
				}
				return nil, status.Errorf(codes.Internal, "get shared agent: %v", err)
			}
			if strings.TrimSpace(agentRecord.OwnerUserID) != strings.TrimSpace(rec.OwnerUserID) {
				continue
			}
			accessibleByID[agentID] = agentRecord
		}

		nextPageToken := strings.TrimSpace(page.NextPageToken)
		if nextPageToken == "" || nextPageToken == pageToken {
			break
		}
		pageToken = nextPageToken
	}
	return mapValues(accessibleByID), nil
}
