package app

import (
	"context"
	"net/http"
	"strings"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

type gatewayStub struct {
	profile SettingsProfile
	locale  string
	keys    []SettingsAIKey
	err     error

	lastUserID     string
	lastProfile    SettingsProfile
	lastLocale     string
	lastLabel      string
	lastSecret     string
	lastCredential string
}

func (g gatewayStub) LoadProfile(context.Context, string) (SettingsProfile, error) {
	if g.err != nil {
		return SettingsProfile{}, g.err
	}
	return g.profile, nil
}
func (g *gatewayStub) SaveProfile(_ context.Context, userID string, profile SettingsProfile) error {
	g.lastUserID = userID
	g.lastProfile = profile
	return g.err
}
func (g gatewayStub) LoadLocale(context.Context, string) (string, error) {
	if g.err != nil {
		return "", g.err
	}
	return g.locale, nil
}
func (g *gatewayStub) SaveLocale(_ context.Context, userID string, locale string) error {
	g.lastUserID = userID
	g.lastLocale = locale
	return g.err
}
func (g gatewayStub) ListAIKeys(context.Context, string) ([]SettingsAIKey, error) {
	if g.err != nil {
		return nil, g.err
	}
	return g.keys, nil
}
func (g *gatewayStub) CreateAIKey(_ context.Context, userID string, label string, secret string) error {
	g.lastUserID = userID
	g.lastLabel = label
	g.lastSecret = secret
	return g.err
}
func (g *gatewayStub) RevokeAIKey(_ context.Context, userID string, credentialID string) error {
	g.lastUserID = userID
	g.lastCredential = credentialID
	return g.err
}

func TestNewServiceFailsClosedWhenGatewayMissing(t *testing.T) {
	t.Parallel()

	svc := NewService(nil)
	_, err := svc.LoadProfile(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestSaveProfileValidatesNameLength(t *testing.T) {
	t.Parallel()

	svc := NewService(&gatewayStub{})
	err := svc.SaveProfile(context.Background(), "user-1", SettingsProfile{Name: strings.Repeat("x", UserProfileNameMaxLength+1)})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestLocaleNormalizationAndParsing(t *testing.T) {
	t.Parallel()

	if got := NormalizeLocale("pt"); got != "pt-BR" {
		t.Fatalf("NormalizeLocale(pt) = %q, want %q", got, "pt-BR")
	}
	if got := NormalizeLocale("bad"); got != "en-US" {
		t.Fatalf("NormalizeLocale(bad) = %q, want %q", got, "en-US")
	}
	if got, ok := ParseLocale("en"); !ok || got != "en-US" {
		t.Fatalf("ParseLocale(en) = (%q,%t), want (%q,true)", got, ok, "en-US")
	}
}

func TestServiceRequiresUserID(t *testing.T) {
	t.Parallel()

	svc := NewService(&gatewayStub{})
	if _, err := svc.LoadProfile(context.Background(), "   "); err == nil {
		t.Fatalf("expected user-id error")
	}
	if err := svc.SaveProfile(context.Background(), "   ", SettingsProfile{}); err == nil {
		t.Fatalf("expected user-id error")
	}
}

func TestSaveLocaleValidatesAndDelegates(t *testing.T) {
	t.Parallel()

	gateway := &gatewayStub{}
	svc := NewService(gateway)
	if err := svc.SaveLocale(context.Background(), "user-1", "pt"); err != nil {
		t.Fatalf("SaveLocale() error = %v", err)
	}
	if gateway.lastLocale != "pt-BR" {
		t.Fatalf("saved locale = %q, want %q", gateway.lastLocale, "pt-BR")
	}
	if err := svc.SaveLocale(context.Background(), "user-1", "es-ES"); err == nil {
		t.Fatalf("expected invalid-locale error")
	}
}

func TestListAIKeysNormalizesRows(t *testing.T) {
	t.Parallel()

	svc := NewService(&gatewayStub{keys: []SettingsAIKey{{
		ID:        "unsafe/id",
		Provider:  "",
		Status:    "",
		CreatedAt: "",
		RevokedAt: "",
		CanRevoke: true,
	}}})
	rows, err := svc.ListAIKeys(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListAIKeys() error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].ID != "" || rows[0].CanRevoke {
		t.Fatalf("unsafe key should be disabled: %+v", rows[0])
	}
	if rows[0].Provider != "Unknown" || rows[0].Status != "Unspecified" {
		t.Fatalf("normalized row = %+v", rows[0])
	}
}

func TestCreateAndRevokeAIKeyValidationAndDelegation(t *testing.T) {
	t.Parallel()

	gateway := &gatewayStub{}
	svc := NewService(gateway)
	if err := svc.CreateAIKey(context.Background(), "user-1", "", "secret"); err == nil {
		t.Fatalf("expected create validation error")
	}
	if err := svc.CreateAIKey(context.Background(), "user-1", "Primary", "sk-secret"); err != nil {
		t.Fatalf("CreateAIKey() error = %v", err)
	}
	if gateway.lastLabel != "Primary" || gateway.lastSecret != "sk-secret" {
		t.Fatalf("create delegation mismatch label=%q secret=%q", gateway.lastLabel, gateway.lastSecret)
	}
	if err := svc.RevokeAIKey(context.Background(), "user-1", ""); err == nil {
		t.Fatalf("expected revoke validation error")
	}
	if err := svc.RevokeAIKey(context.Background(), "user-1", "cred-1"); err != nil {
		t.Fatalf("RevokeAIKey() error = %v", err)
	}
	if gateway.lastCredential != "cred-1" {
		t.Fatalf("revoke delegation credential = %q, want %q", gateway.lastCredential, "cred-1")
	}
}
