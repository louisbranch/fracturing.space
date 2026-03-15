package domain

import (
	"context"
	"testing"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeCampaignAIOrchestrationClient struct {
	queueReqs    []*gamev1.QueueAIGMTurnRequest
	startReqs    []*gamev1.StartAIGMTurnRequest
	failReqs     []*gamev1.FailAIGMTurnRequest
	completeReqs []*gamev1.CompleteAIGMTurnRequest
	queueResp    *gamev1.QueueAIGMTurnResponse
	startResp    *gamev1.StartAIGMTurnResponse
	failResp     *gamev1.FailAIGMTurnResponse
	completeResp *gamev1.CompleteAIGMTurnResponse
	queueErr     error
	startErr     error
	failErr      error
	completeErr  error
}

func (f *fakeCampaignAIOrchestrationClient) QueueAIGMTurn(_ context.Context, in *gamev1.QueueAIGMTurnRequest, _ ...grpc.CallOption) (*gamev1.QueueAIGMTurnResponse, error) {
	f.queueReqs = append(f.queueReqs, in)
	if f.queueErr != nil {
		return nil, f.queueErr
	}
	if f.queueResp != nil {
		return f.queueResp, nil
	}
	return &gamev1.QueueAIGMTurnResponse{
		AiTurn: &gamev1.AITurnState{TurnToken: "turn-1"},
	}, nil
}

func (f *fakeCampaignAIOrchestrationClient) StartAIGMTurn(_ context.Context, in *gamev1.StartAIGMTurnRequest, _ ...grpc.CallOption) (*gamev1.StartAIGMTurnResponse, error) {
	f.startReqs = append(f.startReqs, in)
	if f.startErr != nil {
		return nil, f.startErr
	}
	if f.startResp != nil {
		return f.startResp, nil
	}
	return &gamev1.StartAIGMTurnResponse{}, nil
}

func (f *fakeCampaignAIOrchestrationClient) FailAIGMTurn(_ context.Context, in *gamev1.FailAIGMTurnRequest, _ ...grpc.CallOption) (*gamev1.FailAIGMTurnResponse, error) {
	f.failReqs = append(f.failReqs, in)
	if f.failErr != nil {
		return nil, f.failErr
	}
	if f.failResp != nil {
		return f.failResp, nil
	}
	return &gamev1.FailAIGMTurnResponse{}, nil
}

func (f *fakeCampaignAIOrchestrationClient) CompleteAIGMTurn(_ context.Context, in *gamev1.CompleteAIGMTurnRequest, _ ...grpc.CallOption) (*gamev1.CompleteAIGMTurnResponse, error) {
	f.completeReqs = append(f.completeReqs, in)
	if f.completeErr != nil {
		return nil, f.completeErr
	}
	if f.completeResp != nil {
		return f.completeResp, nil
	}
	return &gamev1.CompleteAIGMTurnResponse{}, nil
}

type fakeCampaignAIServiceClient struct {
	reqs []*gamev1.IssueCampaignAISessionGrantRequest
	resp *gamev1.IssueCampaignAISessionGrantResponse
	err  error
}

func (f *fakeCampaignAIServiceClient) IssueCampaignAISessionGrant(_ context.Context, in *gamev1.IssueCampaignAISessionGrantRequest, _ ...grpc.CallOption) (*gamev1.IssueCampaignAISessionGrantResponse, error) {
	f.reqs = append(f.reqs, in)
	if f.err != nil {
		return nil, f.err
	}
	if f.resp != nil {
		return f.resp, nil
	}
	return &gamev1.IssueCampaignAISessionGrantResponse{
		Grant: &gamev1.AISessionGrant{Token: "grant-1"},
	}, nil
}

type fakeCampaignTurnClient struct {
	reqs []*aiv1.RunCampaignTurnRequest
	resp *aiv1.RunCampaignTurnResponse
	err  error
}

func (f *fakeCampaignTurnClient) RunCampaignTurn(_ context.Context, in *aiv1.RunCampaignTurnRequest, _ ...grpc.CallOption) (*aiv1.RunCampaignTurnResponse, error) {
	f.reqs = append(f.reqs, in)
	if f.err != nil {
		return nil, f.err
	}
	if f.resp != nil {
		return f.resp, nil
	}
	return &aiv1.RunCampaignTurnResponse{OutputText: "Narration"}, nil
}

func TestAIGMTurnRequestedHandlerHandleRunsFullLifecycle(t *testing.T) {
	t.Parallel()

	orchestration := &fakeCampaignAIOrchestrationClient{}
	game := &fakeCampaignAIServiceClient{}
	ai := &fakeCampaignTurnClient{}
	handler := NewAIGMTurnRequestedHandler(orchestration, game, ai)

	err := handler.Handle(context.Background(), outboxEventStub{
		payloadJSON: `{"campaign_id":" camp-1 ","session_id":" sess-1 ","source_event_type":" scene.player_phase_review_started ","source_scene_id":" scene-1 ","source_phase_id":" phase-1 "}`,
	})
	if err != nil {
		t.Fatalf("handle error = %v", err)
	}
	if len(orchestration.queueReqs) != 1 || len(orchestration.startReqs) != 1 || len(orchestration.completeReqs) != 1 {
		t.Fatalf("lifecycle requests = %#v %#v %#v", orchestration.queueReqs, orchestration.startReqs, orchestration.completeReqs)
	}
	if len(game.reqs) != 1 || len(ai.reqs) != 1 {
		t.Fatalf("grant/run requests = %#v %#v", game.reqs, ai.reqs)
	}
	if orchestration.startReqs[0].GetTurnToken() != "turn-1" || orchestration.completeReqs[0].GetTurnToken() != "turn-1" {
		t.Fatalf("turn token propagation = %#v %#v", orchestration.startReqs[0], orchestration.completeReqs[0])
	}
	if ai.reqs[0].GetSessionGrant() != "grant-1" {
		t.Fatalf("run request = %#v", ai.reqs[0])
	}
	if len(orchestration.failReqs) != 0 {
		t.Fatalf("unexpected fail requests = %#v", orchestration.failReqs)
	}
}

func TestAIGMTurnRequestedHandlerHandleNoopsWhenQueueReturnsIdle(t *testing.T) {
	t.Parallel()

	orchestration := &fakeCampaignAIOrchestrationClient{
		queueResp: &gamev1.QueueAIGMTurnResponse{
			AiTurn: &gamev1.AITurnState{Status: gamev1.AITurnStatus_AI_TURN_STATUS_IDLE},
		},
	}
	game := &fakeCampaignAIServiceClient{}
	ai := &fakeCampaignTurnClient{}
	handler := NewAIGMTurnRequestedHandler(orchestration, game, ai)

	err := handler.Handle(context.Background(), outboxEventStub{
		payloadJSON: `{"campaign_id":"camp-1","session_id":"sess-1","source_event_type":"session.started"}`,
	})
	if err != nil {
		t.Fatalf("handle error = %v", err)
	}
	if len(orchestration.startReqs) != 0 || len(orchestration.failReqs) != 0 || len(orchestration.completeReqs) != 0 {
		t.Fatalf("unexpected lifecycle calls = %#v %#v %#v", orchestration.startReqs, orchestration.failReqs, orchestration.completeReqs)
	}
	if len(game.reqs) != 0 || len(ai.reqs) != 0 {
		t.Fatalf("unexpected grant/run requests = %#v %#v", game.reqs, ai.reqs)
	}
}

func TestAIGMTurnRequestedHandlerHandleRejectsInvalidPayloadPermanently(t *testing.T) {
	t.Parallel()

	handler := NewAIGMTurnRequestedHandler(&fakeCampaignAIOrchestrationClient{}, &fakeCampaignAIServiceClient{}, &fakeCampaignTurnClient{})

	err := handler.Handle(context.Background(), outboxEventStub{payloadJSON: `{"campaign_id":"camp-1"}`})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsPermanent(err) {
		t.Fatalf("expected permanent error, got %v", err)
	}
}

func TestAIGMTurnRequestedHandlerHandleTreatsClearedTurnAsSuccessAfterCompleteError(t *testing.T) {
	t.Parallel()

	orchestration := &fakeCampaignAIOrchestrationClient{
		completeErr: status.Error(codes.Unavailable, "response lost"),
		failErr:     status.Error(codes.FailedPrecondition, "turn is no longer active"),
	}
	handler := NewAIGMTurnRequestedHandler(orchestration, &fakeCampaignAIServiceClient{}, &fakeCampaignTurnClient{})

	err := handler.Handle(context.Background(), outboxEventStub{
		payloadJSON: `{"campaign_id":"camp-1","session_id":"sess-1","source_event_type":"session.started"}`,
	})
	if err != nil {
		t.Fatalf("handle error = %v", err)
	}
	if len(orchestration.failReqs) != 1 {
		t.Fatalf("fail requests = %#v, want 1", orchestration.failReqs)
	}
}
func TestAIGMTurnRequestedHandlerHandleClassifiesPreStartErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		err           error
		wantPermanent bool
	}{
		{name: "invalid argument", err: status.Error(codes.InvalidArgument, "bad request"), wantPermanent: true},
		{name: "permission denied", err: status.Error(codes.PermissionDenied, "denied"), wantPermanent: true},
		{name: "unauthenticated", err: status.Error(codes.Unauthenticated, "missing"), wantPermanent: true},
		{name: "failed precondition", err: status.Error(codes.FailedPrecondition, "stale"), wantPermanent: true},
		{name: "unavailable", err: status.Error(codes.Unavailable, "retry"), wantPermanent: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := NewAIGMTurnRequestedHandler(&fakeCampaignAIOrchestrationClient{queueErr: tc.err}, &fakeCampaignAIServiceClient{}, &fakeCampaignTurnClient{})
			err := handler.Handle(context.Background(), outboxEventStub{
				payloadJSON: `{"campaign_id":"camp-1","session_id":"sess-1","source_event_type":"session.active_scene_set"}`,
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if got := IsPermanent(err); got != tc.wantPermanent {
				t.Fatalf("IsPermanent(%v) = %v, want %v", err, got, tc.wantPermanent)
			}
		})
	}
}

func TestAIGMTurnRequestedHandlerHandleFailsTurnOnGrantError(t *testing.T) {
	t.Parallel()

	orchestration := &fakeCampaignAIOrchestrationClient{}
	game := &fakeCampaignAIServiceClient{err: status.Error(codes.Unavailable, "grant down")}
	handler := NewAIGMTurnRequestedHandler(orchestration, game, &fakeCampaignTurnClient{})

	err := handler.Handle(context.Background(), outboxEventStub{
		payloadJSON: `{"campaign_id":"camp-1","session_id":"sess-1","source_event_type":"session.active_scene_set"}`,
	})
	if err != nil {
		t.Fatalf("handle error = %v", err)
	}
	if len(orchestration.failReqs) != 1 {
		t.Fatalf("fail requests = %#v, want 1", orchestration.failReqs)
	}
	if orchestration.failReqs[0].GetTurnToken() != "turn-1" {
		t.Fatalf("fail request = %#v", orchestration.failReqs[0])
	}
}

func TestAIGMTurnRequestedHandlerHandleFailsTurnOnAIError(t *testing.T) {
	t.Parallel()

	orchestration := &fakeCampaignAIOrchestrationClient{}
	ai := &fakeCampaignTurnClient{err: status.Error(codes.Internal, "provider failed")}
	handler := NewAIGMTurnRequestedHandler(orchestration, &fakeCampaignAIServiceClient{}, ai)

	err := handler.Handle(context.Background(), outboxEventStub{
		payloadJSON: `{"campaign_id":"camp-1","session_id":"sess-1","source_event_type":"session.active_scene_set"}`,
	})
	if err != nil {
		t.Fatalf("handle error = %v", err)
	}
	if len(orchestration.failReqs) != 1 {
		t.Fatalf("fail requests = %#v, want 1", orchestration.failReqs)
	}
}

func TestAIGMTurnRequestedHandlerHandleReturnsErrorWhenFailWriteFails(t *testing.T) {
	t.Parallel()

	orchestration := &fakeCampaignAIOrchestrationClient{failErr: status.Error(codes.Unavailable, "still down")}
	game := &fakeCampaignAIServiceClient{err: status.Error(codes.Unavailable, "grant down")}
	handler := NewAIGMTurnRequestedHandler(orchestration, game, &fakeCampaignTurnClient{})

	err := handler.Handle(context.Background(), outboxEventStub{
		payloadJSON: `{"campaign_id":"camp-1","session_id":"sess-1","source_event_type":"session.active_scene_set"}`,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if IsPermanent(err) {
		t.Fatalf("expected retryable error, got %v", err)
	}
}

func TestAIGMTurnRequestedHandlerHandleRequiresDependencies(t *testing.T) {
	t.Parallel()

	var handler *AIGMTurnRequestedHandler
	err := handler.Handle(context.Background(), outboxEventStub{payloadJSON: `{"campaign_id":"camp-1","session_id":"sess-1"}`})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsPermanent(err) {
		t.Fatalf("expected permanent error, got %v", err)
	}
}
