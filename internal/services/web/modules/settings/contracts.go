package settings

import (
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	settingsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/gateway"
)

// SettingsProfile is the transport-facing alias for settings app profile data.
type SettingsProfile = settingsapp.SettingsProfile

// SettingsAIKey is the transport-facing alias for settings app AI key rows.
type SettingsAIKey = settingsapp.SettingsAIKey

// SettingsPasskey is the transport-facing alias for settings passkey rows.
type SettingsPasskey = settingsapp.SettingsPasskey

// SettingsAICredentialOption is the transport-facing alias for agent credential options.
type SettingsAICredentialOption = settingsapp.SettingsAICredentialOption

// SettingsAIModelOption is the transport-facing alias for provider-backed model options.
type SettingsAIModelOption = settingsapp.SettingsAIModelOption

// SettingsAIAgent is the transport-facing alias for settings AI agent rows.
type SettingsAIAgent = settingsapp.SettingsAIAgent

// CreateAIAgentInput is the transport-facing alias for agent creation input.
type CreateAIAgentInput = settingsapp.CreateAIAgentInput

// SettingsGateway is the transport-facing alias for settings app gateway contract.
type SettingsGateway = settingsapp.Gateway

const userProfileNameMaxLength = settingsapp.UserProfileNameMaxLength

// Settings gRPC dependency aliases keep root-module constructors/test seams stable.
type SocialClient = settingsgateway.SocialClient

// AccountClient defines an internal contract used at this web package boundary.
type AccountClient = settingsgateway.AccountClient

// PasskeyClient defines an internal contract used at this web package boundary.
type PasskeyClient = settingsgateway.PasskeyClient

// CredentialClient defines an internal contract used at this web package boundary.
type CredentialClient = settingsgateway.CredentialClient

// AgentClient defines an internal contract used at this web package boundary.
type AgentClient = settingsgateway.AgentClient
