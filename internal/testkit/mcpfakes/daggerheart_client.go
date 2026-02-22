package mcpfakes

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// DaggerheartClient is a configurable fake for Daggerheart MCP tool tests.
type DaggerheartClient struct {
	pb.DaggerheartServiceClient // embed for forward-compatibility

	Response                      *pb.ActionRollResponse
	RollDiceResponse              *pb.RollDiceResponse
	DualityOutcomeResponse        *pb.DualityOutcomeResponse
	DualityExplainResponse        *pb.DualityExplainResponse
	DualityProbabilityResponse    *pb.DualityProbabilityResponse
	RulesVersionResponse          *pb.RulesVersionResponse
	Err                           error
	RollDiceErr                   error
	DualityOutcomeErr             error
	DualityExplainErr             error
	DualityProbabilityErr         error
	RulesVersionErr               error
	LastRequest                   *pb.ActionRollRequest
	LastRollDiceRequest           *pb.RollDiceRequest
	LastDualityOutcomeRequest     *pb.DualityOutcomeRequest
	LastDualityExplainRequest     *pb.DualityExplainRequest
	LastDualityProbabilityRequest *pb.DualityProbabilityRequest
	LastRulesVersionRequest       *pb.RulesVersionRequest
}

// ActionRoll records the request and returns the configured response.
func (f *DaggerheartClient) ActionRoll(ctx context.Context, req *pb.ActionRollRequest, opts ...grpc.CallOption) (*pb.ActionRollResponse, error) {
	f.LastRequest = req
	return f.Response, f.Err
}

// DualityOutcome records the request and returns the configured response.
func (f *DaggerheartClient) DualityOutcome(ctx context.Context, req *pb.DualityOutcomeRequest, opts ...grpc.CallOption) (*pb.DualityOutcomeResponse, error) {
	f.LastDualityOutcomeRequest = req
	return f.DualityOutcomeResponse, f.DualityOutcomeErr
}

// DualityExplain records the request and returns the configured response.
func (f *DaggerheartClient) DualityExplain(ctx context.Context, req *pb.DualityExplainRequest, opts ...grpc.CallOption) (*pb.DualityExplainResponse, error) {
	f.LastDualityExplainRequest = req
	return f.DualityExplainResponse, f.DualityExplainErr
}

// DualityProbability records the request and returns the configured response.
func (f *DaggerheartClient) DualityProbability(ctx context.Context, req *pb.DualityProbabilityRequest, opts ...grpc.CallOption) (*pb.DualityProbabilityResponse, error) {
	f.LastDualityProbabilityRequest = req
	return f.DualityProbabilityResponse, f.DualityProbabilityErr
}

// RulesVersion records the request and returns the configured response.
func (f *DaggerheartClient) RulesVersion(ctx context.Context, req *pb.RulesVersionRequest, opts ...grpc.CallOption) (*pb.RulesVersionResponse, error) {
	f.LastRulesVersionRequest = req
	return f.RulesVersionResponse, f.RulesVersionErr
}

// RollDice records the request and returns the configured response.
func (f *DaggerheartClient) RollDice(ctx context.Context, req *pb.RollDiceRequest, opts ...grpc.CallOption) (*pb.RollDiceResponse, error) {
	f.LastRollDiceRequest = req
	return f.RollDiceResponse, f.RollDiceErr
}

// SessionActionRoll is intentionally not implemented for MCP tests.
func (f *DaggerheartClient) SessionActionRoll(ctx context.Context, req *pb.SessionActionRollRequest, opts ...grpc.CallOption) (*pb.SessionActionRollResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in test fake")
}

// ApplyRollOutcome is intentionally not implemented for MCP tests.
func (f *DaggerheartClient) ApplyRollOutcome(ctx context.Context, req *pb.ApplyRollOutcomeRequest, opts ...grpc.CallOption) (*pb.ApplyRollOutcomeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented in test fake")
}
