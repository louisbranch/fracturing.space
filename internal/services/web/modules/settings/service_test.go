package settings

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestNewServiceFailsClosedWhenGatewayMissing(t *testing.T) {
	t.Parallel()

	svc := newService(nil)
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "load profile", run: func() error { _, err := svc.loadProfile(context.Background(), "user-1"); return err }},
		{name: "load locale", run: func() error { _, err := svc.loadLocale(context.Background(), "user-1"); return err }},
		{name: "list ai keys", run: func() error { _, err := svc.listAIKeys(context.Background(), "user-1"); return err }},
		{name: "save profile", run: func() error {
			return svc.saveProfile(context.Background(), "user-1", SettingsProfile{Username: "adventurer", Name: "Adventurer"})
		}},
		{name: "save locale", run: func() error { return svc.saveLocale(context.Background(), "user-1", "en-US") }},
		{name: "create ai key", run: func() error { return svc.createAIKey(context.Background(), "user-1", "Primary", "sk-test") }},
		{name: "revoke ai key", run: func() error { return svc.revokeAIKey(context.Background(), "user-1", "cred-1") }},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.run()
			if err == nil {
				t.Fatalf("expected unavailable error")
			}
			if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
				t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
			}
		})
	}
}

func TestSaveProfileAllowsOptionalUsernameAndName(t *testing.T) {
	t.Parallel()

	svc := newService(settingsGatewayStub{})
	err := svc.saveProfile(context.Background(), "user-1", SettingsProfile{})
	if err != nil {
		t.Fatalf("saveProfile() error = %v", err)
	}
}

func TestServiceRequiresUserID(t *testing.T) {
	t.Parallel()

	svc := newService(settingsGatewayStub{})
	_, err := svc.loadProfile(context.Background(), "   ")
	if err == nil {
		t.Fatalf("expected user-id error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusUnauthorized {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusUnauthorized)
	}
}

func TestSaveProfileValidatesNameLength(t *testing.T) {
	t.Parallel()

	svc := newService(settingsGatewayStub{})
	err := svc.saveProfile(context.Background(), "user-1", SettingsProfile{
		Username: "rhea",
		Name:     strings.Repeat("x", userProfileNameMaxLength+1),
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestSaveProfileDelegatesToGateway(t *testing.T) {
	t.Parallel()

	gateway := &settingsGatewayRecorder{}
	svc := newService(gateway)
	err := svc.saveProfile(context.Background(), "user-1", SettingsProfile{Username: "rhea", Name: "Rhea Vale"})
	if err != nil {
		t.Fatalf("saveProfile() error = %v", err)
	}
	if gateway.lastRequestedUserID != "user-1" {
		t.Fatalf("user id = %q, want %q", gateway.lastRequestedUserID, "user-1")
	}
	if gateway.lastSavedProfile.Username != "rhea" {
		t.Fatalf("saved username = %q, want %q", gateway.lastSavedProfile.Username, "rhea")
	}
}

func TestSaveLocaleParsesAndDelegates(t *testing.T) {
	t.Parallel()

	gateway := &settingsGatewayRecorder{}
	svc := newService(gateway)
	err := svc.saveLocale(context.Background(), "user-1", "pt-BR")
	if err != nil {
		t.Fatalf("saveLocale() error = %v", err)
	}
	if gateway.lastRequestedUserID != "user-1" {
		t.Fatalf("user id = %q, want %q", gateway.lastRequestedUserID, "user-1")
	}
	if gateway.lastSavedLocale != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("saved locale = %v, want %v", gateway.lastSavedLocale, commonv1.Locale_LOCALE_PT_BR)
	}
}

func TestSaveLocaleRejectsUnknownLocale(t *testing.T) {
	t.Parallel()

	svc := newService(settingsGatewayStub{})
	err := svc.saveLocale(context.Background(), "user-1", "es-ES")
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestCreateAIKeyValidatesInput(t *testing.T) {
	t.Parallel()

	svc := newService(settingsGatewayStub{})
	err := svc.createAIKey(context.Background(), "user-1", "", "secret")
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}

	err = svc.createAIKey(context.Background(), "user-1", "label", "")
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestCreateAIKeyDelegatesToGateway(t *testing.T) {
	t.Parallel()

	gateway := &settingsGatewayRecorder{}
	svc := newService(gateway)
	err := svc.createAIKey(context.Background(), "user-1", "Primary", "sk-secret")
	if err != nil {
		t.Fatalf("createAIKey() error = %v", err)
	}
	if gateway.lastRequestedUserID != "user-1" {
		t.Fatalf("user id = %q, want %q", gateway.lastRequestedUserID, "user-1")
	}
	if gateway.lastCreatedLabel != "Primary" {
		t.Fatalf("label = %q, want %q", gateway.lastCreatedLabel, "Primary")
	}
	if gateway.lastCreatedSecret != "sk-secret" {
		t.Fatalf("secret = %q, want %q", gateway.lastCreatedSecret, "sk-secret")
	}
}

func TestRevokeAIKeyValidatesCredentialID(t *testing.T) {
	t.Parallel()

	svc := newService(settingsGatewayStub{})
	err := svc.revokeAIKey(context.Background(), "user-1", "")
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestLoadProfilePropagatesGatewayError(t *testing.T) {
	t.Parallel()

	svc := newService(settingsGatewayStub{loadProfileErr: errors.New("boom")})
	_, err := svc.loadProfile(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected gateway error")
	}
	if err.Error() != "boom" {
		t.Fatalf("err = %q, want %q", err.Error(), "boom")
	}
}

type settingsGatewayStub struct {
	profile        SettingsProfile
	locale         commonv1.Locale
	keys           []SettingsAIKey
	loadProfileErr error
	loadLocaleErr  error
	listAIKeysErr  error
	saveProfileErr error
	saveLocaleErr  error
	createAIKeyErr error
	revokeAIKeyErr error
}

func (f settingsGatewayStub) LoadProfile(context.Context, string) (SettingsProfile, error) {
	if f.loadProfileErr != nil {
		return SettingsProfile{}, f.loadProfileErr
	}
	if f.profile == (SettingsProfile{}) {
		return SettingsProfile{Username: "adventurer", Name: "Adventurer"}, nil
	}
	return f.profile, nil
}

func (f settingsGatewayStub) SaveProfile(context.Context, string, SettingsProfile) error {
	return f.saveProfileErr
}

func (f settingsGatewayStub) LoadLocale(context.Context, string) (commonv1.Locale, error) {
	if f.loadLocaleErr != nil {
		return commonv1.Locale_LOCALE_UNSPECIFIED, f.loadLocaleErr
	}
	if f.locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		return commonv1.Locale_LOCALE_EN_US, nil
	}
	return f.locale, nil
}

func (f settingsGatewayStub) SaveLocale(context.Context, string, commonv1.Locale) error {
	return f.saveLocaleErr
}

func (f settingsGatewayStub) ListAIKeys(context.Context, string) ([]SettingsAIKey, error) {
	if f.listAIKeysErr != nil {
		return nil, f.listAIKeysErr
	}
	if f.keys == nil {
		return []SettingsAIKey{}, nil
	}
	return f.keys, nil
}

func (f settingsGatewayStub) CreateAIKey(context.Context, string, string, string) error {
	return f.createAIKeyErr
}

func (f settingsGatewayStub) RevokeAIKey(context.Context, string, string) error {
	return f.revokeAIKeyErr
}

type settingsGatewayRecorder struct {
	lastRequestedUserID string
	lastSavedProfile    SettingsProfile
	lastSavedLocale     commonv1.Locale
	lastCreatedLabel    string
	lastCreatedSecret   string
	lastRevokedKeyID    string
}

func (f *settingsGatewayRecorder) LoadProfile(_ context.Context, userID string) (SettingsProfile, error) {
	f.lastRequestedUserID = userID
	return SettingsProfile{Username: "adventurer", Name: "Adventurer"}, nil
}

func (f *settingsGatewayRecorder) SaveProfile(_ context.Context, userID string, profile SettingsProfile) error {
	f.lastRequestedUserID = userID
	f.lastSavedProfile = profile
	return nil
}

func (f *settingsGatewayRecorder) LoadLocale(_ context.Context, userID string) (commonv1.Locale, error) {
	f.lastRequestedUserID = userID
	return commonv1.Locale_LOCALE_EN_US, nil
}

func (f *settingsGatewayRecorder) SaveLocale(_ context.Context, userID string, locale commonv1.Locale) error {
	f.lastRequestedUserID = userID
	f.lastSavedLocale = locale
	return nil
}

func (f *settingsGatewayRecorder) ListAIKeys(_ context.Context, userID string) ([]SettingsAIKey, error) {
	f.lastRequestedUserID = userID
	return []SettingsAIKey{}, nil
}

func (f *settingsGatewayRecorder) CreateAIKey(_ context.Context, userID string, label string, secret string) error {
	f.lastRequestedUserID = userID
	f.lastCreatedLabel = label
	f.lastCreatedSecret = secret
	return nil
}

func (f *settingsGatewayRecorder) RevokeAIKey(_ context.Context, userID string, keyID string) error {
	f.lastRequestedUserID = userID
	f.lastRevokedKeyID = keyID
	return nil
}
