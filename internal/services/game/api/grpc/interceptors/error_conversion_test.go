package interceptors

import (
	"context"
	"errors"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestErrorConversionUnaryPassesThroughNil(t *testing.T) {
	interceptor := ErrorConversionUnaryInterceptor()
	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "ok" {
		t.Fatalf("resp = %v, want ok", resp)
	}
}

func TestErrorConversionUnaryPassesThroughGRPCStatus(t *testing.T) {
	want := status.Error(codes.NotFound, "missing")
	interceptor := ErrorConversionUnaryInterceptor()
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		return nil, want
	})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("code = %s, want %s", status.Code(err), codes.NotFound)
	}
}

func TestErrorConversionUnaryConvertsDomainError(t *testing.T) {
	domainErr := apperrors.New(apperrors.CodeCharacterEmptyName, "name required")
	interceptor := ErrorConversionUnaryInterceptor()
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		return nil, domainErr
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}
}

func TestErrorConversionUnaryWrapsUnknownAsInternal(t *testing.T) {
	interceptor := ErrorConversionUnaryInterceptor()
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req any) (any, error) {
		return nil, errors.New("unexpected")
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %s, want %s", status.Code(err), codes.Internal)
	}
}

func TestErrorConversionStreamPassesThroughNil(t *testing.T) {
	interceptor := ErrorConversionStreamInterceptor()
	err := interceptor(nil, nil, &grpc.StreamServerInfo{}, func(srv any, stream grpc.ServerStream) error {
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestErrorConversionStreamConvertsDomainError(t *testing.T) {
	domainErr := apperrors.New(apperrors.CodeCharacterEmptyName, "name required")
	interceptor := ErrorConversionStreamInterceptor()
	err := interceptor(nil, nil, &grpc.StreamServerInfo{}, func(srv any, stream grpc.ServerStream) error {
		return domainErr
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %s, want %s", status.Code(err), codes.InvalidArgument)
	}
}
