package metadata

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestRequestIDContextHelpers(t *testing.T) {
	if RequestIDFromContext(nil) != "" {
		t.Fatal("expected empty request id for nil context")
	}

	ctx := WithRequestID(nil, "req-1")
	if RequestIDFromContext(ctx) != "req-1" {
		t.Fatalf("expected request id req-1, got %s", RequestIDFromContext(ctx))
	}
}

func TestInvocationIDContextHelpers(t *testing.T) {
	if InvocationIDFromContext(nil) != "" {
		t.Fatal("expected empty invocation id for nil context")
	}

	ctx := WithInvocationID(nil, "inv-1")
	if InvocationIDFromContext(ctx) != "inv-1" {
		t.Fatalf("expected invocation id inv-1, got %s", InvocationIDFromContext(ctx))
	}
}

func TestIsPrintableASCII(t *testing.T) {
	if IsPrintableASCII("") {
		t.Fatal("expected empty string to be non-printable")
	}
	if IsPrintableASCII("hello") != true {
		t.Fatal("expected printable ascii to be accepted")
	}
	if IsPrintableASCII("line\n") {
		t.Fatal("expected newline to be non-printable")
	}
	if IsPrintableASCII(string([]byte{0x7f})) {
		t.Fatal("expected DEL to be non-printable")
	}
}

func TestFirstMetadataValue(t *testing.T) {
	md := metadata.MD{
		"X-Fracturing-Space-Request-Id": {"\n", "req-1"},
		"x-fracturing-space-request-id": {"req-2"},
	}

	value := FirstMetadataValue(md, RequestIDHeader)
	if value != "req-1" && value != "req-2" {
		t.Fatalf("expected printable request id, got %s", value)
	}

	if FirstMetadataValue(metadata.MD{}, RequestIDHeader) != "" {
		t.Fatal("expected empty value for empty metadata")
	}
}

func TestIncomingMetadataAccessors(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		ParticipantIDHeader, "participant-1",
		UserIDHeader, "user-1",
		CampaignIDHeader, "campaign-1",
		SessionIDHeader, "session-1",
	))

	if ParticipantIDFromContext(ctx) != "participant-1" {
		t.Fatal("expected participant id from metadata")
	}
	if UserIDFromContext(ctx) != "user-1" {
		t.Fatal("expected user id from metadata")
	}
	if CampaignIDFromContext(ctx) != "campaign-1" {
		t.Fatal("expected campaign id from metadata")
	}
	if SessionIDFromContext(ctx) != "session-1" {
		t.Fatal("expected session id from metadata")
	}
}

func TestEnsureRequestMetadata(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		RequestIDHeader, "req-1",
		InvocationIDHeader, "inv-1",
	))

	updated, requestID, invocationID, err := ensureRequestMetadata(ctx, func() (string, error) {
		return "generated", nil
	})
	if err != nil {
		t.Fatalf("ensure request metadata: %v", err)
	}
	if requestID != "req-1" || invocationID != "inv-1" {
		t.Fatalf("expected ids from metadata, got %s/%s", requestID, invocationID)
	}
	if RequestIDFromContext(updated) != "req-1" {
		t.Fatal("expected request id stored in context")
	}
	if InvocationIDFromContext(updated) != "inv-1" {
		t.Fatal("expected invocation id stored in context")
	}
}

func TestEnsureRequestMetadataGeneratesID(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})

	updated, requestID, invocationID, err := ensureRequestMetadata(ctx, func() (string, error) {
		return "generated", nil
	})
	if err != nil {
		t.Fatalf("ensure request metadata: %v", err)
	}
	if requestID != "generated" || invocationID != "" {
		t.Fatalf("expected generated request id, got %s/%s", requestID, invocationID)
	}
	if RequestIDFromContext(updated) != "generated" {
		t.Fatal("expected generated request id stored in context")
	}
}

func TestEnsureRequestMetadataGeneratorFailure(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.MD{})

	_, _, _, err := ensureRequestMetadata(ctx, func() (string, error) {
		return "", errors.New("boom")
	})
	if err == nil {
		t.Fatal("expected generator error")
	}
}

func TestResponseHeaders(t *testing.T) {
	md := responseHeaders("req-1", "")
	if FirstMetadataValue(md, RequestIDHeader) != "req-1" {
		t.Fatal("expected request id in response headers")
	}
	if FirstMetadataValue(md, InvocationIDHeader) != "" {
		t.Fatal("expected empty invocation id when missing")
	}

	md = responseHeaders("req-1", "inv-1")
	if FirstMetadataValue(md, InvocationIDHeader) != "inv-1" {
		t.Fatal("expected invocation id in response headers")
	}
}
