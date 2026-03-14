package app

import (
	"context"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// ListAIKeys returns the package view collection for this workflow.
func (unavailableGateway) ListAIKeys(context.Context, string) ([]SettingsAIKey, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

// ListAIAgentCredentials returns active credential options for agent creation.
func (unavailableGateway) ListAIAgentCredentials(context.Context, string) ([]SettingsAICredentialOption, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

// ListAIAgents returns settings agent rows for the AI agents page.
func (unavailableGateway) ListAIAgents(context.Context, string) ([]SettingsAIAgent, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

// ListAIProviderModels returns provider-backed model options for one credential.
func (unavailableGateway) ListAIProviderModels(context.Context, string, string) ([]SettingsAIModelOption, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

// CreateAIKey executes package-scoped creation behavior for this flow.
func (unavailableGateway) CreateAIKey(context.Context, string, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

// CreateAIAgent executes package-scoped agent creation behavior.
func (unavailableGateway) CreateAIAgent(context.Context, string, CreateAIAgentInput) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}

// RevokeAIKey applies this package workflow transition.
func (unavailableGateway) RevokeAIKey(context.Context, string, string) error {
	return apperrors.E(apperrors.KindUnavailable, "settings service is not configured")
}
