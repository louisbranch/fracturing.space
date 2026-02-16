package grpcauthctx

import (
	"context"
	"testing"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc/metadata"
)

func TestWithUserIDAppendsMetadataWhenPresent(t *testing.T) {
	ctx := WithUserID(context.Background(), "user-123")
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("expected outgoing metadata context")
	}
	values := md.Get(grpcmeta.UserIDHeader)
	if len(values) != 1 || values[0] != "user-123" {
		t.Fatalf("metadata %s = %v, want [user-123]", grpcmeta.UserIDHeader, values)
	}
}

func TestWithUserIDNoopWhenEmpty(t *testing.T) {
	ctx := WithUserID(context.Background(), "   ")
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok && len(md.Get(grpcmeta.UserIDHeader)) > 0 {
		t.Fatalf("expected no %s metadata, got %v", grpcmeta.UserIDHeader, md.Get(grpcmeta.UserIDHeader))
	}
}

func TestWithParticipantIDAppendsMetadataWhenPresent(t *testing.T) {
	ctx := WithParticipantID(context.Background(), "part-456")
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("expected outgoing metadata context")
	}
	values := md.Get(grpcmeta.ParticipantIDHeader)
	if len(values) != 1 || values[0] != "part-456" {
		t.Fatalf("metadata %s = %v, want [part-456]", grpcmeta.ParticipantIDHeader, values)
	}
}

func TestWithParticipantIDNoopWhenEmpty(t *testing.T) {
	ctx := WithParticipantID(context.Background(), "")
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok && len(md.Get(grpcmeta.ParticipantIDHeader)) > 0 {
		t.Fatalf("expected no %s metadata, got %v", grpcmeta.ParticipantIDHeader, md.Get(grpcmeta.ParticipantIDHeader))
	}
}
