package interceptors

import (
	"context"
	"testing"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeTelemetryStore struct {
	last  storage.TelemetryEvent
	count int
	err   error
}

func (s *fakeTelemetryStore) AppendTelemetryEvent(ctx context.Context, evt storage.TelemetryEvent) error {
	s.last = evt
	s.count++
	return s.err
}

type campaignRequest struct {
	campaignID string
	sessionID  string
}

func (r campaignRequest) GetCampaignId() string {
	return r.campaignID
}

func (r campaignRequest) GetSessionId() string {
	return r.sessionID
}

type sourceCampaignRequest struct {
	sourceCampaignID string
	sessionID        string
}

func (r sourceCampaignRequest) GetSourceCampaignId() string {
	return r.sourceCampaignID
}

func (r sourceCampaignRequest) GetSessionId() string {
	return r.sessionID
}

func TestIsReadMethod(t *testing.T) {
	if !isReadMethod(campaignv1.CampaignService_GetCampaign_FullMethodName) {
		t.Fatal("expected get campaign to be read method")
	}
	if isReadMethod(campaignv1.CampaignService_CreateCampaign_FullMethodName) {
		t.Fatal("expected create campaign to be write method")
	}
}

func TestExtractScope(t *testing.T) {
	campaignID, sessionID := extractScope(campaignRequest{campaignID: " camp ", sessionID: " sess "})
	if campaignID != "camp" || sessionID != "sess" {
		t.Fatalf("expected trimmed scope, got %q/%q", campaignID, sessionID)
	}

	campaignID, sessionID = extractScope(sourceCampaignRequest{sourceCampaignID: " source ", sessionID: " sess "})
	if campaignID != "source" || sessionID != "sess" {
		t.Fatalf("expected source campaign scope, got %q/%q", campaignID, sessionID)
	}

	campaignID, sessionID = extractScope(nil)
	if campaignID != "" || sessionID != "" {
		t.Fatal("expected empty scope for nil request")
	}
}

func TestTelemetryInterceptorNoStore(t *testing.T) {
	interceptor := TelemetryInterceptor(nil)
	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_GetCampaign_FullMethodName}
	called := false

	_, err := interceptor(context.Background(), &campaignRequest{}, info, func(ctx context.Context, req any) (any, error) {
		called = true
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected handler to be called")
	}
}

func TestTelemetryInterceptorEmitsEvent(t *testing.T) {
	store := &fakeTelemetryStore{}
	interceptor := TelemetryInterceptor(store)
	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_GetCampaign_FullMethodName}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		grpcmeta.ParticipantIDHeader, "participant-1",
	))
	ctx = grpcmeta.WithRequestID(ctx, "req-1")
	ctx = grpcmeta.WithInvocationID(ctx, "inv-1")

	_, err := interceptor(ctx, &campaignRequest{campaignID: "camp-1", sessionID: "sess-1"}, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.count != 1 {
		t.Fatalf("expected event to be emitted, got %d", store.count)
	}
	if store.last.EventName != "telemetry.grpc.read" {
		t.Fatalf("expected event name telemetry.grpc.read, got %s", store.last.EventName)
	}
	if store.last.ActorType != "participant" || store.last.ActorID != "participant-1" {
		t.Fatalf("expected participant actor, got %s/%s", store.last.ActorType, store.last.ActorID)
	}
	if store.last.CampaignID != "camp-1" || store.last.SessionID != "sess-1" {
		t.Fatalf("expected scope camp-1/sess-1, got %s/%s", store.last.CampaignID, store.last.SessionID)
	}
	if store.last.RequestID != "req-1" || store.last.InvocationID != "inv-1" {
		t.Fatalf("expected request/invocation ids, got %s/%s", store.last.RequestID, store.last.InvocationID)
	}
}

func TestTelemetryInterceptorErrorSeverity(t *testing.T) {
	store := &fakeTelemetryStore{}
	interceptor := TelemetryInterceptor(store)
	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_GetCampaign_FullMethodName}

	_, err := interceptor(context.Background(), &campaignRequest{campaignID: "camp-1"}, info, func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.NotFound, "missing")
	})
	if err == nil {
		t.Fatal("expected handler error")
	}
	if store.count != 1 {
		t.Fatalf("expected event to be emitted, got %d", store.count)
	}
	if store.last.Severity != "ERROR" {
		t.Fatalf("expected error severity, got %s", store.last.Severity)
	}
	if store.last.Attributes["code"] != codes.NotFound.String() {
		t.Fatalf("expected code NotFound, got %v", store.last.Attributes["code"])
	}
}

func TestTelemetryInterceptorWriteMethodNoEmit(t *testing.T) {
	store := &fakeTelemetryStore{}
	interceptor := TelemetryInterceptor(store)
	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_CreateCampaign_FullMethodName}

	_, err := interceptor(context.Background(), &campaignRequest{campaignID: "camp-1"}, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.count != 0 {
		t.Fatalf("expected no telemetry emit, got %d", store.count)
	}
}

func TestTelemetryInterceptorSystemActor(t *testing.T) {
	store := &fakeTelemetryStore{}
	interceptor := TelemetryInterceptor(store)
	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_GetCampaign_FullMethodName}

	_, err := interceptor(context.Background(), &campaignRequest{campaignID: "camp-1"}, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.count != 1 {
		t.Fatalf("expected event to be emitted, got %d", store.count)
	}
	if store.last.ActorType != "system" || store.last.ActorID != "" {
		t.Fatalf("expected system actor, got %s/%s", store.last.ActorType, store.last.ActorID)
	}
}

func TestTelemetryInterceptorStoreErrorIgnored(t *testing.T) {
	store := &fakeTelemetryStore{err: context.Canceled}
	interceptor := TelemetryInterceptor(store)
	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_GetCampaign_FullMethodName}

	_, err := interceptor(context.Background(), &campaignRequest{campaignID: "camp-1"}, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.count != 1 {
		t.Fatalf("expected event to be emitted, got %d", store.count)
	}
}
