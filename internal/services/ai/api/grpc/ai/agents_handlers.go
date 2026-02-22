package ai

import (
	"context"
	"errors"
	"sort"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateAgent creates a user-owned AI agent profile.
func (s *Service) CreateAgent(ctx context.Context, in *aiv1.CreateAgentRequest) (*aiv1.CreateAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create agent request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	provider, err := agentProviderFromProto(in.GetProvider())
	if err != nil {
		return nil, err
	}

	createInput, err := agent.NormalizeCreateInput(agent.CreateInput{
		OwnerUserID:     userID,
		Name:            in.GetName(),
		Provider:        provider,
		Model:           in.GetModel(),
		CredentialID:    in.GetCredentialId(),
		ProviderGrantID: in.GetProviderGrantId(),
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.validateAgentAuthReferenceForProvider(ctx, userID, string(provider), createInput.CredentialID, createInput.ProviderGrantID); err != nil {
		return nil, err
	}

	created, err := agent.Create(createInput, s.clock, s.idGenerator)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	record := storage.AgentRecord{
		ID:              created.ID,
		OwnerUserID:     created.OwnerUserID,
		Name:            created.Name,
		Provider:        string(created.Provider),
		Model:           created.Model,
		CredentialID:    created.CredentialID,
		ProviderGrantID: created.ProviderGrantID,
		Status:          string(created.Status),
		CreatedAt:       created.CreatedAt,
		UpdatedAt:       created.UpdatedAt,
	}
	if err := s.agentStore.PutAgent(ctx, record); err != nil {
		return nil, status.Errorf(codes.Internal, "put agent: %v", err)
	}

	return &aiv1.CreateAgentResponse{Agent: agentToProto(record)}, nil
}

// ListAgents returns a page of agents owned by the caller.
func (s *Service) ListAgents(ctx context.Context, in *aiv1.ListAgentsRequest) (*aiv1.ListAgentsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list agents request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	page, err := s.agentStore.ListAgentsByOwner(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list agents: %v", err)
	}

	resp := &aiv1.ListAgentsResponse{
		NextPageToken: page.NextPageToken,
		Agents:        make([]*aiv1.Agent, 0, len(page.Agents)),
	}
	for _, rec := range page.Agents {
		resp.Agents = append(resp.Agents, agentToProto(rec))
	}
	return resp, nil
}

// ListAccessibleAgents returns a page of agents the caller can invoke, combining
// owned agents with approved shared invoke access.
func (s *Service) ListAccessibleAgents(ctx context.Context, in *aiv1.ListAccessibleAgentsRequest) (*aiv1.ListAccessibleAgentsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list accessible agents request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	records, err := s.collectAccessibleAgents(ctx, userID)
	if err != nil {
		return nil, err
	}
	sort.Slice(records, func(i int, j int) bool {
		return records[i].ID < records[j].ID
	})

	pageSize := clampPageSize(in.GetPageSize())
	pageToken := strings.TrimSpace(in.GetPageToken())
	start := findPageStartByID(records, pageToken)
	end := start + pageSize
	nextPageToken := ""
	if end < len(records) {
		nextPageToken = records[end-1].ID
	} else {
		end = len(records)
	}

	resp := &aiv1.ListAccessibleAgentsResponse{
		NextPageToken: nextPageToken,
		Agents:        make([]*aiv1.Agent, 0, end-start),
	}
	for _, rec := range records[start:end] {
		resp.Agents = append(resp.Agents, agentToProto(rec))
	}
	return resp, nil
}

// GetAccessibleAgent returns one agent by ID when the caller can invoke it.
func (s *Service) GetAccessibleAgent(ctx context.Context, in *aiv1.GetAccessibleAgentRequest) (*aiv1.GetAccessibleAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get accessible agent request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	agentID := strings.TrimSpace(in.GetAgentId())
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}

	agentRecord, err := s.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "agent not found")
		}
		return nil, status.Errorf(codes.Internal, "get agent: %v", err)
	}

	// Authorization is intentionally shared with invoke checks so lookup and
	// runtime execution enforce one access policy source.
	authorized, _, _, err := s.isAuthorizedToInvokeAgent(ctx, userID, agentRecord)
	if err != nil {
		return nil, err
	}
	if !authorized {
		// Mask inaccessible resources as not found to avoid tenant probing.
		return nil, status.Error(codes.NotFound, "agent not found")
	}
	return &aiv1.GetAccessibleAgentResponse{Agent: agentToProto(agentRecord)}, nil
}

// UpdateAgent updates mutable fields on one user-owned agent.
func (s *Service) UpdateAgent(ctx context.Context, in *aiv1.UpdateAgentRequest) (*aiv1.UpdateAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update agent request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	agentID := strings.TrimSpace(in.GetAgentId())
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}

	existing, err := s.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "agent not found")
		}
		return nil, status.Errorf(codes.Internal, "get agent: %v", err)
	}
	if strings.TrimSpace(existing.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "agent not found")
	}

	name := firstNonEmpty(strings.TrimSpace(in.GetName()), existing.Name)
	model := firstNonEmpty(strings.TrimSpace(in.GetModel()), existing.Model)
	credentialID := strings.TrimSpace(existing.CredentialID)
	providerGrantID := strings.TrimSpace(existing.ProviderGrantID)
	requestCredentialID := strings.TrimSpace(in.GetCredentialId())
	requestProviderGrantID := strings.TrimSpace(in.GetProviderGrantId())
	if requestCredentialID != "" || requestProviderGrantID != "" {
		credentialID = requestCredentialID
		providerGrantID = requestProviderGrantID
	}
	normalized, err := agent.NormalizeUpdateInput(agent.UpdateInput{
		ID:              existing.ID,
		OwnerUserID:     existing.OwnerUserID,
		Name:            name,
		Model:           model,
		CredentialID:    credentialID,
		ProviderGrantID: providerGrantID,
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.validateAgentAuthReferenceForProvider(ctx, userID, existing.Provider, normalized.CredentialID, normalized.ProviderGrantID); err != nil {
		return nil, err
	}

	record := existing
	record.Name = normalized.Name
	record.Model = normalized.Model
	record.CredentialID = normalized.CredentialID
	record.ProviderGrantID = normalized.ProviderGrantID
	record.UpdatedAt = s.clock().UTC()
	if err := s.agentStore.PutAgent(ctx, record); err != nil {
		return nil, status.Errorf(codes.Internal, "put agent: %v", err)
	}

	return &aiv1.UpdateAgentResponse{Agent: agentToProto(record)}, nil
}

// DeleteAgent deletes one user-owned agent profile.
func (s *Service) DeleteAgent(ctx context.Context, in *aiv1.DeleteAgentRequest) (*aiv1.DeleteAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete agent request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	agentID := strings.TrimSpace(in.GetAgentId())
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}

	if err := s.agentStore.DeleteAgent(ctx, userID, agentID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "agent not found")
		}
		return nil, status.Errorf(codes.Internal, "delete agent: %v", err)
	}
	return &aiv1.DeleteAgentResponse{}, nil
}
func (s *Service) isAuthorizedToInvokeAgent(ctx context.Context, callerUserID string, agentRecord storage.AgentRecord) (bool, bool, string, error) {
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
	if s.accessRequestStore == nil {
		return false, false, "", nil
	}
	rec, err := s.accessRequestStore.GetApprovedInvokeAccessByRequesterForAgent(
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
func (s *Service) collectAccessibleAgents(ctx context.Context, userID string) ([]storage.AgentRecord, error) {
	accessibleByID := make(map[string]storage.AgentRecord)
	pageToken := ""
	for {
		page, err := s.agentStore.ListAgentsByOwner(ctx, userID, maxPageSize, pageToken)
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

	if s.accessRequestStore == nil {
		return mapValues(accessibleByID), nil
	}

	pageToken = ""
	for {
		page, err := s.accessRequestStore.ListApprovedInvokeAccessRequestsByRequester(ctx, userID, maxPageSize, pageToken)
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
			agentRecord, err := s.agentStore.GetAgent(ctx, agentID)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					// Access records can outlive target agents. Ignore stale entries.
					continue
				}
				return nil, status.Errorf(codes.Internal, "get shared agent: %v", err)
			}
			// Require owner match to avoid stale or tampered access rows granting a
			// different owner's agent.
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

func findPageStartByID(records []storage.AgentRecord, pageToken string) int {
	pageToken = strings.TrimSpace(pageToken)
	if pageToken == "" {
		return 0
	}
	for idx, rec := range records {
		if strings.Compare(strings.TrimSpace(rec.ID), pageToken) > 0 {
			return idx
		}
	}
	return len(records)
}

func mapValues(values map[string]storage.AgentRecord) []storage.AgentRecord {
	if len(values) == 0 {
		return []storage.AgentRecord{}
	}
	items := make([]storage.AgentRecord, 0, len(values))
	for _, rec := range values {
		items = append(items, rec)
	}
	return items
}
func (s *Service) validateAgentAuthReferenceForProvider(ctx context.Context, ownerUserID string, provider string, credentialID string, providerGrantID string) error {
	credentialID = strings.TrimSpace(credentialID)
	providerGrantID = strings.TrimSpace(providerGrantID)
	hasCredential := credentialID != ""
	hasProviderGrant := providerGrantID != ""
	if hasCredential == hasProviderGrant {
		return status.Error(codes.InvalidArgument, "exactly one agent auth reference is required")
	}

	if hasCredential {
		if s.credentialStore == nil {
			return status.Error(codes.Internal, "credential store is not configured")
		}
		credentialRecord, err := s.credentialStore.GetCredential(ctx, credentialID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Error(codes.FailedPrecondition, "credential is unavailable")
			}
			return status.Errorf(codes.Internal, "get credential: %v", err)
		}
		if !isCredentialActiveForUser(credentialRecord, ownerUserID, provider) {
			return status.Error(codes.FailedPrecondition, "credential must be active and owned by caller")
		}
		return nil
	}

	if s.providerGrantStore == nil {
		return status.Error(codes.Internal, "provider grant store is not configured")
	}
	grantRecord, err := s.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return status.Error(codes.FailedPrecondition, "provider grant is unavailable")
		}
		return status.Errorf(codes.Internal, "get provider grant: %v", err)
	}
	if !isProviderGrantActiveForUser(grantRecord, ownerUserID, provider) {
		return status.Error(codes.FailedPrecondition, "provider grant must be active and owned by caller")
	}
	return nil
}
func isCredentialActiveForUser(record storage.CredentialRecord, ownerUserID string, provider string) bool {
	if strings.TrimSpace(record.OwnerUserID) != strings.TrimSpace(ownerUserID) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(record.Status), "active") {
		return false
	}
	if provider != "" && !strings.EqualFold(strings.TrimSpace(record.Provider), strings.TrimSpace(provider)) {
		return false
	}
	return true
}

func isProviderGrantActiveForUser(record storage.ProviderGrantRecord, ownerUserID string, provider string) bool {
	if strings.TrimSpace(record.OwnerUserID) != strings.TrimSpace(ownerUserID) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(record.Status), "active") {
		return false
	}
	if provider != "" && !strings.EqualFold(strings.TrimSpace(record.Provider), strings.TrimSpace(provider)) {
		return false
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
