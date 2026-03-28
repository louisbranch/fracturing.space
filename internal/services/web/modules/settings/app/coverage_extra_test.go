package app

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

type securityGatewayStub struct {
	passkeys       []SettingsPasskey
	passkeysErr    error
	beginChallenge PasskeyChallenge
	beginErr       error
	finishErr      error

	lastUserID     string
	lastSessionID  string
	lastCredential json.RawMessage
}

func (s *securityGatewayStub) ListPasskeys(_ context.Context, userID string) ([]SettingsPasskey, error) {
	s.lastUserID = userID
	if s.passkeysErr != nil {
		return nil, s.passkeysErr
	}
	return s.passkeys, nil
}

func (s *securityGatewayStub) BeginPasskeyRegistration(_ context.Context, userID string) (PasskeyChallenge, error) {
	s.lastUserID = userID
	if s.beginErr != nil {
		return PasskeyChallenge{}, s.beginErr
	}
	return s.beginChallenge, nil
}

func (s *securityGatewayStub) FinishPasskeyRegistration(_ context.Context, sessionID string, credential json.RawMessage) error {
	s.lastSessionID = sessionID
	s.lastCredential = credential
	return s.finishErr
}

func TestSecurityServiceFlowsNormalizeValidateAndDelegate(t *testing.T) {
	t.Parallel()

	gateway := &securityGatewayStub{
		passkeys: []SettingsPasskey{{
			Number:     0,
			CreatedAt:  " ",
			LastUsedAt: "",
		}},
		beginChallenge: PasskeyChallenge{
			SessionID: " passkey-session-1 ",
			PublicKey: json.RawMessage(`{"publicKey":{}}`),
		},
	}
	service := NewAccountService(AccountServiceConfig{
		ProfileGateway:  unavailableGateway{},
		LocaleGateway:   unavailableGateway{},
		SecurityGateway: gateway,
	})

	passkeys, err := service.ListPasskeys(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListPasskeys() error = %v", err)
	}
	if len(passkeys) != 1 {
		t.Fatalf("len(passkeys) = %d, want 1", len(passkeys))
	}
	if passkeys[0].Number != 1 || passkeys[0].CreatedAt != "-" || passkeys[0].LastUsedAt != "-" {
		t.Fatalf("normalized passkey = %+v", passkeys[0])
	}
	if gateway.lastUserID != "user-1" {
		t.Fatalf("ListPasskeys user id = %q, want %q", gateway.lastUserID, "user-1")
	}

	challenge, err := service.BeginPasskeyRegistration(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("BeginPasskeyRegistration() error = %v", err)
	}
	if challenge.SessionID != "passkey-session-1" {
		t.Fatalf("challenge.SessionID = %q, want %q", challenge.SessionID, "passkey-session-1")
	}

	if err := service.FinishPasskeyRegistration(context.Background(), " passkey-session-1 ", json.RawMessage(`{"id":"cred-1"}`)); err != nil {
		t.Fatalf("FinishPasskeyRegistration() error = %v", err)
	}
	if gateway.lastSessionID != "passkey-session-1" || string(gateway.lastCredential) != `{"id":"cred-1"}` {
		t.Fatalf("finish delegation = (%q,%s)", gateway.lastSessionID, gateway.lastCredential)
	}
}

func TestSecurityServiceValidationAndUnavailableFallback(t *testing.T) {
	t.Parallel()

	service := NewAccountService(AccountServiceConfig{})

	if passkeys, err := service.ListPasskeys(context.Background(), "user-1"); err == nil {
		t.Fatalf("ListPasskeys() error = nil, want unavailable")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("ListPasskeys() status = %d, want %d", got, http.StatusServiceUnavailable)
	} else if passkeys != nil {
		t.Fatalf("ListPasskeys() = %+v, want nil", passkeys)
	}

	if _, err := service.BeginPasskeyRegistration(context.Background(), "user-1"); err == nil {
		t.Fatalf("BeginPasskeyRegistration() error = nil, want unavailable")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("BeginPasskeyRegistration() status = %d, want %d", got, http.StatusServiceUnavailable)
	}

	if err := service.FinishPasskeyRegistration(context.Background(), "session-1", json.RawMessage(`{"id":"cred-1"}`)); err == nil {
		t.Fatalf("FinishPasskeyRegistration() error = nil, want unavailable")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("FinishPasskeyRegistration() status = %d, want %d", got, http.StatusServiceUnavailable)
	}

	explicit := NewUnavailableGateway()
	if _, err := explicit.ListPasskeys(context.Background(), "user-1"); err == nil {
		t.Fatalf("explicit unavailable ListPasskeys() error = nil")
	}
	if _, err := explicit.BeginPasskeyRegistration(context.Background(), "user-1"); err == nil {
		t.Fatalf("explicit unavailable BeginPasskeyRegistration() error = nil")
	}
	if err := explicit.FinishPasskeyRegistration(context.Background(), "session-1", json.RawMessage(`{"id":"cred-1"}`)); err == nil {
		t.Fatalf("explicit unavailable FinishPasskeyRegistration() error = nil")
	}
}

func TestSecurityServiceRejectsEmptySessionAndCredentialAndEmptyBeginSession(t *testing.T) {
	t.Parallel()

	service := NewAccountService(AccountServiceConfig{
		SecurityGateway: &securityGatewayStub{beginChallenge: PasskeyChallenge{SessionID: " "}},
	})

	if _, err := service.BeginPasskeyRegistration(context.Background(), "user-1"); err == nil {
		t.Fatalf("expected empty session challenge error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}

	if err := service.FinishPasskeyRegistration(context.Background(), " ", json.RawMessage(`{"id":"cred-1"}`)); err == nil {
		t.Fatalf("expected empty session validation error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}

	if err := service.FinishPasskeyRegistration(context.Background(), "session-1", nil); err == nil {
		t.Fatalf("expected empty credential validation error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestAIServiceCoversNilCollectionsInvalidIDsAndDefaultNormalizers(t *testing.T) {
	t.Parallel()

	gateway := &gatewayStub{}
	service := newService(gateway)

	keys, err := service.ListAIKeys(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListAIKeys() error = %v", err)
	}
	if len(keys) != 0 {
		t.Fatalf("ListAIKeys() len = %d, want 0", len(keys))
	}

	credentials, err := service.ListAIAgentCredentials(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListAIAgentCredentials() error = %v", err)
	}
	if len(credentials) != 0 {
		t.Fatalf("ListAIAgentCredentials() len = %d, want 0", len(credentials))
	}

	agents, err := service.ListAIAgents(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListAIAgents() error = %v", err)
	}
	if len(agents) != 0 {
		t.Fatalf("ListAIAgents() len = %d, want 0", len(agents))
	}

	models, err := service.ListAIProviderModels(context.Background(), "user-1", "cred-1")
	if err != nil {
		t.Fatalf("ListAIProviderModels() error = %v", err)
	}
	if len(models) != 0 {
		t.Fatalf("ListAIProviderModels() len = %d, want 0", len(models))
	}

	if err := service.DeleteAIAgent(context.Background(), "user-1", `bad\\id`); err == nil {
		t.Fatalf("expected invalid agent id error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}

	if _, err := service.ListAIProviderModels(context.Background(), "user-1", `bad\\id`); err == nil {
		t.Fatalf("expected invalid credential id error")
	} else if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}

	if IsSafePathID(`bad\\id`) {
		t.Fatal("IsSafePathID() = true for backslash path")
	}

	option := normalizeSettingsAICredentialOption(SettingsAICredentialOption{})
	if option.Provider != "Unknown" || option.Label != "Unknown" {
		t.Fatalf("normalized credential option = %+v", option)
	}

	agent := normalizeSettingsAIAgent(SettingsAIAgent{CanDelete: true})
	if agent.Provider != "Unknown" || agent.AuthState != "Unknown" || agent.CreatedAt != "-" || agent.CanDelete {
		t.Fatalf("normalized agent = %+v", agent)
	}
}
