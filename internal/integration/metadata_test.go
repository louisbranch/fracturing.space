//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/duality/v1"
	"github.com/louisbranch/fracturing.space/internal/grpcmeta"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func runMetadataTests(t *testing.T, suite *integrationSuite, grpcAddr string) {
	t.Helper()

	t.Run("mcp tool metadata", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
		defer cancel()

		result, err := suite.client.CallTool(ctx, &mcp.CallToolParams{Name: "duality_rules_version"})
		if err != nil {
			t.Fatalf("call duality_rules_version: %v", err)
		}
		if result == nil {
			t.Fatal("call duality_rules_version returned nil")
		}
		if result.IsError {
			t.Fatalf("call duality_rules_version returned error content: %+v", result.Content)
		}
		requestID, invocationID := requireResultMetadata(t, result)
		if !grpcmeta.IsPrintableASCII(requestID) {
			t.Fatalf("request id must be printable ASCII, got %q", requestID)
		}
		if !grpcmeta.IsPrintableASCII(invocationID) {
			t.Fatalf("invocation id must be printable ASCII, got %q", invocationID)
		}
	})

	t.Run("grpc headers", func(t *testing.T) {
		conn, err := grpc.NewClient(
			grpcAddr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
		)
		if err != nil {
			t.Fatalf("dial gRPC: %v", err)
		}
		defer conn.Close()

		client := pb.NewDualityServiceClient(conn)
		requestID := "req-abc"
		invocationID := "inv-xyz"
		callCtx := metadata.NewOutgoingContext(
			context.Background(),
			metadata.Pairs(
				grpcmeta.RequestIDHeader, requestID,
				grpcmeta.InvocationIDHeader, invocationID,
			),
		)

		callTimeout, callCancel := context.WithTimeout(callCtx, 2*time.Second)
		var header metadata.MD
		_, err = client.ActionRoll(callTimeout, &pb.ActionRollRequest{}, grpc.Header(&header))
		callCancel()
		if err != nil {
			t.Fatalf("action roll: %v", err)
		}
		assertHeaderValue(t, header, grpcmeta.RequestIDHeader, requestID)
		assertHeaderValue(t, header, grpcmeta.InvocationIDHeader, invocationID)

		callTimeout, callCancel = context.WithTimeout(context.Background(), 2*time.Second)
		var generated metadata.MD
		_, err = client.ActionRoll(callTimeout, &pb.ActionRollRequest{}, grpc.Header(&generated))
		callCancel()
		if err != nil {
			t.Fatalf("action roll without headers: %v", err)
		}
		if len(generated.Get(grpcmeta.RequestIDHeader)) != 1 {
			t.Fatal("expected generated request id header")
		}
	})
}

func requireResultMetadata(t *testing.T, result *mcp.CallToolResult) (string, string) {
	t.Helper()
	if result.Meta == nil {
		t.Fatal("expected response metadata")
	}
	requestID, _ := result.Meta[grpcmeta.RequestIDHeader].(string)
	if requestID == "" {
		t.Fatal("expected request id metadata")
	}
	invocationID, _ := result.Meta[grpcmeta.InvocationIDHeader].(string)
	if invocationID == "" {
		t.Fatal("expected invocation id metadata")
	}
	return requestID, invocationID
}

func assertHeaderValue(t *testing.T, header metadata.MD, key, expected string) {
	t.Helper()
	values := header.Get(key)
	if len(values) != 1 || values[0] != expected {
		t.Fatalf("header %s = %v, want %q", key, values, expected)
	}
}
