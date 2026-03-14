package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// ListAIKeys returns the package view collection for this workflow.
func (s service) ListAIKeys(ctx context.Context, userID string) ([]SettingsAIKey, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return nil, err
	}
	keys, err := s.gateway.ListAIKeys(ctx, resolvedUserID)
	if err != nil {
		return nil, err
	}
	if keys == nil {
		return []SettingsAIKey{}, nil
	}

	normalized := make([]SettingsAIKey, 0, len(keys))
	for _, key := range keys {
		normalized = append(normalized, normalizeSettingsAIKey(key))
	}

	return normalized, nil
}

// ListAIAgentCredentials returns active credential options for agent creation.
func (s service) ListAIAgentCredentials(ctx context.Context, userID string) ([]SettingsAICredentialOption, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return nil, err
	}
	options, err := s.gateway.ListAIAgentCredentials(ctx, resolvedUserID)
	if err != nil {
		return nil, err
	}
	if options == nil {
		return []SettingsAICredentialOption{}, nil
	}

	normalized := make([]SettingsAICredentialOption, 0, len(options))
	for _, option := range options {
		normalized = append(normalized, normalizeSettingsAICredentialOption(option))
	}
	return normalized, nil
}

// ListAIAgents returns existing agent rows for the settings page.
func (s service) ListAIAgents(ctx context.Context, userID string) ([]SettingsAIAgent, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return nil, err
	}
	agents, err := s.gateway.ListAIAgents(ctx, resolvedUserID)
	if err != nil {
		return nil, err
	}
	if agents == nil {
		return []SettingsAIAgent{}, nil
	}

	normalized := make([]SettingsAIAgent, 0, len(agents))
	for _, agent := range agents {
		normalized = append(normalized, normalizeSettingsAIAgent(agent))
	}
	return normalized, nil
}

// ListAIProviderModels returns provider-backed model options for one credential.
func (s service) ListAIProviderModels(ctx context.Context, userID string, credentialID string) ([]SettingsAIModelOption, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return nil, err
	}
	resolvedCredentialID := strings.TrimSpace(credentialID)
	if resolvedCredentialID == "" || !IsSafePathID(resolvedCredentialID) {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "web.settings.ai_agents.error_credential_required", "credential is required")
	}
	models, err := s.gateway.ListAIProviderModels(ctx, resolvedUserID, resolvedCredentialID)
	if err != nil {
		return nil, err
	}
	if models == nil {
		return []SettingsAIModelOption{}, nil
	}

	normalized := make([]SettingsAIModelOption, 0, len(models))
	for _, model := range models {
		normalized = append(normalized, normalizeSettingsAIModelOption(model))
	}
	return normalized, nil
}

// CreateAIKey executes package-scoped creation behavior for this flow.
func (s service) CreateAIKey(ctx context.Context, userID string, label string, secret string) error {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return err
	}
	label = strings.TrimSpace(label)
	secret = strings.TrimSpace(secret)
	if label == "" || secret == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "web.settings.ai_keys.error_required", "label and secret are required")
	}
	return s.gateway.CreateAIKey(ctx, resolvedUserID, label, secret)
}

// CreateAIAgent executes package-scoped agent creation behavior.
func (s service) CreateAIAgent(ctx context.Context, userID string, input CreateAIAgentInput) error {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return err
	}
	input.Label = strings.TrimSpace(input.Label)
	input.CredentialID = strings.TrimSpace(input.CredentialID)
	input.Model = strings.TrimSpace(input.Model)
	input.Instructions = strings.TrimSpace(input.Instructions)
	if input.Label == "" || input.CredentialID == "" || input.Model == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "web.settings.ai_agents.error_required", "label, credential, and model are required")
	}
	if !aiAgentLabelPattern.MatchString(input.Label) {
		return apperrors.EK(apperrors.KindInvalidInput, "web.settings.ai_agents.error_label_invalid", "agent label is invalid")
	}
	if !IsSafePathID(input.CredentialID) {
		return apperrors.EK(apperrors.KindInvalidInput, "web.settings.ai_agents.error_credential_required", "credential is required")
	}
	return s.gateway.CreateAIAgent(ctx, resolvedUserID, input)
}

// DeleteAIAgent removes one user-owned AI agent when it is not in use.
func (s service) DeleteAIAgent(ctx context.Context, userID string, agentID string) error {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return err
	}
	resolvedAgentID := strings.TrimSpace(agentID)
	if resolvedAgentID == "" || !IsSafePathID(resolvedAgentID) {
		return apperrors.EK(apperrors.KindInvalidInput, "web.settings.ai_agents.error_agent_required", "agent is required")
	}
	return s.gateway.DeleteAIAgent(ctx, resolvedUserID, resolvedAgentID)
}

// RevokeAIKey applies this package workflow transition.
func (s service) RevokeAIKey(ctx context.Context, userID string, credentialID string) error {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return err
	}
	resolvedCredentialID := strings.TrimSpace(credentialID)
	if resolvedCredentialID == "" {
		return apperrors.EK(
			apperrors.KindInvalidInput,
			"error.web.message.ai_key_id_is_required",
			"credential id is required",
		)
	}
	return s.gateway.RevokeAIKey(ctx, resolvedUserID, resolvedCredentialID)
}

// normalizeSettingsAIKey normalizes one credential row for stable template rendering.
func normalizeSettingsAIKey(key SettingsAIKey) SettingsAIKey {
	key.ID = strings.TrimSpace(key.ID)
	key.Label = strings.TrimSpace(key.Label)
	key.Provider = strings.TrimSpace(key.Provider)
	key.Status = strings.TrimSpace(key.Status)
	key.CreatedAt = strings.TrimSpace(key.CreatedAt)
	key.RevokedAt = strings.TrimSpace(key.RevokedAt)

	if key.Provider == "" {
		key.Provider = "Unknown"
	}
	if key.Status == "" {
		key.Status = "Unspecified"
	}
	if key.CreatedAt == "" {
		key.CreatedAt = "-"
	}
	if key.RevokedAt == "" {
		key.RevokedAt = "-"
	}
	if !IsSafePathID(key.ID) {
		key.ID = ""
		key.CanRevoke = false
	}
	return key
}

// normalizeSettingsAICredentialOption ensures credential options render predictably.
func normalizeSettingsAICredentialOption(option SettingsAICredentialOption) SettingsAICredentialOption {
	option.ID = strings.TrimSpace(option.ID)
	option.Label = strings.TrimSpace(option.Label)
	option.Provider = strings.TrimSpace(option.Provider)
	if option.Provider == "" {
		option.Provider = "Unknown"
	}
	if option.Label == "" {
		option.Label = option.Provider
	}
	return option
}

// normalizeSettingsAIAgent normalizes one AI agent row for stable template rendering.
func normalizeSettingsAIAgent(agent SettingsAIAgent) SettingsAIAgent {
	agent.ID = strings.TrimSpace(agent.ID)
	agent.Label = strings.TrimSpace(agent.Label)
	agent.Provider = strings.TrimSpace(agent.Provider)
	agent.Model = strings.TrimSpace(agent.Model)
	agent.AuthState = strings.TrimSpace(agent.AuthState)
	agent.CreatedAt = strings.TrimSpace(agent.CreatedAt)
	agent.Instructions = strings.TrimSpace(agent.Instructions)

	if agent.Provider == "" {
		agent.Provider = "Unknown"
	}
	if agent.AuthState == "" {
		agent.AuthState = "Unknown"
	}
	if agent.CreatedAt == "" {
		agent.CreatedAt = "-"
	}
	agent.CanDelete = agent.CanDelete && agent.ID != ""
	return agent
}

// normalizeSettingsAIModelOption ensures model options render predictably.
func normalizeSettingsAIModelOption(option SettingsAIModelOption) SettingsAIModelOption {
	option.ID = strings.TrimSpace(option.ID)
	option.OwnedBy = strings.TrimSpace(option.OwnedBy)
	return option
}
