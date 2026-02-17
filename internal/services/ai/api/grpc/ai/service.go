package ai

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	"github.com/louisbranch/fracturing.space/internal/services/ai/accessrequest"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/providergrant"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// userIDHeader is injected by trusted edge/auth layers and consumed here for
	// ownership enforcement. Direct callers must not be allowed to spoof it.
	userIDHeader = "x-fracturing-space-user-id"

	defaultPageSize = 10
	maxPageSize     = 50

	providerGrantRefreshWindow = 2 * time.Minute
)

// SecretSealer encrypts secret values before persistence.
type SecretSealer interface {
	Seal(value string) (string, error)
	Open(sealed string) (string, error)
}

// ProviderOAuthAdapter handles provider-specific OAuth URL/token exchange logic.
type ProviderOAuthAdapter interface {
	BuildAuthorizationURL(input ProviderAuthorizationURLInput) (string, error)
	ExchangeAuthorizationCode(ctx context.Context, input ProviderAuthorizationCodeInput) (ProviderTokenExchangeResult, error)
	RefreshToken(ctx context.Context, input ProviderRefreshTokenInput) (ProviderTokenExchangeResult, error)
	RevokeToken(ctx context.Context, input ProviderRevokeTokenInput) error
}

// ProviderAuthorizationURLInput contains parameters for building provider auth URL.
type ProviderAuthorizationURLInput struct {
	State           string
	CodeChallenge   string
	RequestedScopes []string
}

// ProviderAuthorizationCodeInput contains token-exchange input fields.
type ProviderAuthorizationCodeInput struct {
	AuthorizationCode string
	CodeVerifier      string
}

// ProviderRefreshTokenInput contains refresh-token input fields.
type ProviderRefreshTokenInput struct {
	RefreshToken string
}

// ProviderRevokeTokenInput contains token-revocation input fields.
type ProviderRevokeTokenInput struct {
	Token string
}

// ProviderTokenExchangeResult contains provider token exchange output.
type ProviderTokenExchangeResult struct {
	TokenPlaintext   string
	RefreshSupported bool
	ExpiresAt        *time.Time
	LastRefreshError string
}

// ProviderInvocationAdapter handles provider-specific inference invocation.
type ProviderInvocationAdapter interface {
	Invoke(ctx context.Context, input ProviderInvokeInput) (ProviderInvokeResult, error)
}

// ProviderInvokeInput contains provider invocation input fields.
type ProviderInvokeInput struct {
	Model string
	Input string
	// CredentialSecret is decrypted only at call-time and must never be logged.
	CredentialSecret string
}

// ProviderInvokeResult contains invocation output.
type ProviderInvokeResult struct {
	OutputText string
}

// Service implements ai.v1 credential and agent services.
//
// It is the orchestration root where credential/grant/agent state is validated,
// authorized, and projected into protocol responses for callers.
type Service struct {
	aiv1.UnimplementedCredentialServiceServer
	aiv1.UnimplementedAgentServiceServer
	aiv1.UnimplementedInvocationServiceServer
	aiv1.UnimplementedProviderGrantServiceServer
	aiv1.UnimplementedAccessRequestServiceServer

	credentialStore            storage.CredentialStore
	agentStore                 storage.AgentStore
	providerGrantStore         storage.ProviderGrantStore
	connectSessionStore        storage.ProviderConnectSessionStore
	accessRequestStore         storage.AccessRequestStore
	auditEventStore            storage.AuditEventStore
	providerOAuthAdapters      map[providergrant.Provider]ProviderOAuthAdapter
	providerInvocationAdapters map[providergrant.Provider]ProviderInvocationAdapter
	sealer                     SecretSealer

	clock       func() time.Time
	idGenerator func() (string, error)
	// codeVerifierGenerator is injectable for tests; production uses
	// cryptographic randomness for PKCE verifier generation.
	codeVerifierGenerator func() (string, error)
}

// NewService builds a new ai.v1 service implementation.
//
// Passing the service and credential stores separately allows one persisted
// snapshot to satisfy multiple interfaces while preserving explicit dependency intent.
func NewService(credentialStore storage.CredentialStore, agentStore storage.AgentStore, sealer SecretSealer) *Service {
	var providerGrantStore storage.ProviderGrantStore
	if store, ok := credentialStore.(storage.ProviderGrantStore); ok {
		providerGrantStore = store
	}
	if providerGrantStore == nil {
		if store, ok := agentStore.(storage.ProviderGrantStore); ok {
			providerGrantStore = store
		}
	}
	var connectSessionStore storage.ProviderConnectSessionStore
	if store, ok := credentialStore.(storage.ProviderConnectSessionStore); ok {
		connectSessionStore = store
	}
	if connectSessionStore == nil {
		if store, ok := agentStore.(storage.ProviderConnectSessionStore); ok {
			connectSessionStore = store
		}
	}
	var accessRequestStore storage.AccessRequestStore
	if store, ok := credentialStore.(storage.AccessRequestStore); ok {
		accessRequestStore = store
	}
	if accessRequestStore == nil {
		if store, ok := agentStore.(storage.AccessRequestStore); ok {
			accessRequestStore = store
		}
	}
	var auditEventStore storage.AuditEventStore
	if store, ok := credentialStore.(storage.AuditEventStore); ok {
		auditEventStore = store
	}
	if auditEventStore == nil {
		if store, ok := agentStore.(storage.AuditEventStore); ok {
			auditEventStore = store
		}
	}
	return &Service{
		credentialStore:     credentialStore,
		agentStore:          agentStore,
		providerGrantStore:  providerGrantStore,
		connectSessionStore: connectSessionStore,
		accessRequestStore:  accessRequestStore,
		auditEventStore:     auditEventStore,
		providerOAuthAdapters: map[providergrant.Provider]ProviderOAuthAdapter{
			providergrant.ProviderOpenAI: &defaultOpenAIOAuthAdapter{},
		},
		providerInvocationAdapters: map[providergrant.Provider]ProviderInvocationAdapter{
			providergrant.ProviderOpenAI: NewOpenAIInvokeAdapter(OpenAIInvokeConfig{}),
		},
		sealer:      sealer,
		clock:       time.Now,
		idGenerator: id.NewID,
		codeVerifierGenerator: func() (string, error) {
			return generatePKCECodeVerifier()
		},
	}
}

// SetOpenAIOAuthAdapter overrides the OpenAI OAuth adapter implementation.
func (s *Service) SetOpenAIOAuthAdapter(adapter ProviderOAuthAdapter) {
	if s == nil || adapter == nil {
		return
	}
	if s.providerOAuthAdapters == nil {
		s.providerOAuthAdapters = make(map[providergrant.Provider]ProviderOAuthAdapter)
	}
	s.providerOAuthAdapters[providergrant.ProviderOpenAI] = adapter
}

// SetOpenAIInvocationAdapter overrides the OpenAI invocation adapter.
func (s *Service) SetOpenAIInvocationAdapter(adapter ProviderInvocationAdapter) {
	if s == nil || adapter == nil {
		return
	}
	if s.providerInvocationAdapters == nil {
		s.providerInvocationAdapters = make(map[providergrant.Provider]ProviderInvocationAdapter)
	}
	s.providerInvocationAdapters[providergrant.ProviderOpenAI] = adapter
}

type defaultOpenAIOAuthAdapter struct{}

// OpenAIOAuthConfig configures OpenAI OAuth endpoints and credentials.
type OpenAIOAuthConfig struct {
	AuthorizationURL string
	TokenURL         string
	ClientID         string
	ClientSecret     string
	RedirectURI      string
	HTTPClient       *http.Client
}

type openAIOAuthAdapter struct {
	cfg OpenAIOAuthConfig
}

// OpenAIInvokeConfig configures OpenAI responses endpoint and HTTP behavior.
type OpenAIInvokeConfig struct {
	ResponsesURL string
	HTTPClient   *http.Client
}

type openAIInvokeAdapter struct {
	cfg OpenAIInvokeConfig
}

// NewOpenAIOAuthAdapter builds an OpenAI OAuth adapter using HTTP token exchange.
func NewOpenAIOAuthAdapter(cfg OpenAIOAuthConfig) ProviderOAuthAdapter {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	return &openAIOAuthAdapter{cfg: cfg}
}

// NewOpenAIInvokeAdapter builds an OpenAI invocation adapter.
func NewOpenAIInvokeAdapter(cfg OpenAIInvokeConfig) ProviderInvocationAdapter {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	if strings.TrimSpace(cfg.ResponsesURL) == "" {
		cfg.ResponsesURL = "https://api.openai.com/v1/responses"
	}
	return &openAIInvokeAdapter{cfg: cfg}
}

func (a *defaultOpenAIOAuthAdapter) BuildAuthorizationURL(input ProviderAuthorizationURLInput) (string, error) {
	return fmt.Sprintf("https://oauth.fracturing.space/openai?state=%s", strings.TrimSpace(input.State)), nil
}

func (a *defaultOpenAIOAuthAdapter) ExchangeAuthorizationCode(_ context.Context, input ProviderAuthorizationCodeInput) (ProviderTokenExchangeResult, error) {
	code := strings.TrimSpace(input.AuthorizationCode)
	if code == "" {
		return ProviderTokenExchangeResult{}, fmt.Errorf("authorization code is required")
	}
	token := "token:" + code
	return ProviderTokenExchangeResult{
		TokenPlaintext:   token,
		RefreshSupported: true,
	}, nil
}

func (a *defaultOpenAIOAuthAdapter) RefreshToken(_ context.Context, input ProviderRefreshTokenInput) (ProviderTokenExchangeResult, error) {
	refreshToken := strings.TrimSpace(input.RefreshToken)
	if refreshToken == "" {
		return ProviderTokenExchangeResult{}, fmt.Errorf("refresh token is required")
	}
	token := "token:refresh:" + refreshToken
	return ProviderTokenExchangeResult{
		TokenPlaintext:   token,
		RefreshSupported: true,
	}, nil
}

func (a *defaultOpenAIOAuthAdapter) RevokeToken(_ context.Context, input ProviderRevokeTokenInput) error {
	if strings.TrimSpace(input.Token) == "" {
		return fmt.Errorf("token is required")
	}
	return nil
}

func (a *openAIOAuthAdapter) BuildAuthorizationURL(input ProviderAuthorizationURLInput) (string, error) {
	authURL := strings.TrimSpace(a.cfg.AuthorizationURL)
	clientID := strings.TrimSpace(a.cfg.ClientID)
	redirectURI := strings.TrimSpace(a.cfg.RedirectURI)
	state := strings.TrimSpace(input.State)
	challenge := strings.TrimSpace(input.CodeChallenge)
	if authURL == "" || clientID == "" || redirectURI == "" || state == "" || challenge == "" {
		return "", fmt.Errorf("authorization url configuration is incomplete")
	}

	u, err := url.Parse(authURL)
	if err != nil {
		return "", fmt.Errorf("parse authorization url: %w", err)
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	scopes := strings.TrimSpace(strings.Join(normalizeScopes(input.RequestedScopes), " "))
	if scopes != "" {
		q.Set("scope", scopes)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (a *openAIOAuthAdapter) ExchangeAuthorizationCode(ctx context.Context, input ProviderAuthorizationCodeInput) (ProviderTokenExchangeResult, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", strings.TrimSpace(input.AuthorizationCode))
	form.Set("code_verifier", strings.TrimSpace(input.CodeVerifier))
	form.Set("client_id", strings.TrimSpace(a.cfg.ClientID))
	form.Set("client_secret", strings.TrimSpace(a.cfg.ClientSecret))
	form.Set("redirect_uri", strings.TrimSpace(a.cfg.RedirectURI))
	return a.tokenRequest(ctx, form)
}

func (a *openAIOAuthAdapter) RefreshToken(ctx context.Context, input ProviderRefreshTokenInput) (ProviderTokenExchangeResult, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", strings.TrimSpace(input.RefreshToken))
	form.Set("client_id", strings.TrimSpace(a.cfg.ClientID))
	form.Set("client_secret", strings.TrimSpace(a.cfg.ClientSecret))
	return a.tokenRequest(ctx, form)
}

func (a *openAIOAuthAdapter) RevokeToken(_ context.Context, input ProviderRevokeTokenInput) error {
	if strings.TrimSpace(input.Token) == "" {
		return fmt.Errorf("token is required")
	}
	// OpenAI revocation endpoint support is optional at this phase boundary.
	// Returning nil here avoids leaking token material into error/log paths.
	return nil
}

func (a *openAIInvokeAdapter) Invoke(ctx context.Context, input ProviderInvokeInput) (ProviderInvokeResult, error) {
	responsesURL := strings.TrimSpace(a.cfg.ResponsesURL)
	credentialSecret := strings.TrimSpace(input.CredentialSecret)
	model := strings.TrimSpace(input.Model)
	prompt := strings.TrimSpace(input.Input)
	if responsesURL == "" {
		return ProviderInvokeResult{}, fmt.Errorf("responses url is required")
	}
	if credentialSecret == "" {
		return ProviderInvokeResult{}, fmt.Errorf("credential secret is required")
	}
	if model == "" {
		return ProviderInvokeResult{}, fmt.Errorf("model is required")
	}
	if prompt == "" {
		return ProviderInvokeResult{}, fmt.Errorf("input is required")
	}

	requestBody, err := json.Marshal(map[string]any{
		"model": model,
		"input": prompt,
	})
	if err != nil {
		return ProviderInvokeResult{}, fmt.Errorf("marshal invoke request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, responsesURL, strings.NewReader(string(requestBody)))
	if err != nil {
		return ProviderInvokeResult{}, fmt.Errorf("build invoke request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	// Credential material is sent only as an Authorization header and is never
	// echoed in errors or response payloads.
	req.Header.Set("Authorization", "Bearer "+credentialSecret)

	res, err := a.cfg.HTTPClient.Do(req)
	if err != nil {
		return ProviderInvokeResult{}, fmt.Errorf("invoke request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
		if err != nil {
			return ProviderInvokeResult{}, fmt.Errorf("read invoke error body: %w", err)
		}
		return ProviderInvokeResult{}, fmt.Errorf("invoke request status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload struct {
		OutputText string `json:"output_text"`
		Output     []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return ProviderInvokeResult{}, fmt.Errorf("decode invoke response: %w", err)
	}
	outputText := strings.TrimSpace(payload.OutputText)
	if outputText == "" {
		for _, item := range payload.Output {
			for _, content := range item.Content {
				if strings.TrimSpace(content.Text) != "" {
					outputText = strings.TrimSpace(content.Text)
					break
				}
			}
			if outputText != "" {
				break
			}
		}
	}
	if outputText == "" {
		return ProviderInvokeResult{}, fmt.Errorf("invoke response missing output text")
	}
	return ProviderInvokeResult{OutputText: outputText}, nil
}

func (a *openAIOAuthAdapter) tokenRequest(ctx context.Context, form url.Values) (ProviderTokenExchangeResult, error) {
	tokenURL := strings.TrimSpace(a.cfg.TokenURL)
	if tokenURL == "" {
		return ProviderTokenExchangeResult{}, fmt.Errorf("token url is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return ProviderTokenExchangeResult{}, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := a.cfg.HTTPClient.Do(req)
	if err != nil {
		return ProviderTokenExchangeResult{}, fmt.Errorf("token request failed: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, err := io.ReadAll(io.LimitReader(res.Body, 4096))
		if err != nil {
			return ProviderTokenExchangeResult{}, fmt.Errorf("read token error body: %w", err)
		}
		return ProviderTokenExchangeResult{}, fmt.Errorf("token request status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload map[string]any
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		return ProviderTokenExchangeResult{}, fmt.Errorf("decode token response: %w", err)
	}

	accessToken := asString(payload["access_token"])
	refreshToken := asString(payload["refresh_token"])
	tokenType := asString(payload["token_type"])
	scope := asString(payload["scope"])
	if accessToken == "" {
		return ProviderTokenExchangeResult{}, fmt.Errorf("token response missing access_token")
	}
	tokenPlaintextBytes, err := json.Marshal(map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    tokenType,
		"scope":         scope,
	})
	if err != nil {
		return ProviderTokenExchangeResult{}, fmt.Errorf("marshal token payload: %w", err)
	}

	var expiresAt *time.Time
	switch value := payload["expires_in"].(type) {
	case float64:
		if value > 0 {
			exp := time.Now().UTC().Add(time.Duration(value) * time.Second)
			expiresAt = &exp
		}
	case int:
		if value > 0 {
			exp := time.Now().UTC().Add(time.Duration(value) * time.Second)
			expiresAt = &exp
		}
	}
	return ProviderTokenExchangeResult{
		TokenPlaintext:   string(tokenPlaintextBytes),
		RefreshSupported: refreshToken != "",
		ExpiresAt:        expiresAt,
	}, nil
}

func asString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return ""
	}
}

// CreateCredential creates one user-owned provider credential.
func (s *Service) CreateCredential(ctx context.Context, in *aiv1.CreateCredentialRequest) (*aiv1.CreateCredentialResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create credential request is required")
	}
	if s.credentialStore == nil {
		return nil, status.Error(codes.Internal, "credential store is not configured")
	}
	if s.sealer == nil {
		return nil, status.Error(codes.Internal, "secret sealer is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	provider, err := credentialProviderFromProto(in.GetProvider())
	if err != nil {
		return nil, err
	}

	created, err := credential.Create(credential.CreateInput{
		OwnerUserID: userID,
		Provider:    provider,
		Label:       in.GetLabel(),
		Secret:      in.GetSecret(),
	}, s.clock, s.idGenerator)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Encrypt before persistence so storage only receives ciphertext. The API
	// never returns plaintext secrets after this boundary.
	sealedSecret, err := s.sealer.Seal(created.Secret)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "seal credential secret: %v", err)
	}

	record := storage.CredentialRecord{
		ID:               created.ID,
		OwnerUserID:      created.OwnerUserID,
		Provider:         string(created.Provider),
		Label:            created.Label,
		SecretCiphertext: sealedSecret,
		Status:           string(created.Status),
		CreatedAt:        created.CreatedAt,
		UpdatedAt:        created.UpdatedAt,
	}
	if err := s.credentialStore.PutCredential(ctx, record); err != nil {
		return nil, status.Errorf(codes.Internal, "put credential: %v", err)
	}

	return &aiv1.CreateCredentialResponse{
		Credential: credentialToProto(record),
	}, nil
}

// ListCredentials returns a page of credentials owned by the caller.
func (s *Service) ListCredentials(ctx context.Context, in *aiv1.ListCredentialsRequest) (*aiv1.ListCredentialsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list credentials request is required")
	}
	if s.credentialStore == nil {
		return nil, status.Error(codes.Internal, "credential store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	page, err := s.credentialStore.ListCredentialsByOwner(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list credentials: %v", err)
	}

	resp := &aiv1.ListCredentialsResponse{
		NextPageToken: page.NextPageToken,
		Credentials:   make([]*aiv1.Credential, 0, len(page.Credentials)),
	}
	for _, rec := range page.Credentials {
		resp.Credentials = append(resp.Credentials, credentialToProto(rec))
	}
	return resp, nil
}

// RevokeCredential revokes one credential owned by the caller.
func (s *Service) RevokeCredential(ctx context.Context, in *aiv1.RevokeCredentialRequest) (*aiv1.RevokeCredentialResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "revoke credential request is required")
	}
	if s.credentialStore == nil {
		return nil, status.Error(codes.Internal, "credential store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	credentialID := strings.TrimSpace(in.GetCredentialId())
	if credentialID == "" {
		return nil, status.Error(codes.InvalidArgument, "credential_id is required")
	}

	revokedAt := s.clock().UTC()
	if err := s.credentialStore.RevokeCredential(ctx, userID, credentialID, revokedAt); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "credential not found")
		}
		return nil, status.Errorf(codes.Internal, "revoke credential: %v", err)
	}

	record, err := s.credentialStore.GetCredential(ctx, credentialID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "credential not found")
		}
		return nil, status.Errorf(codes.Internal, "get credential: %v", err)
	}
	if strings.TrimSpace(record.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "credential not found")
	}

	return &aiv1.RevokeCredentialResponse{
		Credential: credentialToProto(record),
	}, nil
}

// CreateAgent creates a user-owned AI agent profile.
func (s *Service) CreateAgent(ctx context.Context, in *aiv1.CreateAgentRequest) (*aiv1.CreateAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create agent request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	provider, err := agentProviderFromProto(in.GetProvider())
	if err != nil {
		return nil, err
	}

	createInput, err := agent.NormalizeCreateInput(agent.CreateInput{
		OwnerUserID:     userID,
		Name:            in.GetName(),
		Provider:        provider,
		Model:           in.GetModel(),
		CredentialID:    in.GetCredentialId(),
		ProviderGrantID: in.GetProviderGrantId(),
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.validateAgentAuthReferenceForProvider(ctx, userID, string(provider), createInput.CredentialID, createInput.ProviderGrantID); err != nil {
		return nil, err
	}

	created, err := agent.Create(createInput, s.clock, s.idGenerator)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	record := storage.AgentRecord{
		ID:              created.ID,
		OwnerUserID:     created.OwnerUserID,
		Name:            created.Name,
		Provider:        string(created.Provider),
		Model:           created.Model,
		CredentialID:    created.CredentialID,
		ProviderGrantID: created.ProviderGrantID,
		Status:          string(created.Status),
		CreatedAt:       created.CreatedAt,
		UpdatedAt:       created.UpdatedAt,
	}
	if err := s.agentStore.PutAgent(ctx, record); err != nil {
		return nil, status.Errorf(codes.Internal, "put agent: %v", err)
	}

	return &aiv1.CreateAgentResponse{Agent: agentToProto(record)}, nil
}

// ListAgents returns a page of agents owned by the caller.
func (s *Service) ListAgents(ctx context.Context, in *aiv1.ListAgentsRequest) (*aiv1.ListAgentsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list agents request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	page, err := s.agentStore.ListAgentsByOwner(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list agents: %v", err)
	}

	resp := &aiv1.ListAgentsResponse{
		NextPageToken: page.NextPageToken,
		Agents:        make([]*aiv1.Agent, 0, len(page.Agents)),
	}
	for _, rec := range page.Agents {
		resp.Agents = append(resp.Agents, agentToProto(rec))
	}
	return resp, nil
}

// ListAccessibleAgents returns a page of agents the caller can invoke, combining
// owned agents with approved shared invoke access.
func (s *Service) ListAccessibleAgents(ctx context.Context, in *aiv1.ListAccessibleAgentsRequest) (*aiv1.ListAccessibleAgentsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list accessible agents request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	records, err := s.collectAccessibleAgents(ctx, userID)
	if err != nil {
		return nil, err
	}
	sort.Slice(records, func(i int, j int) bool {
		return records[i].ID < records[j].ID
	})

	pageSize := clampPageSize(in.GetPageSize())
	pageToken := strings.TrimSpace(in.GetPageToken())
	start := findPageStartByID(records, pageToken)
	end := start + pageSize
	nextPageToken := ""
	if end < len(records) {
		nextPageToken = records[end-1].ID
	} else {
		end = len(records)
	}

	resp := &aiv1.ListAccessibleAgentsResponse{
		NextPageToken: nextPageToken,
		Agents:        make([]*aiv1.Agent, 0, end-start),
	}
	for _, rec := range records[start:end] {
		resp.Agents = append(resp.Agents, agentToProto(rec))
	}
	return resp, nil
}

// GetAccessibleAgent returns one agent by ID when the caller can invoke it.
func (s *Service) GetAccessibleAgent(ctx context.Context, in *aiv1.GetAccessibleAgentRequest) (*aiv1.GetAccessibleAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get accessible agent request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	agentID := strings.TrimSpace(in.GetAgentId())
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}

	agentRecord, err := s.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "agent not found")
		}
		return nil, status.Errorf(codes.Internal, "get agent: %v", err)
	}

	// Authorization is intentionally shared with invoke checks so lookup and
	// runtime execution enforce one access policy source.
	authorized, _, _, err := s.isAuthorizedToInvokeAgent(ctx, userID, agentRecord)
	if err != nil {
		return nil, err
	}
	if !authorized {
		// Mask inaccessible resources as not found to avoid tenant probing.
		return nil, status.Error(codes.NotFound, "agent not found")
	}
	return &aiv1.GetAccessibleAgentResponse{Agent: agentToProto(agentRecord)}, nil
}

// UpdateAgent updates mutable fields on one user-owned agent.
func (s *Service) UpdateAgent(ctx context.Context, in *aiv1.UpdateAgentRequest) (*aiv1.UpdateAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "update agent request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	agentID := strings.TrimSpace(in.GetAgentId())
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}

	existing, err := s.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "agent not found")
		}
		return nil, status.Errorf(codes.Internal, "get agent: %v", err)
	}
	if strings.TrimSpace(existing.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "agent not found")
	}

	name := firstNonEmpty(strings.TrimSpace(in.GetName()), existing.Name)
	model := firstNonEmpty(strings.TrimSpace(in.GetModel()), existing.Model)
	credentialID := strings.TrimSpace(existing.CredentialID)
	providerGrantID := strings.TrimSpace(existing.ProviderGrantID)
	requestCredentialID := strings.TrimSpace(in.GetCredentialId())
	requestProviderGrantID := strings.TrimSpace(in.GetProviderGrantId())
	if requestCredentialID != "" || requestProviderGrantID != "" {
		credentialID = requestCredentialID
		providerGrantID = requestProviderGrantID
	}
	normalized, err := agent.NormalizeUpdateInput(agent.UpdateInput{
		ID:              existing.ID,
		OwnerUserID:     existing.OwnerUserID,
		Name:            name,
		Model:           model,
		CredentialID:    credentialID,
		ProviderGrantID: providerGrantID,
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.validateAgentAuthReferenceForProvider(ctx, userID, existing.Provider, normalized.CredentialID, normalized.ProviderGrantID); err != nil {
		return nil, err
	}

	record := existing
	record.Name = normalized.Name
	record.Model = normalized.Model
	record.CredentialID = normalized.CredentialID
	record.ProviderGrantID = normalized.ProviderGrantID
	record.UpdatedAt = s.clock().UTC()
	if err := s.agentStore.PutAgent(ctx, record); err != nil {
		return nil, status.Errorf(codes.Internal, "put agent: %v", err)
	}

	return &aiv1.UpdateAgentResponse{Agent: agentToProto(record)}, nil
}

// DeleteAgent deletes one user-owned agent profile.
func (s *Service) DeleteAgent(ctx context.Context, in *aiv1.DeleteAgentRequest) (*aiv1.DeleteAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "delete agent request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	agentID := strings.TrimSpace(in.GetAgentId())
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}

	if err := s.agentStore.DeleteAgent(ctx, userID, agentID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "agent not found")
		}
		return nil, status.Errorf(codes.Internal, "delete agent: %v", err)
	}
	return &aiv1.DeleteAgentResponse{}, nil
}

// CreateAccessRequest stores a requester-owned pending access request for an agent.
func (s *Service) CreateAccessRequest(ctx context.Context, in *aiv1.CreateAccessRequestRequest) (*aiv1.CreateAccessRequestResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create access request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}
	if s.accessRequestStore == nil {
		return nil, status.Error(codes.Internal, "access request store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	agentID := strings.TrimSpace(in.GetAgentId())
	agentRecord, err := s.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			// Return generic unavailability so callers cannot infer resource ownership.
			return nil, status.Error(codes.FailedPrecondition, "agent is unavailable")
		}
		return nil, status.Errorf(codes.Internal, "get agent: %v", err)
	}
	if !strings.EqualFold(strings.TrimSpace(agentRecord.Status), "active") {
		return nil, status.Error(codes.FailedPrecondition, "agent is unavailable")
	}

	createInput, err := accessrequest.NormalizeCreateInput(accessrequest.CreateInput{
		RequesterUserID: userID,
		OwnerUserID:     agentRecord.OwnerUserID,
		AgentID:         agentRecord.ID,
		Scope:           accessrequest.Scope(in.GetScope()),
		RequestNote:     in.GetRequestNote(),
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	created, err := accessrequest.Create(createInput, s.clock, s.idGenerator)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	record := storage.AccessRequestRecord{
		ID:              created.ID,
		RequesterUserID: created.RequesterUserID,
		OwnerUserID:     created.OwnerUserID,
		AgentID:         created.AgentID,
		Scope:           string(created.Scope),
		RequestNote:     created.RequestNote,
		Status:          string(created.Status),
		ReviewerUserID:  created.ReviewerUserID,
		ReviewNote:      created.ReviewNote,
		CreatedAt:       created.CreatedAt,
		UpdatedAt:       created.UpdatedAt,
		ReviewedAt:      created.ReviewedAt,
	}
	if err := s.accessRequestStore.PutAccessRequest(ctx, record); err != nil {
		return nil, status.Errorf(codes.Internal, "put access request: %v", err)
	}
	if err := s.putAuditEvent(ctx, storage.AuditEventRecord{
		EventName:       "access_request.created",
		ActorUserID:     userID,
		OwnerUserID:     record.OwnerUserID,
		RequesterUserID: record.RequesterUserID,
		AgentID:         record.AgentID,
		AccessRequestID: record.ID,
		Outcome:         record.Status,
		CreatedAt:       record.CreatedAt,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "put audit event: %v", err)
	}

	return &aiv1.CreateAccessRequestResponse{AccessRequest: accessRequestToProto(record)}, nil
}

// ListAccessRequests returns one role-scoped page of access requests.
func (s *Service) ListAccessRequests(ctx context.Context, in *aiv1.ListAccessRequestsRequest) (*aiv1.ListAccessRequestsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list access requests request is required")
	}
	if s.accessRequestStore == nil {
		return nil, status.Error(codes.Internal, "access request store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	var (
		page storage.AccessRequestPage
		err  error
	)
	switch in.GetRole() {
	case aiv1.AccessRequestRole_ACCESS_REQUEST_ROLE_REQUESTER:
		page, err = s.accessRequestStore.ListAccessRequestsByRequester(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken())
	case aiv1.AccessRequestRole_ACCESS_REQUEST_ROLE_OWNER:
		page, err = s.accessRequestStore.ListAccessRequestsByOwner(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken())
	default:
		return nil, status.Error(codes.InvalidArgument, "role is required")
	}
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list access requests: %v", err)
	}

	resp := &aiv1.ListAccessRequestsResponse{
		NextPageToken:  page.NextPageToken,
		AccessRequests: make([]*aiv1.AccessRequest, 0, len(page.AccessRequests)),
	}
	for _, record := range page.AccessRequests {
		resp.AccessRequests = append(resp.AccessRequests, accessRequestToProto(record))
	}
	return resp, nil
}

// ListAuditEvents returns one owner-scoped page of AI audit events.
func (s *Service) ListAuditEvents(ctx context.Context, in *aiv1.ListAuditEventsRequest) (*aiv1.ListAuditEventsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list audit events request is required")
	}
	if s.auditEventStore == nil {
		return nil, status.Error(codes.Internal, "audit event store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	// Filters are caller-supplied and only narrow rows within this authenticated
	// owner scope. Ownership comes from trusted auth metadata, never from input.
	filter, err := listAuditEventFilterFromRequest(in)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	page, err := s.auditEventStore.ListAuditEventsByOwner(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken(), filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list audit events: %v", err)
	}
	resp := &aiv1.ListAuditEventsResponse{
		NextPageToken: page.NextPageToken,
		AuditEvents:   make([]*aiv1.AuditEvent, 0, len(page.AuditEvents)),
	}
	for _, record := range page.AuditEvents {
		resp.AuditEvents = append(resp.AuditEvents, auditEventToProto(record))
	}
	return resp, nil
}

// ReviewAccessRequest applies one owner decision to a pending access request.
func (s *Service) ReviewAccessRequest(ctx context.Context, in *aiv1.ReviewAccessRequestRequest) (*aiv1.ReviewAccessRequestResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "review access request is required")
	}
	if s.accessRequestStore == nil {
		return nil, status.Error(codes.Internal, "access request store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	accessRequestID := strings.TrimSpace(in.GetAccessRequestId())
	if accessRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "access_request_id is required")
	}

	existing, err := s.accessRequestStore.GetAccessRequest(ctx, accessRequestID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		return nil, status.Errorf(codes.Internal, "get access request: %v", err)
	}
	if strings.TrimSpace(existing.OwnerUserID) != userID {
		// Hide unauthorized resources to avoid cross-tenant enumeration.
		return nil, status.Error(codes.NotFound, "access request not found")
	}

	decision, err := accessRequestDecisionFromProto(in.GetDecision())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	updatedDomain, err := accessrequest.Review(accessrequest.AccessRequest{
		ID:              existing.ID,
		RequesterUserID: existing.RequesterUserID,
		OwnerUserID:     existing.OwnerUserID,
		AgentID:         existing.AgentID,
		Scope:           accessrequest.Scope(existing.Scope),
		RequestNote:     existing.RequestNote,
		Status:          accessrequest.Status(existing.Status),
		ReviewerUserID:  existing.ReviewerUserID,
		ReviewNote:      existing.ReviewNote,
		CreatedAt:       existing.CreatedAt,
		UpdatedAt:       existing.UpdatedAt,
		ReviewedAt:      existing.ReviewedAt,
	}, accessrequest.ReviewInput{
		ID:             existing.ID,
		ReviewerUserID: userID,
		Decision:       decision,
		ReviewNote:     in.GetReviewNote(),
	}, s.clock)
	if err != nil {
		if errors.Is(err, accessrequest.ErrNotPending) {
			return nil, status.Error(codes.FailedPrecondition, "access request is already reviewed")
		}
		if errors.Is(err, accessrequest.ErrReviewerNotOwner) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if updatedDomain.ReviewedAt == nil {
		return nil, status.Error(codes.Internal, "review timestamp is unavailable")
	}
	if err := s.accessRequestStore.ReviewAccessRequest(
		ctx,
		userID,
		existing.ID,
		string(updatedDomain.Status),
		updatedDomain.ReviewerUserID,
		updatedDomain.ReviewNote,
		*updatedDomain.ReviewedAt,
	); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		if errors.Is(err, storage.ErrConflict) {
			return nil, status.Error(codes.FailedPrecondition, "access request is already reviewed")
		}
		return nil, status.Errorf(codes.Internal, "review access request: %v", err)
	}

	updatedRecord, err := s.accessRequestStore.GetAccessRequest(ctx, existing.ID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		return nil, status.Errorf(codes.Internal, "get access request: %v", err)
	}
	if err := s.putAuditEvent(ctx, storage.AuditEventRecord{
		EventName:       "access_request.reviewed",
		ActorUserID:     userID,
		OwnerUserID:     updatedRecord.OwnerUserID,
		RequesterUserID: updatedRecord.RequesterUserID,
		AgentID:         updatedRecord.AgentID,
		AccessRequestID: updatedRecord.ID,
		Outcome:         updatedRecord.Status,
		CreatedAt:       updatedRecord.UpdatedAt,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "put audit event: %v", err)
	}
	return &aiv1.ReviewAccessRequestResponse{AccessRequest: accessRequestToProto(updatedRecord)}, nil
}

// RevokeAccessRequest removes delegated access for one approved request.
func (s *Service) RevokeAccessRequest(ctx context.Context, in *aiv1.RevokeAccessRequestRequest) (*aiv1.RevokeAccessRequestResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "revoke access request is required")
	}
	if s.accessRequestStore == nil {
		return nil, status.Error(codes.Internal, "access request store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	accessRequestID := strings.TrimSpace(in.GetAccessRequestId())
	if accessRequestID == "" {
		return nil, status.Error(codes.InvalidArgument, "access_request_id is required")
	}

	existing, err := s.accessRequestStore.GetAccessRequest(ctx, accessRequestID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		return nil, status.Errorf(codes.Internal, "get access request: %v", err)
	}
	if strings.TrimSpace(existing.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "access request not found")
	}

	updatedDomain, err := accessrequest.Revoke(accessrequest.AccessRequest{
		ID:              existing.ID,
		RequesterUserID: existing.RequesterUserID,
		OwnerUserID:     existing.OwnerUserID,
		AgentID:         existing.AgentID,
		Scope:           accessrequest.Scope(existing.Scope),
		RequestNote:     existing.RequestNote,
		Status:          accessrequest.Status(existing.Status),
		ReviewerUserID:  existing.ReviewerUserID,
		ReviewNote:      existing.ReviewNote,
		CreatedAt:       existing.CreatedAt,
		UpdatedAt:       existing.UpdatedAt,
		ReviewedAt:      existing.ReviewedAt,
	}, accessrequest.RevokeInput{
		ID:            existing.ID,
		RevokerUserID: userID,
		RevokeNote:    in.GetRevokeNote(),
	}, s.clock)
	if err != nil {
		if errors.Is(err, accessrequest.ErrNotApproved) {
			return nil, status.Error(codes.FailedPrecondition, "access request is not approved")
		}
		if errors.Is(err, accessrequest.ErrReviewerNotOwner) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err := s.accessRequestStore.RevokeAccessRequest(
		ctx,
		userID,
		existing.ID,
		string(updatedDomain.Status),
		updatedDomain.ReviewerUserID,
		updatedDomain.ReviewNote,
		updatedDomain.UpdatedAt,
	); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		if errors.Is(err, storage.ErrConflict) {
			return nil, status.Error(codes.FailedPrecondition, "access request is not approved")
		}
		return nil, status.Errorf(codes.Internal, "revoke access request: %v", err)
	}

	updatedRecord, err := s.accessRequestStore.GetAccessRequest(ctx, existing.ID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "access request not found")
		}
		return nil, status.Errorf(codes.Internal, "get access request: %v", err)
	}
	if err := s.putAuditEvent(ctx, storage.AuditEventRecord{
		EventName:       "access_request.revoked",
		ActorUserID:     userID,
		OwnerUserID:     updatedRecord.OwnerUserID,
		RequesterUserID: updatedRecord.RequesterUserID,
		AgentID:         updatedRecord.AgentID,
		AccessRequestID: updatedRecord.ID,
		Outcome:         updatedRecord.Status,
		CreatedAt:       updatedRecord.UpdatedAt,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "put audit event: %v", err)
	}
	return &aiv1.RevokeAccessRequestResponse{AccessRequest: accessRequestToProto(updatedRecord)}, nil
}

// InvokeAgent executes one provider call using an owned active agent auth reference.
func (s *Service) InvokeAgent(ctx context.Context, in *aiv1.InvokeAgentRequest) (*aiv1.InvokeAgentResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "invoke agent request is required")
	}
	if s.agentStore == nil {
		return nil, status.Error(codes.Internal, "agent store is not configured")
	}
	if s.sealer == nil {
		return nil, status.Error(codes.Internal, "secret sealer is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}
	agentID := strings.TrimSpace(in.GetAgentId())
	if agentID == "" {
		return nil, status.Error(codes.InvalidArgument, "agent_id is required")
	}
	input := strings.TrimSpace(in.GetInput())
	if input == "" {
		return nil, status.Error(codes.InvalidArgument, "input is required")
	}

	agentRecord, err := s.agentStore.GetAgent(ctx, agentID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "agent not found")
		}
		return nil, status.Errorf(codes.Internal, "get agent: %v", err)
	}

	authorized, sharedAccess, accessRequestID, err := s.isAuthorizedToInvokeAgent(ctx, userID, agentRecord)
	if err != nil {
		return nil, err
	}
	if !authorized {
		return nil, status.Error(codes.NotFound, "agent not found")
	}

	provider := providergrant.Provider(strings.ToLower(strings.TrimSpace(agentRecord.Provider)))
	adapter, ok := s.providerInvocationAdapters[provider]
	if !ok || adapter == nil {
		return nil, status.Error(codes.FailedPrecondition, "provider invocation adapter is unavailable")
	}

	invokeToken, err := s.resolveAgentInvokeToken(ctx, strings.TrimSpace(agentRecord.OwnerUserID), agentRecord)
	if err != nil {
		return nil, err
	}
	result, err := adapter.Invoke(ctx, ProviderInvokeInput{
		Model:            agentRecord.Model,
		Input:            input,
		CredentialSecret: invokeToken,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "invoke provider: %v", err)
	}
	outputText := strings.TrimSpace(result.OutputText)
	if outputText == "" {
		return nil, status.Error(codes.Internal, "provider returned empty output")
	}
	if sharedAccess {
		if err := s.putAuditEvent(ctx, storage.AuditEventRecord{
			EventName:       "agent.invoke.shared",
			ActorUserID:     userID,
			OwnerUserID:     strings.TrimSpace(agentRecord.OwnerUserID),
			RequesterUserID: userID,
			AgentID:         strings.TrimSpace(agentRecord.ID),
			AccessRequestID: accessRequestID,
			Outcome:         "success",
			CreatedAt:       s.clock().UTC(),
		}); err != nil {
			return nil, status.Errorf(codes.Internal, "put audit event: %v", err)
		}
	}
	return &aiv1.InvokeAgentResponse{
		OutputText: outputText,
		Provider:   providerToProto(agentRecord.Provider),
		Model:      agentRecord.Model,
	}, nil
}

// isAuthorizedToInvokeAgent returns true when caller is owner or has an
// approved invoke access request for the target agent. Unauthorized callers
// receive not-found responses at the handler boundary to avoid tenant probing.
func (s *Service) isAuthorizedToInvokeAgent(ctx context.Context, callerUserID string, agentRecord storage.AgentRecord) (bool, bool, string, error) {
	ownerUserID := strings.TrimSpace(agentRecord.OwnerUserID)
	if ownerUserID == "" {
		return false, false, "", status.Error(codes.FailedPrecondition, "agent owner is unavailable")
	}
	callerUserID = strings.TrimSpace(callerUserID)
	if callerUserID == "" {
		return false, false, "", nil
	}
	if callerUserID == ownerUserID {
		return true, false, "", nil
	}
	if s.accessRequestStore == nil {
		return false, false, "", nil
	}
	rec, err := s.accessRequestStore.GetApprovedInvokeAccessByRequesterForAgent(
		ctx,
		callerUserID,
		ownerUserID,
		strings.TrimSpace(agentRecord.ID),
	)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return false, false, "", nil
		}
		return false, false, "", status.Errorf(codes.Internal, "get approved invoke access request: %v", err)
	}
	return true, true, strings.TrimSpace(rec.ID), nil
}

// putAuditEvent records one append-only security event for privileged access
// workflows. Events intentionally omit secret material and include only identity
// and lifecycle metadata.
func (s *Service) putAuditEvent(ctx context.Context, record storage.AuditEventRecord) error {
	if s.auditEventStore == nil {
		return fmt.Errorf("audit event store is not configured")
	}
	record.CreatedAt = record.CreatedAt.UTC()
	return s.auditEventStore.PutAuditEvent(ctx, record)
}

func (s *Service) collectAccessibleAgents(ctx context.Context, userID string) ([]storage.AgentRecord, error) {
	accessibleByID := make(map[string]storage.AgentRecord)
	pageToken := ""
	for {
		page, err := s.agentStore.ListAgentsByOwner(ctx, userID, maxPageSize, pageToken)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list agents: %v", err)
		}
		for _, rec := range page.Agents {
			if strings.TrimSpace(rec.ID) == "" {
				continue
			}
			accessibleByID[rec.ID] = rec
		}

		nextPageToken := strings.TrimSpace(page.NextPageToken)
		if nextPageToken == "" || nextPageToken == pageToken {
			break
		}
		pageToken = nextPageToken
	}

	if s.accessRequestStore == nil {
		return mapValues(accessibleByID), nil
	}

	pageToken = ""
	for {
		page, err := s.accessRequestStore.ListApprovedInvokeAccessRequestsByRequester(ctx, userID, maxPageSize, pageToken)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list approved invoke access requests: %v", err)
		}
		for _, rec := range page.AccessRequests {
			agentID := strings.TrimSpace(rec.AgentID)
			if agentID == "" {
				continue
			}
			if _, exists := accessibleByID[agentID]; exists {
				continue
			}
			agentRecord, err := s.agentStore.GetAgent(ctx, agentID)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					// Access records can outlive target agents. Ignore stale entries.
					continue
				}
				return nil, status.Errorf(codes.Internal, "get shared agent: %v", err)
			}
			// Require owner match to avoid stale or tampered access rows granting a
			// different owner's agent.
			if strings.TrimSpace(agentRecord.OwnerUserID) != strings.TrimSpace(rec.OwnerUserID) {
				continue
			}
			accessibleByID[agentID] = agentRecord
		}

		nextPageToken := strings.TrimSpace(page.NextPageToken)
		if nextPageToken == "" || nextPageToken == pageToken {
			break
		}
		pageToken = nextPageToken
	}
	return mapValues(accessibleByID), nil
}

func findPageStartByID(records []storage.AgentRecord, pageToken string) int {
	pageToken = strings.TrimSpace(pageToken)
	if pageToken == "" {
		return 0
	}
	for idx, rec := range records {
		if strings.Compare(strings.TrimSpace(rec.ID), pageToken) > 0 {
			return idx
		}
	}
	return len(records)
}

func mapValues(values map[string]storage.AgentRecord) []storage.AgentRecord {
	if len(values) == 0 {
		return []storage.AgentRecord{}
	}
	items := make([]storage.AgentRecord, 0, len(values))
	for _, rec := range values {
		items = append(items, rec)
	}
	return items
}

// StartProviderConnect starts a provider OAuth grant handshake.
func (s *Service) StartProviderConnect(ctx context.Context, in *aiv1.StartProviderConnectRequest) (*aiv1.StartProviderConnectResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "start provider connect request is required")
	}
	if s.providerGrantStore == nil {
		return nil, status.Error(codes.Internal, "provider grant store is not configured")
	}
	if s.connectSessionStore == nil {
		return nil, status.Error(codes.Internal, "provider connect session store is not configured")
	}
	if s.sealer == nil {
		return nil, status.Error(codes.Internal, "secret sealer is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	provider, err := providerGrantProviderFromProto(in.GetProvider())
	if err != nil {
		return nil, err
	}

	sessionID, err := s.idGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate connect session id: %v", err)
	}
	state, err := s.idGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate connect state: %v", err)
	}
	codeVerifierGenerator := s.codeVerifierGenerator
	if codeVerifierGenerator == nil {
		codeVerifierGenerator = generatePKCECodeVerifier
	}
	codeVerifier, err := codeVerifierGenerator()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "generate code verifier: %v", err)
	}
	codeVerifier = strings.TrimSpace(codeVerifier)
	if !isValidPKCECodeVerifier(codeVerifier) {
		return nil, status.Error(codes.Internal, "generate code verifier: value is invalid")
	}
	codeChallenge := pkceCodeChallengeS256(codeVerifier)
	codeVerifierCiphertext, err := s.sealer.Seal(codeVerifier)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "seal code verifier: %v", err)
	}

	now := s.clock().UTC()
	expiresAt := now.Add(10 * time.Minute)
	record := storage.ProviderConnectSessionRecord{
		ID:                     sessionID,
		OwnerUserID:            userID,
		Provider:               string(provider),
		Status:                 "pending",
		RequestedScopes:        normalizeScopes(in.GetRequestedScopes()),
		StateHash:              hashState(state),
		CodeVerifierCiphertext: codeVerifierCiphertext,
		CreatedAt:              now,
		UpdatedAt:              now,
		ExpiresAt:              expiresAt,
	}
	if err := s.connectSessionStore.PutProviderConnectSession(ctx, record); err != nil {
		return nil, status.Errorf(codes.Internal, "put provider connect session: %v", err)
	}

	adapter, ok := s.providerOAuthAdapters[provider]
	if !ok || adapter == nil {
		return nil, status.Error(codes.FailedPrecondition, "provider oauth adapter is unavailable")
	}
	authorizationURL, err := adapter.BuildAuthorizationURL(ProviderAuthorizationURLInput{
		State:           state,
		CodeChallenge:   codeChallenge,
		RequestedScopes: record.RequestedScopes,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "build authorization url: %v", err)
	}
	return &aiv1.StartProviderConnectResponse{
		ConnectSessionId: sessionID,
		State:            state,
		AuthorizationUrl: authorizationURL,
		ExpiresAt:        timestamppb.New(expiresAt),
	}, nil
}

// FinishProviderConnect completes a provider OAuth grant handshake.
func (s *Service) FinishProviderConnect(ctx context.Context, in *aiv1.FinishProviderConnectRequest) (*aiv1.FinishProviderConnectResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "finish provider connect request is required")
	}
	if s.providerGrantStore == nil {
		return nil, status.Error(codes.Internal, "provider grant store is not configured")
	}
	if s.connectSessionStore == nil {
		return nil, status.Error(codes.Internal, "provider connect session store is not configured")
	}
	if s.sealer == nil {
		return nil, status.Error(codes.Internal, "secret sealer is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	sessionID := strings.TrimSpace(in.GetConnectSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "connect_session_id is required")
	}
	state := strings.TrimSpace(in.GetState())
	if state == "" {
		return nil, status.Error(codes.InvalidArgument, "state is required")
	}
	authorizationCode := strings.TrimSpace(in.GetAuthorizationCode())
	if authorizationCode == "" {
		return nil, status.Error(codes.InvalidArgument, "authorization_code is required")
	}

	session, err := s.connectSessionStore.GetProviderConnectSession(ctx, sessionID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "connect session not found")
		}
		return nil, status.Errorf(codes.Internal, "get connect session: %v", err)
	}
	if strings.TrimSpace(session.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "connect session not found")
	}
	if !strings.EqualFold(strings.TrimSpace(session.Status), "pending") {
		return nil, status.Error(codes.FailedPrecondition, "connect session is no longer pending")
	}
	if s.clock().UTC().After(session.ExpiresAt) {
		return nil, status.Error(codes.FailedPrecondition, "connect session expired")
	}
	// State check is the CSRF boundary for the connect handshake.
	if hashState(state) != strings.TrimSpace(session.StateHash) {
		return nil, status.Error(codes.FailedPrecondition, "state mismatch")
	}
	completedAt := s.clock().UTC()
	if err := s.connectSessionStore.CompleteProviderConnectSession(ctx, userID, sessionID, completedAt); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "connect session not found")
		}
		return nil, status.Errorf(codes.Internal, "complete connect session: %v", err)
	}

	provider := providergrant.Provider(strings.ToLower(strings.TrimSpace(session.Provider)))
	if provider != providergrant.ProviderOpenAI {
		return nil, status.Error(codes.FailedPrecondition, "provider is unavailable")
	}
	adapter, ok := s.providerOAuthAdapters[provider]
	if !ok || adapter == nil {
		return nil, status.Error(codes.FailedPrecondition, "provider oauth adapter is unavailable")
	}
	codeVerifier, err := s.sealer.Open(session.CodeVerifierCiphertext)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "open code verifier: %v", err)
	}
	exchanged, err := adapter.ExchangeAuthorizationCode(ctx, ProviderAuthorizationCodeInput{
		AuthorizationCode: authorizationCode,
		CodeVerifier:      codeVerifier,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "exchange authorization code: %v", err)
	}
	tokenPlaintext := strings.TrimSpace(exchanged.TokenPlaintext)
	if tokenPlaintext == "" {
		return nil, status.Error(codes.Internal, "provider returned empty token payload")
	}
	tokenCiphertext, err := s.sealer.Seal(tokenPlaintext)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "seal provider token: %v", err)
	}

	created, err := providergrant.Create(providergrant.CreateInput{
		OwnerUserID:      userID,
		Provider:         provider,
		GrantedScopes:    session.RequestedScopes,
		TokenCiphertext:  tokenCiphertext,
		RefreshSupported: exchanged.RefreshSupported,
		ExpiresAt:        exchanged.ExpiresAt,
	}, s.clock, s.idGenerator)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	record := storage.ProviderGrantRecord{
		ID:               created.ID,
		OwnerUserID:      created.OwnerUserID,
		Provider:         string(created.Provider),
		GrantedScopes:    created.GrantedScopes,
		TokenCiphertext:  created.TokenCiphertext,
		RefreshSupported: created.RefreshSupported,
		Status:           string(created.Status),
		LastRefreshError: strings.TrimSpace(exchanged.LastRefreshError),
		CreatedAt:        created.CreatedAt,
		UpdatedAt:        created.UpdatedAt,
		RevokedAt:        created.RevokedAt,
		ExpiresAt:        created.ExpiresAt,
		LastRefreshedAt:  created.RefreshedAt,
	}
	if err := s.providerGrantStore.PutProviderGrant(ctx, record); err != nil {
		return nil, status.Errorf(codes.Internal, "put provider grant: %v", err)
	}
	return &aiv1.FinishProviderConnectResponse{ProviderGrant: providerGrantToProto(record)}, nil
}

// ListProviderGrants returns a page of provider grants owned by the caller.
func (s *Service) ListProviderGrants(ctx context.Context, in *aiv1.ListProviderGrantsRequest) (*aiv1.ListProviderGrantsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list provider grants request is required")
	}
	if s.providerGrantStore == nil {
		return nil, status.Error(codes.Internal, "provider grant store is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	// Filters are caller-supplied and can only narrow rows inside the
	// authenticated owner scope derived from trusted auth metadata.
	filter, err := providerGrantFilterFromRequest(in)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	page, err := s.providerGrantStore.ListProviderGrantsByOwner(ctx, userID, clampPageSize(in.GetPageSize()), in.GetPageToken(), filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list provider grants: %v", err)
	}

	resp := &aiv1.ListProviderGrantsResponse{
		NextPageToken:  page.NextPageToken,
		ProviderGrants: make([]*aiv1.ProviderGrant, 0, len(page.ProviderGrants)),
	}
	for _, rec := range page.ProviderGrants {
		resp.ProviderGrants = append(resp.ProviderGrants, providerGrantToProto(rec))
	}
	return resp, nil
}

// RevokeProviderGrant revokes one provider grant owned by the caller.
func (s *Service) RevokeProviderGrant(ctx context.Context, in *aiv1.RevokeProviderGrantRequest) (*aiv1.RevokeProviderGrantResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "revoke provider grant request is required")
	}
	if s.providerGrantStore == nil {
		return nil, status.Error(codes.Internal, "provider grant store is not configured")
	}
	if s.sealer == nil {
		return nil, status.Error(codes.Internal, "secret sealer is not configured")
	}

	userID := userIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.PermissionDenied, "missing user identity")
	}

	providerGrantID := strings.TrimSpace(in.GetProviderGrantId())
	if providerGrantID == "" {
		return nil, status.Error(codes.InvalidArgument, "provider_grant_id is required")
	}

	record, err := s.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "provider grant not found")
		}
		return nil, status.Errorf(codes.Internal, "get provider grant: %v", err)
	}
	if strings.TrimSpace(record.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "provider grant not found")
	}

	provider := providergrant.Provider(strings.ToLower(strings.TrimSpace(record.Provider)))
	if adapter, ok := s.providerOAuthAdapters[provider]; ok && adapter != nil {
		tokenPlaintext, err := s.sealer.Open(record.TokenCiphertext)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "open provider token: %v", err)
		}
		tokenForRevoke, err := revokeTokenFromTokenPayload(tokenPlaintext)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "derive provider revoke token: %v", err)
		}
		if err := adapter.RevokeToken(ctx, ProviderRevokeTokenInput{Token: tokenForRevoke}); err != nil {
			return nil, status.Errorf(codes.Internal, "revoke provider token: %v", err)
		}
	}

	revokedAt := s.clock().UTC()
	if err := s.providerGrantStore.RevokeProviderGrant(ctx, userID, providerGrantID, revokedAt); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "provider grant not found")
		}
		return nil, status.Errorf(codes.Internal, "revoke provider grant: %v", err)
	}

	record, err = s.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "provider grant not found")
		}
		return nil, status.Errorf(codes.Internal, "get provider grant: %v", err)
	}
	if strings.TrimSpace(record.OwnerUserID) != userID {
		return nil, status.Error(codes.NotFound, "provider grant not found")
	}
	return &aiv1.RevokeProviderGrantResponse{ProviderGrant: providerGrantToProto(record)}, nil
}

func (s *Service) refreshProviderGrant(ctx context.Context, ownerUserID string, providerGrantID string) (storage.ProviderGrantRecord, error) {
	if s.providerGrantStore == nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("provider grant store is not configured")
	}
	if s.sealer == nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("secret sealer is not configured")
	}

	ownerUserID = strings.TrimSpace(ownerUserID)
	if ownerUserID == "" {
		return storage.ProviderGrantRecord{}, fmt.Errorf("owner user id is required")
	}
	providerGrantID = strings.TrimSpace(providerGrantID)
	if providerGrantID == "" {
		return storage.ProviderGrantRecord{}, fmt.Errorf("provider grant id is required")
	}

	record, err := s.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("get provider grant: %w", err)
	}
	if strings.TrimSpace(record.OwnerUserID) != ownerUserID {
		return storage.ProviderGrantRecord{}, storage.ErrNotFound
	}
	if !record.RefreshSupported {
		return storage.ProviderGrantRecord{}, fmt.Errorf("provider grant does not support refresh")
	}

	provider := providergrant.Provider(strings.ToLower(strings.TrimSpace(record.Provider)))
	adapter, ok := s.providerOAuthAdapters[provider]
	if !ok || adapter == nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("provider oauth adapter is unavailable")
	}

	tokenPlaintext, err := s.sealer.Open(record.TokenCiphertext)
	if err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("open provider token: %w", err)
	}
	refreshToken, err := refreshTokenFromTokenPayload(tokenPlaintext)
	if err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("extract refresh token: %w", err)
	}

	refreshedAt := s.clock().UTC()
	exchanged, err := adapter.RefreshToken(ctx, ProviderRefreshTokenInput{
		RefreshToken: refreshToken,
	})
	if err != nil {
		if markErr := s.markProviderGrantRefreshFailed(ctx, record, refreshedAt, err); markErr != nil {
			return storage.ProviderGrantRecord{}, markErr
		}
		return storage.ProviderGrantRecord{}, fmt.Errorf("refresh provider token: %w", err)
	}
	newTokenPlaintext := strings.TrimSpace(exchanged.TokenPlaintext)
	if newTokenPlaintext == "" {
		emptyResultErr := fmt.Errorf("provider returned empty token payload")
		if markErr := s.markProviderGrantRefreshFailed(ctx, record, refreshedAt, emptyResultErr); markErr != nil {
			return storage.ProviderGrantRecord{}, markErr
		}
		return storage.ProviderGrantRecord{}, emptyResultErr
	}
	tokenCiphertext, err := s.sealer.Seal(newTokenPlaintext)
	if err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("seal provider token: %w", err)
	}

	// Refresh errors are stored as metadata only; token ciphertext stays sealed.
	if err := s.providerGrantStore.UpdateProviderGrantToken(
		ctx,
		ownerUserID,
		providerGrantID,
		tokenCiphertext,
		refreshedAt,
		exchanged.ExpiresAt,
		string(providergrant.StatusActive),
		strings.TrimSpace(exchanged.LastRefreshError),
	); err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("update provider grant token: %w", err)
	}
	updated, err := s.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		return storage.ProviderGrantRecord{}, fmt.Errorf("get provider grant: %w", err)
	}
	if strings.TrimSpace(updated.OwnerUserID) != ownerUserID {
		return storage.ProviderGrantRecord{}, storage.ErrNotFound
	}
	return updated, nil
}

func (s *Service) markProviderGrantRefreshFailed(ctx context.Context, record storage.ProviderGrantRecord, refreshedAt time.Time, refreshErr error) error {
	message := "provider token refresh failed"
	if refreshErr != nil && strings.TrimSpace(refreshErr.Error()) != "" {
		message = strings.TrimSpace(refreshErr.Error())
	}
	// Keep prior ciphertext on failure; only status/metadata mutates here.
	if err := s.providerGrantStore.UpdateProviderGrantToken(
		ctx,
		record.OwnerUserID,
		record.ID,
		record.TokenCiphertext,
		refreshedAt,
		record.ExpiresAt,
		string(providergrant.StatusRefreshFailed),
		message,
	); err != nil {
		return fmt.Errorf("mark provider grant refresh failed: %w", err)
	}
	return nil
}

// validateAgentAuthReferenceForProvider enforces that agent auth references are
// exclusive and usable for the caller before persistence.
// This prevents cross-tenant bindings and provider-mismatch configurations.
func (s *Service) validateAgentAuthReferenceForProvider(ctx context.Context, ownerUserID string, provider string, credentialID string, providerGrantID string) error {
	credentialID = strings.TrimSpace(credentialID)
	providerGrantID = strings.TrimSpace(providerGrantID)
	hasCredential := credentialID != ""
	hasProviderGrant := providerGrantID != ""
	if hasCredential == hasProviderGrant {
		return status.Error(codes.InvalidArgument, "exactly one agent auth reference is required")
	}

	if hasCredential {
		if s.credentialStore == nil {
			return status.Error(codes.Internal, "credential store is not configured")
		}
		credentialRecord, err := s.credentialStore.GetCredential(ctx, credentialID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return status.Error(codes.FailedPrecondition, "credential is unavailable")
			}
			return status.Errorf(codes.Internal, "get credential: %v", err)
		}
		if !isCredentialActiveForUser(credentialRecord, ownerUserID, provider) {
			return status.Error(codes.FailedPrecondition, "credential must be active and owned by caller")
		}
		return nil
	}

	if s.providerGrantStore == nil {
		return status.Error(codes.Internal, "provider grant store is not configured")
	}
	grantRecord, err := s.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return status.Error(codes.FailedPrecondition, "provider grant is unavailable")
		}
		return status.Errorf(codes.Internal, "get provider grant: %v", err)
	}
	if !isProviderGrantActiveForUser(grantRecord, ownerUserID, provider) {
		return status.Error(codes.FailedPrecondition, "provider grant must be active and owned by caller")
	}
	return nil
}

// resolveAgentInvokeToken resolves the active auth reference and returns a
// short-lived plaintext token for in-memory provider dispatch only.
func (s *Service) resolveAgentInvokeToken(ctx context.Context, ownerUserID string, agentRecord storage.AgentRecord) (string, error) {
	credentialID := strings.TrimSpace(agentRecord.CredentialID)
	providerGrantID := strings.TrimSpace(agentRecord.ProviderGrantID)
	hasCredential := credentialID != ""
	hasProviderGrant := providerGrantID != ""
	if hasCredential == hasProviderGrant {
		return "", status.Error(codes.FailedPrecondition, "agent auth reference is invalid")
	}

	if hasCredential {
		if s.credentialStore == nil {
			return "", status.Error(codes.Internal, "credential store is not configured")
		}
		credentialRecord, err := s.credentialStore.GetCredential(ctx, credentialID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				return "", status.Error(codes.FailedPrecondition, "credential is unavailable")
			}
			return "", status.Errorf(codes.Internal, "get credential: %v", err)
		}
		if !isCredentialActiveForUser(credentialRecord, ownerUserID, agentRecord.Provider) {
			return "", status.Error(codes.FailedPrecondition, "credential must be active and owned by caller")
		}
		// Secrets are decrypted only for in-memory request dispatch and never returned.
		credentialSecret, err := s.sealer.Open(credentialRecord.SecretCiphertext)
		if err != nil {
			return "", status.Errorf(codes.Internal, "open credential secret: %v", err)
		}
		return credentialSecret, nil
	}

	grantRecord, err := s.resolveProviderGrantForInvocation(ctx, ownerUserID, providerGrantID, agentRecord.Provider)
	if err != nil {
		return "", err
	}
	tokenPlaintext, err := s.sealer.Open(grantRecord.TokenCiphertext)
	if err != nil {
		return "", status.Errorf(codes.Internal, "open provider token: %v", err)
	}
	accessToken, err := accessTokenFromTokenPayload(tokenPlaintext)
	if err != nil {
		return "", status.Errorf(codes.FailedPrecondition, "provider token payload is invalid: %v", err)
	}
	return accessToken, nil
}

// resolveProviderGrantForInvocation validates owner/provider/status and refreshes
// grant token material when needed. Ownership/provider mismatches are reported
// as unavailable so callers cannot enumerate foreign grants.
func (s *Service) resolveProviderGrantForInvocation(ctx context.Context, ownerUserID string, providerGrantID string, provider string) (storage.ProviderGrantRecord, error) {
	if s.providerGrantStore == nil {
		return storage.ProviderGrantRecord{}, status.Error(codes.Internal, "provider grant store is not configured")
	}

	record, err := s.providerGrantStore.GetProviderGrant(ctx, providerGrantID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
		}
		return storage.ProviderGrantRecord{}, status.Errorf(codes.Internal, "get provider grant: %v", err)
	}
	if strings.TrimSpace(record.OwnerUserID) != strings.TrimSpace(ownerUserID) {
		return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
	}
	if provider != "" && !strings.EqualFold(strings.TrimSpace(record.Provider), strings.TrimSpace(provider)) {
		return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
	}

	statusValue := strings.ToLower(strings.TrimSpace(record.Status))
	now := s.clock().UTC()
	switch statusValue {
	case "active":
		if isProviderGrantExpired(record, now) && !record.RefreshSupported {
			return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
		}
		if shouldRefreshProviderGrantForInvocation(record, now) {
			refreshed, err := s.refreshProviderGrant(ctx, ownerUserID, providerGrantID)
			if err != nil {
				return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant refresh failed")
			}
			record = refreshed
		}
	case "refresh_failed", "expired":
		if !record.RefreshSupported {
			return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
		}
		refreshed, err := s.refreshProviderGrant(ctx, ownerUserID, providerGrantID)
		if err != nil {
			return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant refresh failed")
		}
		record = refreshed
	default:
		return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
	}
	if !strings.EqualFold(strings.TrimSpace(record.Status), "active") {
		return storage.ProviderGrantRecord{}, status.Error(codes.FailedPrecondition, "provider grant is unavailable")
	}
	return record, nil
}

// shouldRefreshProviderGrantForInvocation applies a small pre-expiry window so
// invocation paths can refresh before provider-side token rejection.
func shouldRefreshProviderGrantForInvocation(record storage.ProviderGrantRecord, now time.Time) bool {
	if !record.RefreshSupported {
		return false
	}
	if record.ExpiresAt == nil {
		return false
	}
	return !record.ExpiresAt.After(now.Add(providerGrantRefreshWindow))
}

func isProviderGrantExpired(record storage.ProviderGrantRecord, now time.Time) bool {
	if record.ExpiresAt == nil {
		return false
	}
	return !record.ExpiresAt.After(now)
}

// userIDFromContext extracts the authenticated user ID from incoming metadata.
// The returned value is only trusted when set by authenticated transport.
func userIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(userIDHeader)
	if len(values) == 0 {
		return ""
	}
	return strings.TrimSpace(values[0])
}

func listAuditEventFilterFromRequest(in *aiv1.ListAuditEventsRequest) (storage.AuditEventFilter, error) {
	filter := storage.AuditEventFilter{
		EventName: strings.TrimSpace(in.GetEventName()),
		AgentID:   strings.TrimSpace(in.GetAgentId()),
	}
	if in.GetCreatedAfter() != nil {
		if err := in.GetCreatedAfter().CheckValid(); err != nil {
			return storage.AuditEventFilter{}, fmt.Errorf("created_after is invalid")
		}
		createdAfter := in.GetCreatedAfter().AsTime().UTC()
		filter.CreatedAfter = &createdAfter
	}
	if in.GetCreatedBefore() != nil {
		if err := in.GetCreatedBefore().CheckValid(); err != nil {
			return storage.AuditEventFilter{}, fmt.Errorf("created_before is invalid")
		}
		createdBefore := in.GetCreatedBefore().AsTime().UTC()
		filter.CreatedBefore = &createdBefore
	}
	if filter.CreatedAfter != nil && filter.CreatedBefore != nil && filter.CreatedAfter.After(*filter.CreatedBefore) {
		return storage.AuditEventFilter{}, fmt.Errorf("created_after must be before or equal to created_before")
	}
	return filter, nil
}

func providerGrantFilterFromRequest(in *aiv1.ListProviderGrantsRequest) (storage.ProviderGrantFilter, error) {
	filter := storage.ProviderGrantFilter{}
	switch in.GetProvider() {
	case aiv1.Provider_PROVIDER_UNSPECIFIED:
	case aiv1.Provider_PROVIDER_OPENAI:
		filter.Provider = "openai"
	default:
		return storage.ProviderGrantFilter{}, fmt.Errorf("provider filter is invalid")
	}

	switch in.GetStatus() {
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_UNSPECIFIED:
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_ACTIVE:
		filter.Status = "active"
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REVOKED:
		filter.Status = "revoked"
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_EXPIRED:
		filter.Status = "expired"
	case aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REFRESH_FAILED:
		filter.Status = "refresh_failed"
	default:
		return storage.ProviderGrantFilter{}, fmt.Errorf("status filter is invalid")
	}
	return filter, nil
}

func credentialProviderFromProto(value aiv1.Provider) (credential.Provider, error) {
	switch value {
	case aiv1.Provider_PROVIDER_OPENAI:
		return credential.ProviderOpenAI, nil
	default:
		return "", status.Error(codes.InvalidArgument, "provider is required")
	}
}

func agentProviderFromProto(value aiv1.Provider) (agent.Provider, error) {
	switch value {
	case aiv1.Provider_PROVIDER_OPENAI:
		return agent.ProviderOpenAI, nil
	default:
		return "", status.Error(codes.InvalidArgument, "provider is required")
	}
}

func providerGrantProviderFromProto(value aiv1.Provider) (providergrant.Provider, error) {
	switch value {
	case aiv1.Provider_PROVIDER_OPENAI:
		return providergrant.ProviderOpenAI, nil
	default:
		return "", status.Error(codes.InvalidArgument, "provider is required")
	}
}

func providerToProto(value string) aiv1.Provider {
	if strings.EqualFold(strings.TrimSpace(value), "openai") {
		return aiv1.Provider_PROVIDER_OPENAI
	}
	return aiv1.Provider_PROVIDER_UNSPECIFIED
}

func credentialStatusToProto(value string) aiv1.CredentialStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "active":
		return aiv1.CredentialStatus_CREDENTIAL_STATUS_ACTIVE
	case "revoked":
		return aiv1.CredentialStatus_CREDENTIAL_STATUS_REVOKED
	default:
		return aiv1.CredentialStatus_CREDENTIAL_STATUS_UNSPECIFIED
	}
}

func agentStatusToProto(value string) aiv1.AgentStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "active":
		return aiv1.AgentStatus_AGENT_STATUS_ACTIVE
	default:
		return aiv1.AgentStatus_AGENT_STATUS_UNSPECIFIED
	}
}

func providerGrantStatusToProto(value string) aiv1.ProviderGrantStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "active":
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_ACTIVE
	case "revoked":
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REVOKED
	case "expired":
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_EXPIRED
	case "refresh_failed":
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_REFRESH_FAILED
	default:
		return aiv1.ProviderGrantStatus_PROVIDER_GRANT_STATUS_UNSPECIFIED
	}
}

func accessRequestStatusToProto(value string) aiv1.AccessRequestStatus {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pending":
		return aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_PENDING
	case "approved":
		return aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_APPROVED
	case "denied":
		return aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_DENIED
	case "revoked":
		return aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_REVOKED
	default:
		return aiv1.AccessRequestStatus_ACCESS_REQUEST_STATUS_UNSPECIFIED
	}
}

func accessRequestDecisionFromProto(value aiv1.AccessRequestDecision) (accessrequest.Decision, error) {
	switch value {
	case aiv1.AccessRequestDecision_ACCESS_REQUEST_DECISION_APPROVE:
		return accessrequest.DecisionApprove, nil
	case aiv1.AccessRequestDecision_ACCESS_REQUEST_DECISION_DENY:
		return accessrequest.DecisionDeny, nil
	default:
		return "", status.Error(codes.InvalidArgument, "decision is required")
	}
}

func credentialToProto(record storage.CredentialRecord) *aiv1.Credential {
	// Intentionally omits SecretCiphertext to avoid exposing encrypted credential
	// material over read APIs.
	proto := &aiv1.Credential{
		Id:          record.ID,
		OwnerUserId: record.OwnerUserID,
		Provider:    providerToProto(record.Provider),
		Label:       record.Label,
		Status:      credentialStatusToProto(record.Status),
		CreatedAt:   timestamppb.New(record.CreatedAt),
		UpdatedAt:   timestamppb.New(record.UpdatedAt),
	}
	if record.RevokedAt != nil {
		proto.RevokedAt = timestamppb.New(*record.RevokedAt)
	}
	return proto
}

func agentToProto(record storage.AgentRecord) *aiv1.Agent {
	return &aiv1.Agent{
		Id:              record.ID,
		OwnerUserId:     record.OwnerUserID,
		Name:            record.Name,
		Provider:        providerToProto(record.Provider),
		Model:           record.Model,
		CredentialId:    record.CredentialID,
		ProviderGrantId: record.ProviderGrantID,
		Status:          agentStatusToProto(record.Status),
		CreatedAt:       timestamppb.New(record.CreatedAt),
		UpdatedAt:       timestamppb.New(record.UpdatedAt),
	}
}

func providerGrantToProto(record storage.ProviderGrantRecord) *aiv1.ProviderGrant {
	proto := &aiv1.ProviderGrant{
		Id:               record.ID,
		OwnerUserId:      record.OwnerUserID,
		Provider:         providerToProto(record.Provider),
		GrantedScopes:    append([]string(nil), record.GrantedScopes...),
		RefreshSupported: record.RefreshSupported,
		Status:           providerGrantStatusToProto(record.Status),
		LastRefreshError: record.LastRefreshError,
		CreatedAt:        timestamppb.New(record.CreatedAt),
		UpdatedAt:        timestamppb.New(record.UpdatedAt),
	}
	if record.RevokedAt != nil {
		proto.RevokedAt = timestamppb.New(*record.RevokedAt)
	}
	if record.ExpiresAt != nil {
		proto.ExpiresAt = timestamppb.New(*record.ExpiresAt)
	}
	if record.LastRefreshedAt != nil {
		proto.LastRefreshedAt = timestamppb.New(*record.LastRefreshedAt)
	}
	return proto
}

func accessRequestToProto(record storage.AccessRequestRecord) *aiv1.AccessRequest {
	proto := &aiv1.AccessRequest{
		Id:              record.ID,
		RequesterUserId: record.RequesterUserID,
		OwnerUserId:     record.OwnerUserID,
		AgentId:         record.AgentID,
		Scope:           record.Scope,
		RequestNote:     record.RequestNote,
		Status:          accessRequestStatusToProto(record.Status),
		ReviewerUserId:  record.ReviewerUserID,
		ReviewNote:      record.ReviewNote,
		CreatedAt:       timestamppb.New(record.CreatedAt),
		UpdatedAt:       timestamppb.New(record.UpdatedAt),
	}
	if record.ReviewedAt != nil {
		proto.ReviewedAt = timestamppb.New(*record.ReviewedAt)
	}
	return proto
}

func auditEventToProto(record storage.AuditEventRecord) *aiv1.AuditEvent {
	return &aiv1.AuditEvent{
		Id:              record.ID,
		EventName:       record.EventName,
		ActorUserId:     record.ActorUserID,
		OwnerUserId:     record.OwnerUserID,
		RequesterUserId: record.RequesterUserID,
		AgentId:         record.AgentID,
		AccessRequestId: record.AccessRequestID,
		Outcome:         record.Outcome,
		CreatedAt:       timestamppb.New(record.CreatedAt),
	}
}

func clampPageSize(requested int32) int {
	if requested <= 0 {
		return defaultPageSize
	}
	if requested > maxPageSize {
		return maxPageSize
	}
	return int(requested)
}

// isCredentialActiveForUser centralizes credential authorization checks used by
// agent write flows.
func isCredentialActiveForUser(record storage.CredentialRecord, ownerUserID string, provider string) bool {
	if strings.TrimSpace(record.OwnerUserID) != strings.TrimSpace(ownerUserID) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(record.Status), "active") {
		return false
	}
	if provider != "" && !strings.EqualFold(strings.TrimSpace(record.Provider), strings.TrimSpace(provider)) {
		return false
	}
	return true
}

func isProviderGrantActiveForUser(record storage.ProviderGrantRecord, ownerUserID string, provider string) bool {
	if strings.TrimSpace(record.OwnerUserID) != strings.TrimSpace(ownerUserID) {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(record.Status), "active") {
		return false
	}
	if provider != "" && !strings.EqualFold(strings.TrimSpace(record.Provider), strings.TrimSpace(provider)) {
		return false
	}
	return true
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type providerTokenPayload struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func decodeProviderTokenPayload(tokenPlaintext string) (providerTokenPayload, error) {
	tokenPlaintext = strings.TrimSpace(tokenPlaintext)
	if tokenPlaintext == "" {
		return providerTokenPayload{}, fmt.Errorf("token payload is empty")
	}
	var payload providerTokenPayload
	if err := json.Unmarshal([]byte(tokenPlaintext), &payload); err != nil {
		return providerTokenPayload{}, fmt.Errorf("decode provider token payload: %w", err)
	}
	payload.AccessToken = strings.TrimSpace(payload.AccessToken)
	payload.RefreshToken = strings.TrimSpace(payload.RefreshToken)
	return payload, nil
}

func refreshTokenFromTokenPayload(tokenPlaintext string) (string, error) {
	payload, err := decodeProviderTokenPayload(tokenPlaintext)
	if err != nil {
		return "", err
	}
	refreshToken := strings.TrimSpace(payload.RefreshToken)
	if refreshToken == "" {
		return "", fmt.Errorf("refresh token is unavailable")
	}
	return refreshToken, nil
}

// accessTokenFromTokenPayload extracts only the provider access token used for
// invocation and avoids leaking unrelated token payload fields downstream.
func accessTokenFromTokenPayload(tokenPlaintext string) (string, error) {
	payload, err := decodeProviderTokenPayload(tokenPlaintext)
	if err != nil {
		return "", err
	}
	accessToken := strings.TrimSpace(payload.AccessToken)
	if accessToken == "" {
		return "", fmt.Errorf("access token is unavailable")
	}
	return accessToken, nil
}

func revokeTokenFromTokenPayload(tokenPlaintext string) (string, error) {
	payload, err := decodeProviderTokenPayload(tokenPlaintext)
	if err == nil {
		token := firstNonEmpty(payload.RefreshToken, payload.AccessToken)
		if token == "" {
			return "", fmt.Errorf("token payload is missing revoke token")
		}
		return token, nil
	}
	token := strings.TrimSpace(tokenPlaintext)
	if token == "" {
		return "", fmt.Errorf("token payload is empty")
	}
	return token, nil
}

func normalizeScopes(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	scopes := make([]string, 0, len(values))
	for _, value := range values {
		scope := strings.TrimSpace(value)
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		scopes = append(scopes, scope)
	}
	if len(scopes) == 0 {
		return nil
	}
	return scopes
}

// generatePKCECodeVerifier returns an RFC 7636-compliant verifier string with
// cryptographic entropy suitable for S256 code challenge derivation.
func generatePKCECodeVerifier() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("read pkce entropy: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func pkceCodeChallengeS256(codeVerifier string) string {
	sum := sha256.Sum256([]byte(codeVerifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func isValidPKCECodeVerifier(value string) bool {
	if len(value) < 43 || len(value) > 128 {
		return false
	}
	for _, r := range value {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-', r == '.', r == '_', r == '~':
		default:
			return false
		}
	}
	return true
}

func hashState(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}
