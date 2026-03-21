package ai

import (
	"context"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/aisessiongrant"
	"google.golang.org/grpc/codes"
)

func TestRunCampaignTurnRejectsInvalidGrant(t *testing.T) {
	cfg := newCampaignOrchestrationHandlersConfigWithStores(newFakeStore(), newFakeStore(), &fakeSealer{})
	cfg.CampaignTurnRunner = &fakeCampaignTurnRunner{}
	cfg.GameCampaignAIClient = &fakeCampaignAIAuthStateClient{}
	sessionGrantConfig := testAISessionGrantConfig()
	cfg.SessionGrantConfig = &sessionGrantConfig
	svc := NewCampaignOrchestrationHandlers(cfg)

	_, err := svc.RunCampaignTurn(context.Background(), &aiv1.RunCampaignTurnRequest{
		SessionGrant: "bad-token",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestRunCampaignTurnRejectsStaleGrant(t *testing.T) {
	cfg := newCampaignOrchestrationHandlersConfigWithStores(newFakeStore(), newFakeStore(), &fakeSealer{})
	cfg.CampaignTurnRunner = &fakeCampaignTurnRunner{}
	sessionGrantConfig := testAISessionGrantConfig()
	cfg.SessionGrantConfig = &sessionGrantConfig
	cfg.GameCampaignAIClient = &fakeCampaignAIAuthStateClient{
		authState: &gamev1.GetCampaignAIAuthStateResponse{
			CampaignId:      "camp-1",
			AiAgentId:       "agent-1",
			ActiveSessionId: "sess-1",
			AuthEpoch:       1,
			ParticipantId:   "gm-2",
		},
	}
	svc := NewCampaignOrchestrationHandlers(cfg)

	_, err := svc.RunCampaignTurn(context.Background(), &aiv1.RunCampaignTurnRequest{
		SessionGrant: mustIssueAISessionGrant(t, testAISessionGrantConfig(), aisessiongrant.IssueInput{
			GrantID:       "grant-1",
			CampaignID:    "camp-1",
			SessionID:     "sess-1",
			ParticipantID: "gm-1",
			AuthEpoch:     1,
		}),
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestRunCampaignTurnMapsRunnerTimeout(t *testing.T) {
	store := newFakeStore()
	now := time.Now().UTC()
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "main",
		Status:           "active",
		SecretCiphertext: "enc:sk-1",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "gm",
		Provider:     "openai",
		Model:        "gpt-4.1-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	cfg := newCampaignOrchestrationHandlersConfigWithStores(store, store, &fakeSealer{})
	cfg.CampaignTurnRunner = &fakeCampaignTurnRunner{runErr: context.DeadlineExceeded}
	sessionGrantConfig := testAISessionGrantConfig()
	cfg.SessionGrantConfig = &sessionGrantConfig
	cfg.GameCampaignAIClient = &fakeCampaignAIAuthStateClient{
		authState: &gamev1.GetCampaignAIAuthStateResponse{
			CampaignId:      "camp-1",
			AiAgentId:       "agent-1",
			ActiveSessionId: "sess-1",
			AuthEpoch:       7,
			ParticipantId:   "gm-1",
		},
	}
	svc := NewCampaignOrchestrationHandlers(cfg)

	_, err := svc.RunCampaignTurn(context.Background(), &aiv1.RunCampaignTurnRequest{
		SessionGrant: mustIssueAISessionGrant(t, testAISessionGrantConfig(), aisessiongrant.IssueInput{
			GrantID:       "grant-1",
			CampaignID:    "camp-1",
			SessionID:     "sess-1",
			ParticipantID: "gm-1",
			AuthEpoch:     7,
		}),
	})
	assertStatusCode(t, err, codes.DeadlineExceeded)
	assertStatusReason(t, err, apperrors.CodeAIOrchestrationTimedOut)
}

func TestRunCampaignTurnMapsRunnerStepLimit(t *testing.T) {
	store := newFakeStore()
	now := time.Now().UTC()
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "main",
		Status:           "active",
		SecretCiphertext: "enc:sk-1",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "gm",
		Provider:     "openai",
		Model:        "gpt-4.1-mini",
		CredentialID: "cred-1",
		Status:       "active",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	cfg := newCampaignOrchestrationHandlersConfigWithStores(store, store, &fakeSealer{})
	cfg.CampaignTurnRunner = &fakeCampaignTurnRunner{runErr: orchestration.ErrStepLimit}
	sessionGrantConfig := testAISessionGrantConfig()
	cfg.SessionGrantConfig = &sessionGrantConfig
	cfg.GameCampaignAIClient = &fakeCampaignAIAuthStateClient{
		authState: &gamev1.GetCampaignAIAuthStateResponse{
			CampaignId:      "camp-1",
			AiAgentId:       "agent-1",
			ActiveSessionId: "sess-1",
			AuthEpoch:       7,
			ParticipantId:   "gm-1",
		},
	}
	svc := NewCampaignOrchestrationHandlers(cfg)

	_, err := svc.RunCampaignTurn(context.Background(), &aiv1.RunCampaignTurnRequest{
		SessionGrant: mustIssueAISessionGrant(t, testAISessionGrantConfig(), aisessiongrant.IssueInput{
			GrantID:       "grant-1",
			CampaignID:    "camp-1",
			SessionID:     "sess-1",
			ParticipantID: "gm-1",
			AuthEpoch:     7,
		}),
	})
	assertStatusCode(t, err, codes.Internal)
	assertStatusReason(t, err, apperrors.CodeAIOrchestrationStepLimitExceeded)
}

func TestRunCampaignTurnRunsOrchestration(t *testing.T) {
	store := newFakeStore()
	now := time.Now().UTC()
	store.Credentials["cred-1"] = storage.CredentialRecord{
		ID:               "cred-1",
		OwnerUserID:      "user-1",
		Provider:         "openai",
		Label:            "main",
		Status:           "active",
		SecretCiphertext: "enc:sk-1",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	store.Agents["agent-1"] = storage.AgentRecord{
		ID:           "agent-1",
		OwnerUserID:  "user-1",
		Label:        "gm",
		Provider:     "openai",
		Model:        "gpt-4.1-mini",
		CredentialID: "cred-1",
		Status:       "active",
		Instructions: "Be the GM.",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	runner := &fakeCampaignTurnRunner{
		runResult: orchestration.Result{
			OutputText: "The storm breaks over the ridge.",
			Usage:      provider.Usage{InputTokens: 14, OutputTokens: 9, ReasoningTokens: 4, TotalTokens: 23},
		},
	}
	cfg := newCampaignOrchestrationHandlersConfigWithStores(store, store, &fakeSealer{})
	cfg.CampaignTurnRunner = runner
	sessionGrantConfig := testAISessionGrantConfig()
	cfg.SessionGrantConfig = &sessionGrantConfig
	cfg.GameCampaignAIClient = &fakeCampaignAIAuthStateClient{
		authState: &gamev1.GetCampaignAIAuthStateResponse{
			CampaignId:      "camp-1",
			AiAgentId:       "agent-1",
			ActiveSessionId: "sess-1",
			AuthEpoch:       7,
			ParticipantId:   "gm-1",
		},
	}
	svc := NewCampaignOrchestrationHandlers(cfg)

	resp, err := svc.RunCampaignTurn(context.Background(), &aiv1.RunCampaignTurnRequest{
		SessionGrant: mustIssueAISessionGrant(t, testAISessionGrantConfig(), aisessiongrant.IssueInput{
			GrantID:       "grant-1",
			CampaignID:    "camp-1",
			SessionID:     "sess-1",
			ParticipantID: "gm-1",
			AuthEpoch:     7,
		}),
		Input:           "Frame the next scene.",
		ReasoningEffort: "medium",
	})
	if err != nil {
		t.Fatalf("run campaign turn: %v", err)
	}
	if resp.GetOutputText() != "The storm breaks over the ridge." {
		t.Fatalf("output_text = %q", resp.GetOutputText())
	}
	if resp.GetProvider() != aiv1.Provider_PROVIDER_OPENAI {
		t.Fatalf("provider = %v", resp.GetProvider())
	}
	if runner.lastInput.CampaignID != "camp-1" || runner.lastInput.SessionID != "sess-1" {
		t.Fatalf("runner claims = %+v", runner.lastInput)
	}
	if runner.lastInput.ParticipantID != "gm-1" {
		t.Fatalf("runner participant = %q", runner.lastInput.ParticipantID)
	}
	if runner.lastInput.Model != "gpt-4.1-mini" {
		t.Fatalf("runner model = %q", runner.lastInput.Model)
	}
	if runner.lastInput.Instructions != "Be the GM." {
		t.Fatalf("runner instructions = %q", runner.lastInput.Instructions)
	}
	if runner.lastInput.CredentialSecret != "sk-1" {
		t.Fatalf("runner credential secret = %q", runner.lastInput.CredentialSecret)
	}
	if runner.lastInput.Input != "Frame the next scene." {
		t.Fatalf("runner input = %q", runner.lastInput.Input)
	}
	if runner.lastInput.ReasoningEffort != "medium" {
		t.Fatalf("runner reasoning effort = %q", runner.lastInput.ReasoningEffort)
	}
	if runner.lastInput.Provider == nil {
		t.Fatal("runner provider = nil")
	}
	if resp.GetUsage().GetTotalTokens() != 23 {
		t.Fatalf("usage.total_tokens = %d", resp.GetUsage().GetTotalTokens())
	}
}
