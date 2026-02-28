package settings

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

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

	svc := newService(&fakeGateway{})
	err := svc.saveProfile(context.Background(), "user-1", SettingsProfile{})
	if err != nil {
		t.Fatalf("saveProfile() error = %v", err)
	}
}

func TestServiceRequiresUserID(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeGateway{})
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

	svc := newService(&fakeGateway{})
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

	gateway := &fakeGateway{}
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

	gateway := &fakeGateway{}
	svc := newService(gateway)
	err := svc.saveLocale(context.Background(), "user-1", "pt-BR")
	if err != nil {
		t.Fatalf("saveLocale() error = %v", err)
	}
	if gateway.lastRequestedUserID != "user-1" {
		t.Fatalf("user id = %q, want %q", gateway.lastRequestedUserID, "user-1")
	}
	if gateway.lastSavedLocale != "pt-BR" {
		t.Fatalf("saved locale = %v, want %v", gateway.lastSavedLocale, "pt-BR")
	}
}

func TestSaveLocaleRejectsUnknownLocale(t *testing.T) {
	t.Parallel()

	svc := newService(&fakeGateway{})
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

	svc := newService(&fakeGateway{})
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

	gateway := &fakeGateway{}
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

	svc := newService(&fakeGateway{})
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

	svc := newService(&fakeGateway{loadProfileErr: errors.New("boom")})
	_, err := svc.loadProfile(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected gateway error")
	}
	if err.Error() != "boom" {
		t.Fatalf("err = %q, want %q", err.Error(), "boom")
	}
}
