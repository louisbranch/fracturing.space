package interceptors

import (
	"context"
	"testing"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit/events"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeAuditStore struct {
	last  storage.AuditEvent
	count int
	err   error
}

func (s *fakeAuditStore) AppendAuditEvent(ctx context.Context, evt storage.AuditEvent) error {
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

func TestClassifyMethodKind(t *testing.T) {
	if classifyMethodKind(campaignv1.CampaignService_GetCampaign_FullMethodName) != "read" {
		t.Fatal("expected get campaign to be read method")
	}
	if classifyMethodKind(campaignv1.CampaignService_CreateCampaign_FullMethodName) != "write" {
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

func TestAuditInterceptorNoStore(t *testing.T) {
	interceptor := AuditInterceptor(nil)
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

func TestAuditInterceptorEmitsEventForRead(t *testing.T) {
	store := &fakeAuditStore{}
	interceptor := AuditInterceptor(store)
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
	if store.last.EventName != events.GRPCRead {
		t.Fatalf("expected event name %s, got %s", events.GRPCRead, store.last.EventName)
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

func TestAuditInterceptorErrorSeverity(t *testing.T) {
	store := &fakeAuditStore{}
	interceptor := AuditInterceptor(store)
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

func TestAuditInterceptorEmitsEventForWrite(t *testing.T) {
	store := &fakeAuditStore{}
	interceptor := AuditInterceptor(store)
	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_CreateCampaign_FullMethodName}

	_, err := interceptor(context.Background(), &campaignRequest{campaignID: "camp-1"}, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.count != 1 {
		t.Fatalf("expected one audit emit, got %d", store.count)
	}
	if store.last.EventName != events.GRPCWrite {
		t.Fatalf("expected event name %s, got %s", events.GRPCWrite, store.last.EventName)
	}
	if got, ok := store.last.Attributes["method_kind"].(string); !ok || got != "write" {
		t.Fatalf("expected method_kind write, got %#v", store.last.Attributes["method_kind"])
	}
}

func TestAuditInterceptorSystemActor(t *testing.T) {
	store := &fakeAuditStore{}
	interceptor := AuditInterceptor(store)
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

func TestAuditInterceptorOTelTraceContext(t *testing.T) {
	store := &fakeAuditStore{}
	interceptor := AuditInterceptor(store)
	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_GetCampaign_FullMethodName}

	// Create an OTel span so the context carries a valid trace/span ID.
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	defer func() { _ = tp.Shutdown(context.Background()) }()

	ctx, span := tp.Tracer("test").Start(context.Background(), "test-span")
	defer span.End()

	sc := trace.SpanFromContext(ctx).SpanContext()
	wantTraceID := sc.TraceID().String()
	wantSpanID := sc.SpanID().String()

	_, err := interceptor(ctx, &campaignRequest{campaignID: "camp-1"}, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.last.TraceID != wantTraceID {
		t.Fatalf("expected trace_id %s, got %s", wantTraceID, store.last.TraceID)
	}
	if store.last.SpanID != wantSpanID {
		t.Fatalf("expected span_id %s, got %s", wantSpanID, store.last.SpanID)
	}
}

func TestAuditInterceptorNoSpanEmptyIDs(t *testing.T) {
	store := &fakeAuditStore{}
	interceptor := AuditInterceptor(store)
	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_GetCampaign_FullMethodName}

	// No OTel span in context — trace/span IDs should remain empty.
	_, err := interceptor(context.Background(), &campaignRequest{campaignID: "camp-1"}, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.last.TraceID != "" {
		t.Fatalf("expected empty trace_id, got %s", store.last.TraceID)
	}
	if store.last.SpanID != "" {
		t.Fatalf("expected empty span_id, got %s", store.last.SpanID)
	}
}

func TestAuditInterceptorStoreErrorIgnored(t *testing.T) {
	store := &fakeAuditStore{err: context.Canceled}
	interceptor := AuditInterceptor(store)
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
