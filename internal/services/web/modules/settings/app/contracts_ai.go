package app

import "context"

// AIKeyGateway loads and mutates AI credential settings.
type AIKeyGateway interface {
	ListAIKeys(context.Context, string) ([]SettingsAIKey, error)
	CreateAIKey(context.Context, string, CreateAIKeyInput) error
	RevokeAIKey(context.Context, string, string) error
}

// AIAgentGateway loads and mutates AI agent settings.
type AIAgentGateway interface {
	ListAIAgentCredentials(context.Context, string) ([]SettingsAICredentialOption, error)
	ListAIAgents(context.Context, string) ([]SettingsAIAgent, error)
	ListAIProviderModels(context.Context, string, string) ([]SettingsAIModelOption, error)
	CreateAIAgent(context.Context, string, CreateAIAgentInput) error
	DeleteAIAgent(context.Context, string, string) error
}

// AIGateway groups AI-owned settings gateway behavior.
type AIGateway interface {
	AIKeyGateway
	AIAgentGateway
}

// AIKeyService exposes AI credential orchestration used by transport handlers.
type AIKeyService interface {
	ListAIKeys(context.Context, string) ([]SettingsAIKey, error)
	CreateAIKey(context.Context, string, CreateAIKeyInput) error
	RevokeAIKey(context.Context, string, string) error
}

// AIAgentService exposes AI agent orchestration used by transport handlers.
type AIAgentService interface {
	ListAIAgentCredentials(context.Context, string) ([]SettingsAICredentialOption, error)
	ListAIAgents(context.Context, string) ([]SettingsAIAgent, error)
	ListAIProviderModels(context.Context, string, string) ([]SettingsAIModelOption, error)
	CreateAIAgent(context.Context, string, CreateAIAgentInput) error
	DeleteAIAgent(context.Context, string, string) error
}

// AIService groups AI-owned settings orchestration.
type AIService interface {
	AIKeyService
	AIAgentService
}
