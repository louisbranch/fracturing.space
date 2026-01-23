// Package mcp tests the MCP server wiring.
package mcp

import (
	"context"
	"errors"
	"reflect"
	"testing"

	pb "github.com/louisbranch/duality-protocol/api/gen/go/duality/v1"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
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

// TestActionRollHandlerPassesNegativeDifficulty ensures gRPC receives invalid difficulty.
func TestActionRollHandlerPassesNegativeDifficulty(t *testing.T) {
	client := &fakeDualityClient{err: errors.New("boom")}
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

// TestDualityOutcomeHandlerPassesInvalidDice ensures gRPC receives invalid dice.
func TestDualityOutcomeHandlerPassesInvalidDice(t *testing.T) {
	client := &fakeDualityClient{dualityOutcomeErr: errors.New("boom")}
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

// TestDualityExplainHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestDualityExplainHandlerReturnsClientError(t *testing.T) {
	client := &fakeDualityClient{dualityExplainErr: errors.New("boom")}
	handler := dualityExplainHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, DualityExplainInput{
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
	handler := dualityExplainHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, DualityExplainInput{
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

// TestRollDiceHandlerPassesMissingDice ensures gRPC receives empty dice.
func TestRollDiceHandlerPassesMissingDice(t *testing.T) {
	client := &fakeDualityClient{rollDiceErr: errors.New("boom")}
	handler := rollDiceHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, RollDiceInput{})
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
	client := &fakeDualityClient{rollDiceResponse: &pb.RollDiceResponse{
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

// TestRulesVersionHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestRulesVersionHandlerReturnsClientError(t *testing.T) {
	client := &fakeDualityClient{rulesVersionErr: errors.New("boom")}
	handler := rulesVersionHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, RulesVersionInput{})
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
	handler := rulesVersionHandler(client)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, RulesVersionInput{})
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

// intPointer returns an int pointer for test inputs.
func intPointer(value int) *int {
	return &value
}
