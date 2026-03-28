package gateway

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	settingsapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/settings/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type socialStub struct{}

func (socialStub) GetUserProfile(context.Context, *socialv1.GetUserProfileRequest, ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error) {
	return &socialv1.GetUserProfileResponse{UserProfile: &socialv1.UserProfile{Name: "  Rhea  "}}, nil
}
func (socialStub) SetUserProfile(context.Context, *socialv1.SetUserProfileRequest, ...grpc.CallOption) (*socialv1.SetUserProfileResponse, error) {
	return &socialv1.SetUserProfileResponse{}, nil
}

type accountStub struct {
	lastUpdateReq *authv1.UpdateProfileRequest
}

func (accountStub) GetProfile(context.Context, *authv1.GetProfileRequest, ...grpc.CallOption) (*authv1.GetProfileResponse, error) {
	return &authv1.GetProfileResponse{Profile: &authv1.AccountProfile{Username: "  rhea  ", Locale: commonv1.Locale_LOCALE_PT_BR}}, nil
}
func (a *accountStub) UpdateProfile(_ context.Context, req *authv1.UpdateProfileRequest, _ ...grpc.CallOption) (*authv1.UpdateProfileResponse, error) {
	a.lastUpdateReq = req
	return &authv1.UpdateProfileResponse{}, nil
}

type passkeyStub struct{}

func (passkeyStub) ListPasskeys(context.Context, *authv1.ListPasskeysRequest, ...grpc.CallOption) (*authv1.ListPasskeysResponse, error) {
	return &authv1.ListPasskeysResponse{Passkeys: []*authv1.PasskeyCredentialSummary{{}}}, nil
}
func (passkeyStub) BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return &authv1.BeginPasskeyRegistrationResponse{SessionId: "passkey-session-1", CredentialCreationOptionsJson: []byte(`{"publicKey":{}}`)}, nil
}
func (passkeyStub) FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return &authv1.FinishPasskeyRegistrationResponse{}, nil
}

type credentialStub struct {
	listResp      *aiv1.ListCredentialsResponse
	listResponses map[string]*aiv1.ListCredentialsResponse
	listErr       error
	listReqs      []*aiv1.ListCredentialsRequest
	lastCreateReq *aiv1.CreateCredentialRequest
	lastRevokeReq *aiv1.RevokeCredentialRequest
}

func (c *credentialStub) ListCredentials(_ context.Context, req *aiv1.ListCredentialsRequest, _ ...grpc.CallOption) (*aiv1.ListCredentialsResponse, error) {
	c.listReqs = append(c.listReqs, req)
	if c.listErr != nil {
		return nil, c.listErr
	}
	if c.listResponses != nil {
		if resp, ok := c.listResponses[strings.TrimSpace(req.GetPageToken())]; ok {
			return resp, nil
		}
	}
	if c.listResp != nil {
		return c.listResp, nil
	}
	return &aiv1.ListCredentialsResponse{Credentials: []*aiv1.Credential{{
		Id:       "unsafe/id",
		Label:    "Primary",
		Provider: aiv1.Provider_PROVIDER_OPENAI,
		Status:   aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
	}}}, nil
}
func (c *credentialStub) CreateCredential(_ context.Context, req *aiv1.CreateCredentialRequest, _ ...grpc.CallOption) (*aiv1.CreateCredentialResponse, error) {
	c.lastCreateReq = req
	return &aiv1.CreateCredentialResponse{}, nil
}
func (c *credentialStub) RevokeCredential(_ context.Context, req *aiv1.RevokeCredentialRequest, _ ...grpc.CallOption) (*aiv1.RevokeCredentialResponse, error) {
	c.lastRevokeReq = req
	return &aiv1.RevokeCredentialResponse{}, nil
}

type agentStub struct {
	lastCreateReq     *aiv1.CreateAgentRequest
	lastDeleteReq     *aiv1.DeleteAgentRequest
	lastListModelsReq *aiv1.ListProviderModelsRequest
	lastListAgentsReq *aiv1.ListAgentsRequest
	listAgentsCalls   int
}

func (a *agentStub) ListAgents(_ context.Context, req *aiv1.ListAgentsRequest, _ ...grpc.CallOption) (*aiv1.ListAgentsResponse, error) {
	a.lastListAgentsReq = req
	a.listAgentsCalls++
	if strings.TrimSpace(req.GetPageToken()) == "" {
		return &aiv1.ListAgentsResponse{Agents: []*aiv1.Agent{{Id: "agent-1", Label: "narrator", Provider: aiv1.Provider_PROVIDER_OPENAI, Model: "gpt-4o-mini", Status: aiv1.AgentStatus_AGENT_STATUS_ACTIVE, AuthState: aiv1.AgentAuthState_AGENT_AUTH_STATE_READY, ActiveCampaignCount: 1, Instructions: "Keep the session moving."}}, NextPageToken: "page-2"}, nil
	}
	return &aiv1.ListAgentsResponse{Agents: []*aiv1.Agent{{Id: "agent-2", Label: "oracle", Provider: aiv1.Provider_PROVIDER_OPENAI, Model: "gpt-4o", Status: aiv1.AgentStatus_AGENT_STATUS_ACTIVE, AuthState: aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_REVOKED, ActiveCampaignCount: 0, Instructions: "Answer briefly."}}}, nil
}

func (a *agentStub) ListProviderModels(_ context.Context, req *aiv1.ListProviderModelsRequest, _ ...grpc.CallOption) (*aiv1.ListProviderModelsResponse, error) {
	a.lastListModelsReq = req
	return &aiv1.ListProviderModelsResponse{Models: []*aiv1.ProviderModel{{Id: "gpt-4o-mini"}}}, nil
}

func (a *agentStub) CreateAgent(_ context.Context, req *aiv1.CreateAgentRequest, _ ...grpc.CallOption) (*aiv1.CreateAgentResponse, error) {
	a.lastCreateReq = req
	return &aiv1.CreateAgentResponse{}, nil
}

func (a *agentStub) DeleteAgent(_ context.Context, req *aiv1.DeleteAgentRequest, _ ...grpc.CallOption) (*aiv1.DeleteAgentResponse, error) {
	a.lastDeleteReq = req
	return &aiv1.DeleteAgentResponse{}, nil
}

func TestNewGRPCGatewayWithoutConfiguredClientsFailsClosed(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{}
	_, err := gateway.LoadProfile(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected unavailable error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestGRPCGatewayMapsProfileAndLocale(t *testing.T) {
	t.Parallel()

	account := &accountStub{}
	passkeys := passkeyStub{}
	credentials := &credentialStub{}
	agents := &agentStub{}
	gateway := GRPCGateway{
		SocialClient:     socialStub{},
		AccountClient:    account,
		PasskeyClient:    passkeys,
		CredentialClient: credentials,
		AgentClient:      agents,
	}
	profile, err := gateway.LoadProfile(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	if profile.Username != "rhea" || profile.Name != "Rhea" {
		t.Fatalf("unexpected profile: %+v", profile)
	}
	locale, err := gateway.LoadLocale(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("LoadLocale() error = %v", err)
	}
	if locale != "pt-BR" {
		t.Fatalf("locale = %q, want %q", locale, "pt-BR")
	}
	if err := gateway.SaveLocale(context.Background(), "user-1", "en-US"); err != nil {
		t.Fatalf("SaveLocale() error = %v", err)
	}
	if account.lastUpdateReq == nil || account.lastUpdateReq.GetLocale() != commonv1.Locale_LOCALE_EN_US {
		t.Fatalf("unexpected update req: %+v", account.lastUpdateReq)
	}
	if rows, err := gateway.ListAIKeys(context.Background(), "user-1"); err != nil || len(rows) != 1 || rows[0].CanRevoke {
		t.Fatalf("unexpected AI keys: rows=%+v err=%v", rows, err)
	}
}

func TestGRPCGatewayListAIProviderModelsUsesTypedAuthReference(t *testing.T) {
	t.Parallel()

	agents := &agentStub{}
	credentials := &credentialStub{listResp: &aiv1.ListCredentialsResponse{Credentials: []*aiv1.Credential{{
		Id:       "cred-1",
		Label:    "Primary",
		Provider: aiv1.Provider_PROVIDER_ANTHROPIC,
		Status:   aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
	}}}}
	gateway := GRPCGateway{CredentialClient: credentials, AgentClient: agents}

	rows, err := gateway.ListAIProviderModels(context.Background(), "user-1", "cred-1")
	if err != nil {
		t.Fatalf("ListAIProviderModels() error = %v", err)
	}
	if len(rows) != 1 || rows[0].ID != "gpt-4o-mini" {
		t.Fatalf("unexpected models: %+v", rows)
	}
	if agents.lastListModelsReq == nil {
		t.Fatal("expected ListProviderModels request")
	}
	if agents.lastListModelsReq.GetAuthReference().GetType() != aiv1.AgentAuthReferenceType_AGENT_AUTH_REFERENCE_TYPE_CREDENTIAL {
		t.Fatalf("auth reference type = %v, want credential", agents.lastListModelsReq.GetAuthReference().GetType())
	}
	if agents.lastListModelsReq.GetAuthReference().GetId() != "cred-1" {
		t.Fatalf("auth reference id = %q, want %q", agents.lastListModelsReq.GetAuthReference().GetId(), "cred-1")
	}
	if agents.lastListModelsReq.GetProvider() != aiv1.Provider_PROVIDER_ANTHROPIC {
		t.Fatalf("provider = %v, want %v", agents.lastListModelsReq.GetProvider(), aiv1.Provider_PROVIDER_ANTHROPIC)
	}
}

func TestGRPCGatewayCreateAIAgentUsesTypedAuthReference(t *testing.T) {
	t.Parallel()

	agents := &agentStub{}
	credentials := &credentialStub{listResp: &aiv1.ListCredentialsResponse{Credentials: []*aiv1.Credential{{
		Id:       "cred-1",
		Label:    "Primary",
		Provider: aiv1.Provider_PROVIDER_ANTHROPIC,
		Status:   aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
	}}}}
	gateway := GRPCGateway{CredentialClient: credentials, AgentClient: agents}

	err := gateway.CreateAIAgent(context.Background(), "user-1", settingsapp.CreateAIAgentInput{
		Label:        "narrator",
		Model:        "gpt-4o-mini",
		CredentialID: "cred-1",
		Instructions: "Keep the scene moving.",
	})
	if err != nil {
		t.Fatalf("CreateAIAgent() error = %v", err)
	}
	if agents.lastCreateReq == nil {
		t.Fatal("expected CreateAgent request")
	}
	if agents.lastCreateReq.GetAuthReference().GetType() != aiv1.AgentAuthReferenceType_AGENT_AUTH_REFERENCE_TYPE_CREDENTIAL {
		t.Fatalf("auth reference type = %v, want credential", agents.lastCreateReq.GetAuthReference().GetType())
	}
	if agents.lastCreateReq.GetAuthReference().GetId() != "cred-1" {
		t.Fatalf("auth reference id = %q, want %q", agents.lastCreateReq.GetAuthReference().GetId(), "cred-1")
	}
	if agents.lastCreateReq.GetProvider() != aiv1.Provider_PROVIDER_ANTHROPIC {
		t.Fatalf("provider = %v, want %v", agents.lastCreateReq.GetProvider(), aiv1.Provider_PROVIDER_ANTHROPIC)
	}
}

func TestGRPCGatewayListAIAgentCredentialsAndAgents(t *testing.T) {
	t.Parallel()

	credentials := &credentialStub{listResp: &aiv1.ListCredentialsResponse{Credentials: []*aiv1.Credential{
		{Id: "cred-1", Label: "Primary", Provider: aiv1.Provider_PROVIDER_OPENAI, Status: aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE},
		{Id: "bad/id", Label: "Unsafe", Provider: aiv1.Provider_PROVIDER_OPENAI, Status: aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE},
		{Id: "cred-2", Label: "Revoked", Provider: aiv1.Provider_PROVIDER_ANTHROPIC, Status: aiv1.CredentialStatus_CREDENTIAL_STATUS_REVOKED},
	}}}
	agents := &agentStub{}
	gateway := GRPCGateway{CredentialClient: credentials, AgentClient: agents}

	options, err := gateway.ListAIAgentCredentials(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListAIAgentCredentials() error = %v", err)
	}
	if len(options) != 1 || options[0].ID != "cred-1" {
		t.Fatalf("credential options = %+v, want cred-1 only", options)
	}

	rows, err := gateway.ListAIAgents(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListAIAgents() error = %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want %d", len(rows), 2)
	}
	if rows[0].Provider != "OpenAI" || rows[1].Provider != "OpenAI" {
		t.Fatalf("unexpected provider labels: %+v", rows)
	}
	if !rows[1].CanDelete {
		t.Fatalf("rows[1].CanDelete = false, want true")
	}
}

func TestGRPCGatewayDeleteAndRevokeAIResources(t *testing.T) {
	t.Parallel()

	credentials := &credentialStub{}
	agents := &agentStub{}
	gateway := GRPCGateway{CredentialClient: credentials, AgentClient: agents}

	if err := gateway.DeleteAIAgent(context.Background(), "user-1", "agent-1"); err != nil {
		t.Fatalf("DeleteAIAgent() error = %v", err)
	}
	if agents.lastDeleteReq == nil || agents.lastDeleteReq.GetAgentId() != "agent-1" {
		t.Fatalf("unexpected delete req: %+v", agents.lastDeleteReq)
	}

	if err := gateway.RevokeAIKey(context.Background(), "user-1", "cred-1"); err != nil {
		t.Fatalf("RevokeAIKey() error = %v", err)
	}
	if credentials.lastRevokeReq == nil || credentials.lastRevokeReq.GetCredentialId() != "cred-1" {
		t.Fatalf("unexpected revoke req: %+v", credentials.lastRevokeReq)
	}
}

func TestGRPCGatewayCreateAIKeyParsesProvider(t *testing.T) {
	t.Parallel()

	credentials := &credentialStub{}
	gateway := GRPCGateway{CredentialClient: credentials}

	err := gateway.CreateAIKey(context.Background(), "user-1", settingsapp.CreateAIKeyInput{
		Label:    "Primary",
		Provider: " anthropic ",
		Secret:   "secret-key",
	})
	if err != nil {
		t.Fatalf("CreateAIKey() error = %v", err)
	}
	if credentials.lastCreateReq == nil {
		t.Fatal("expected CreateCredential request")
	}
	if credentials.lastCreateReq.GetProvider() != aiv1.Provider_PROVIDER_ANTHROPIC {
		t.Fatalf("provider = %v, want %v", credentials.lastCreateReq.GetProvider(), aiv1.Provider_PROVIDER_ANTHROPIC)
	}
}

func TestProviderProtoFromWeb(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    aiv1.Provider
		wantErr bool
	}{
		{name: "openai", input: "openai", want: aiv1.Provider_PROVIDER_OPENAI},
		{name: "anthropic trim", input: "  anthropic ", want: aiv1.Provider_PROVIDER_ANTHROPIC},
		{name: "invalid", input: "", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := providerProtoFromWeb(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if apperrors.HTTPStatus(err) != http.StatusBadRequest {
					t.Fatalf("HTTPStatus(err) = %d, want %d", apperrors.HTTPStatus(err), http.StatusBadRequest)
				}
				return
			}
			if err != nil {
				t.Fatalf("providerProtoFromWeb() error = %v", err)
			}
			if got != tc.want {
				t.Fatalf("providerProtoFromWeb() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestGRPCGatewayLookupCredentialPagesUntilMatch(t *testing.T) {
	t.Parallel()

	credentials := &credentialStub{listResponses: map[string]*aiv1.ListCredentialsResponse{
		"": {
			Credentials:   []*aiv1.Credential{{Id: "cred-1", Provider: aiv1.Provider_PROVIDER_OPENAI}},
			NextPageToken: "page-2",
		},
		"page-2": {
			Credentials: []*aiv1.Credential{{Id: "cred-2", Provider: aiv1.Provider_PROVIDER_ANTHROPIC}},
		},
	}}
	gateway := GRPCGateway{CredentialClient: credentials}

	credential, err := gateway.lookupCredential(context.Background(), "cred-2")
	if err != nil {
		t.Fatalf("lookupCredential() error = %v", err)
	}
	if credential.GetId() != "cred-2" {
		t.Fatalf("credential id = %q, want %q", credential.GetId(), "cred-2")
	}
	if len(credentials.listReqs) != 2 {
		t.Fatalf("len(listReqs) = %d, want %d", len(credentials.listReqs), 2)
	}
}

func TestGRPCGatewayLookupCredentialErrors(t *testing.T) {
	t.Parallel()

	t.Run("missing credential", func(t *testing.T) {
		t.Parallel()
		gateway := GRPCGateway{CredentialClient: &credentialStub{listResp: &aiv1.ListCredentialsResponse{}}}
		_, err := gateway.lookupCredential(context.Background(), "missing")
		if err == nil {
			t.Fatal("expected error")
		}
		if apperrors.HTTPStatus(err) != http.StatusBadRequest {
			t.Fatalf("HTTPStatus(err) = %d, want %d", apperrors.HTTPStatus(err), http.StatusBadRequest)
		}
	})

	t.Run("list failure", func(t *testing.T) {
		t.Parallel()
		gateway := GRPCGateway{CredentialClient: &credentialStub{listErr: errors.New("boom")}}
		_, err := gateway.lookupCredential(context.Background(), "cred-1")
		if err == nil || !strings.Contains(err.Error(), "list credentials") {
			t.Fatalf("lookupCredential() error = %v, want wrapped list error", err)
		}
	})
}

func TestGRPCGatewayAIHelperFormattingAndMapping(t *testing.T) {
	t.Parallel()

	if got := providerDisplayLabel(aiv1.Provider_PROVIDER_ANTHROPIC); got != "Anthropic" {
		t.Fatalf("providerDisplayLabel() = %q, want %q", got, "Anthropic")
	}
	if got := credentialStatusDisplayLabel(aiv1.CredentialStatus_CREDENTIAL_STATUS_REVOKED); got != "Revoked" {
		t.Fatalf("credentialStatusDisplayLabel() = %q, want %q", got, "Revoked")
	}
	if got := agentStatusDisplayLabel(aiv1.AgentStatus_AGENT_STATUS_ACTIVE); got != "Active" {
		t.Fatalf("agentStatusDisplayLabel() = %q, want %q", got, "Active")
	}
	if got := agentAuthStateDisplayLabel(aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE); got != "Auth unavailable" {
		t.Fatalf("agentAuthStateDisplayLabel() = %q, want %q", got, "Auth unavailable")
	}
	if got := formatProtoTimestamp(nil); got != "-" {
		t.Fatalf("formatProtoTimestamp(nil) = %q, want %q", got, "-")
	}
	if got := formatProtoTimestamp(timestamppb.New(time.Date(2026, time.March, 10, 14, 30, 0, 0, time.UTC))); got != "2026-03-10 14:30 UTC" {
		t.Fatalf("formatProtoTimestamp(valid) = %q, want %q", got, "2026-03-10 14:30 UTC")
	}

	keyErr := mapAIKeyMutationError(status.Error(codes.AlreadyExists, "duplicate"))
	if apperrors.HTTPStatus(keyErr) != http.StatusConflict {
		t.Fatalf("HTTPStatus(keyErr) = %d, want %d", apperrors.HTTPStatus(keyErr), http.StatusConflict)
	}
	agentErr := mapAIAgentMutationError(status.Error(codes.FailedPrecondition, "in use"))
	if apperrors.HTTPStatus(agentErr) != http.StatusConflict {
		t.Fatalf("HTTPStatus(agentErr) = %d, want %d", apperrors.HTTPStatus(agentErr), http.StatusConflict)
	}
}

func TestGRPCGatewayListPasskeysSortsByLastUsedThenCreated(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, time.March, 9, 12, 0, 0, 0, time.UTC)
	gateway := GRPCGateway{
		PasskeyClient: passkeyListStub{resp: &authv1.ListPasskeysResponse{Passkeys: []*authv1.PasskeyCredentialSummary{
			{CreatedAt: timestamppb.New(base.Add(-3 * time.Hour))},
			{CreatedAt: timestamppb.New(base.Add(-2 * time.Hour)), LastUsedAt: timestamppb.New(base.Add(-1 * time.Hour))},
			{CreatedAt: timestamppb.New(base.Add(-90 * time.Minute)), LastUsedAt: timestamppb.New(base.Add(-1 * time.Hour))},
		}}},
	}

	rows, err := gateway.ListPasskeys(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("ListPasskeys() error = %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("len(rows) = %d, want %d", len(rows), 3)
	}
	if rows[0].CreatedAt != "2026-03-09 10:30 UTC" {
		t.Fatalf("rows[0].CreatedAt = %q, want %q", rows[0].CreatedAt, "2026-03-09 10:30 UTC")
	}
	if rows[1].CreatedAt != "2026-03-09 10:00 UTC" {
		t.Fatalf("rows[1].CreatedAt = %q, want %q", rows[1].CreatedAt, "2026-03-09 10:00 UTC")
	}
	if rows[2].CreatedAt != "2026-03-09 09:00 UTC" {
		t.Fatalf("rows[2].CreatedAt = %q, want %q", rows[2].CreatedAt, "2026-03-09 09:00 UTC")
	}
}

func TestGRPCGatewayPasskeyRegistrationFlowsAndConstructors(t *testing.T) {
	t.Parallel()

	passkeys := passkeyStub{}
	accountGateway := NewAccountGateway(socialStub{}, &accountStub{}, passkeys)
	aiGateway := NewAIGateway(&credentialStub{}, &agentStub{})
	if accountGateway.PasskeyClient == nil || aiGateway.CredentialClient == nil || aiGateway.AgentClient == nil {
		t.Fatal("constructors should wire provided clients")
	}

	challenge, err := accountGateway.BeginPasskeyRegistration(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("BeginPasskeyRegistration() error = %v", err)
	}
	if challenge.SessionID != "passkey-session-1" {
		t.Fatalf("challenge.SessionID = %q, want %q", challenge.SessionID, "passkey-session-1")
	}

	if err := accountGateway.FinishPasskeyRegistration(context.Background(), "passkey-session-1", []byte(`{"id":"cred-1"}`)); err != nil {
		t.Fatalf("FinishPasskeyRegistration() error = %v", err)
	}
}

type passkeyListStub struct {
	resp *authv1.ListPasskeysResponse
}

func (s passkeyListStub) ListPasskeys(context.Context, *authv1.ListPasskeysRequest, ...grpc.CallOption) (*authv1.ListPasskeysResponse, error) {
	return s.resp, nil
}

func (passkeyListStub) BeginPasskeyRegistration(context.Context, *authv1.BeginPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.BeginPasskeyRegistrationResponse, error) {
	return &authv1.BeginPasskeyRegistrationResponse{}, nil
}

func (passkeyListStub) FinishPasskeyRegistration(context.Context, *authv1.FinishPasskeyRegistrationRequest, ...grpc.CallOption) (*authv1.FinishPasskeyRegistrationResponse, error) {
	return &authv1.FinishPasskeyRegistrationResponse{}, nil
}

func TestGRPCGatewayMissingClientBehavior(t *testing.T) {
	t.Parallel()

	gateway := GRPCGateway{}
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "load profile", run: func() error { _, err := gateway.LoadProfile(context.Background(), "user-1"); return err }},
		{name: "save profile", run: func() error {
			return gateway.SaveProfile(context.Background(), "user-1", settingsapp.SettingsProfile{})
		}},
		{name: "load locale", run: func() error { _, err := gateway.LoadLocale(context.Background(), "user-1"); return err }},
		{name: "save locale", run: func() error { return gateway.SaveLocale(context.Background(), "user-1", "en-US") }},
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
