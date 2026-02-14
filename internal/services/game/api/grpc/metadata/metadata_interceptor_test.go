package metadata

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
	md  metadata.MD
}

func (f *fakeServerStream) Context() context.Context {
	if f.ctx == nil {
		return context.Background()
	}
	return f.ctx
}

func (f *fakeServerStream) SetHeader(md metadata.MD) error {
	f.md = md
	return nil
}

type fakeTransportStream struct {
	md metadata.MD
}

func (f *fakeTransportStream) Method() string {
	return "game.v1.Test/Unary"
}

func (f *fakeTransportStream) SetHeader(md metadata.MD) error {
	f.md = md
	return nil
}

func (f *fakeTransportStream) SendHeader(md metadata.MD) error {
	f.md = md
	return nil
}

func (f *fakeTransportStream) SetTrailer(md metadata.MD) error {
	return nil
}

func TestUnaryServerInterceptorAddsHeaders(t *testing.T) {
	baseCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		RequestIDHeader, "req-1",
		InvocationIDHeader, "inv-1",
	))
	transport := &fakeTransportStream{}
	ctx := grpc.NewContextWithServerTransportStream(baseCtx, transport)

	interceptor := UnaryServerInterceptor(func() (string, error) {
		return "gen", nil
	})
	info := &grpc.UnaryServerInfo{FullMethod: "game.v1.Test/Unary"}

	_, err := interceptor(ctx, nil, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if FirstMetadataValue(transport.md, RequestIDHeader) != "req-1" {
		t.Fatalf("expected request id header, got %v", transport.md)
	}
	if FirstMetadataValue(transport.md, InvocationIDHeader) != "inv-1" {
		t.Fatalf("expected invocation id header, got %v", transport.md)
	}
}

func TestUnaryServerInterceptorGeneratorError(t *testing.T) {
	interceptor := UnaryServerInterceptor(func() (string, error) {
		return "", errors.New("boom")
	})
	info := &grpc.UnaryServerInfo{FullMethod: "game.v1.Test/Unary"}

	_, err := interceptor(context.Background(), nil, info, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error, got %v", err)
	}
}

func TestStreamServerInterceptorAddsHeaders(t *testing.T) {
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		RequestIDHeader, "req-1",
	))
	stream := &fakeServerStream{ctx: ctx}
	interceptor := StreamServerInterceptor(func() (string, error) {
		return "gen", nil
	})
	info := &grpc.StreamServerInfo{FullMethod: "game.v1.Test/Stream"}

	err := interceptor(nil, stream, info, func(srv any, stream grpc.ServerStream) error {
		if RequestIDFromContext(stream.Context()) != "req-1" {
			return status.Error(codes.Internal, "missing request id")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if FirstMetadataValue(stream.md, RequestIDHeader) != "req-1" {
		t.Fatalf("expected response header request id, got %v", stream.md)
	}
}

func TestStreamServerInterceptorGeneratorError(t *testing.T) {
	stream := &fakeServerStream{ctx: context.Background()}
	interceptor := StreamServerInterceptor(func() (string, error) {
		return "", errors.New("boom")
	})
	info := &grpc.StreamServerInfo{FullMethod: "game.v1.Test/Stream"}

	err := interceptor(nil, stream, info, func(srv any, stream grpc.ServerStream) error {
		return nil
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("expected internal error, got %v", err)
	}
}
