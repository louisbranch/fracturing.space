package settings

import (
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	settingsgateway "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/gateway"
)

// SettingsProfile is the transport-facing alias for settings app profile data.
type SettingsProfile = settingsapp.SettingsProfile

// SettingsAIKey is the transport-facing alias for settings app AI key rows.
type SettingsAIKey = settingsapp.SettingsAIKey

// SettingsGateway is the transport-facing alias for settings app gateway contract.
type SettingsGateway = settingsapp.Gateway

const userProfileNameMaxLength = settingsapp.UserProfileNameMaxLength

// Settings gRPC dependency aliases keep root-module constructors/test seams stable.
type SocialClient = settingsgateway.SocialClient
type AccountClient = settingsgateway.AccountClient
type CredentialClient = settingsgateway.CredentialClient
