package settings

import (
	"context"
	"encoding/json"

	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
)

// fakeGateway implements the split settings gateway seams for tests with configurable return
// values, error injection, and call recording.
type fakeGateway struct {
	profile          settingsapp.SettingsProfile
	locale           string
	keys             []settingsapp.SettingsAIKey
	passkeys         []settingsapp.SettingsPasskey
	credentials      []settingsapp.SettingsAICredentialOption
	models           []settingsapp.SettingsAIModelOption
	agents           []settingsapp.SettingsAIAgent
	loadProfileErr   error
	loadLocaleErr    error
	listAIKeysErr    error
	listModelsErr    error
	listAgentsErr    error
	saveProfileErr   error
	saveLocaleErr    error
	createAIKeyErr   error
	createAIAgentErr error
	deleteAIAgentErr error
	revokeAIKeyErr   error

	lastSavedProfile         settingsapp.SettingsProfile
	lastSavedLocale          string
	lastCreatedLabel         string
	lastCreatedSecret        string
	lastCreatedAgent         settingsapp.CreateAIAgentInput
	lastDeletedAgentID       string
	lastSelectedCredentialID string
	lastRevokedCredentialID  string
	lastRequestedUserID      string
	lastPasskeySessionID     string
	lastPasskeyCredential    json.RawMessage
}

// newPopulatedFakeGateway returns a fakeGateway pre-loaded with rich canned
// data suitable for integration-level module tests.
func newPopulatedFakeGateway() *fakeGateway {
	return &fakeGateway{
		profile: settingsapp.SettingsProfile{
			Username:      "rhea",
			Name:          "Rhea Vale",
			AvatarSetID:   "set-a",
			AvatarAssetID: "asset-1",
			Bio:           "Traveler",
		},
		locale: "pt-BR",
		keys: []settingsapp.SettingsAIKey{{
			ID:        "cred-1",
			Label:     "Primary Key",
			Provider:  "OpenAI",
			Status:    "Active",
			CreatedAt: "2026-01-01 00:00 UTC",
			RevokedAt: "-",
			CanRevoke: true,
		}},
		passkeys: []settingsapp.SettingsPasskey{{
			Number:     1,
			CreatedAt:  "2026-01-01 00:00 UTC",
			LastUsedAt: "2026-01-02 00:00 UTC",
		}},
		credentials: []settingsapp.SettingsAICredentialOption{{
			ID:       "cred-1",
			Label:    "Primary Key",
			Provider: "OpenAI",
		}},
		models: []settingsapp.SettingsAIModelOption{{
			ID:      "gpt-4o-mini",
			OwnedBy: "openai",
		}},
		agents: []settingsapp.SettingsAIAgent{{
			ID:                  "agent-1",
			Label:               "narrator",
			Provider:            "OpenAI",
			Model:               "gpt-4o-mini",
			AuthState:           "Ready",
			CanDelete:           true,
			ActiveCampaignCount: 0,
			CreatedAt:           "2026-01-01 00:00 UTC",
			Instructions:        "Keep the session moving.",
		}},
	}
}

func (f *fakeGateway) LoadProfile(_ context.Context, userID string) (settingsapp.SettingsProfile, error) {
	f.lastRequestedUserID = userID
	if f.loadProfileErr != nil {
		return settingsapp.SettingsProfile{}, f.loadProfileErr
	}
	if f.profile == (settingsapp.SettingsProfile{}) {
		return settingsapp.SettingsProfile{Username: "adventurer", Name: "Adventurer"}, nil
	}
	return f.profile, nil
}

func (f *fakeGateway) SaveProfile(_ context.Context, userID string, profile settingsapp.SettingsProfile) error {
	f.lastRequestedUserID = userID
	f.lastSavedProfile = profile
	return f.saveProfileErr
}

func (f *fakeGateway) LoadLocale(_ context.Context, userID string) (string, error) {
	f.lastRequestedUserID = userID
	if f.loadLocaleErr != nil {
		return "", f.loadLocaleErr
	}
	if f.locale == "" {
		return "en-US", nil
	}
	return f.locale, nil
}

func (f *fakeGateway) SaveLocale(_ context.Context, userID string, locale string) error {
	f.lastRequestedUserID = userID
	f.lastSavedLocale = locale
	return f.saveLocaleErr
}

func (f *fakeGateway) ListPasskeys(_ context.Context, userID string) ([]settingsapp.SettingsPasskey, error) {
	f.lastRequestedUserID = userID
	if f.passkeys == nil {
		return []settingsapp.SettingsPasskey{}, nil
	}
	return f.passkeys, nil
}

func (f *fakeGateway) BeginPasskeyRegistration(_ context.Context, userID string) (settingsapp.PasskeyChallenge, error) {
	f.lastRequestedUserID = userID
	return settingsapp.PasskeyChallenge{
		SessionID: "passkey-session-1",
		PublicKey: json.RawMessage(`{"publicKey":{}}`),
	}, nil
}

func (f *fakeGateway) FinishPasskeyRegistration(_ context.Context, sessionID string, credential json.RawMessage) error {
	f.lastPasskeySessionID = sessionID
	f.lastPasskeyCredential = credential
	return nil
}

func (f *fakeGateway) ListAIKeys(_ context.Context, userID string) ([]settingsapp.SettingsAIKey, error) {
	f.lastRequestedUserID = userID
	if f.listAIKeysErr != nil {
		return nil, f.listAIKeysErr
	}
	if f.keys == nil {
		return []settingsapp.SettingsAIKey{}, nil
	}
	return f.keys, nil
}

func (f *fakeGateway) CreateAIKey(_ context.Context, userID string, label string, secret string) error {
	f.lastRequestedUserID = userID
	f.lastCreatedLabel = label
	f.lastCreatedSecret = secret
	return f.createAIKeyErr
}

func (f *fakeGateway) ListAIAgentCredentials(_ context.Context, userID string) ([]settingsapp.SettingsAICredentialOption, error) {
	f.lastRequestedUserID = userID
	if f.listAIKeysErr != nil {
		return nil, f.listAIKeysErr
	}
	if f.credentials == nil {
		return []settingsapp.SettingsAICredentialOption{}, nil
	}
	return f.credentials, nil
}

func (f *fakeGateway) ListAIAgents(_ context.Context, userID string) ([]settingsapp.SettingsAIAgent, error) {
	f.lastRequestedUserID = userID
	if f.listAgentsErr != nil {
		return nil, f.listAgentsErr
	}
	if f.agents == nil {
		return []settingsapp.SettingsAIAgent{}, nil
	}
	return f.agents, nil
}

func (f *fakeGateway) ListAIProviderModels(_ context.Context, userID string, credentialID string) ([]settingsapp.SettingsAIModelOption, error) {
	f.lastRequestedUserID = userID
	f.lastSelectedCredentialID = credentialID
	if f.listModelsErr != nil {
		return nil, f.listModelsErr
	}
	if f.models == nil {
		return []settingsapp.SettingsAIModelOption{}, nil
	}
	return f.models, nil
}

func (f *fakeGateway) CreateAIAgent(_ context.Context, userID string, input settingsapp.CreateAIAgentInput) error {
	f.lastRequestedUserID = userID
	f.lastCreatedAgent = input
	return f.createAIAgentErr
}

func (f *fakeGateway) DeleteAIAgent(_ context.Context, userID string, agentID string) error {
	f.lastRequestedUserID = userID
	f.lastDeletedAgentID = agentID
	return f.deleteAIAgentErr
}

func (f *fakeGateway) RevokeAIKey(_ context.Context, userID string, credentialID string) error {
	f.lastRequestedUserID = userID
	f.lastRevokedCredentialID = credentialID
	return f.revokeAIKeyErr
}
