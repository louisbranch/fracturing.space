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

// AccountGateway groups account-owned settings gateway behavior.
type AccountGateway interface {
	ProfileGateway
	LocaleGateway
	SecurityGateway
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

// AccountService groups account-owned settings orchestration.
type AccountService interface {
	ProfileService
	LocaleService
	SecurityService
}
