// Package service tests the MCP server wiring.
package service

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	campaignpb "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	pb "github.com/louisbranch/duality-engine/api/gen/go/duality/v1"
	"github.com/louisbranch/duality-engine/internal/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// fakeDualityClient implements DualityServiceClient for tests.
type fakeDualityClient struct {
	response                      *pb.ActionRollResponse
	rollDiceResponse              *pb.RollDiceResponse
	dualityOutcomeResponse        *pb.DualityOutcomeResponse
	dualityExplainResponse        *pb.DualityExplainResponse
	dualityProbabilityResponse    *pb.DualityProbabilityResponse
	rulesVersionResponse          *pb.RulesVersionResponse
	err                           error
	rollDiceErr                   error
	dualityOutcomeErr             error
	dualityExplainErr             error
	dualityProbabilityErr         error
	rulesVersionErr               error
	lastRequest                   *pb.ActionRollRequest
	lastRollDiceRequest           *pb.RollDiceRequest
	lastDualityOutcomeRequest     *pb.DualityOutcomeRequest
	lastDualityExplainRequest     *pb.DualityExplainRequest
	lastDualityProbabilityRequest *pb.DualityProbabilityRequest
	lastRulesVersionRequest       *pb.RulesVersionRequest
}

// fakeCampaignClient implements CampaignServiceClient for tests.
type fakeCampaignClient struct {
	response    *campaignpb.CreateCampaignResponse
	err         error
	lastRequest *campaignpb.CreateCampaignRequest
}

// failingTransport returns a connection error for tests.
type failingTransport struct{}

// Connect returns the configured error for tests.
func (f failingTransport) Connect(context.Context) (mcp.Connection, error) {
	return nil, errors.New("transport failure")
}

// ActionRoll records the request and returns the configured response.
func (f *fakeDualityClient) ActionRoll(ctx context.Context, req *pb.ActionRollRequest, opts ...grpc.CallOption) (*pb.ActionRollResponse, error) {
	f.lastRequest = req
	return f.response, f.err
}

// DualityOutcome records the request and returns the configured response.
func (f *fakeDualityClient) DualityOutcome(ctx context.Context, req *pb.DualityOutcomeRequest, opts ...grpc.CallOption) (*pb.DualityOutcomeResponse, error) {
	f.lastDualityOutcomeRequest = req
	return f.dualityOutcomeResponse, f.dualityOutcomeErr
}

// DualityExplain records the request and returns the configured response.
func (f *fakeDualityClient) DualityExplain(ctx context.Context, req *pb.DualityExplainRequest, opts ...grpc.CallOption) (*pb.DualityExplainResponse, error) {
	f.lastDualityExplainRequest = req
	return f.dualityExplainResponse, f.dualityExplainErr
}

// DualityProbability records the request and returns the configured response.
func (f *fakeDualityClient) DualityProbability(ctx context.Context, req *pb.DualityProbabilityRequest, opts ...grpc.CallOption) (*pb.DualityProbabilityResponse, error) {
	f.lastDualityProbabilityRequest = req
	return f.dualityProbabilityResponse, f.dualityProbabilityErr
}

// RulesVersion records the request and returns the configured response.
func (f *fakeDualityClient) RulesVersion(ctx context.Context, req *pb.RulesVersionRequest, opts ...grpc.CallOption) (*pb.RulesVersionResponse, error) {
	f.lastRulesVersionRequest = req
	return f.rulesVersionResponse, f.rulesVersionErr
}

// RollDice records the request and returns the configured response.
func (f *fakeDualityClient) RollDice(ctx context.Context, req *pb.RollDiceRequest, opts ...grpc.CallOption) (*pb.RollDiceResponse, error) {
	f.lastRollDiceRequest = req
	return f.rollDiceResponse, f.rollDiceErr
}

// CreateCampaign records the request and returns the configured response.
func (f *fakeCampaignClient) CreateCampaign(ctx context.Context, req *campaignpb.CreateCampaignRequest, opts ...grpc.CallOption) (*campaignpb.CreateCampaignResponse, error) {
	f.lastRequest = req
	return f.response, f.err
}

// TestGRPCAddressPrefersEnv ensures env configuration overrides defaults.
func TestGRPCAddressPrefersEnv(t *testing.T) {
	t.Setenv("DUALITY_GRPC_ADDR", "env:123")
	if got := grpcAddress("fallback"); got != "env:123" {
		t.Fatalf("expected env address, got %q", got)
	}
}

// TestGRPCAddressFallback ensures the fallback address is used when env is empty.
func TestGRPCAddressFallback(t *testing.T) {
	t.Setenv("DUALITY_GRPC_ADDR", "")
	if got := grpcAddress("fallback"); got != "fallback" {
		t.Fatalf("expected fallback address, got %q", got)
	}
}

// TestServeRequiresConfiguredServer ensures Serve returns an error when unconfigured.
func TestServeRequiresConfiguredServer(t *testing.T) {
	tests := []struct {
		name   string
		server *Server
	}{
		{name: "nil server", server: nil},
		{name: "missing mcp server", server: &Server{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.server.Serve(context.Background()); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

// TestServeStopsOnContext ensures Serve exits when the context is cancelled.
func TestServeStopsOnContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, err := New("localhost:8080")
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.serveWithTransport(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	clientCtx, clientCancel := context.WithTimeout(context.Background(), time.Second)
	defer clientCancel()
	clientSession, err := client.Connect(clientCtx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	defer clientSession.Close()

	cancel()

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("serve returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after cancel")
	}
}

// TestRunStopsOnContext ensures Run exits when the context is cancelled.
func TestRunStopsOnContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- runWithTransport(ctx, addr, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{Name: "client", Version: "v0.0.1"}, nil)
	clientCtx, clientCancel := context.WithTimeout(context.Background(), time.Second)
	defer clientCancel()
	clientSession, err := client.Connect(clientCtx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	defer clientSession.Close()

	cancel()

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("run did not stop after cancel")
	}
}

// TestRunReturnsTransportError ensures Run reports transport failures.
func TestRunReturnsTransportError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	if err := runWithTransport(ctx, addr, failingTransport{}); err == nil {
		t.Fatal("expected transport error")
	}
}

// TestNewConfiguresServer ensures New returns a configured server.
func TestNewConfiguresServer(t *testing.T) {
	server, err := New("localhost:8080")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if server == nil || server.mcpServer == nil {
		t.Fatal("expected configured server")
	}
}

// TestActionRollHandlerPassesNegativeDifficulty ensures gRPC receives invalid difficulty.
func TestActionRollHandlerPassesNegativeDifficulty(t *testing.T) {
	client := &fakeDualityClient{err: errors.New("boom")}
	handler := domain.ActionRollHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.ActionRollInput{
		Modifier:   1,
		Difficulty: intPointer(-1),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if client.lastRequest == nil {
		t.Fatal("expected gRPC call on invalid input")
	}
	if client.lastRequest.Difficulty == nil || *client.lastRequest.Difficulty != -1 {
		t.Fatalf("expected difficulty -1, got %v", client.lastRequest.Difficulty)
	}
}

// TestActionRollHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestActionRollHandlerReturnsClientError(t *testing.T) {
	client := &fakeDualityClient{err: errors.New("boom")}
	handler := domain.ActionRollHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.ActionRollInput{Modifier: 2})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestActionRollHandlerMapsRequestAndResponse ensures inputs and outputs are mapped consistently.
func TestActionRollHandlerMapsRequestAndResponse(t *testing.T) {
	difficulty := int32(7)
	client := &fakeDualityClient{
		response: &pb.ActionRollResponse{
			Hope:            4,
			Fear:            6,
			Modifier:        7,
			Total:           17,
			IsCrit:          false,
			MeetsDifficulty: true,
			Difficulty:      &difficulty,
			Outcome:         pb.Outcome_SUCCESS_WITH_FEAR,
		},
	}
	handler := domain.ActionRollHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.ActionRollInput{
		Modifier:   7,
		Difficulty: intPointer(7),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != nil {
		t.Fatal("expected nil result on success")
	}
	if client.lastRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastRequest.Modifier != 7 {
		t.Fatalf("expected modifier 7, got %d", client.lastRequest.Modifier)
	}
	if client.lastRequest.Difficulty == nil || *client.lastRequest.Difficulty != 7 {
		t.Fatalf("expected difficulty 7, got %v", client.lastRequest.Difficulty)
	}

	if output.Hope != 4 || output.Fear != 6 || output.Total != 17 {
		t.Fatalf("unexpected dice output: %+v", output)
	}
	if output.Modifier != 7 {
		t.Fatalf("expected modifier 7, got %d", output.Modifier)
	}
	if output.IsCrit {
		t.Fatal("expected is_crit false")
	}
	if !output.MeetsDifficulty {
		t.Fatal("expected meets_difficulty true")
	}
	if output.Outcome != pb.Outcome_SUCCESS_WITH_FEAR.String() {
		t.Fatalf("expected outcome %q, got %q", pb.Outcome_SUCCESS_WITH_FEAR.String(), output.Outcome)
	}
	if output.Difficulty == nil || *output.Difficulty != 7 {
		t.Fatalf("expected difficulty 7, got %v", output.Difficulty)
	}
}

// TestDualityOutcomeHandlerPassesInvalidDice ensures gRPC receives invalid dice.
func TestDualityOutcomeHandlerPassesInvalidDice(t *testing.T) {
	client := &fakeDualityClient{dualityOutcomeErr: errors.New("boom")}
	handler := domain.DualityOutcomeHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.DualityOutcomeInput{
		Hope: 0,
		Fear: 12,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if client.lastDualityOutcomeRequest == nil {
		t.Fatal("expected gRPC call on invalid input")
	}
	if client.lastDualityOutcomeRequest.GetHope() != 0 || client.lastDualityOutcomeRequest.GetFear() != 12 {
		t.Fatalf("unexpected dice in request: %+v", client.lastDualityOutcomeRequest)
	}
}

// TestDualityOutcomeHandlerPassesNegativeDifficulty ensures gRPC receives invalid difficulty.
func TestDualityOutcomeHandlerPassesNegativeDifficulty(t *testing.T) {
	client := &fakeDualityClient{dualityOutcomeErr: errors.New("boom")}
	handler := domain.DualityOutcomeHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.DualityOutcomeInput{
		Hope:       6,
		Fear:       5,
		Difficulty: intPointer(-1),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if client.lastDualityOutcomeRequest == nil {
		t.Fatal("expected gRPC call on invalid input")
	}
	if client.lastDualityOutcomeRequest.Difficulty == nil || *client.lastDualityOutcomeRequest.Difficulty != -1 {
		t.Fatalf("expected difficulty -1, got %v", client.lastDualityOutcomeRequest.Difficulty)
	}
}

// TestDualityOutcomeHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestDualityOutcomeHandlerReturnsClientError(t *testing.T) {
	client := &fakeDualityClient{dualityOutcomeErr: errors.New("boom")}
	handler := domain.DualityOutcomeHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.DualityOutcomeInput{
		Hope:     6,
		Fear:     5,
		Modifier: 1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestDualityOutcomeHandlerMapsRequestAndResponse ensures inputs and outputs map consistently.
func TestDualityOutcomeHandlerMapsRequestAndResponse(t *testing.T) {
	difficulty := int32(10)
	client := &fakeDualityClient{dualityOutcomeResponse: &pb.DualityOutcomeResponse{
		Hope:            10,
		Fear:            4,
		Modifier:        1,
		Total:           15,
		IsCrit:          false,
		MeetsDifficulty: true,
		Difficulty:      &difficulty,
		Outcome:         pb.Outcome_SUCCESS_WITH_HOPE,
	}}

	handler := domain.DualityOutcomeHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.DualityOutcomeInput{
		Hope:       10,
		Fear:       4,
		Modifier:   1,
		Difficulty: intPointer(10),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != nil {
		t.Fatal("expected nil result on success")
	}
	if client.lastDualityOutcomeRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastDualityOutcomeRequest.GetHope() != 10 || client.lastDualityOutcomeRequest.GetFear() != 4 {
		t.Fatalf("unexpected dice in request: %+v", client.lastDualityOutcomeRequest)
	}
	if output.Total != 15 {
		t.Fatalf("expected total 15, got %d", output.Total)
	}
	if output.MeetsDifficulty != true {
		t.Fatal("expected meets_difficulty true")
	}
	if output.Outcome != pb.Outcome_SUCCESS_WITH_HOPE.String() {
		t.Fatalf("expected outcome %q, got %q", pb.Outcome_SUCCESS_WITH_HOPE.String(), output.Outcome)
	}
}

// TestDualityExplainHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestDualityExplainHandlerReturnsClientError(t *testing.T) {
	client := &fakeDualityClient{dualityExplainErr: errors.New("boom")}
	handler := domain.DualityExplainHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.DualityExplainInput{
		Hope:     6,
		Fear:     5,
		Modifier: 1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestDualityExplainHandlerMapsRequestAndResponse ensures inputs and outputs map consistently.
func TestDualityExplainHandlerMapsRequestAndResponse(t *testing.T) {
	difficulty := int32(10)
	stepData, err := structpb.NewStruct(map[string]any{"base_total": int64(14)})
	if err != nil {
		t.Fatalf("expected step data, got %v", err)
	}
	client := &fakeDualityClient{dualityExplainResponse: &pb.DualityExplainResponse{
		Hope:            10,
		Fear:            4,
		Modifier:        1,
		Total:           15,
		IsCrit:          false,
		MeetsDifficulty: true,
		Difficulty:      &difficulty,
		Outcome:         pb.Outcome_SUCCESS_WITH_HOPE,
		RulesVersion:    "1.0.0",
		Intermediates: &pb.Intermediates{
			BaseTotal:       14,
			Total:           15,
			IsCrit:          false,
			MeetsDifficulty: true,
			HopeGtFear:      true,
			FearGtHope:      false,
		},
		Steps: []*pb.ExplainStep{{
			Code:    "SUM_DICE",
			Message: "Sum Hope and Fear dice",
			Data:    stepData,
		}},
	}}

	requestID := "trace-123"
	handler := domain.DualityExplainHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.DualityExplainInput{
		Hope:       10,
		Fear:       4,
		Modifier:   1,
		Difficulty: intPointer(10),
		RequestID:  &requestID,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != nil {
		t.Fatal("expected nil result on success")
	}
	if client.lastDualityExplainRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastDualityExplainRequest.GetHope() != 10 || client.lastDualityExplainRequest.GetFear() != 4 {
		t.Fatalf("unexpected dice in request: %+v", client.lastDualityExplainRequest)
	}
	if output.Total != 15 {
		t.Fatalf("expected total 15, got %d", output.Total)
	}
	if output.RulesVersion != "1.0.0" {
		t.Fatalf("expected rules version %q, got %q", "1.0.0", output.RulesVersion)
	}
	if output.Intermediates.BaseTotal != 14 {
		t.Fatalf("expected base_total 14, got %d", output.Intermediates.BaseTotal)
	}
	if len(output.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(output.Steps))
	}
	if output.Steps[0].Code != "SUM_DICE" {
		t.Fatalf("expected step code %q, got %q", "SUM_DICE", output.Steps[0].Code)
	}
	if stepValue := output.Steps[0].Data["base_total"]; stepValue != 14.0 {
		t.Fatalf("expected base_total 14, got %v", stepValue)
	}
}

// TestDualityProbabilityHandlerPassesNegativeDifficulty ensures gRPC receives invalid difficulty.
func TestDualityProbabilityHandlerPassesNegativeDifficulty(t *testing.T) {
	client := &fakeDualityClient{dualityProbabilityErr: errors.New("boom")}
	handler := domain.DualityProbabilityHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.DualityProbabilityInput{
		Modifier:   1,
		Difficulty: -1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if client.lastDualityProbabilityRequest == nil {
		t.Fatal("expected gRPC call on invalid input")
	}
	if client.lastDualityProbabilityRequest.GetDifficulty() != -1 {
		t.Fatalf("expected difficulty -1, got %d", client.lastDualityProbabilityRequest.GetDifficulty())
	}
}

// TestDualityProbabilityHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestDualityProbabilityHandlerReturnsClientError(t *testing.T) {
	client := &fakeDualityClient{dualityProbabilityErr: errors.New("boom")}
	handler := domain.DualityProbabilityHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.DualityProbabilityInput{
		Modifier:   0,
		Difficulty: 10,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestDualityProbabilityHandlerMapsRequestAndResponse ensures inputs and outputs map consistently.
func TestDualityProbabilityHandlerMapsRequestAndResponse(t *testing.T) {
	client := &fakeDualityClient{dualityProbabilityResponse: &pb.DualityProbabilityResponse{
		TotalOutcomes: 144,
		CritCount:     12,
		SuccessCount:  70,
		FailureCount:  74,
		OutcomeCounts: []*pb.OutcomeCount{
			{Outcome: pb.Outcome_CRITICAL_SUCCESS, Count: 12},
			{Outcome: pb.Outcome_SUCCESS_WITH_HOPE, Count: 34},
			{Outcome: pb.Outcome_SUCCESS_WITH_FEAR, Count: 24},
			{Outcome: pb.Outcome_FAILURE_WITH_HOPE, Count: 40},
			{Outcome: pb.Outcome_FAILURE_WITH_FEAR, Count: 34},
		},
	}}
	handler := domain.DualityProbabilityHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.DualityProbabilityInput{
		Modifier:   1,
		Difficulty: 10,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != nil {
		t.Fatal("expected nil result on success")
	}
	if client.lastDualityProbabilityRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if output.TotalOutcomes != 144 {
		t.Fatalf("expected total 144, got %d", output.TotalOutcomes)
	}
	if output.CritCount != 12 {
		t.Fatalf("expected crit 12, got %d", output.CritCount)
	}
	if len(output.OutcomeCounts) != 5 {
		t.Fatalf("expected 5 outcome counts, got %d", len(output.OutcomeCounts))
	}
}

// TestRollDiceHandlerPassesMissingDice ensures gRPC receives empty dice.
func TestRollDiceHandlerPassesMissingDice(t *testing.T) {
	client := &fakeDualityClient{rollDiceErr: errors.New("boom")}
	handler := domain.RollDiceHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.RollDiceInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if client.lastRollDiceRequest == nil {
		t.Fatal("expected gRPC call on invalid input")
	}
	if len(client.lastRollDiceRequest.GetDice()) != 0 {
		t.Fatalf("expected empty dice, got %d", len(client.lastRollDiceRequest.GetDice()))
	}
}

// TestRollDiceHandlerPassesInvalidDice ensures gRPC receives invalid dice specs.
func TestRollDiceHandlerPassesInvalidDice(t *testing.T) {
	client := &fakeDualityClient{rollDiceErr: errors.New("boom")}
	handler := domain.RollDiceHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.RollDiceInput{
		Dice: []domain.RollDiceSpec{{Sides: -1, Count: 2}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if client.lastRollDiceRequest == nil {
		t.Fatal("expected gRPC call on invalid input")
	}
	if len(client.lastRollDiceRequest.GetDice()) != 1 {
		t.Fatalf("expected 1 dice spec, got %d", len(client.lastRollDiceRequest.GetDice()))
	}
	if client.lastRollDiceRequest.Dice[0].Sides != -1 || client.lastRollDiceRequest.Dice[0].Count != 2 {
		t.Fatalf("unexpected dice spec: %+v", client.lastRollDiceRequest.Dice[0])
	}
}

// TestRollDiceHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestRollDiceHandlerReturnsClientError(t *testing.T) {
	client := &fakeDualityClient{rollDiceErr: errors.New("boom")}
	handler := domain.RollDiceHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.RollDiceInput{
		Dice: []domain.RollDiceSpec{{Sides: 6, Count: 1}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestRollDiceHandlerMapsRequestAndResponse ensures inputs and outputs are mapped consistently.
func TestRollDiceHandlerMapsRequestAndResponse(t *testing.T) {
	client := &fakeDualityClient{rollDiceResponse: &pb.RollDiceResponse{
		Rolls: []*pb.DiceRoll{
			{Sides: 6, Results: []int32{2, 5}, Total: 7},
			{Sides: 8, Results: []int32{4}, Total: 4},
		},
		Total: 11,
	}}

	handler := domain.RollDiceHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.RollDiceInput{
		Dice: []domain.RollDiceSpec{{Sides: 6, Count: 2}, {Sides: 8, Count: 1}},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != nil {
		t.Fatal("expected nil result on success")
	}
	if client.lastRollDiceRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if len(client.lastRollDiceRequest.Dice) != 2 {
		t.Fatalf("expected 2 dice specs, got %d", len(client.lastRollDiceRequest.Dice))
	}
	if client.lastRollDiceRequest.Dice[0].Sides != 6 || client.lastRollDiceRequest.Dice[0].Count != 2 {
		t.Fatalf("unexpected first dice spec: %+v", client.lastRollDiceRequest.Dice[0])
	}
	if client.lastRollDiceRequest.Dice[1].Sides != 8 || client.lastRollDiceRequest.Dice[1].Count != 1 {
		t.Fatalf("unexpected second dice spec: %+v", client.lastRollDiceRequest.Dice[1])
	}

	if output.Total != 11 {
		t.Fatalf("expected total 11, got %d", output.Total)
	}
	if len(output.Rolls) != 2 {
		t.Fatalf("expected 2 rolls, got %d", len(output.Rolls))
	}
	if output.Rolls[0].Sides != 6 || output.Rolls[0].Total != 7 {
		t.Fatalf("unexpected first roll: %+v", output.Rolls[0])
	}
	if output.Rolls[1].Sides != 8 || output.Rolls[1].Total != 4 {
		t.Fatalf("unexpected second roll: %+v", output.Rolls[1])
	}
}

// TestRulesVersionHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestRulesVersionHandlerReturnsClientError(t *testing.T) {
	client := &fakeDualityClient{rulesVersionErr: errors.New("boom")}
	handler := domain.RulesVersionHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.RulesVersionInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestRulesVersionHandlerMapsResponse ensures metadata is passed through.
func TestRulesVersionHandlerMapsResponse(t *testing.T) {
	client := &fakeDualityClient{rulesVersionResponse: &pb.RulesVersionResponse{
		System:         "Daggerheart",
		Module:         "Duality",
		RulesVersion:   "1.0.0",
		DiceModel:      "2d12",
		TotalFormula:   "hope + fear + modifier",
		CritRule:       "critical success on matching hope/fear; overrides difficulty",
		DifficultyRule: "difficulty optional; total >= difficulty succeeds; critical success always succeeds",
		Outcomes: []pb.Outcome{
			pb.Outcome_ROLL_WITH_HOPE,
			pb.Outcome_ROLL_WITH_FEAR,
			pb.Outcome_SUCCESS_WITH_HOPE,
			pb.Outcome_SUCCESS_WITH_FEAR,
			pb.Outcome_FAILURE_WITH_HOPE,
			pb.Outcome_FAILURE_WITH_FEAR,
			pb.Outcome_CRITICAL_SUCCESS,
		},
	}}
	handler := domain.RulesVersionHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.RulesVersionInput{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != nil {
		t.Fatal("expected nil result on success")
	}
	if client.lastRulesVersionRequest == nil {
		t.Fatal("expected gRPC request")
	}

	expectedOutcomes := []string{
		pb.Outcome_ROLL_WITH_HOPE.String(),
		pb.Outcome_ROLL_WITH_FEAR.String(),
		pb.Outcome_SUCCESS_WITH_HOPE.String(),
		pb.Outcome_SUCCESS_WITH_FEAR.String(),
		pb.Outcome_FAILURE_WITH_HOPE.String(),
		pb.Outcome_FAILURE_WITH_FEAR.String(),
		pb.Outcome_CRITICAL_SUCCESS.String(),
	}
	if !reflect.DeepEqual(output.Outcomes, expectedOutcomes) {
		t.Fatalf("expected outcomes %v, got %v", expectedOutcomes, output.Outcomes)
	}
	if output.System != client.rulesVersionResponse.System {
		t.Fatalf("expected system %q, got %q", client.rulesVersionResponse.System, output.System)
	}
	if output.Module != client.rulesVersionResponse.Module {
		t.Fatalf("expected module %q, got %q", client.rulesVersionResponse.Module, output.Module)
	}
	if output.RulesVersion != client.rulesVersionResponse.RulesVersion {
		t.Fatalf("expected rules version %q, got %q", client.rulesVersionResponse.RulesVersion, output.RulesVersion)
	}
	if output.DiceModel != client.rulesVersionResponse.DiceModel {
		t.Fatalf("expected dice model %q, got %q", client.rulesVersionResponse.DiceModel, output.DiceModel)
	}
	if output.TotalFormula != client.rulesVersionResponse.TotalFormula {
		t.Fatalf("expected total formula %q, got %q", client.rulesVersionResponse.TotalFormula, output.TotalFormula)
	}
	if output.CritRule != client.rulesVersionResponse.CritRule {
		t.Fatalf("expected crit rule %q, got %q", client.rulesVersionResponse.CritRule, output.CritRule)
	}
	if output.DifficultyRule != client.rulesVersionResponse.DifficultyRule {
		t.Fatalf("expected difficulty rule %q, got %q", client.rulesVersionResponse.DifficultyRule, output.DifficultyRule)
	}
}

// TestCampaignCreateHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestCampaignCreateHandlerReturnsClientError(t *testing.T) {
	client := &fakeCampaignClient{err: errors.New("boom")}
	handler := domain.CampaignCreateHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CampaignCreateInput{
		Name:        "New Campaign",
		GmMode:      "HUMAN",
		PlayerSlots: 4,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCampaignCreateHandlerMapsRequestAndResponse ensures inputs and outputs map consistently.
func TestCampaignCreateHandlerMapsRequestAndResponse(t *testing.T) {
	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	client := &fakeCampaignClient{response: &campaignpb.CreateCampaignResponse{
		Campaign: &campaignpb.Campaign{
			Id:          "camp-123",
			Name:        "Snowbound",
			GmMode:      campaignpb.GmMode_AI,
			PlayerSlots: 5,
			ThemePrompt: "ice and steel",
			CreatedAt:   timestamppb.New(now),
			UpdatedAt:   timestamppb.New(now),
		},
	}}
	result, output, err := domain.CampaignCreateHandler(client)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.CampaignCreateInput{
			Name:        "Snowbound",
			GmMode:      "HUMAN",
			PlayerSlots: 5,
			ThemePrompt: "ice and steel",
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result != nil {
		t.Fatal("expected nil result on success")
	}
	if client.lastRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastRequest.GetGmMode() != campaignpb.GmMode_HUMAN {
		t.Fatalf("expected gm mode HUMAN, got %v", client.lastRequest.GetGmMode())
	}
	if output.ID != "camp-123" {
		t.Fatalf("expected id camp-123, got %q", output.ID)
	}
	if output.GmMode != "AI" {
		t.Fatalf("expected gm mode AI, got %q", output.GmMode)
	}
	if output.PlayerSlots != 5 {
		t.Fatalf("expected player slots 5, got %d", output.PlayerSlots)
	}
}

// intPointer returns an int pointer for test inputs.
func intPointer(value int) *int {
	return &value
}

func TestWaitForHealthSuccess(t *testing.T) {
	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_SERVING)
	defer stop()

	conn, err := newGRPCConn(addr)
	if err != nil {
		t.Fatalf("dial health server: %v", err)
	}
	defer conn.Close()

	server := &Server{conn: conn}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.waitForHealth(ctx); err != nil {
		t.Fatalf("wait for health: %v", err)
	}
}

func TestWaitForHealthRetriesUntilServing(t *testing.T) {
	addr, setStatus, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	defer stop()

	conn, err := newGRPCConn(addr)
	if err != nil {
		t.Fatalf("dial health server: %v", err)
	}
	defer conn.Close()

	server := &Server{conn: conn}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	setStatusTimer := time.NewTimer(200 * time.Millisecond)
	defer setStatusTimer.Stop()
	go func() {
		<-setStatusTimer.C
		setStatus(grpc_health_v1.HealthCheckResponse_SERVING)
	}()

	if err := server.waitForHealth(ctx); err != nil {
		t.Fatalf("wait for health: %v", err)
	}
}

func TestWaitForHealthTimeout(t *testing.T) {
	addr, _, stop := startHealthServer(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	defer stop()

	conn, err := newGRPCConn(addr)
	if err != nil {
		t.Fatalf("dial health server: %v", err)
	}
	defer conn.Close()

	server := &Server{conn: conn}
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if err := server.waitForHealth(ctx); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestWaitForHealthMissingConn(t *testing.T) {
	server := &Server{}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.waitForHealth(ctx); err == nil {
		t.Fatal("expected error for missing connection")
	}
}

func startHealthServer(t *testing.T, status grpc_health_v1.HealthCheckResponse_ServingStatus) (string, func(grpc_health_v1.HealthCheckResponse_ServingStatus), func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)
	healthServer.SetServingStatus("", status)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	setStatus := func(next grpc_health_v1.HealthCheckResponse_ServingStatus) {
		healthServer.SetServingStatus("", next)
	}

	stop := func() {
		healthServer.Shutdown()
		grpcServer.GracefulStop()
		_ = listener.Close()
	}

	return listener.Addr().String(), setStatus, stop
}
