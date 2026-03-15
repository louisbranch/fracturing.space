package domain

import (
	"context"
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeCampaignAIOrchestrationClient struct {
	requests []*gamev1.QueueAIGMTurnRequest
	err      error
}

func (f *fakeCampaignAIOrchestrationClient) QueueAIGMTurn(_ context.Context, in *gamev1.QueueAIGMTurnRequest, _ ...grpc.CallOption) (*gamev1.QueueAIGMTurnResponse, error) {
	f.requests = append(f.requests, in)
	if f.err != nil {
		return nil, f.err
	}
	return &gamev1.QueueAIGMTurnResponse{}, nil
}

func TestAIGMTurnRequestedHandlerHandleQueuesEligibleTurn(t *testing.T) {
	t.Parallel()

	client := &fakeCampaignAIOrchestrationClient{}
	handler := NewAIGMTurnRequestedHandler(client)

	err := handler.Handle(context.Background(), outboxEventStub{
		payloadJSON: `{"campaign_id":" camp-1 ","session_id":" sess-1 ","source_event_type":" scene.player_phase_review_started ","source_scene_id":" scene-1 ","source_phase_id":" phase-1 "}`,
	})
	if err != nil {
		t.Fatalf("handle error = %v", err)
	}
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d, want 1", len(client.requests))
	}
	req := client.requests[0]
	if req.GetCampaignId() != "camp-1" || req.GetSessionId() != "sess-1" {
		t.Fatalf("request ids = %#v", req)
	}
	if req.GetSourceEventType() != "scene.player_phase_review_started" || req.GetSourceSceneId() != "scene-1" || req.GetSourcePhaseId() != "phase-1" {
		t.Fatalf("request source = %#v", req)
	}
}

func TestAIGMTurnRequestedHandlerHandleRejectsInvalidPayloadPermanently(t *testing.T) {
	t.Parallel()

	handler := NewAIGMTurnRequestedHandler(&fakeCampaignAIOrchestrationClient{})

	err := handler.Handle(context.Background(), outboxEventStub{payloadJSON: `{"campaign_id":"camp-1"}`})
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsPermanent(err) {
		t.Fatalf("expected permanent error, got %v", err)
	}
}

func TestAIGMTurnRequestedHandlerHandleClassifiesOrchestrationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		err           error
		wantPermanent bool
	}{
		{name: "invalid argument", err: status.Error(codes.InvalidArgument, "bad request"), wantPermanent: true},
		{name: "permission denied", err: status.Error(codes.PermissionDenied, "denied"), wantPermanent: true},
		{name: "unauthenticated", err: status.Error(codes.Unauthenticated, "missing"), wantPermanent: true},
		{name: "unavailable", err: status.Error(codes.Unavailable, "retry"), wantPermanent: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler := NewAIGMTurnRequestedHandler(&fakeCampaignAIOrchestrationClient{err: tc.err})
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
