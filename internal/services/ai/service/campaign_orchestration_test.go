package service

import (
	"context"
	"errors"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/credential"
	"github.com/louisbranch/fracturing.space/internal/services/ai/debugtrace"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
)

type campaignTurnRunnerStub struct {
	lastInput orchestration.Input
	result    orchestration.Result
	err       error
}

func (s *campaignTurnRunnerStub) Run(_ context.Context, input orchestration.Input) (orchestration.Result, error) {
	s.lastInput = input
	if s.err != nil {
		return orchestration.Result{}, s.err
	}
	return s.result, nil
}

type providerAdapterStub struct{}

func (providerAdapterStub) Run(context.Context, orchestration.ProviderInput) (orchestration.ProviderOutput, error) {
	return orchestration.ProviderOutput{}, nil
}

type campaignAuthStateReaderStub struct {
	authState *gamev1.GetCampaignAIAuthStateResponse
	authErr   error
}

func (s *campaignAuthStateReaderStub) CampaignAuthState(context.Context, string) (*gamev1.GetCampaignAIAuthStateResponse, error) {
	if s.authErr != nil {
		return nil, s.authErr
	}
	return s.authState, nil
}

func TestNewCampaignOrchestrationServiceValidationAndCopy(t *testing.T) {
	t.Parallel()

	if _, err := NewCampaignOrchestrationService(CampaignOrchestrationServiceConfig{}); err == nil {
		t.Fatal("expected missing dependencies error")
	}
	if _, err := NewCampaignOrchestrationService(CampaignOrchestrationServiceConfig{
		AgentStore: aifakes.NewAgentStore(),
	}); err == nil {
		t.Fatal("expected missing auth material resolver error")
	}

	grantCfg := aisessiongrant.Config{
		Issuer:   "issuer",
		Audience: "audience",
		HMACKey:  []byte("12345678901234567890123456789012"),
		TTL:      time.Minute,
		Now:      func() time.Time { return time.Unix(1711111111, 0).UTC() },
	}
	grantCfgPtr := &grantCfg
	svc, err := NewCampaignOrchestrationService(CampaignOrchestrationServiceConfig{
		AgentStore: aifakes.NewAgentStore(),
		AuthMaterialResolver: NewAuthMaterialResolver(AuthMaterialResolverConfig{
			ProviderGrantRuntime: NewProviderGrantRuntime(ProviderGrantRuntimeConfig{}),
		}),
		ProviderRegistry: mustProviderRegistryForTests(t, nil, nil, nil, map[provider.Provider]orchestration.Provider{
			provider.OpenAI: providerAdapterStub{},
		}),
		SessionGrantConfig: grantCfgPtr,
	})
	if err != nil {
		t.Fatalf("NewCampaignOrchestrationService: %v", err)
	}
	if svc.sessionGrantConfig == grantCfgPtr {
		t.Fatal("sessionGrantConfig should be copied, not aliased")
	}
	grantCfg.Issuer = "changed"
	if svc.sessionGrantConfig.Issuer != "issuer" {
		t.Fatalf("sessionGrantConfig issuer = %q, want %q", svc.sessionGrantConfig.Issuer, "issuer")
	}
}

func TestCampaignOrchestrationServiceRunCampaignTurn(t *testing.T) {
	t.Parallel()

	now := time.Unix(1711111111, 0).UTC()
	grantConfig := aisessiongrant.Config{
		Issuer:   "fracturing-space-game",
		Audience: "fracturing-space-ai",
		HMACKey:  []byte("12345678901234567890123456789012"),
		TTL:      5 * time.Minute,
		Now:      func() time.Time { return now },
	}
	sessionGrant, _, err := aisessiongrant.Issue(grantConfig, aisessiongrant.IssueInput{
		GrantID:         "grant-1",
		CampaignID:      "campaign-1",
		SessionID:       "session-1",
		ParticipantID:   "participant-1",
		AuthEpoch:       7,
		IssuedForUserID: "user-1",
	})
	if err != nil {
		t.Fatalf("Issue session grant: %v", err)
	}

	credentialStore := aifakes.NewCredentialStore()
	credentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "primary",
		SecretCiphertext: "enc:token-1",
		Status:           credential.StatusActive,
	}
	authMaterialResolver := NewAuthMaterialResolver(AuthMaterialResolverConfig{
		CredentialStore: credentialStore,
		Sealer:          &aifakes.Sealer{},
	})

	agentStore := aifakes.NewAgentStore()
	agentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "gm-runtime",
		Instructions:  "Run the session.",
		Provider:      provider.OpenAI,
		Model:         "gpt-5.4",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
	}
	runner := &campaignTurnRunnerStub{
		result: orchestration.Result{
			OutputText: "The harbor wakes.",
			Usage: provider.Usage{
				InputTokens:  10,
				OutputTokens: 5,
				TotalTokens:  15,
			},
		},
	}
	traceStore := newDebugTraceStoreStub()
	svc, err := NewCampaignOrchestrationService(CampaignOrchestrationServiceConfig{
		AgentStore:              agentStore,
		CampaignAuthStateReader: &campaignAuthStateReaderStub{authState: matchingCampaignAIAuthState()},
		ProviderRegistry:        mustProviderRegistryForTests(t, nil, nil, nil, map[provider.Provider]orchestration.Provider{provider.OpenAI: providerAdapterStub{}}),
		CampaignTurnRunner:      runner,
		DebugTraceStore:         traceStore,
		DebugUpdateBroker:       NewCampaignDebugUpdateBroker(),
		SessionGrantConfig:      &grantConfig,
		AuthMaterialResolver:    authMaterialResolver,
		Clock:                   func() time.Time { return now },
		IDGenerator:             func() (string, error) { return "turn-1", nil },
	})
	if err != nil {
		t.Fatalf("NewCampaignOrchestrationService: %v", err)
	}

	result, err := svc.RunCampaignTurn(context.Background(), RunCampaignTurnInput{
		SessionGrant:    sessionGrant,
		Input:           "Start the scene.",
		ReasoningEffort: "medium",
		TurnToken:       "turn-token-1",
	})
	if err != nil {
		t.Fatalf("RunCampaignTurn: %v", err)
	}
	if result.OutputText != "The harbor wakes." || result.Provider != provider.OpenAI || result.Model != "gpt-5.4" || result.Usage.TotalTokens != 15 {
		t.Fatalf("result = %#v", result)
	}
	if runner.lastInput.CampaignID != "campaign-1" || runner.lastInput.SessionID != "session-1" || runner.lastInput.ParticipantID != "participant-1" {
		t.Fatalf("runner input IDs = %#v", runner.lastInput)
	}
	if runner.lastInput.AuthToken != "token-1" || runner.lastInput.Input != "Start the scene." || runner.lastInput.TraceRecorder == nil {
		t.Fatalf("runner input = %#v", runner.lastInput)
	}
	if turn := traceStore.turns["turn-1"]; turn.Status != debugtrace.StatusSucceeded || turn.TurnToken != "turn-token-1" || turn.Provider != provider.OpenAI {
		t.Fatalf("trace turn = %#v", turn)
	}
}

func TestCampaignOrchestrationServiceRunCampaignTurnValidationAndFailures(t *testing.T) {
	t.Parallel()

	now := time.Unix(1711111111, 0).UTC()
	validGrantConfig := aisessiongrant.Config{
		Issuer:   "fracturing-space-game",
		Audience: "fracturing-space-ai",
		HMACKey:  []byte("12345678901234567890123456789012"),
		TTL:      5 * time.Minute,
		Now:      func() time.Time { return now },
	}
	validGrant, _, err := aisessiongrant.Issue(validGrantConfig, aisessiongrant.IssueInput{
		GrantID:       "grant-1",
		CampaignID:    "campaign-1",
		SessionID:     "session-1",
		ParticipantID: "participant-1",
		AuthEpoch:     7,
	})
	if err != nil {
		t.Fatalf("Issue valid grant: %v", err)
	}

	expiredGrantConfig := validGrantConfig
	expiredGrantConfig.Now = func() time.Time { return now.Add(-10 * time.Minute) }
	expiredGrantConfig.TTL = time.Minute
	expiredGrant, _, err := aisessiongrant.Issue(expiredGrantConfig, aisessiongrant.IssueInput{
		GrantID:       "grant-expired",
		CampaignID:    "campaign-1",
		SessionID:     "session-1",
		ParticipantID: "participant-1",
		AuthEpoch:     7,
	})
	if err != nil {
		t.Fatalf("Issue expired grant: %v", err)
	}

	baseAgentStore := aifakes.NewAgentStore()
	baseAgentStore.Agents["agent-1"] = agent.Agent{
		ID:            "agent-1",
		OwnerUserID:   "user-1",
		Label:         "gm-runtime",
		Instructions:  "Run the session.",
		Provider:      provider.OpenAI,
		Model:         "gpt-5.4",
		AuthReference: agent.CredentialAuthReference("cred-1"),
		Status:        agent.StatusActive,
	}
	baseCredentialStore := aifakes.NewCredentialStore()
	baseCredentialStore.Credentials["cred-1"] = credential.Credential{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         provider.OpenAI,
		Label:            "primary",
		SecretCiphertext: "enc:token-1",
		Status:           credential.StatusActive,
	}
	baseAuthResolver := NewAuthMaterialResolver(AuthMaterialResolverConfig{
		CredentialStore: baseCredentialStore,
		Sealer:          &aifakes.Sealer{},
	})
	baseRunner := &campaignTurnRunnerStub{
		result: orchestration.Result{OutputText: "ok"},
	}

	newService := func() *CampaignOrchestrationService {
		svc, svcErr := NewCampaignOrchestrationService(CampaignOrchestrationServiceConfig{
			AgentStore:              baseAgentStore,
			CampaignAuthStateReader: &campaignAuthStateReaderStub{authState: matchingCampaignAIAuthState()},
			ProviderRegistry:        mustProviderRegistryForTests(t, nil, nil, nil, map[provider.Provider]orchestration.Provider{provider.OpenAI: providerAdapterStub{}}),
			CampaignTurnRunner:      baseRunner,
			SessionGrantConfig:      &validGrantConfig,
			AuthMaterialResolver:    baseAuthResolver,
			Clock:                   func() time.Time { return now },
			IDGenerator:             func() (string, error) { return "turn-1", nil },
		})
		if svcErr != nil {
			t.Fatalf("NewCampaignOrchestrationService: %v", svcErr)
		}
		return svc
	}

	runErr := errors.New("runner failed")
	tests := []struct {
		name      string
		mutate    func(*CampaignOrchestrationService)
		input     RunCampaignTurnInput
		wantKind  ErrorKind
		wantErrIs error
	}{
		{
			name: "missing runner",
			mutate: func(s *CampaignOrchestrationService) {
				s.campaignTurnRunner = nil
			},
			input:    RunCampaignTurnInput{SessionGrant: validGrant},
			wantKind: ErrKindFailedPrecondition,
		},
		{
			name: "missing grant config",
			mutate: func(s *CampaignOrchestrationService) {
				s.sessionGrantConfig = nil
			},
			input:    RunCampaignTurnInput{SessionGrant: validGrant},
			wantKind: ErrKindFailedPrecondition,
		},
		{
			name: "missing auth state client",
			mutate: func(s *CampaignOrchestrationService) {
				s.campaignAuthStateReader = nil
			},
			input:    RunCampaignTurnInput{SessionGrant: validGrant},
			wantKind: ErrKindFailedPrecondition,
		},
		{
			name:     "missing session grant",
			input:    RunCampaignTurnInput{},
			wantKind: ErrKindInvalidArgument,
		},
		{
			name:     "invalid session grant",
			input:    RunCampaignTurnInput{SessionGrant: "bad-token"},
			wantKind: ErrKindPermissionDenied,
		},
		{
			name:     "expired session grant",
			input:    RunCampaignTurnInput{SessionGrant: expiredGrant},
			wantKind: ErrKindPermissionDenied,
		},
		{
			name: "stale auth state",
			mutate: func(s *CampaignOrchestrationService) {
				s.campaignAuthStateReader = &campaignAuthStateReaderStub{authState: &gamev1.GetCampaignAIAuthStateResponse{
					CampaignId:      "campaign-1",
					ActiveSessionId: "session-2",
					ParticipantId:   "participant-1",
					AuthEpoch:       7,
					AiAgentId:       "agent-1",
				}}
			},
			input:    RunCampaignTurnInput{SessionGrant: validGrant},
			wantKind: ErrKindFailedPrecondition,
		},
		{
			name: "missing agent id",
			mutate: func(s *CampaignOrchestrationService) {
				s.campaignAuthStateReader = &campaignAuthStateReaderStub{authState: &gamev1.GetCampaignAIAuthStateResponse{
					CampaignId:      "campaign-1",
					ActiveSessionId: "session-1",
					ParticipantId:   "participant-1",
					AuthEpoch:       7,
				}}
			},
			input:    RunCampaignTurnInput{SessionGrant: validGrant},
			wantKind: ErrKindFailedPrecondition,
		},
		{
			name: "missing agent record",
			mutate: func(s *CampaignOrchestrationService) {
				s.agentStore = aifakes.NewAgentStore()
			},
			input:    RunCampaignTurnInput{SessionGrant: validGrant},
			wantKind: ErrKindFailedPrecondition,
		},
		{
			name: "inactive agent",
			mutate: func(s *CampaignOrchestrationService) {
				store := aifakes.NewAgentStore()
				store.Agents["agent-1"] = agent.Agent{
					ID:            "agent-1",
					OwnerUserID:   "user-1",
					Label:         "gm-runtime",
					Provider:      provider.OpenAI,
					Model:         "gpt-5.4",
					AuthReference: agent.CredentialAuthReference("cred-1"),
					Status:        "",
				}
				s.agentStore = store
			},
			input:    RunCampaignTurnInput{SessionGrant: validGrant},
			wantKind: ErrKindFailedPrecondition,
		},
		{
			name: "missing provider adapter",
			mutate: func(s *CampaignOrchestrationService) {
				s.providerRegistry = mustProviderRegistryForTests(t, nil, nil, nil, nil)
			},
			input:    RunCampaignTurnInput{SessionGrant: validGrant},
			wantKind: ErrKindFailedPrecondition,
		},
		{
			name: "credential unavailable",
			mutate: func(s *CampaignOrchestrationService) {
				s.authMaterialResolver = NewAuthMaterialResolver(AuthMaterialResolverConfig{
					CredentialStore: aifakes.NewCredentialStore(),
					Sealer:          &aifakes.Sealer{},
				})
			},
			input:    RunCampaignTurnInput{SessionGrant: validGrant},
			wantKind: ErrKindFailedPrecondition,
		},
		{
			name: "runner error",
			mutate: func(s *CampaignOrchestrationService) {
				s.campaignTurnRunner = &campaignTurnRunnerStub{err: runErr}
			},
			input:     RunCampaignTurnInput{SessionGrant: validGrant},
			wantErrIs: runErr,
		},
		{
			name: "empty output",
			mutate: func(s *CampaignOrchestrationService) {
				s.campaignTurnRunner = &campaignTurnRunnerStub{result: orchestration.Result{}}
			},
			input:     RunCampaignTurnInput{SessionGrant: validGrant},
			wantErrIs: orchestration.ErrEmptyOutput,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newService()
			if tt.mutate != nil {
				tt.mutate(svc)
			}
			_, err := svc.RunCampaignTurn(context.Background(), tt.input)
			if tt.wantErrIs != nil {
				if !errors.Is(err, tt.wantErrIs) {
					t.Fatalf("error = %v, want %v", err, tt.wantErrIs)
				}
				return
			}
			if got := ErrorKindOf(err); got != tt.wantKind {
				t.Fatalf("ErrorKindOf(err) = %v, want %v (err=%v)", got, tt.wantKind, err)
			}
		})
	}
}

func TestStaleGrant(t *testing.T) {
	t.Parallel()

	claims := aisessiongrant.Claims{
		CampaignID:    "campaign-1",
		SessionID:     "session-1",
		ParticipantID: "participant-1",
		AuthEpoch:     7,
	}
	tests := []struct {
		name  string
		state *gamev1.GetCampaignAIAuthStateResponse
		want  bool
	}{
		{name: "nil state", want: true},
		{name: "campaign mismatch", state: &gamev1.GetCampaignAIAuthStateResponse{CampaignId: "campaign-2", ActiveSessionId: "session-1", ParticipantId: "participant-1", AuthEpoch: 7}, want: true},
		{name: "session mismatch", state: &gamev1.GetCampaignAIAuthStateResponse{CampaignId: "campaign-1", ActiveSessionId: "session-2", ParticipantId: "participant-1", AuthEpoch: 7}, want: true},
		{name: "participant mismatch", state: &gamev1.GetCampaignAIAuthStateResponse{CampaignId: "campaign-1", ActiveSessionId: "session-1", ParticipantId: "participant-2", AuthEpoch: 7}, want: true},
		{name: "epoch mismatch", state: &gamev1.GetCampaignAIAuthStateResponse{CampaignId: "campaign-1", ActiveSessionId: "session-1", ParticipantId: "participant-1", AuthEpoch: 8}, want: true},
		{name: "matching state", state: matchingCampaignAIAuthState(), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := staleGrant(claims, tt.state); got != tt.want {
				t.Fatalf("staleGrant() = %v, want %v", got, tt.want)
			}
		})
	}
}

func matchingCampaignAIAuthState() *gamev1.GetCampaignAIAuthStateResponse {
	return &gamev1.GetCampaignAIAuthStateResponse{
		CampaignId:      "campaign-1",
		ActiveSessionId: "session-1",
		ParticipantId:   "participant-1",
		AuthEpoch:       7,
		AiAgentId:       "agent-1",
	}
}
