package campaigntransport

import (
	"context"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeAgentServiceClient struct {
	lastValidateRequest *aiv1.ValidateCampaignAgentBindingRequest
	lastValidateUserID  string
	validateErr         error
}

func (f *fakeAgentServiceClient) CreateAgent(context.Context, *aiv1.CreateAgentRequest, ...grpc.CallOption) (*aiv1.CreateAgentResponse, error) {
	return nil, status.Error(codes.Unimplemented, "CreateAgent not implemented in tests")
}

func (f *fakeAgentServiceClient) ListAgents(context.Context, *aiv1.ListAgentsRequest, ...grpc.CallOption) (*aiv1.ListAgentsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListAgents not implemented in tests")
}

func (f *fakeAgentServiceClient) ListProviderModels(context.Context, *aiv1.ListProviderModelsRequest, ...grpc.CallOption) (*aiv1.ListProviderModelsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListProviderModels not implemented in tests")
}

func (f *fakeAgentServiceClient) ListAccessibleAgents(context.Context, *aiv1.ListAccessibleAgentsRequest, ...grpc.CallOption) (*aiv1.ListAccessibleAgentsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "ListAccessibleAgents not implemented in tests")
}

func (f *fakeAgentServiceClient) GetAccessibleAgent(context.Context, *aiv1.GetAccessibleAgentRequest, ...grpc.CallOption) (*aiv1.GetAccessibleAgentResponse, error) {
	return nil, status.Error(codes.Unimplemented, "GetAccessibleAgent not implemented in tests")
}

func (f *fakeAgentServiceClient) ValidateCampaignAgentBinding(ctx context.Context, req *aiv1.ValidateCampaignAgentBindingRequest, _ ...grpc.CallOption) (*aiv1.ValidateCampaignAgentBindingResponse, error) {
	f.lastValidateRequest = req
	f.lastValidateUserID = grpcauthctx.UserIDFromOutgoingContext(ctx)
	if f.validateErr != nil {
		return nil, f.validateErr
	}
	return &aiv1.ValidateCampaignAgentBindingResponse{}, nil
}

func (f *fakeAgentServiceClient) UpdateAgent(context.Context, *aiv1.UpdateAgentRequest, ...grpc.CallOption) (*aiv1.UpdateAgentResponse, error) {
	return nil, status.Error(codes.Unimplemented, "UpdateAgent not implemented in tests")
}

func (f *fakeAgentServiceClient) DeleteAgent(context.Context, *aiv1.DeleteAgentRequest, ...grpc.CallOption) (*aiv1.DeleteAgentResponse, error) {
	return nil, status.Error(codes.Unimplemented, "DeleteAgent not implemented in tests")
}

func TestSetCampaignAIBindingSuccess(t *testing.T) {
	ts := newTestDeps()
	now := time.Date(2026, 3, 14, 11, 0, 0, 0, time.UTC)
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {
			ID:             "owner-1",
			CampaignID:     "c1",
			UserID:         "user-1",
			CampaignAccess: participant.CampaignAccessOwner,
		},
	}
	domain := &fakeDomainEngine{
		store: ts.Event,
		resultsByType: map[command.Type]engine.Result{
			handler.CommandTypeCampaignAIBind: {
				Decision: command.Accept(event.Event{
					CampaignID:  "c1",
					Type:        handler.EventTypeCampaignAIBound,
					Timestamp:   now,
					ActorType:   event.ActorTypeParticipant,
					EntityType:  "campaign",
					EntityID:    "c1",
					PayloadJSON: mustJSON(t, campaign.AIBindPayload{AIAgentID: "agent-7"}),
				}),
			},
			handler.CommandTypeCampaignAIAuthRotate: {
				Decision: command.Accept(event.Event{
					CampaignID:  "c1",
					Type:        handler.EventTypeCampaignAIAuthRotated,
					Timestamp:   now,
					ActorType:   event.ActorTypeParticipant,
					EntityType:  "campaign",
					EntityID:    "c1",
					PayloadJSON: mustJSON(t, campaign.AIAuthRotatePayload{EpochAfter: 1, Reason: aiAuthRotateReasonCampaignAIBound}),
				}),
			},
		},
	}
	aiClient := &fakeAgentServiceClient{}

	deps := ts.withDomain(domain).build()
	deps.AIClient = aiClient
	svc := newTestCampaignService(deps, runtimekit.FixedClock(now), nil)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcmeta.ParticipantIDHeader, "owner-1",
		grpcmeta.RequestIDHeader, "req-1",
		grpcmeta.InvocationIDHeader, "inv-1",
	))

	resp, err := svc.SetCampaignAIBinding(ctx, &statev1.SetCampaignAIBindingRequest{
		CampaignId: "c1",
		AiAgentId:  "agent-7",
	})
	if err != nil {
		t.Fatalf("SetCampaignAIBinding() error = %v", err)
	}
	if resp.GetCampaign().GetAiAgentId() != "agent-7" {
		t.Fatalf("campaign ai agent id = %q, want %q", resp.GetCampaign().GetAiAgentId(), "agent-7")
	}
	storedCampaign, err := ts.Campaign.Get(context.Background(), "c1")
	if err != nil {
		t.Fatalf("load stored campaign: %v", err)
	}
	if storedCampaign.AIAuthEpoch != 1 {
		t.Fatalf("stored ai auth epoch = %d, want %d", storedCampaign.AIAuthEpoch, 1)
	}
	if aiClient.lastValidateRequest == nil {
		t.Fatal("expected ValidateCampaignAgentBinding request")
	}
	if aiClient.lastValidateRequest.GetCampaignId() != "c1" || aiClient.lastValidateRequest.GetAgentId() != "agent-7" {
		t.Fatalf("validate request = %+v", aiClient.lastValidateRequest)
	}
	if aiClient.lastValidateUserID != "user-1" {
		t.Fatalf("validate user id = %q, want %q", aiClient.lastValidateUserID, "user-1")
	}
	if domain.calls != 2 {
		t.Fatalf("domain calls = %d, want %d", domain.calls, 2)
	}
	if domain.commands[0].Type != handler.CommandTypeCampaignAIBind {
		t.Fatalf("first command type = %q, want %q", domain.commands[0].Type, handler.CommandTypeCampaignAIBind)
	}
	if domain.commands[1].Type != handler.CommandTypeCampaignAIAuthRotate {
		t.Fatalf("second command type = %q, want %q", domain.commands[1].Type, handler.CommandTypeCampaignAIAuthRotate)
	}
}

func TestSetCampaignAIBindingRequiresConfiguredAIClient(t *testing.T) {
	ts := newTestDeps()
	ts.Campaign.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {
			ID:             "owner-1",
			CampaignID:     "c1",
			UserID:         "user-1",
			CampaignAccess: participant.CampaignAccessOwner,
		},
	}

	svc := NewCampaignService(ts.build())
	_, err := svc.SetCampaignAIBinding(requestctx.WithParticipantID("owner-1"), &statev1.SetCampaignAIBindingRequest{
		CampaignId: "c1",
		AiAgentId:  "agent-7",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestClearCampaignAIBindingSuccess(t *testing.T) {
	ts := newTestDeps()
	now := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	campaignRecord := gametest.ActiveCampaignRecord("c1")
	campaignRecord.AIAgentID = "agent-7"
	campaignRecord.AIAuthEpoch = 1
	ts.Campaign.Campaigns["c1"] = campaignRecord
	ts.Participant.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": {
			ID:             "owner-1",
			CampaignID:     "c1",
			UserID:         "user-1",
			CampaignAccess: participant.CampaignAccessOwner,
		},
	}
	domain := &fakeDomainEngine{
		store: ts.Event,
		resultsByType: map[command.Type]engine.Result{
			handler.CommandTypeCampaignAIUnbind: {
				Decision: command.Accept(event.Event{
					CampaignID:  "c1",
					Type:        handler.EventTypeCampaignAIUnbound,
					Timestamp:   now,
					ActorType:   event.ActorTypeParticipant,
					EntityType:  "campaign",
					EntityID:    "c1",
					PayloadJSON: mustJSON(t, campaign.AIUnbindPayload{}),
				}),
			},
			handler.CommandTypeCampaignAIAuthRotate: {
				Decision: command.Accept(event.Event{
					CampaignID:  "c1",
					Type:        handler.EventTypeCampaignAIAuthRotated,
					Timestamp:   now,
					ActorType:   event.ActorTypeParticipant,
					EntityType:  "campaign",
					EntityID:    "c1",
					PayloadJSON: mustJSON(t, campaign.AIAuthRotatePayload{EpochAfter: 2, Reason: aiAuthRotateReasonCampaignAIUnbound}),
				}),
			},
		},
	}

	svc := newTestCampaignService(ts.withDomain(domain).build(), runtimekit.FixedClock(now), nil)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcmeta.ParticipantIDHeader, "owner-1",
		grpcmeta.RequestIDHeader, "req-2",
		grpcmeta.InvocationIDHeader, "inv-2",
	))
	resp, err := svc.ClearCampaignAIBinding(ctx, &statev1.ClearCampaignAIBindingRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ClearCampaignAIBinding() error = %v", err)
	}
	if resp.GetCampaign().GetAiAgentId() != "" {
		t.Fatalf("campaign ai agent id = %q, want empty", resp.GetCampaign().GetAiAgentId())
	}
	storedCampaign, err := ts.Campaign.Get(context.Background(), "c1")
	if err != nil {
		t.Fatalf("load stored campaign: %v", err)
	}
	if storedCampaign.AIAuthEpoch != 2 {
		t.Fatalf("stored ai auth epoch = %d, want %d", storedCampaign.AIAuthEpoch, 2)
	}
	if domain.calls != 2 {
		t.Fatalf("domain calls = %d, want %d", domain.calls, 2)
	}
}

func TestRequireCampaignOwnerRequiresOwnerUserIdentity(t *testing.T) {
	campaignRecord := gametest.ActiveCampaignRecord("c1")

	t.Run("manager access is insufficient", func(t *testing.T) {
		participants := gametest.NewFakeParticipantStore()
		participants.Participants["c1"] = map[string]storage.ParticipantRecord{
			"manager-1": {
				ID:             "manager-1",
				CampaignID:     "c1",
				UserID:         "user-1",
				CampaignAccess: participant.CampaignAccessManager,
			},
		}

		_, err := requireCampaignOwner(requestctx.WithParticipantID("manager-1"), authz.PolicyDeps{Participant: participants}, campaignRecord)
		assertStatusCode(t, err, codes.PermissionDenied)
	})

	t.Run("owner user identity is required", func(t *testing.T) {
		participants := gametest.NewFakeParticipantStore()
		participants.Participants["c1"] = map[string]storage.ParticipantRecord{
			"owner-1": {
				ID:             "owner-1",
				CampaignID:     "c1",
				CampaignAccess: participant.CampaignAccessOwner,
			},
		}

		_, err := requireCampaignOwner(requestctx.WithParticipantID("owner-1"), authz.PolicyDeps{Participant: participants}, campaignRecord)
		assertStatusCode(t, err, codes.PermissionDenied)
	})
}

func TestNewClearCampaignAIBindingFuncClearsStoredBinding(t *testing.T) {
	ts := newTestDeps()
	now := time.Date(2026, 3, 14, 13, 0, 0, 0, time.UTC)
	campaignRecord := gametest.ActiveCampaignRecord("c1")
	campaignRecord.AIAgentID = "agent-7"
	campaignRecord.AIAuthEpoch = 3
	ts.Campaign.Campaigns["c1"] = campaignRecord
	domain := &fakeDomainEngine{
		store: ts.Event,
		resultsByType: map[command.Type]engine.Result{
			handler.CommandTypeCampaignAIUnbind: {
				Decision: command.Accept(event.Event{
					CampaignID:  "c1",
					Type:        handler.EventTypeCampaignAIUnbound,
					Timestamp:   now,
					ActorType:   event.ActorTypeParticipant,
					EntityType:  "campaign",
					EntityID:    "c1",
					PayloadJSON: mustJSON(t, campaign.AIUnbindPayload{}),
				}),
			},
			handler.CommandTypeCampaignAIAuthRotate: {
				Decision: command.Accept(event.Event{
					CampaignID:  "c1",
					Type:        handler.EventTypeCampaignAIAuthRotated,
					Timestamp:   now,
					ActorType:   event.ActorTypeParticipant,
					EntityType:  "campaign",
					EntityID:    "c1",
					PayloadJSON: mustJSON(t, campaign.AIAuthRotatePayload{EpochAfter: 4, Reason: aiAuthRotateReasonCampaignAIUnbound}),
				}),
			},
		},
	}

	clearBinding := NewClearCampaignAIBindingFunc(
		ts.Campaign,
		domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		projection.Applier{Campaign: ts.Campaign},
	)
	updated, err := clearBinding(
		metadata.NewIncomingContext(context.Background(), metadata.Pairs(
			grpcmeta.RequestIDHeader, "req-3",
			grpcmeta.InvocationIDHeader, "inv-3",
		)),
		"c1",
		"owner-1",
		command.ActorTypeParticipant,
		"req-3",
		"inv-3",
	)
	if err != nil {
		t.Fatalf("clear binding callback error = %v", err)
	}
	if updated.AIAgentID != "" {
		t.Fatalf("updated ai agent id = %q, want empty", updated.AIAgentID)
	}
	if updated.AIAuthEpoch != 4 {
		t.Fatalf("updated ai auth epoch = %d, want %d", updated.AIAuthEpoch, 4)
	}
	if domain.calls != 2 {
		t.Fatalf("domain calls = %d, want %d", domain.calls, 2)
	}
}
