package ai

import (
	"context"
	"errors"
	"sort"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CreateAgent creates a user-owned AI agent profile.
func (h *AgentHandlers) CreateAgent(ctx context.Context, in *aiv1.CreateAgentRequest) (*aiv1.CreateAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create agent request is required")
	}
	if h.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	providerID, err := providerFromProto(in.GetProvider())
	if err != nil {
		return nil, err
	}

	authReference, err := agent.AuthReferenceFromIDs(in.GetCredentialId(), in.GetProviderGrantId(), true)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	createInput, err := agent.NormalizeCreateInput(agent.CreateInput{
		OwnerUserID:   userID,
		Label:         in.GetLabel(),
		Instructions:  in.GetInstructions(),
		Provider:      providerID,
		Model:         in.GetModel(),
		AuthReference: authReference,
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := h.validateAgentAuthReferenceForProvider(ctx, userID, providerID, createInput.AuthReference); err != nil {
		return nil, err
	}
	if err := h.validateProviderModelAvailable(ctx, userID, providerID, createInput.AuthReference, createInput.Model); err != nil {
		return nil, err
	}

	created, err := agent.Create(createInput, h.clock, h.idGenerator)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	record := storage.AgentRecord{
		ID:           created.ID,
		OwnerUserID:  created.OwnerUserID,
		Label:        created.Label,
		Instructions: created.Instructions,
		Provider:     string(created.Provider),
		Model:        created.Model,
		Status:       string(created.Status),
		CreatedAt:    created.CreatedAt,
		UpdatedAt:    created.UpdatedAt,
	}
	applyAgentAuthReference(&record, created.AuthReference)
	if err := h.agentStore.PutAgent(ctx, record); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			return nil, status.Error(codes.AlreadyExists, "agent label already exists")
		}
		return nil, status.Errorf(codes.Internal, "put agent: %v", err)
	}

	return &aiv1.CreateAgentResponse{Agent: h.agentProtoWithAuthState(ctx, record)}, nil
}

// ListAgents returns a page of agents owned by the caller.
func (h *AgentHandlers) ListAgents(ctx context.Context, in *aiv1.ListAgentsRequest) (*aiv1.ListAgentsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list agents request is required")
	}
	if h.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	page, err := h.agentStore.ListAgentsByOwner(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list agents: %v", err)
	}

	resp := &aiv1.ListAgentsResponse{
		NextPageToken: page.NextPageToken,
		Agents:        make([]*aiv1.Agent, 0, len(page.Agents)),
	}
	for _, rec := range page.Agents {
		proto, err := h.agentProtoWithUsage(ctx, rec)
		if err != nil {
			return nil, err
		}
		resp.Agents = append(resp.Agents, proto)
	}
	return resp, nil
}

// ListProviderModels returns provider-backed model options for one owned auth reference.
func (h *AgentHandlers) ListProviderModels(ctx context.Context, in *aiv1.ListProviderModelsRequest) (*aiv1.ListProviderModelsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list provider models request is required")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	providerID, err := providerFromProto(in.GetProvider())
	if err != nil {
		return nil, err
	}
	authReference, err := agent.AuthReferenceFromIDs(in.GetCredentialId(), in.GetProviderGrantId(), true)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	token, err := h.authTokenResolver.resolveAuthReferenceToken(
		ctx,
		userID,
		providerID,
		authReference,
	)
	if err != nil {
		return nil, err
	}

	modelProvider := providerID
	adapter, ok := h.providerModelAdapters[modelProvider]
	if !ok || adapter == nil {
		return nil, status.Error(codes.FailedPrecondition, "provider model adapter is unavailable")
	}
	models, err := adapter.ListModels(ctx, provider.ListModelsInput{CredentialSecret: token})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list provider models: %v", err)
	}
	sort.Slice(models, func(i int, j int) bool {
		if models[i].Created != models[j].Created {
			return models[i].Created > models[j].Created
		}
		return strings.Compare(strings.TrimSpace(models[i].ID), strings.TrimSpace(models[j].ID)) > 0
	})

	resp := &aiv1.ListProviderModelsResponse{Models: make([]*aiv1.ProviderModel, 0, len(models))}
	for _, model := range models {
		modelID := strings.TrimSpace(model.ID)
		if modelID == "" {
			continue
		}
		resp.Models = append(resp.Models, &aiv1.ProviderModel{
			Id:      modelID,
			OwnedBy: strings.TrimSpace(model.OwnedBy),
		})
	}
	return resp, nil
}

// ListAccessibleAgents returns a page of agents the caller can invoke, combining
// owned agents with approved shared invoke access.
func (h *AgentHandlers) ListAccessibleAgents(ctx context.Context, in *aiv1.ListAccessibleAgentsRequest) (*aiv1.ListAccessibleAgentsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list accessible agents request is required")
	}
	if h.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	records, err := newAccessibleAgentResolver(h.agentStore, h.accessRequestStore).collectAccessibleAgents(ctx, userID)
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
		resp.Agents = append(resp.Agents, h.agentProtoWithAuthState(ctx, rec))
	}
	return resp, nil
}

// GetAccessibleAgent returns one agent by ID when the caller can invoke it.
func (h *AgentHandlers) GetAccessibleAgent(ctx context.Context, in *aiv1.GetAccessibleAgentRequest) (*aiv1.GetAccessibleAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get accessible agent request is required")
	}
	if h.agentStore == nil {
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

	agentRecord, err := h.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "agent not found")
		}
		return nil, status.Errorf(codes.Internal, "get agent: %v", err)
	}

	// Authorization is intentionally shared with invoke checks so lookup and
	// runtime execution enforce one access policy source.
	authorized, _, _, err := newAccessibleAgentResolver(h.agentStore, h.accessRequestStore).isAuthorizedToInvokeAgent(ctx, userID, agentRecord)
	if err != nil {
		return nil, err
	}
	if !authorized {
		// Mask inaccessible resources as not found to avoid tenant probing.
		return nil, status.Error(codes.NotFound, "agent not found")
	}
	return &aiv1.GetAccessibleAgentResponse{Agent: h.agentProtoWithAuthState(ctx, agentRecord)}, nil
}

// ValidateCampaignAgentBinding verifies owner-scoped bind eligibility for one agent.
func (h *AgentHandlers) ValidateCampaignAgentBinding(ctx context.Context, in *aiv1.ValidateCampaignAgentBindingRequest) (*aiv1.ValidateCampaignAgentBindingResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "validate campaign agent binding request is required")
	}
	if h.agentStore == nil {
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

	agentRecord, err := h.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "agent not found")
		}
		return nil, status.Errorf(codes.Internal, "get agent: %v", err)
	}
	if strings.TrimSpace(agentRecord.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "agent not found")
	}
	if agentStatusToProto(agentRecord.Status) != aiv1.AgentStatus_AGENT_STATUS_ACTIVE {
		return nil, status.Error(codes.FailedPrecondition, "agent is not active")
	}
	authReference, err := agentAuthReferenceFromRecord(agentRecord)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "agent auth reference is invalid")
	}
	if err := h.validateAgentAuthReferenceForProvider(ctx, userID, providerFromString(agentRecord.Provider), authReference); err != nil {
		return nil, err
	}
	return &aiv1.ValidateCampaignAgentBindingResponse{Agent: h.agentProtoWithAuthState(ctx, agentRecord)}, nil
}

// UpdateAgent updates mutable fields on one user-owned agent.
func (h *AgentHandlers) UpdateAgent(ctx context.Context, in *aiv1.UpdateAgentRequest) (*aiv1.UpdateAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update agent request is required")
	}
	if h.agentStore == nil {
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

	existing, err := h.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "agent not found")
		}
		return nil, status.Errorf(codes.Internal, "get agent: %v", err)
	}
	if strings.TrimSpace(existing.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "agent not found")
	}
	if err := h.ensureAgentNotBoundToActiveCampaigns(ctx, existing.ID); err != nil {
		return nil, err
	}

	label := firstNonEmpty(strings.TrimSpace(in.GetLabel()), existing.Label)
	instructions := firstNonEmpty(strings.TrimSpace(in.GetInstructions()), existing.Instructions)
	model := firstNonEmpty(strings.TrimSpace(in.GetModel()), existing.Model)
	authReference, err := agentAuthReferenceFromRecord(existing)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "agent auth reference is invalid")
	}
	requestCredentialID := strings.TrimSpace(in.GetCredentialId())
	requestProviderGrantID := strings.TrimSpace(in.GetProviderGrantId())
	if requestCredentialID != "" || requestProviderGrantID != "" {
		authReference, err = agent.AuthReferenceFromIDs(requestCredentialID, requestProviderGrantID, true)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	normalized, err := agent.NormalizeUpdateInput(agent.UpdateInput{
		ID:            existing.ID,
		OwnerUserID:   existing.OwnerUserID,
		Label:         label,
		Instructions:  instructions,
		Model:         model,
		AuthReference: authReference,
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := h.validateAgentAuthReferenceForProvider(ctx, userID, providerFromString(existing.Provider), normalized.AuthReference); err != nil {
		return nil, err
	}
	existingAuthReference, err := agentAuthReferenceFromRecord(existing)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, "agent auth reference is invalid")
	}
	if normalized.Model != existing.Model ||
		normalized.AuthReference != existingAuthReference {
		if err := h.validateProviderModelAvailable(ctx, userID, providerFromString(existing.Provider), normalized.AuthReference, normalized.Model); err != nil {
			return nil, err
		}
	}

	record := existing
	record.Label = normalized.Label
	record.Instructions = normalized.Instructions
	record.Model = normalized.Model
	record.UpdatedAt = h.clock().UTC()
	applyAgentAuthReference(&record, normalized.AuthReference)
	if err := h.agentStore.PutAgent(ctx, record); err != nil {
		if errors.Is(err, storage.ErrConflict) {
			return nil, status.Error(codes.AlreadyExists, "agent label already exists")
		}
		return nil, status.Errorf(codes.Internal, "put agent: %v", err)
	}

	return &aiv1.UpdateAgentResponse{Agent: h.agentProtoWithAuthState(ctx, record)}, nil
}

// DeleteAgent deletes one user-owned agent profile.
func (h *AgentHandlers) DeleteAgent(ctx context.Context, in *aiv1.DeleteAgentRequest) (*aiv1.DeleteAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete agent request is required")
	}
	if h.agentStore == nil {
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
	if err := h.ensureAgentNotBoundToActiveCampaigns(ctx, agentID); err != nil {
		return nil, err
	}

	if err := h.agentStore.DeleteAgent(ctx, userID, agentID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "agent not found")
		}
		return nil, status.Errorf(codes.Internal, "delete agent: %v", err)
	}
	return &aiv1.DeleteAgentResponse{}, nil
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

func (h *AgentHandlers) ensureAgentNotBoundToActiveCampaigns(ctx context.Context, agentID string) error {
	return newAuthReferenceUsageGuard(h.agentStore, h.gameCampaignAIClient).ensureAgentNotBoundToActiveCampaigns(ctx, agentID)
}

func (h *AgentHandlers) validateAgentAuthReferenceForProvider(ctx context.Context, ownerUserID string, requestedProvider provider.Provider, authReference agent.AuthReference) error {
	switch authReference.Kind {
	case agent.AuthReferenceKindCredential:
		if h.credentialStore == nil {
			return status.Error(codes.Internal, "credential store is not configured")
		}
		credentialRecord, err := h.credentialStore.GetCredential(ctx, authReference.CredentialID())
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Error(codes.FailedPrecondition, "credential is unavailable")
			}
			return status.Errorf(codes.Internal, "get credential: %v", err)
		}
		if !credentialFromRecord(credentialRecord).IsUsableBy(ownerUserID, requestedProvider) {
			return status.Error(codes.FailedPrecondition, "credential must be active and owned by caller")
		}
		return nil
	case agent.AuthReferenceKindProviderGrant:
		if h.providerGrantStore == nil {
			return status.Error(codes.Internal, "provider grant store is not configured")
		}
		grantRecord, err := h.providerGrantStore.GetProviderGrant(ctx, authReference.ProviderGrantID())
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Error(codes.FailedPrecondition, "provider grant is unavailable")
			}
			return status.Errorf(codes.Internal, "get provider grant: %v", err)
		}
		if !providerGrantFromRecord(grantRecord).IsUsableBy(ownerUserID, requestedProvider) {
			return status.Error(codes.FailedPrecondition, "provider grant must be active and owned by caller")
		}
		return nil
	default:
		return status.Error(codes.InvalidArgument, "exactly one agent auth reference is required")
	}
}

func (h *AgentHandlers) validateProviderModelAvailable(ctx context.Context, ownerUserID string, requestedProvider provider.Provider, authReference agent.AuthReference, model string) error {
	model = strings.TrimSpace(model)
	if model == "" {
		return status.Error(codes.InvalidArgument, "model is required")
	}

	token, err := h.authTokenResolver.resolveAuthReferenceToken(ctx, ownerUserID, requestedProvider, authReference)
	if err != nil {
		return err
	}
	adapter, ok := h.providerModelAdapters[requestedProvider]
	if !ok || adapter == nil {
		return status.Error(codes.FailedPrecondition, "provider model adapter is unavailable")
	}
	models, err := adapter.ListModels(ctx, provider.ListModelsInput{CredentialSecret: token})
	if err != nil {
		return status.Errorf(codes.Internal, "list provider models: %v", err)
	}
	for _, candidate := range models {
		if strings.TrimSpace(candidate.ID) == model {
			return nil
		}
	}
	return status.Error(codes.FailedPrecondition, "model is unavailable for the selected auth reference")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
