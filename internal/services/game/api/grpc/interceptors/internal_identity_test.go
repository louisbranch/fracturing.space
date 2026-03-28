package interceptors

import (
	"context"
	"testing"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testServerStream struct {
	ctx context.Context
}

func (s testServerStream) SetHeader(metadata.MD) error  { return nil }
func (s testServerStream) SendHeader(metadata.MD) error { return nil }
func (s testServerStream) SetTrailer(metadata.MD)       {}
func (s testServerStream) Context() context.Context     { return s.ctx }
func (s testServerStream) SendMsg(any) error            { return nil }
func (s testServerStream) RecvMsg(any) error            { return nil }

func TestInternalIdentityMethodScoped(t *testing.T) {
	if internalIdentityMethodScoped("", []string{"/game.v1.CampaignAIService/"}) {
		t.Fatal("expected empty full method to be out of scope")
	}
	if internalIdentityMethodScoped("/game.v1.CampaignAIService/GetCampaignAIAuthState", nil) {
		t.Fatal("expected empty prefixes to be out of scope")
	}
	if !internalIdentityMethodScoped("/game.v1.CampaignAIService/GetCampaignAIAuthState", []string{" /game.v1.CampaignAIService/ "}) {
		t.Fatal("expected method to match scoped prefix")
	}
}

func TestValidateInternalServiceIdentity(t *testing.T) {
	err := validateInternalServiceIdentity(context.Background(), nil)
	if status.Code(err) != codes.Internal {
		t.Fatalf("status = %v, want %v", status.Code(err), codes.Internal)
	}

	err = validateInternalServiceIdentity(context.Background(), map[string]struct{}{"chat": {}})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status = %v, want %v", status.Code(err), codes.PermissionDenied)
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ServiceIDHeader, "worker"))
	err = validateInternalServiceIdentity(ctx, map[string]struct{}{"chat": {}})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status = %v, want %v", status.Code(err), codes.PermissionDenied)
	}

	ctx = metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ServiceIDHeader, "CHAT"))
	err = validateInternalServiceIdentity(ctx, map[string]struct{}{"chat": {}})
	if err != nil {
		t.Fatalf("validate identity: %v", err)
	}
}

func TestInternalServiceIdentityUnaryInterceptor(t *testing.T) {
	cfg := InternalServiceIdentityConfig{
		MethodPrefixes:    []string{"/game.v1.CampaignAIService/"},
		AllowedServiceIDs: map[string]struct{}{"chat": {}},
	}
	interceptor := InternalServiceIdentityUnaryInterceptor(cfg)

	handlerCalled := false
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/game.v1.CampaignService/GetCampaign"}, func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unscoped unary call returned error: %v", err)
	}
	if !handlerCalled {
		t.Fatal("expected unscoped call to reach handler")
	}

	handlerCalled = false
	_, err = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/game.v1.CampaignAIService/GetCampaignAIAuthState"}, func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "ok", nil
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status = %v, want %v", status.Code(err), codes.PermissionDenied)
	}
	if handlerCalled {
		t.Fatal("expected scoped call without identity to be rejected before handler")
	}

	allowedCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ServiceIDHeader, "chat"))
	handlerCalled = false
	_, err = interceptor(allowedCtx, nil, &grpc.UnaryServerInfo{FullMethod: "/game.v1.CampaignAIService/GetCampaignAIAuthState"}, func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("scoped unary call returned error: %v", err)
	}
	if !handlerCalled {
		t.Fatal("expected scoped allowed call to reach handler")
	}
}

func TestInternalServiceIdentityStreamInterceptor(t *testing.T) {
	cfg := InternalServiceIdentityConfig{
		MethodPrefixes:    []string{"/game.v1.CampaignAIService/"},
		AllowedServiceIDs: map[string]struct{}{"chat": {}},
	}
	interceptor := InternalServiceIdentityStreamInterceptor(cfg)

	handlerCalled := false
	err := interceptor(nil, testServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/game.v1.CampaignAIService/GetCampaignAIAuthState"}, func(srv any, stream grpc.ServerStream) error {
		handlerCalled = true
		return nil
	})
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status = %v, want %v", status.Code(err), codes.PermissionDenied)
	}
	if handlerCalled {
		t.Fatal("expected scoped stream call without identity to be rejected before handler")
	}

	allowedCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ServiceIDHeader, "chat"))
	handlerCalled = false
	err = interceptor(nil, testServerStream{ctx: allowedCtx}, &grpc.StreamServerInfo{FullMethod: "/game.v1.CampaignAIService/GetCampaignAIAuthState"}, func(srv any, stream grpc.ServerStream) error {
		handlerCalled = true
		return nil
	})
	if err != nil {
		t.Fatalf("scoped stream call returned error: %v", err)
	}
	if !handlerCalled {
		t.Fatal("expected scoped allowed stream call to reach handler")
	}
}
