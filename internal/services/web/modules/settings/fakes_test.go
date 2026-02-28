package settings

import "context"

// fakeGateway implements SettingsGateway for tests with configurable return
// values, error injection, and call recording.
type fakeGateway struct {
	profile        SettingsProfile
	locale         string
	keys           []SettingsAIKey
	loadProfileErr error
	loadLocaleErr  error
	listAIKeysErr  error
	saveProfileErr error
	saveLocaleErr  error
	createAIKeyErr error
	revokeAIKeyErr error

	lastSavedProfile        SettingsProfile
	lastSavedLocale         string
	lastCreatedLabel        string
	lastCreatedSecret       string
	lastRevokedCredentialID string
	lastRequestedUserID     string
}

// newPopulatedFakeGateway returns a fakeGateway pre-loaded with rich canned
// data suitable for integration-level module tests.
func newPopulatedFakeGateway() *fakeGateway {
	return &fakeGateway{
		profile: SettingsProfile{
			Username:      "rhea",
			Name:          "Rhea Vale",
			AvatarSetID:   "set-a",
			AvatarAssetID: "asset-1",
			Bio:           "Traveler",
		},
		locale: "pt-BR",
		keys: []SettingsAIKey{{
			ID:        "cred-1",
			Label:     "Primary Key",
			Provider:  "OpenAI",
			Status:    "Active",
			CreatedAt: "2026-01-01 00:00 UTC",
			RevokedAt: "-",
			CanRevoke: true,
		}},
	}
}

func (f *fakeGateway) LoadProfile(_ context.Context, userID string) (SettingsProfile, error) {
	f.lastRequestedUserID = userID
	if f.loadProfileErr != nil {
		return SettingsProfile{}, f.loadProfileErr
	}
	if f.profile == (SettingsProfile{}) {
		return SettingsProfile{Username: "adventurer", Name: "Adventurer"}, nil
	}
	return f.profile, nil
}

func (f *fakeGateway) SaveProfile(_ context.Context, userID string, profile SettingsProfile) error {
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

func (f *fakeGateway) ListAIKeys(_ context.Context, userID string) ([]SettingsAIKey, error) {
	f.lastRequestedUserID = userID
	if f.listAIKeysErr != nil {
		return nil, f.listAIKeysErr
	}
	if f.keys == nil {
		return []SettingsAIKey{}, nil
	}
	return f.keys, nil
}

func (f *fakeGateway) CreateAIKey(_ context.Context, userID string, label string, secret string) error {
	f.lastRequestedUserID = userID
	f.lastCreatedLabel = label
	f.lastCreatedSecret = secret
	return f.createAIKeyErr
}

func (f *fakeGateway) RevokeAIKey(_ context.Context, userID string, credentialID string) error {
	f.lastRequestedUserID = userID
	f.lastRevokedCredentialID = credentialID
	return f.revokeAIKeyErr
}
