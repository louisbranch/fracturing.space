package app

import (
	"context"
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/userid"
	"golang.org/x/text/language"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// UserProfileNameMaxLength is the maximum allowed rune length for profile names.
const UserProfileNameMaxLength = 64

// SettingsProfile stores editable user profile settings.
type SettingsProfile struct {
	Username      string
	Name          string
	AvatarSetID   string
	AvatarAssetID string
	Pronouns      string
	Bio           string
}

// SettingsAIKey stores a credential row displayed in the AI keys page.
type SettingsAIKey struct {
	ID        string
	Label     string
	Provider  string
	Status    string
	CreatedAt string
	RevokedAt string
	CanRevoke bool
}

// SettingsPasskey stores one passkey summary row rendered on the security page.
type SettingsPasskey struct {
	Number     int
	CreatedAt  string
	LastUsedAt string
}

// SettingsAICredentialOption stores an active credential option for agent creation.
type SettingsAICredentialOption struct {
	ID       string
	Label    string
	Provider string
}

// SettingsAIModelOption stores one provider-backed model option for agent creation.
type SettingsAIModelOption struct {
	ID      string
	OwnedBy string
}

// SettingsAIAgent stores an agent row displayed in the AI agents page.
type SettingsAIAgent struct {
	ID                  string
	Label               string
	Provider            string
	Model               string
	AuthState           string
	CanDelete           bool
	ActiveCampaignCount int32
	CreatedAt           string
	Instructions        string
}

// CreateAIAgentInput stores validated agent creation input.
type CreateAIAgentInput struct {
	Label        string
	CredentialID string
	Model        string
	Instructions string
}

var aiAgentLabelPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{2,31}$`)

// settingsLocale defines an internal contract used at this web package boundary.
type settingsLocale string

const (
	settingsLocaleUnspecified settingsLocale = ""
	settingsLocaleEnUS        settingsLocale = "en-US"
	settingsLocalePtBR        settingsLocale = "pt-BR"
)

var settingsLocaleByTag = map[string]settingsLocale{
	"en":    settingsLocaleEnUS,
	"en-us": settingsLocaleEnUS,
	"pt":    settingsLocalePtBR,
	"pt-br": settingsLocalePtBR,
}

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

// Service exposes settings orchestration methods used by transport handlers.
type Service interface {
	AccountService
	AIService
}

// PasskeyChallenge stores authenticated passkey enrollment begin state.
type PasskeyChallenge struct {
	SessionID string
	PublicKey json.RawMessage
}

// RequireUserID validates and returns a trimmed user ID, or returns an
// unauthorized error if it is blank.
func RequireUserID(userID string) (string, error) {
	return userid.Require(userID)
}

// parseSettingsLocale parses inbound values into package-safe forms.
func parseSettingsLocale(value string) (settingsLocale, bool) {
	tag, err := language.Parse(strings.TrimSpace(value))
	if err != nil {
		return settingsLocaleUnspecified, false
	}
	normalized := strings.ToLower(tag.String())
	if locale, ok := settingsLocaleByTag[normalized]; ok {
		return locale, true
	}
	return settingsLocaleUnspecified, false
}

// normalizeSettingsLocale centralizes this web behavior in one helper seam.
func normalizeSettingsLocale(value settingsLocale) settingsLocale {
	locale, ok := parseSettingsLocale(string(value))
	if ok {
		return locale
	}
	return settingsLocaleEnUS
}

// ParseLocale validates a locale and returns the normalized value.
func ParseLocale(value string) (string, bool) {
	locale, ok := parseSettingsLocale(value)
	if !ok {
		return "", false
	}
	return string(locale), true
}

// NormalizeLocale returns a supported locale value, defaulting to en-US.
func NormalizeLocale(value string) string {
	return string(normalizeSettingsLocale(settingsLocale(value)))
}

// isSafeCredentialPathID reports whether this package condition is satisfied.
func isSafeCredentialPathID(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	return !strings.Contains(value, "/") && !strings.Contains(value, "\\")
}

// validateNameLength centralizes this web behavior in one helper seam.
func validateNameLength(name string) error {
	if utf8.RuneCountInString(name) > UserProfileNameMaxLength {
		return apperrors.EK(apperrors.KindInvalidInput, "web.settings.user_profile.error_name_too_long", "name is too long")
	}
	return nil
}
