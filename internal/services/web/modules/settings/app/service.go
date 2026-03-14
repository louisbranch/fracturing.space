package app

import (
	"context"
	"encoding/json"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// service defines an internal contract used at this web package boundary.
type service struct {
	gateway Gateway
}

// NewService constructs a settings service with fail-closed gateway defaults.
func NewService(gateway Gateway) Service {
	if gateway == nil {
		gateway = unavailableGateway{}
	}
	return service{gateway: gateway}
}

// LoadProfile loads the package state needed for this request path.
func (s service) LoadProfile(ctx context.Context, userID string) (SettingsProfile, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return SettingsProfile{}, err
	}
	profile, err := s.gateway.LoadProfile(ctx, resolvedUserID)
	if err != nil {
		return SettingsProfile{}, err
	}
	return normalizeSettingsProfile(profile), nil
}

// SaveProfile centralizes this web behavior in one helper seam.
func (s service) SaveProfile(ctx context.Context, userID string, profile SettingsProfile) error {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return err
	}
	profile = normalizeSettingsProfile(profile)
	if err := validateNameLength(profile.Name); err != nil {
		return err
	}
	return s.gateway.SaveProfile(ctx, resolvedUserID, profile)
}

// LoadLocale loads the package state needed for this request path.
func (s service) LoadLocale(ctx context.Context, userID string) (string, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return "", err
	}
	locale, err := s.gateway.LoadLocale(ctx, resolvedUserID)
	if err != nil {
		return "", err
	}
	return NormalizeLocale(locale), nil
}

// SaveLocale centralizes this web behavior in one helper seam.
func (s service) SaveLocale(ctx context.Context, userID string, value string) error {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return err
	}
	locale, ok := ParseLocale(value)
	if !ok {
		return apperrors.EK(apperrors.KindInvalidInput, "error.http.invalid_locale", "locale is invalid")
	}
	return s.gateway.SaveLocale(ctx, resolvedUserID, locale)
}

// ListPasskeys returns read-only passkey summaries for the security page.
func (s service) ListPasskeys(ctx context.Context, userID string) ([]SettingsPasskey, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return nil, err
	}
	passkeys, err := s.gateway.ListPasskeys(ctx, resolvedUserID)
	if err != nil {
		return nil, err
	}
	if passkeys == nil {
		return []SettingsPasskey{}, nil
	}
	normalized := make([]SettingsPasskey, 0, len(passkeys))
	for _, passkey := range passkeys {
		normalized = append(normalized, normalizeSettingsPasskey(passkey))
	}
	return normalized, nil
}

// BeginPasskeyRegistration starts authenticated passkey enrollment for the current user.
func (s service) BeginPasskeyRegistration(ctx context.Context, userID string) (PasskeyChallenge, error) {
	resolvedUserID, err := RequireUserID(userID)
	if err != nil {
		return PasskeyChallenge{}, err
	}
	challenge, err := s.gateway.BeginPasskeyRegistration(ctx, resolvedUserID)
	if err != nil {
		return PasskeyChallenge{}, err
	}
	challenge.SessionID = strings.TrimSpace(challenge.SessionID)
	if challenge.SessionID == "" {
		return PasskeyChallenge{}, apperrors.E(apperrors.KindUnavailable, "passkey session is unavailable")
	}
	return challenge, nil
}

// FinishPasskeyRegistration completes authenticated passkey enrollment.
func (s service) FinishPasskeyRegistration(ctx context.Context, sessionID string, credential json.RawMessage) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "session id is required")
	}
	if len(credential) == 0 {
		return apperrors.E(apperrors.KindInvalidInput, "credential is required")
	}
	return s.gateway.FinishPasskeyRegistration(ctx, sessionID, credential)
}

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
	if resolvedCredentialID == "" {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "web.settings.ai_agents.error_credential_required", "credential is required")
	}
	if !isSafeCredentialPathID(resolvedCredentialID) {
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
	input.Name = strings.TrimSpace(input.Name)
	input.CredentialID = strings.TrimSpace(input.CredentialID)
	input.Model = strings.TrimSpace(input.Model)
	input.Instructions = strings.TrimSpace(input.Instructions)
	if input.Name == "" || input.CredentialID == "" || input.Model == "" {
		return apperrors.EK(apperrors.KindInvalidInput, "web.settings.ai_agents.error_required", "name, credential, and model are required")
	}
	if !isSafeCredentialPathID(input.CredentialID) {
		return apperrors.EK(apperrors.KindInvalidInput, "web.settings.ai_agents.error_credential_required", "credential is required")
	}
	return s.gateway.CreateAIAgent(ctx, resolvedUserID, input)
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

// normalizeSettingsProfile centralizes profile field normalization before service flows.
func normalizeSettingsProfile(profile SettingsProfile) SettingsProfile {
	profile.Username = strings.TrimSpace(profile.Username)
	profile.Name = strings.TrimSpace(profile.Name)
	profile.AvatarSetID = strings.TrimSpace(profile.AvatarSetID)
	profile.AvatarAssetID = strings.TrimSpace(profile.AvatarAssetID)
	profile.Bio = strings.TrimSpace(profile.Bio)
	profile.Pronouns = strings.TrimSpace(profile.Pronouns)
	return profile
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
	if !isSafeCredentialPathID(key.ID) {
		key.ID = ""
		key.CanRevoke = false
	}
	return key
}

// normalizeSettingsPasskey normalizes one passkey row for stable rendering.
func normalizeSettingsPasskey(passkey SettingsPasskey) SettingsPasskey {
	if passkey.Number <= 0 {
		passkey.Number = 1
	}
	passkey.CreatedAt = strings.TrimSpace(passkey.CreatedAt)
	passkey.LastUsedAt = strings.TrimSpace(passkey.LastUsedAt)
	if passkey.CreatedAt == "" {
		passkey.CreatedAt = "-"
	}
	if passkey.LastUsedAt == "" {
		passkey.LastUsedAt = "-"
	}
	return passkey
}

// normalizeSettingsAICredentialOption ensures credential options render predictably.
func normalizeSettingsAICredentialOption(option SettingsAICredentialOption) SettingsAICredentialOption {
	option.ID = strings.TrimSpace(option.ID)
	option.Label = strings.TrimSpace(option.Label)
	option.Provider = strings.TrimSpace(option.Provider)
	if option.Label == "" {
		option.Label = option.ID
	}
	if option.Provider == "" {
		option.Provider = "Unknown"
	}
	if !isSafeCredentialPathID(option.ID) {
		option.ID = ""
	}
	return option
}

// normalizeSettingsAIModelOption ensures model options render predictably.
func normalizeSettingsAIModelOption(model SettingsAIModelOption) SettingsAIModelOption {
	model.ID = strings.TrimSpace(model.ID)
	model.OwnedBy = strings.TrimSpace(model.OwnedBy)
	return model
}

// normalizeSettingsAIAgent ensures agent rows render predictably.
func normalizeSettingsAIAgent(agent SettingsAIAgent) SettingsAIAgent {
	agent.ID = strings.TrimSpace(agent.ID)
	agent.Name = strings.TrimSpace(agent.Name)
	agent.Provider = strings.TrimSpace(agent.Provider)
	agent.Model = strings.TrimSpace(agent.Model)
	agent.Status = strings.TrimSpace(agent.Status)
	agent.CreatedAt = strings.TrimSpace(agent.CreatedAt)
	agent.Instructions = strings.TrimSpace(agent.Instructions)
	if agent.Provider == "" {
		agent.Provider = "Unknown"
	}
	if agent.Status == "" {
		agent.Status = "Unspecified"
	}
	if agent.CreatedAt == "" {
		agent.CreatedAt = "-"
	}
	return agent
}
