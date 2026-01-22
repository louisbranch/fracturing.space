// Package mcp tests the MCP server wiring.
package mcp

import (
	"context"
	"errors"
	"testing"

	pb "github.com/louisbranch/duality-protocol/api/gen/go/duality/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
)

// fakeDiceRollClient implements DiceRollServiceClient for tests.
type fakeDiceRollClient struct {
	response                      *pb.ActionRollResponse
	rollDiceResponse              *pb.RollDiceResponse
	dualityOutcomeResponse        *pb.DualityOutcomeResponse
	dualityProbabilityResponse    *pb.DualityProbabilityResponse
	err                           error
	rollDiceErr                   error
	dualityOutcomeErr             error
	dualityProbabilityErr         error
	lastRequest                   *pb.ActionRollRequest
	lastRollDiceRequest           *pb.RollDiceRequest
	lastDualityOutcomeRequest     *pb.DualityOutcomeRequest
	lastDualityProbabilityRequest *pb.DualityProbabilityRequest
}

// ActionRoll records the request and returns the configured response.
func (f *fakeDiceRollClient) ActionRoll(ctx context.Context, req *pb.ActionRollRequest, opts ...grpc.CallOption) (*pb.ActionRollResponse, error) {
	f.lastRequest = req
	return f.response, f.err
}

// DualityOutcome records the request and returns the configured response.
func (f *fakeDiceRollClient) DualityOutcome(ctx context.Context, req *pb.DualityOutcomeRequest, opts ...grpc.CallOption) (*pb.DualityOutcomeResponse, error) {
	f.lastDualityOutcomeRequest = req
	return f.dualityOutcomeResponse, f.dualityOutcomeErr
}

// DualityProbability records the request and returns the configured response.
func (f *fakeDiceRollClient) DualityProbability(ctx context.Context, req *pb.DualityProbabilityRequest, opts ...grpc.CallOption) (*pb.DualityProbabilityResponse, error) {
	f.lastDualityProbabilityRequest = req
	return f.dualityProbabilityResponse, f.dualityProbabilityErr
}

// RollDice records the request and returns the configured response.
func (f *fakeDiceRollClient) RollDice(ctx context.Context, req *pb.RollDiceRequest, opts ...grpc.CallOption) (*pb.RollDiceResponse, error) {
	f.lastRollDiceRequest = req
	return f.rollDiceResponse, f.rollDiceErr
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
			if err := tt.server.Serve(); err == nil {
				t.Fatal("expected error")
			}
		})
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

// TestActionRollHandlerRejectsNegativeDifficulty ensures invalid difficulty returns an error result.
func TestActionRollHandlerRejectsNegativeDifficulty(t *testing.T) {
	client := &fakeDiceRollClient{}
	handler := actionRollHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, ActionRollInput{
		Modifier:   1,
		Difficulty: intPointer(-1),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if client.lastRequest != nil {
		t.Fatal("expected no gRPC call on invalid input")
	}
}

// TestActionRollHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestActionRollHandlerReturnsClientError(t *testing.T) {
	client := &fakeDiceRollClient{err: errors.New("boom")}
	handler := actionRollHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, ActionRollInput{Modifier: 2})
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
	client := &fakeDiceRollClient{
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
	handler := actionRollHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, ActionRollInput{
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

// TestDualityOutcomeHandlerRejectsInvalidDice ensures invalid dice return errors.
func TestDualityOutcomeHandlerRejectsInvalidDice(t *testing.T) {
	client := &fakeDiceRollClient{}
	handler := dualityOutcomeHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, DualityOutcomeInput{
		Hope: 0,
		Fear: 12,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if client.lastDualityOutcomeRequest != nil {
		t.Fatal("expected no gRPC call on invalid input")
	}
}

// TestDualityOutcomeHandlerRejectsNegativeDifficulty ensures invalid difficulty returns errors.
func TestDualityOutcomeHandlerRejectsNegativeDifficulty(t *testing.T) {
	client := &fakeDiceRollClient{}
	handler := dualityOutcomeHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, DualityOutcomeInput{
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
	if client.lastDualityOutcomeRequest != nil {
		t.Fatal("expected no gRPC call on invalid input")
	}
}

// TestDualityOutcomeHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestDualityOutcomeHandlerReturnsClientError(t *testing.T) {
	client := &fakeDiceRollClient{dualityOutcomeErr: errors.New("boom")}
	handler := dualityOutcomeHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, DualityOutcomeInput{
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
	client := &fakeDiceRollClient{dualityOutcomeResponse: &pb.DualityOutcomeResponse{
		Hope:            10,
		Fear:            4,
		Modifier:        1,
		Total:           15,
		IsCrit:          false,
		MeetsDifficulty: true,
		Difficulty:      &difficulty,
		Outcome:         pb.Outcome_SUCCESS_WITH_HOPE,
	}}

	handler := dualityOutcomeHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, DualityOutcomeInput{
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

// TestDualityProbabilityHandlerRejectsNegativeDifficulty ensures invalid difficulty returns errors.
func TestDualityProbabilityHandlerRejectsNegativeDifficulty(t *testing.T) {
	client := &fakeDiceRollClient{}
	handler := dualityProbabilityHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, DualityProbabilityInput{
		Modifier:   1,
		Difficulty: -1,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if client.lastDualityProbabilityRequest != nil {
		t.Fatal("expected no gRPC call on invalid input")
	}
}

// TestDualityProbabilityHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestDualityProbabilityHandlerReturnsClientError(t *testing.T) {
	client := &fakeDiceRollClient{dualityProbabilityErr: errors.New("boom")}
	handler := dualityProbabilityHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, DualityProbabilityInput{
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
	client := &fakeDiceRollClient{dualityProbabilityResponse: &pb.DualityProbabilityResponse{
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
	handler := dualityProbabilityHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, DualityProbabilityInput{
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

// TestRollDiceHandlerRejectsMissingDice ensures empty dice requests return an error result.
func TestRollDiceHandlerRejectsMissingDice(t *testing.T) {
	client := &fakeDiceRollClient{}
	handler := rollDiceHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, RollDiceInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if client.lastRollDiceRequest != nil {
		t.Fatal("expected no gRPC call on invalid input")
	}
}

// TestRollDiceHandlerRejectsInvalidDice ensures invalid dice specs return an error result.
func TestRollDiceHandlerRejectsInvalidDice(t *testing.T) {
	client := &fakeDiceRollClient{}
	handler := rollDiceHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, RollDiceInput{
		Dice: []RollDiceSpec{{Sides: -1, Count: 2}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if client.lastRollDiceRequest != nil {
		t.Fatal("expected no gRPC call on invalid input")
	}
}

// TestRollDiceHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestRollDiceHandlerReturnsClientError(t *testing.T) {
	client := &fakeDiceRollClient{rollDiceErr: errors.New("boom")}
	handler := rollDiceHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, RollDiceInput{
		Dice: []RollDiceSpec{{Sides: 6, Count: 1}},
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
	client := &fakeDiceRollClient{rollDiceResponse: &pb.RollDiceResponse{
		Rolls: []*pb.DiceRoll{
			{Sides: 6, Results: []int32{2, 5}, Total: 7},
			{Sides: 8, Results: []int32{4}, Total: 4},
		},
		Total: 11,
	}}

	handler := rollDiceHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, RollDiceInput{
		Dice: []RollDiceSpec{{Sides: 6, Count: 2}, {Sides: 8, Count: 1}},
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

// intPointer returns an int pointer for test inputs.
func intPointer(value int) *int {
	return &value
}
