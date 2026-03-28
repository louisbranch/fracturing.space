package requestctx

import (
	"context"
	"testing"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"google.golang.org/grpc/metadata"
)

func TestWithParticipantIDInjectsIncomingMetadata(t *testing.T) {
	ctx := WithParticipantID(context.Background(), "participant-1")
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		t.Fatal("expected incoming metadata")
	}
	if got := md.Get(grpcmeta.ParticipantIDHeader); len(got) != 1 || got[0] != "participant-1" {
		t.Fatalf("participant header = %v, want [participant-1]", got)
	}
}

func TestWithUserIDInjectsIncomingMetadata(t *testing.T) {
	ctx := WithUserID(context.Background(), "user-1")
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		t.Fatal("expected incoming metadata")
	}
	if got := md.Get(grpcmeta.UserIDHeader); len(got) != 1 || got[0] != "user-1" {
		t.Fatalf("user header = %v, want [user-1]", got)
	}
}

func TestEmptyHelpersReturnBackgroundWithoutIncomingMetadata(t *testing.T) {
	if ctx := WithParticipantID(context.Background(), ""); ctx != context.Background() {
		t.Fatal("expected WithParticipantID(\"\") to return context.Background()")
	}
	if ctx := WithUserID(context.Background(), ""); ctx != context.Background() {
		t.Fatal("expected WithUserID(\"\") to return context.Background()")
	}
	if _, ok := metadata.FromIncomingContext(context.Background()); ok {
		t.Fatal("expected bare background context to have no incoming metadata")
	}
}

func TestWithAdminOverrideInjectsDefaultAndTrimmedMetadata(t *testing.T) {
	ctx := WithAdminOverride(context.Background(), "  reason-1  ")
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		t.Fatal("expected incoming metadata")
	}
	if got := md.Get(grpcmeta.PlatformRoleHeader); len(got) != 1 || got[0] != grpcmeta.PlatformRoleAdmin {
		t.Fatalf("platform role header = %v, want [%s]", got, grpcmeta.PlatformRoleAdmin)
	}
	if got := md.Get(grpcmeta.AuthzOverrideReasonHeader); len(got) != 1 || got[0] != "reason-1" {
		t.Fatalf("override reason header = %v, want [reason-1]", got)
	}
	if got := md.Get(grpcmeta.UserIDHeader); len(got) != 1 || got[0] != "user-admin-test" {
		t.Fatalf("user header = %v, want [user-admin-test]", got)
	}

	defaultCtx := WithAdminOverride(context.Background(), "   ")
	defaultMD, ok := metadata.FromIncomingContext(defaultCtx)
	if !ok {
		t.Fatal("expected incoming metadata for default override")
	}
	if got := defaultMD.Get(grpcmeta.AuthzOverrideReasonHeader); len(got) != 1 || got[0] != "test-override" {
		t.Fatalf("default override reason header = %v, want [test-override]", got)
	}
}
