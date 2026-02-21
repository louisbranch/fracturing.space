package grpcauthctx

import (
	"context"
	"testing"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
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

func TestWithAdminOverrideAppendsMetadataWhenReasonPresent(t *testing.T) {
	ctx := WithAdminOverride(context.Background(), "admin_dashboard")
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("expected outgoing metadata context")
	}
	roleValues := md.Get(grpcmeta.PlatformRoleHeader)
	if len(roleValues) != 1 || roleValues[0] != grpcmeta.PlatformRoleAdmin {
		t.Fatalf("metadata %s = %v, want [%s]", grpcmeta.PlatformRoleHeader, roleValues, grpcmeta.PlatformRoleAdmin)
	}
	reasonValues := md.Get(grpcmeta.AuthzOverrideReasonHeader)
	if len(reasonValues) != 1 || reasonValues[0] != "admin_dashboard" {
		t.Fatalf("metadata %s = %v, want [admin_dashboard]", grpcmeta.AuthzOverrideReasonHeader, reasonValues)
	}
}

func TestWithAdminOverrideNoopWhenReasonEmpty(t *testing.T) {
	ctx := WithAdminOverride(context.Background(), "   ")
	md, ok := metadata.FromOutgoingContext(ctx)
	if ok && len(md.Get(grpcmeta.PlatformRoleHeader)) > 0 {
		t.Fatalf("expected no %s metadata, got %v", grpcmeta.PlatformRoleHeader, md.Get(grpcmeta.PlatformRoleHeader))
	}
}

func TestAdminOverrideUnaryClientInterceptorAddsMetadata(t *testing.T) {
	interceptor := AdminOverrideUnaryClientInterceptor("mcp_service")

	err := interceptor(
		context.Background(),
		"/game.v1.CampaignService/ListCampaigns",
		nil,
		nil,
		nil,
		func(ctx context.Context, _ string, _, _ any, _ *grpc.ClientConn, _ ...grpc.CallOption) error {
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				t.Fatalf("expected outgoing metadata context")
			}
			if got := md.Get(grpcmeta.PlatformRoleHeader); len(got) != 1 || got[0] != grpcmeta.PlatformRoleAdmin {
				t.Fatalf("metadata %s = %v, want [%s]", grpcmeta.PlatformRoleHeader, got, grpcmeta.PlatformRoleAdmin)
			}
			if got := md.Get(grpcmeta.AuthzOverrideReasonHeader); len(got) != 1 || got[0] != "mcp_service" {
				t.Fatalf("metadata %s = %v, want [mcp_service]", grpcmeta.AuthzOverrideReasonHeader, got)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
}

func TestAdminOverrideStreamClientInterceptorAddsMetadata(t *testing.T) {
	interceptor := AdminOverrideStreamClientInterceptor("admin_dashboard")

	_, err := interceptor(
		context.Background(),
		&grpc.StreamDesc{},
		nil,
		"/game.v1.CampaignService/ListCampaigns",
		func(ctx context.Context, _ *grpc.StreamDesc, _ *grpc.ClientConn, _ string, _ ...grpc.CallOption) (grpc.ClientStream, error) {
			md, ok := metadata.FromOutgoingContext(ctx)
			if !ok {
				t.Fatalf("expected outgoing metadata context")
			}
			if got := md.Get(grpcmeta.PlatformRoleHeader); len(got) != 1 || got[0] != grpcmeta.PlatformRoleAdmin {
				t.Fatalf("metadata %s = %v, want [%s]", grpcmeta.PlatformRoleHeader, got, grpcmeta.PlatformRoleAdmin)
			}
			if got := md.Get(grpcmeta.AuthzOverrideReasonHeader); len(got) != 1 || got[0] != "admin_dashboard" {
				t.Fatalf("metadata %s = %v, want [admin_dashboard]", grpcmeta.AuthzOverrideReasonHeader, got)
			}
			return nil, nil
		},
	)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
}
