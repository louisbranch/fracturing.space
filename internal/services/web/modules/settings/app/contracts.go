package app

import (
	"context"
	"encoding/json"
)

// ProfileGateway loads and updates editable profile fields.
type ProfileGateway interface {
	LoadProfile(context.Context, string) (SettingsProfile, error)
	SaveProfile(context.Context, string, SettingsProfile) error
}

// LocaleGateway loads and updates account locale settings.
type LocaleGateway interface {
	LoadLocale(context.Context, string) (string, error)
	SaveLocale(context.Context, string, string) error
}

// SecurityGateway loads and mutates authenticated passkey settings.
type SecurityGateway interface {
	ListPasskeys(context.Context, string) ([]SettingsPasskey, error)
	BeginPasskeyRegistration(context.Context, string) (PasskeyChallenge, error)
	FinishPasskeyRegistration(context.Context, string, json.RawMessage) error
}

// AIKeyGateway loads and mutates AI credential settings.
type AIKeyGateway interface {
	ListAIKeys(context.Context, string) ([]SettingsAIKey, error)
	CreateAIKey(context.Context, string, string, string) error
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

// AccountGateway groups account-owned settings gateway behavior.
type AccountGateway interface {
	ProfileGateway
	LocaleGateway
	SecurityGateway
}

// AIGateway groups AI-owned settings gateway behavior.
type AIGateway interface {
	AIKeyGateway
	AIAgentGateway
}

// Gateway loads and updates settings data for web handlers.
type Gateway interface {
	AccountGateway
	AIGateway
}

// ProfileService exposes profile orchestration used by transport handlers.
type ProfileService interface {
	LoadProfile(context.Context, string) (SettingsProfile, error)
	SaveProfile(context.Context, string, SettingsProfile) error
}

// LocaleService exposes locale orchestration used by transport handlers.
type LocaleService interface {
	LoadLocale(context.Context, string) (string, error)
	SaveLocale(context.Context, string, string) error
}

// SecurityService exposes passkey security orchestration used by transport handlers.
type SecurityService interface {
	ListPasskeys(context.Context, string) ([]SettingsPasskey, error)
	BeginPasskeyRegistration(context.Context, string) (PasskeyChallenge, error)
	FinishPasskeyRegistration(context.Context, string, json.RawMessage) error
}

// AIKeyService exposes AI credential orchestration used by transport handlers.
type AIKeyService interface {
	ListAIKeys(context.Context, string) ([]SettingsAIKey, error)
	CreateAIKey(context.Context, string, string, string) error
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

// AccountService groups account-owned settings orchestration.
type AccountService interface {
	ProfileService
	LocaleService
	SecurityService
}

// AIService groups AI-owned settings orchestration.
type AIService interface {
	AIKeyService
	AIAgentService
}
