package app

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

type gatewayStub struct {
	profile     SettingsProfile
	locale      string
	keys        []SettingsAIKey
	passkeys    []SettingsPasskey
	credentials []SettingsAICredentialOption
	models      []SettingsAIModelOption
	agents      []SettingsAIAgent
	err         error

	lastUserID       string
	lastProfile      SettingsProfile
	lastLocale       string
	lastLabel        string
	lastSecret       string
	lastAgent        CreateAIAgentInput
	lastAgentID      string
	lastCredentialID string
	lastCredential   string
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
func (g gatewayStub) ListPasskeys(context.Context, string) ([]SettingsPasskey, error) {
	if g.err != nil {
		return nil, g.err
	}
	return g.passkeys, nil
}
func (g *gatewayStub) BeginPasskeyRegistration(_ context.Context, userID string) (PasskeyChallenge, error) {
	g.lastUserID = userID
	if g.err != nil {
		return PasskeyChallenge{}, g.err
	}
	return PasskeyChallenge{SessionID: "passkey-session-1", PublicKey: json.RawMessage(`{"publicKey":{}}`)}, nil
}
func (g *gatewayStub) FinishPasskeyRegistration(_ context.Context, sessionID string, credential json.RawMessage) error {
	g.lastCredentialID = sessionID
	g.lastCredential = string(credential)
	return g.err
}
func (g gatewayStub) ListAIKeys(context.Context, string) ([]SettingsAIKey, error) {
	if g.err != nil {
		return nil, g.err
	}
	return g.keys, nil
}
func (g gatewayStub) ListAIAgentCredentials(context.Context, string) ([]SettingsAICredentialOption, error) {
	if g.err != nil {
		return nil, g.err
	}
	return g.credentials, nil
}
func (g gatewayStub) ListAIAgents(context.Context, string) ([]SettingsAIAgent, error) {
	if g.err != nil {
		return nil, g.err
	}
	return g.agents, nil
}
func (g *gatewayStub) ListAIProviderModels(_ context.Context, userID string, credentialID string) ([]SettingsAIModelOption, error) {
	g.lastUserID = userID
	g.lastCredentialID = credentialID
	if g.err != nil {
		return nil, g.err
	}
	return g.models, nil
}
func (g *gatewayStub) CreateAIKey(_ context.Context, userID string, label string, secret string) error {
	g.lastUserID = userID
	g.lastLabel = label
	g.lastSecret = secret
	return g.err
}
func (g *gatewayStub) CreateAIAgent(_ context.Context, userID string, input CreateAIAgentInput) error {
	g.lastUserID = userID
	g.lastAgent = input
	return g.err
}
func (g *gatewayStub) DeleteAIAgent(_ context.Context, userID string, agentID string) error {
	g.lastUserID = userID
	g.lastAgentID = agentID
	return g.err
}
func (g *gatewayStub) RevokeAIKey(_ context.Context, userID string, credentialID string) error {
	g.lastUserID = userID
	g.lastCredential = credentialID
	return g.err
}

func TestNewServiceFailsClosedWhenGatewayMissing(t *testing.T) {
	t.Parallel()

	svc := newService(nil)
	_, err := svc.LoadProfile(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestUnavailableGatewayFailsClosed(t *testing.T) {
	t.Parallel()

	gateway := NewUnavailableGateway()

	ctx := context.Background()
	if profile, err := gateway.LoadProfile(ctx, "user-1"); err == nil {
		t.Fatalf("LoadProfile() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("LoadProfile() status = %d, want %d", got, http.StatusServiceUnavailable)
	} else if profile != (SettingsProfile{}) {
		t.Fatalf("LoadProfile() profile = %+v, want zero value", profile)
	}
	if err := gateway.SaveProfile(ctx, "user-1", SettingsProfile{Name: "Rhea"}); err == nil {
		t.Fatalf("SaveProfile() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("SaveProfile() status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if locale, err := gateway.LoadLocale(ctx, "user-1"); err == nil {
		t.Fatalf("LoadLocale() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("LoadLocale() status = %d, want %d", got, http.StatusServiceUnavailable)
	} else if locale != "" {
		t.Fatalf("LoadLocale() locale = %q, want empty", locale)
	}
	if err := gateway.SaveLocale(ctx, "user-1", "en-US"); err == nil {
		t.Fatalf("SaveLocale() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("SaveLocale() status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if keys, err := gateway.ListAIKeys(ctx, "user-1"); err == nil {
		t.Fatalf("ListAIKeys() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("ListAIKeys() status = %d, want %d", got, http.StatusServiceUnavailable)
	} else if keys != nil {
		t.Fatalf("ListAIKeys() keys = %+v, want nil", keys)
	}
	if err := gateway.CreateAIKey(ctx, "user-1", "Primary", "sk-secret"); err == nil {
		t.Fatalf("CreateAIKey() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("CreateAIKey() status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if credentials, err := gateway.ListAIAgentCredentials(ctx, "user-1"); err == nil {
		t.Fatalf("ListAIAgentCredentials() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("ListAIAgentCredentials() status = %d, want %d", got, http.StatusServiceUnavailable)
	} else if credentials != nil {
		t.Fatalf("ListAIAgentCredentials() = %+v, want nil", credentials)
	}
	if agents, err := gateway.ListAIAgents(ctx, "user-1"); err == nil {
		t.Fatalf("ListAIAgents() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("ListAIAgents() status = %d, want %d", got, http.StatusServiceUnavailable)
	} else if agents != nil {
		t.Fatalf("ListAIAgents() = %+v, want nil", agents)
	}
	if models, err := gateway.ListAIProviderModels(ctx, "user-1", "cred-1"); err == nil {
		t.Fatalf("ListAIProviderModels() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("ListAIProviderModels() status = %d, want %d", got, http.StatusServiceUnavailable)
	} else if models != nil {
		t.Fatalf("ListAIProviderModels() = %+v, want nil", models)
	}
	if err := gateway.CreateAIAgent(ctx, "user-1", CreateAIAgentInput{Label: "narrator", CredentialID: "cred-1", Model: "gpt-4o-mini"}); err == nil {
		t.Fatalf("CreateAIAgent() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("CreateAIAgent() status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if err := gateway.RevokeAIKey(ctx, "user-1", "cred-1"); err == nil {
		t.Fatalf("RevokeAIKey() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("RevokeAIKey() status = %d, want %d", got, http.StatusServiceUnavailable)
	}
	if err := gateway.DeleteAIAgent(ctx, "user-1", "agent-1"); err == nil {
		t.Fatalf("DeleteAIAgent() error = nil, want unavailable error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("DeleteAIAgent() status = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestSaveProfileValidatesNameLength(t *testing.T) {
	t.Parallel()

	svc := newService(&gatewayStub{})
	err := svc.SaveProfile(context.Background(), "user-1", SettingsProfile{Name: strings.Repeat("x", UserProfileNameMaxLength+1)})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestLoadAndSaveProfileNormalizeFields(t *testing.T) {
	t.Parallel()

	gateway := &gatewayStub{
		profile: SettingsProfile{
			Username:      "  rhea  ",
			Name:          "  Rhea Vale  ",
			AvatarSetID:   "  portraits ",
			AvatarAssetID: "  ranger ",
			Pronouns:      " she/they ",
			Bio:           "  cartographer ",
		},
	}
	svc := newService(gateway)

	loaded, err := svc.LoadProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	if loaded.Username != "rhea" || loaded.Name != "Rhea Vale" {
		t.Fatalf("LoadProfile() normalization failed: %+v", loaded)
	}
	if err := svc.SaveProfile(context.Background(), "user-1", SettingsProfile{
		Username:      "  rhea  ",
		Name:          "  Rhea Vale  ",
		AvatarSetID:   "  portraits ",
		AvatarAssetID: "  ranger ",
		Pronouns:      " she/they ",
		Bio:           "  cartographer ",
	}); err != nil {
		t.Fatalf("SaveProfile() error = %v", err)
	}
	if gateway.lastProfile.Username != "rhea" || gateway.lastProfile.Name != "Rhea Vale" {
		t.Fatalf("SaveProfile() normalization failed: %+v", gateway.lastProfile)
	}
	if gateway.lastProfile.AvatarSetID != "portraits" || gateway.lastProfile.AvatarAssetID != "ranger" {
		t.Fatalf("SaveProfile() avatar normalization failed: %+v", gateway.lastProfile)
	}
	if gateway.lastProfile.Pronouns != "she/they" || gateway.lastProfile.Bio != "cartographer" {
		t.Fatalf("SaveProfile() text normalization failed: %+v", gateway.lastProfile)
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

	svc := newService(&gatewayStub{})
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
	svc := newService(gateway)
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

func TestLoadLocaleValidatesAndNormalizes(t *testing.T) {
	t.Parallel()

	svc := newService(&gatewayStub{locale: "pt"})
	locale, err := svc.LoadLocale(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadLocale() error = %v", err)
	}
	if locale != "pt-BR" {
		t.Fatalf("LoadLocale() = %q, want %q", locale, "pt-BR")
	}

	if _, err := svc.LoadLocale(context.Background(), "   "); err == nil {
		t.Fatalf("expected user-id validation error")
	}
}

func TestListAIKeysNormalizesRows(t *testing.T) {
	t.Parallel()

	svc := newService(&gatewayStub{keys: []SettingsAIKey{{
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
	svc := newService(gateway)
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
	if err := svc.RevokeAIKey(context.Background(), "user-1", " cred-1 "); err != nil {
		t.Fatalf("RevokeAIKey() error = %v", err)
	}
	if gateway.lastCredential != "cred-1" {
		t.Fatalf("revoke delegation credential = %q, want %q", gateway.lastCredential, "cred-1")
	}
}

func TestAIAgentServiceFlowsValidateNormalizeAndDelegate(t *testing.T) {
	t.Parallel()

	gateway := &gatewayStub{
		credentials: []SettingsAICredentialOption{{ID: " cred-1 ", Label: " Primary ", Provider: " OpenAI "}},
		models:      []SettingsAIModelOption{{ID: " gpt-4o-mini ", OwnedBy: " openai "}},
		agents:      []SettingsAIAgent{{ID: " agent-1 ", Label: " narrator ", Provider: " OpenAI ", Model: " gpt-4o-mini ", AuthState: " Ready ", CanDelete: true, ActiveCampaignCount: 2, CreatedAt: " 2026-01-01 00:00 UTC "}},
	}
	svc := newService(gateway)

	credentials, err := svc.ListAIAgentCredentials(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListAIAgentCredentials() error = %v", err)
	}
	if len(credentials) != 1 || credentials[0].ID != "cred-1" || credentials[0].Label != "Primary" {
		t.Fatalf("credentials = %+v", credentials)
	}

	models, err := svc.ListAIProviderModels(context.Background(), "user-1", " cred-1 ")
	if err != nil {
		t.Fatalf("ListAIProviderModels() error = %v", err)
	}
	if len(models) != 1 || models[0].ID != "gpt-4o-mini" {
		t.Fatalf("models = %+v", models)
	}
	if gateway.lastCredentialID != "cred-1" {
		t.Fatalf("credential id = %q, want %q", gateway.lastCredentialID, "cred-1")
	}

	agents, err := svc.ListAIAgents(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListAIAgents() error = %v", err)
	}
	if len(agents) != 1 || agents[0].Label != "narrator" || agents[0].AuthState != "Ready" || agents[0].ActiveCampaignCount != 2 {
		t.Fatalf("agents = %+v", agents)
	}

	if err := svc.CreateAIAgent(context.Background(), "user-1", CreateAIAgentInput{}); err == nil {
		t.Fatalf("expected create validation error")
	}
	if err := svc.CreateAIAgent(context.Background(), "user-1", CreateAIAgentInput{
		Label:        " narrator ",
		CredentialID: " cred-1 ",
		Model:        " gpt-4o-mini ",
		Instructions: " Keep the session moving. ",
	}); err != nil {
		t.Fatalf("CreateAIAgent() error = %v", err)
	}
	if gateway.lastAgent.Label != "narrator" || gateway.lastAgent.CredentialID != "cred-1" || gateway.lastAgent.Model != "gpt-4o-mini" {
		t.Fatalf("delegated agent input = %+v", gateway.lastAgent)
	}
	if gateway.lastAgent.Instructions != "Keep the session moving." {
		t.Fatalf("instructions = %q, want %q", gateway.lastAgent.Instructions, "Keep the session moving.")
	}
	if err := svc.DeleteAIAgent(context.Background(), "user-1", " agent-1 "); err != nil {
		t.Fatalf("DeleteAIAgent() error = %v", err)
	}
	if gateway.lastAgentID != "agent-1" {
		t.Fatalf("agent id = %q, want %q", gateway.lastAgentID, "agent-1")
	}
}
