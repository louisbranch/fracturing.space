// Package service tests the MCP server wiring.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	pb "github.com/louisbranch/duality-engine/api/gen/go/duality/v1"
	sessionv1 "github.com/louisbranch/duality-engine/api/gen/go/session/v1"
	"github.com/louisbranch/duality-engine/internal/grpcmeta"
	"github.com/louisbranch/duality-engine/internal/mcp/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
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

// requireToolMetadata asserts tool result metadata includes correlation IDs.
func requireToolMetadata(t *testing.T, result *mcp.CallToolResult) (string, string) {
	t.Helper()
	if result == nil {
		t.Fatal("expected result metadata")
	}
	if result.Meta == nil {
		t.Fatal("expected result metadata map")
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

// fakeCampaignClient implements CampaignServiceClient for tests.
type fakeCampaignClient struct {
	response                     *campaignv1.CreateCampaignResponse
	listResponse                 *campaignv1.ListCampaignsResponse
	getCampaignResponse          *campaignv1.GetCampaignResponse
	createParticipantResponse    *campaignv1.CreateParticipantResponse
	listParticipantsResponse     *campaignv1.ListParticipantsResponse
	getParticipantResponse       *campaignv1.GetParticipantResponse
	createCharacterResponse      *campaignv1.CreateCharacterResponse
	listCharactersResponse       *campaignv1.ListCharactersResponse
	setDefaultControlResponse    *campaignv1.SetDefaultControlResponse
	err                          error
	listErr                      error
	getCampaignErr               error
	createParticipantErr         error
	listParticipantsErr          error
	getParticipantErr            error
	createCharacterErr           error
	listCharactersErr            error
	setDefaultControlErr         error
	lastRequest                  *campaignv1.CreateCampaignRequest
	lastListRequest              *campaignv1.ListCampaignsRequest
	lastGetCampaignRequest       *campaignv1.GetCampaignRequest
	lastCreateParticipantRequest *campaignv1.CreateParticipantRequest
	lastListParticipantsRequest  *campaignv1.ListParticipantsRequest
	lastGetParticipantRequest    *campaignv1.GetParticipantRequest
	lastCreateCharacterRequest   *campaignv1.CreateCharacterRequest
	lastListCharactersRequest    *campaignv1.ListCharactersRequest
	lastSetDefaultControlRequest *campaignv1.SetDefaultControlRequest
	listCalls                    int
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
func (f *fakeCampaignClient) CreateCampaign(ctx context.Context, req *campaignv1.CreateCampaignRequest, opts ...grpc.CallOption) (*campaignv1.CreateCampaignResponse, error) {
	f.lastRequest = req
	return f.response, f.err
}

// ListCampaigns records the request and returns the configured response.
func (f *fakeCampaignClient) ListCampaigns(ctx context.Context, req *campaignv1.ListCampaignsRequest, opts ...grpc.CallOption) (*campaignv1.ListCampaignsResponse, error) {
	f.lastListRequest = req
	f.listCalls++
	return f.listResponse, f.listErr
}

// GetCampaign records the request and returns the configured response.
func (f *fakeCampaignClient) GetCampaign(ctx context.Context, req *campaignv1.GetCampaignRequest, opts ...grpc.CallOption) (*campaignv1.GetCampaignResponse, error) {
	f.lastGetCampaignRequest = req
	return f.getCampaignResponse, f.getCampaignErr
}

// CreateParticipant records the request and returns the configured response.
func (f *fakeCampaignClient) CreateParticipant(ctx context.Context, req *campaignv1.CreateParticipantRequest, opts ...grpc.CallOption) (*campaignv1.CreateParticipantResponse, error) {
	f.lastCreateParticipantRequest = req
	return f.createParticipantResponse, f.createParticipantErr
}

// ListParticipants records the request and returns the configured response.
func (f *fakeCampaignClient) ListParticipants(ctx context.Context, req *campaignv1.ListParticipantsRequest, opts ...grpc.CallOption) (*campaignv1.ListParticipantsResponse, error) {
	f.lastListParticipantsRequest = req
	return f.listParticipantsResponse, f.listParticipantsErr
}

// GetParticipant records the request and returns the configured response.
func (f *fakeCampaignClient) GetParticipant(ctx context.Context, req *campaignv1.GetParticipantRequest, opts ...grpc.CallOption) (*campaignv1.GetParticipantResponse, error) {
	f.lastGetParticipantRequest = req
	return f.getParticipantResponse, f.getParticipantErr
}

// CreateCharacter records the request and returns the configured response.
func (f *fakeCampaignClient) CreateCharacter(ctx context.Context, req *campaignv1.CreateCharacterRequest, opts ...grpc.CallOption) (*campaignv1.CreateCharacterResponse, error) {
	f.lastCreateCharacterRequest = req
	return f.createCharacterResponse, f.createCharacterErr
}

// ListCharacters records the request and returns the configured response.
func (f *fakeCampaignClient) ListCharacters(ctx context.Context, req *campaignv1.ListCharactersRequest, opts ...grpc.CallOption) (*campaignv1.ListCharactersResponse, error) {
	f.lastListCharactersRequest = req
	return f.listCharactersResponse, f.listCharactersErr
}

// SetDefaultControl records the request and returns the configured response.
func (f *fakeCampaignClient) SetDefaultControl(ctx context.Context, req *campaignv1.SetDefaultControlRequest, opts ...grpc.CallOption) (*campaignv1.SetDefaultControlResponse, error) {
	f.lastSetDefaultControlRequest = req
	return f.setDefaultControlResponse, f.setDefaultControlErr
}

// GetCharacterSheet records the request and returns the configured response.
func (f *fakeCampaignClient) GetCharacterSheet(ctx context.Context, req *campaignv1.GetCharacterSheetRequest, opts ...grpc.CallOption) (*campaignv1.GetCharacterSheetResponse, error) {
	return nil, errors.New("not implemented in fake client")
}

// PatchCharacterProfile records the request and returns the configured response.
func (f *fakeCampaignClient) PatchCharacterProfile(ctx context.Context, req *campaignv1.PatchCharacterProfileRequest, opts ...grpc.CallOption) (*campaignv1.PatchCharacterProfileResponse, error) {
	return nil, errors.New("not implemented in fake client")
}

// PatchCharacterState records the request and returns the configured response.
func (f *fakeCampaignClient) PatchCharacterState(ctx context.Context, req *campaignv1.PatchCharacterStateRequest, opts ...grpc.CallOption) (*campaignv1.PatchCharacterStateResponse, error) {
	return nil, errors.New("not implemented in fake client")
}

// fakeSessionClient implements SessionServiceClient for tests.
type fakeSessionClient struct {
	startSessionResponse    *sessionv1.StartSessionResponse
	listSessionsResponse    *sessionv1.ListSessionsResponse
	getSessionResponse      *sessionv1.GetSessionResponse
	err                     error
	listSessionsErr         error
	getSessionErr           error
	lastRequest             *sessionv1.StartSessionRequest
	lastListSessionsRequest *sessionv1.ListSessionsRequest
	lastGetSessionRequest   *sessionv1.GetSessionRequest
}

// StartSession records the request and returns the configured response.
func (f *fakeSessionClient) StartSession(ctx context.Context, req *sessionv1.StartSessionRequest, opts ...grpc.CallOption) (*sessionv1.StartSessionResponse, error) {
	f.lastRequest = req
	return f.startSessionResponse, f.err
}

// ListSessions records the request and returns the configured response.
func (f *fakeSessionClient) ListSessions(ctx context.Context, req *sessionv1.ListSessionsRequest, opts ...grpc.CallOption) (*sessionv1.ListSessionsResponse, error) {
	f.lastListSessionsRequest = req
	return f.listSessionsResponse, f.listSessionsErr
}

// GetSession records the request and returns the configured response.
func (f *fakeSessionClient) GetSession(ctx context.Context, req *sessionv1.GetSessionRequest, opts ...grpc.CallOption) (*sessionv1.GetSessionResponse, error) {
	f.lastGetSessionRequest = req
	return f.getSessionResponse, f.getSessionErr
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
	requireToolMetadata(t, result)
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
	requireToolMetadata(t, result)
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
	requireToolMetadata(t, result)
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
	requireToolMetadata(t, result)
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
	expectedOutcomes := []pb.Outcome{
		pb.Outcome_CRITICAL_SUCCESS,
		pb.Outcome_SUCCESS_WITH_HOPE,
		pb.Outcome_SUCCESS_WITH_FEAR,
		pb.Outcome_FAILURE_WITH_HOPE,
		pb.Outcome_FAILURE_WITH_FEAR,
	}
	for i, expectedOutcome := range expectedOutcomes {
		if output.OutcomeCounts[i].Outcome != expectedOutcome.String() {
			t.Fatalf("outcome[%d] = %q, want %q", i, output.OutcomeCounts[i].Outcome, expectedOutcome.String())
		}
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
	requireToolMetadata(t, result)
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
	requireToolMetadata(t, result)
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
		Name:   "New Campaign",
		GmMode: "HUMAN",
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
	client := &fakeCampaignClient{response: &campaignv1.CreateCampaignResponse{
		Campaign: &campaignv1.Campaign{
			Id:               "camp-123",
			Name:             "Snowbound",
			GmMode:           campaignv1.GmMode_AI,
			ParticipantCount: 5,
			CharacterCount:   3,
			ThemePrompt:      "ice and steel",
			CreatedAt:        timestamppb.New(now),
			UpdatedAt:        timestamppb.New(now),
		},
	}}
	result, output, err := domain.CampaignCreateHandler(client)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.CampaignCreateInput{
			Name:        "Snowbound",
			GmMode:      "HUMAN",
			ThemePrompt: "ice and steel",
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastRequest.GetGmMode() != campaignv1.GmMode_HUMAN {
		t.Fatalf("expected gm mode HUMAN, got %v", client.lastRequest.GetGmMode())
	}
	if output.ID != "camp-123" {
		t.Fatalf("expected id camp-123, got %q", output.ID)
	}
	if output.GmMode != "AI" {
		t.Fatalf("expected gm mode AI, got %q", output.GmMode)
	}
	if output.ParticipantCount != 5 {
		t.Fatalf("expected participant count 5, got %d", output.ParticipantCount)
	}
	if output.CharacterCount != 3 {
		t.Fatalf("expected character count 3, got %d", output.CharacterCount)
	}
}

// TestCampaignListResourceHandlerReturnsClientError ensures list errors are returned.
func TestCampaignListResourceHandlerReturnsClientError(t *testing.T) {
	client := &fakeCampaignClient{listErr: errors.New("boom")}
	handler := domain.CampaignListResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if client.listCalls != 1 {
		t.Fatalf("expected 1 list call, got %d", client.listCalls)
	}
}

// TestCampaignListResourceHandlerRejectsEmptyResponse ensures nil responses are rejected.
func TestCampaignListResourceHandlerRejectsEmptyResponse(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CampaignListResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCampaignListResourceHandlerMapsResponse ensures JSON payload is formatted.
func TestCampaignListResourceHandlerMapsResponse(t *testing.T) {
	now := time.Date(2026, 1, 23, 13, 0, 0, 0, time.UTC)
	client := &fakeCampaignClient{listResponse: &campaignv1.ListCampaignsResponse{
		Campaigns: []*campaignv1.Campaign{{
			Id:               "camp-1",
			Name:             "Red Sands",
			GmMode:           campaignv1.GmMode_HUMAN,
			ParticipantCount: 4,
			CharacterCount:   2,
			ThemePrompt:      "desert skies",
			CreatedAt:        timestamppb.New(now),
			UpdatedAt:        timestamppb.New(now.Add(time.Hour)),
		}},
		NextPageToken: "next",
	}}

	handler := domain.CampaignListResourceHandler(client)
	result, err := handler(context.Background(), &mcp.ReadResourceRequest{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || len(result.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %v", result)
	}
	if client.lastListRequest == nil {
		t.Fatal("expected list request")
	}
	if client.lastListRequest.GetPageSize() != 10 {
		t.Fatalf("expected page size 10, got %d", client.lastListRequest.GetPageSize())
	}
	if client.lastListRequest.GetPageToken() != "" {
		t.Fatalf("expected empty page token, got %q", client.lastListRequest.GetPageToken())
	}
	if client.listCalls != 1 {
		t.Fatalf("expected 1 list call, got %d", client.listCalls)
	}

	var payload struct {
		Campaigns []struct {
			ID               string `json:"id"`
			Name             string `json:"name"`
			GmMode           string `json:"gm_mode"`
			ParticipantCount int    `json:"participant_count"`
			CharacterCount   int    `json:"character_count"`
			ThemePrompt      string `json:"theme_prompt"`
			CreatedAt        string `json:"created_at"`
			UpdatedAt        string `json:"updated_at"`
		} `json:"campaigns"`
	}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(payload.Campaigns) != 1 {
		t.Fatalf("expected 1 campaign, got %d", len(payload.Campaigns))
	}
	if payload.Campaigns[0].ID != "camp-1" {
		t.Fatalf("expected id camp-1, got %q", payload.Campaigns[0].ID)
	}
	if payload.Campaigns[0].GmMode != "HUMAN" {
		t.Fatalf("expected gm mode HUMAN, got %q", payload.Campaigns[0].GmMode)
	}
	if payload.Campaigns[0].CreatedAt != now.Format(time.RFC3339) {
		t.Fatalf("expected created_at %q, got %q", now.Format(time.RFC3339), payload.Campaigns[0].CreatedAt)
	}
	if payload.Campaigns[0].UpdatedAt != now.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("expected updated_at %q, got %q", now.Add(time.Hour).Format(time.RFC3339), payload.Campaigns[0].UpdatedAt)
	}
}

// TestCampaignResourceHandlerMapsResponse ensures JSON payload is formatted.
func TestCampaignResourceHandlerMapsResponse(t *testing.T) {
	now := time.Date(2026, 1, 23, 13, 0, 0, 0, time.UTC)
	client := &fakeCampaignClient{getCampaignResponse: &campaignv1.GetCampaignResponse{
		Campaign: &campaignv1.Campaign{
			Id:               "camp-1",
			Name:             "Red Sands",
			GmMode:           campaignv1.GmMode_HUMAN,
			ParticipantCount: 4,
			CharacterCount:   2,
			ThemePrompt:      "desert skies",
			CreatedAt:        timestamppb.New(now),
			UpdatedAt:        timestamppb.New(now.Add(time.Hour)),
		},
	}}

	handler := domain.CampaignResourceHandler(client)
	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "campaign://camp-1",
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || len(result.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %v", result)
	}
	if client.lastGetCampaignRequest == nil {
		t.Fatal("expected get campaign request")
	}
	if client.lastGetCampaignRequest.GetCampaignId() != "camp-1" {
		t.Fatalf("expected campaign id camp-1, got %q", client.lastGetCampaignRequest.GetCampaignId())
	}

	var payload struct {
		Campaign struct {
			ID               string `json:"id"`
			Name             string `json:"name"`
			GmMode           string `json:"gm_mode"`
			ParticipantCount int    `json:"participant_count"`
			CharacterCount   int    `json:"character_count"`
			ThemePrompt      string `json:"theme_prompt"`
			CreatedAt        string `json:"created_at"`
			UpdatedAt        string `json:"updated_at"`
		} `json:"campaign"`
	}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Campaign.ID != "camp-1" {
		t.Fatalf("expected id camp-1, got %q", payload.Campaign.ID)
	}
	if payload.Campaign.Name != "Red Sands" {
		t.Fatalf("expected name Red Sands, got %q", payload.Campaign.Name)
	}
	if payload.Campaign.GmMode != "HUMAN" {
		t.Fatalf("expected gm mode HUMAN, got %q", payload.Campaign.GmMode)
	}
	if payload.Campaign.ParticipantCount != 4 {
		t.Fatalf("expected participant_count 4, got %d", payload.Campaign.ParticipantCount)
	}
	if payload.Campaign.CharacterCount != 2 {
		t.Fatalf("expected character_count 2, got %d", payload.Campaign.CharacterCount)
	}
	if payload.Campaign.CreatedAt != now.Format(time.RFC3339) {
		t.Fatalf("expected created_at %q, got %q", now.Format(time.RFC3339), payload.Campaign.CreatedAt)
	}
	if payload.Campaign.UpdatedAt != now.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("expected updated_at %q, got %q", now.Add(time.Hour).Format(time.RFC3339), payload.Campaign.UpdatedAt)
	}
}

// TestCampaignResourceHandlerReturnsNotFound ensures NotFound errors are returned.
func TestCampaignResourceHandlerReturnsNotFound(t *testing.T) {
	client := &fakeCampaignClient{getCampaignErr: status.Error(codes.NotFound, "campaign not found")}
	handler := domain.CampaignResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "campaign://camp-999",
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if !strings.Contains(err.Error(), "campaign not found") {
		t.Fatalf("expected 'campaign not found' in error, got %q", err.Error())
	}
}

// TestCampaignResourceHandlerReturnsInvalidArgument ensures InvalidArgument errors are returned.
func TestCampaignResourceHandlerReturnsInvalidArgument(t *testing.T) {
	client := &fakeCampaignClient{getCampaignErr: status.Error(codes.InvalidArgument, "campaign id is required")}
	handler := domain.CampaignResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "campaign://invalid-id",
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if !strings.Contains(err.Error(), "invalid campaign_id") {
		t.Fatalf("expected 'invalid campaign_id' in error, got %q", err.Error())
	}
}

// TestCampaignResourceHandlerRejectsPlaceholder ensures placeholder URI is rejected.
func TestCampaignResourceHandlerRejectsPlaceholder(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CampaignResourceHandler(client)

	// When the URI matches the registered placeholder, it returns early with a specific error
	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "campaign://_",
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	// The handler checks if URI matches registered placeholder first, so we get the early return message
	if !strings.Contains(err.Error(), "campaign ID is required") {
		t.Fatalf("expected 'campaign ID is required' in error, got %q", err.Error())
	}
}

// TestCampaignResourceHandlerRejectsEmptyID ensures empty campaign ID is rejected.
func TestCampaignResourceHandlerRejectsEmptyID(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CampaignResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{
			URI: "campaign://",
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if !strings.Contains(err.Error(), "campaign ID is required") {
		t.Fatalf("expected 'campaign ID is required' in error, got %q", err.Error())
	}
}

// TestCampaignResourceHandlerRejectsSuffixedURI ensures URIs with path segments are rejected.
func TestCampaignResourceHandlerRejectsSuffixedURI(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CampaignResourceHandler(client)

	testCases := []struct {
		name string
		uri  string
	}{
		{"path segment", "campaign://camp-1/participants"},
		{"query parameter", "campaign://camp-1?foo=bar"},
		{"fragment", "campaign://camp-1#section"},
		{"path and query", "campaign://camp-1/participants?foo=bar"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := handler(context.Background(), &mcp.ReadResourceRequest{
				Params: &mcp.ReadResourceParams{
					URI: tc.uri,
				},
			})
			if err == nil {
				t.Fatal("expected error")
			}
			if result != nil {
				t.Fatal("expected nil result on error")
			}
			if !strings.Contains(err.Error(), "path segments") && !strings.Contains(err.Error(), "query parameters") && !strings.Contains(err.Error(), "fragments") {
				t.Fatalf("expected error about path segments/query parameters/fragments, got %q", err.Error())
			}
		})
	}
}

// TestParticipantCreateHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestParticipantCreateHandlerReturnsClientError(t *testing.T) {
	client := &fakeCampaignClient{createParticipantErr: errors.New("boom")}
	handler := domain.ParticipantCreateHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.ParticipantCreateInput{
		CampaignID:  "camp-123",
		DisplayName: "Test Player",
		Role:        "PLAYER",
		Controller:  "HUMAN",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestParticipantCreateHandlerMapsRequestAndResponse ensures inputs and outputs map consistently.
func TestParticipantCreateHandlerMapsRequestAndResponse(t *testing.T) {
	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	client := &fakeCampaignClient{createParticipantResponse: &campaignv1.CreateParticipantResponse{
		Participant: &campaignv1.Participant{
			Id:          "part-456",
			CampaignId:  "camp-123",
			DisplayName: "Test Player",
			Role:        campaignv1.ParticipantRole_PLAYER,
			Controller:  campaignv1.Controller_CONTROLLER_HUMAN,
			CreatedAt:   timestamppb.New(now),
			UpdatedAt:   timestamppb.New(now.Add(time.Hour)),
		},
	}}
	result, output, err := domain.ParticipantCreateHandler(client)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.ParticipantCreateInput{
			CampaignID:  "camp-123",
			DisplayName: "Test Player",
			Role:        "PLAYER",
			Controller:  "HUMAN",
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastCreateParticipantRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastCreateParticipantRequest.GetCampaignId() != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", client.lastCreateParticipantRequest.GetCampaignId())
	}
	if client.lastCreateParticipantRequest.GetDisplayName() != "Test Player" {
		t.Fatalf("expected display name Test Player, got %q", client.lastCreateParticipantRequest.GetDisplayName())
	}
	if client.lastCreateParticipantRequest.GetRole() != campaignv1.ParticipantRole_PLAYER {
		t.Fatalf("expected role PLAYER, got %v", client.lastCreateParticipantRequest.GetRole())
	}
	if client.lastCreateParticipantRequest.GetController() != campaignv1.Controller_CONTROLLER_HUMAN {
		t.Fatalf("expected controller HUMAN, got %v", client.lastCreateParticipantRequest.GetController())
	}
	if output.ID != "part-456" {
		t.Fatalf("expected id part-456, got %q", output.ID)
	}
	if output.CampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", output.CampaignID)
	}
	if output.DisplayName != "Test Player" {
		t.Fatalf("expected display name Test Player, got %q", output.DisplayName)
	}
	if output.Role != "PLAYER" {
		t.Fatalf("expected role PLAYER, got %q", output.Role)
	}
	if output.Controller != "HUMAN" {
		t.Fatalf("expected controller HUMAN, got %q", output.Controller)
	}
	if output.CreatedAt != now.Format(time.RFC3339) {
		t.Fatalf("expected created_at %q, got %q", now.Format(time.RFC3339), output.CreatedAt)
	}
	if output.UpdatedAt != now.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("expected updated_at %q, got %q", now.Add(time.Hour).Format(time.RFC3339), output.UpdatedAt)
	}
}

// TestParticipantCreateHandlerOptionalController ensures optional controller field works.
func TestParticipantCreateHandlerOptionalController(t *testing.T) {
	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	client := &fakeCampaignClient{createParticipantResponse: &campaignv1.CreateParticipantResponse{
		Participant: &campaignv1.Participant{
			Id:          "part-789",
			CampaignId:  "camp-123",
			DisplayName: "Test GM",
			Role:        campaignv1.ParticipantRole_GM,
			Controller:  campaignv1.Controller_CONTROLLER_HUMAN,
			CreatedAt:   timestamppb.New(now),
			UpdatedAt:   timestamppb.New(now),
		},
	}}
	result, output, err := domain.ParticipantCreateHandler(client)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.ParticipantCreateInput{
			CampaignID:  "camp-123",
			DisplayName: "Test GM",
			Role:        "GM",
			// Controller omitted
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastCreateParticipantRequest == nil {
		t.Fatal("expected gRPC request")
	}
	// Controller should be unspecified when not provided
	if client.lastCreateParticipantRequest.GetController() != campaignv1.Controller_CONTROLLER_UNSPECIFIED {
		t.Fatalf("expected controller UNSPECIFIED when omitted, got %v", client.lastCreateParticipantRequest.GetController())
	}
	if output.Role != "GM" {
		t.Fatalf("expected role GM, got %q", output.Role)
	}
}

// TestParticipantCreateHandlerRejectsEmptyResponse ensures nil responses are rejected.
func TestParticipantCreateHandlerRejectsEmptyResponse(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.ParticipantCreateHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.ParticipantCreateInput{
		CampaignID:  "camp-123",
		DisplayName: "Test Player",
		Role:        "PLAYER",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCharacterCreateHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestCharacterCreateHandlerReturnsClientError(t *testing.T) {
	client := &fakeCampaignClient{createCharacterErr: errors.New("boom")}
	handler := domain.CharacterCreateHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CharacterCreateInput{
		CampaignID: "camp-123",
		Name:       "Test Character",
		Kind:       "PC",
		Notes:      "A brave warrior",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCharacterCreateHandlerMapsRequestAndResponse ensures inputs and outputs map consistently.
func TestCharacterCreateHandlerMapsRequestAndResponse(t *testing.T) {
	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	client := &fakeCampaignClient{createCharacterResponse: &campaignv1.CreateCharacterResponse{
		Character: &campaignv1.Character{
			Id:         "character-456",
			CampaignId: "camp-123",
			Name:       "Test Character",
			Kind:       campaignv1.CharacterKind_PC,
			Notes:      "A brave warrior",
			CreatedAt:  timestamppb.New(now),
			UpdatedAt:  timestamppb.New(now.Add(time.Hour)),
		},
	}}
	result, output, err := domain.CharacterCreateHandler(client)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.CharacterCreateInput{
			CampaignID: "camp-123",
			Name:       "Test Character",
			Kind:       "PC",
			Notes:      "A brave warrior",
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastCreateCharacterRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastCreateCharacterRequest.GetCampaignId() != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", client.lastCreateCharacterRequest.GetCampaignId())
	}
	if client.lastCreateCharacterRequest.GetName() != "Test Character" {
		t.Fatalf("expected name Test Character, got %q", client.lastCreateCharacterRequest.GetName())
	}
	if client.lastCreateCharacterRequest.GetKind() != campaignv1.CharacterKind_PC {
		t.Fatalf("expected kind PC, got %v", client.lastCreateCharacterRequest.GetKind())
	}
	if client.lastCreateCharacterRequest.GetNotes() != "A brave warrior" {
		t.Fatalf("expected notes A brave warrior, got %q", client.lastCreateCharacterRequest.GetNotes())
	}
	if output.ID != "character-456" {
		t.Fatalf("expected id character-456, got %q", output.ID)
	}
	if output.CampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", output.CampaignID)
	}
	if output.Name != "Test Character" {
		t.Fatalf("expected name Test Character, got %q", output.Name)
	}
	if output.Kind != "PC" {
		t.Fatalf("expected kind PC, got %q", output.Kind)
	}
	if output.Notes != "A brave warrior" {
		t.Fatalf("expected notes A brave warrior, got %q", output.Notes)
	}
	if output.CreatedAt != now.Format(time.RFC3339) {
		t.Fatalf("expected created_at %q, got %q", now.Format(time.RFC3339), output.CreatedAt)
	}
	if output.UpdatedAt != now.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("expected updated_at %q, got %q", now.Add(time.Hour).Format(time.RFC3339), output.UpdatedAt)
	}
}

// TestCharacterCreateHandlerOptionalNotes ensures optional notes field works.
func TestCharacterCreateHandlerOptionalNotes(t *testing.T) {
	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	client := &fakeCampaignClient{createCharacterResponse: &campaignv1.CreateCharacterResponse{
		Character: &campaignv1.Character{
			Id:         "character-789",
			CampaignId: "camp-123",
			Name:       "Test NPC",
			Kind:       campaignv1.CharacterKind_NPC,
			Notes:      "",
			CreatedAt:  timestamppb.New(now),
			UpdatedAt:  timestamppb.New(now),
		},
	}}
	result, output, err := domain.CharacterCreateHandler(client)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.CharacterCreateInput{
			CampaignID: "camp-123",
			Name:       "Test NPC",
			Kind:       "NPC",
			// Notes omitted
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastCreateCharacterRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastCreateCharacterRequest.GetNotes() != "" {
		t.Fatalf("expected empty notes when omitted, got %q", client.lastCreateCharacterRequest.GetNotes())
	}
	if output.Kind != "NPC" {
		t.Fatalf("expected kind NPC, got %q", output.Kind)
	}
}

// TestCharacterCreateHandlerRejectsEmptyResponse ensures nil responses are rejected.
func TestCharacterCreateHandlerRejectsEmptyResponse(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CharacterCreateHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CharacterCreateInput{
		CampaignID: "camp-123",
		Name:       "Test Character",
		Kind:       "PC",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCharacterControlSetHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestCharacterControlSetHandlerReturnsClientError(t *testing.T) {
	client := &fakeCampaignClient{setDefaultControlErr: errors.New("boom")}
	handler := domain.CharacterControlSetHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CharacterControlSetInput{
		CampaignID:  "camp-123",
		CharacterID: "character-456",
		Controller:  "GM",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCharacterControlSetHandlerMapsRequestAndResponseGM ensures GM controller inputs and outputs map consistently.
func TestCharacterControlSetHandlerMapsRequestAndResponseGM(t *testing.T) {
	client := &fakeCampaignClient{setDefaultControlResponse: &campaignv1.SetDefaultControlResponse{
		CampaignId:  "camp-123",
		CharacterId: "character-456",
		Controller: &campaignv1.CharacterController{
			Controller: &campaignv1.CharacterController_Gm{
				Gm: &campaignv1.GmController{},
			},
		},
	}}
	result, output, err := domain.CharacterControlSetHandler(client)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.CharacterControlSetInput{
			CampaignID:  "camp-123",
			CharacterID: "character-456",
			Controller:  "GM",
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastSetDefaultControlRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastSetDefaultControlRequest.GetCampaignId() != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", client.lastSetDefaultControlRequest.GetCampaignId())
	}
	if client.lastSetDefaultControlRequest.GetCharacterId() != "character-456" {
		t.Fatalf("expected character id character-456, got %q", client.lastSetDefaultControlRequest.GetCharacterId())
	}
	controller := client.lastSetDefaultControlRequest.GetController()
	if controller == nil {
		t.Fatal("expected controller in request")
	}
	if _, ok := controller.GetController().(*campaignv1.CharacterController_Gm); !ok {
		t.Fatalf("expected GM controller, got %T", controller.GetController())
	}
	if output.CampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", output.CampaignID)
	}
	if output.CharacterID != "character-456" {
		t.Fatalf("expected character id character-456, got %q", output.CharacterID)
	}
	if output.Controller != "GM" {
		t.Fatalf("expected controller GM, got %q", output.Controller)
	}
}

// TestCharacterControlSetHandlerMapsRequestAndResponseParticipant ensures participant controller inputs and outputs map consistently.
func TestCharacterControlSetHandlerMapsRequestAndResponseParticipant(t *testing.T) {
	participantID := "part-789"
	client := &fakeCampaignClient{setDefaultControlResponse: &campaignv1.SetDefaultControlResponse{
		CampaignId:  "camp-123",
		CharacterId: "character-456",
		Controller: &campaignv1.CharacterController{
			Controller: &campaignv1.CharacterController_Participant{
				Participant: &campaignv1.ParticipantController{
					ParticipantId: participantID,
				},
			},
		},
	}}
	result, output, err := domain.CharacterControlSetHandler(client)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.CharacterControlSetInput{
			CampaignID:  "camp-123",
			CharacterID: "character-456",
			Controller:  participantID,
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastSetDefaultControlRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastSetDefaultControlRequest.GetCampaignId() != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", client.lastSetDefaultControlRequest.GetCampaignId())
	}
	if client.lastSetDefaultControlRequest.GetCharacterId() != "character-456" {
		t.Fatalf("expected character id character-456, got %q", client.lastSetDefaultControlRequest.GetCharacterId())
	}
	controller := client.lastSetDefaultControlRequest.GetController()
	if controller == nil {
		t.Fatal("expected controller in request")
	}
	participantCtrl, ok := controller.GetController().(*campaignv1.CharacterController_Participant)
	if !ok {
		t.Fatalf("expected participant controller, got %T", controller.GetController())
	}
	if participantCtrl.Participant.GetParticipantId() != participantID {
		t.Fatalf("expected participant id %q, got %q", participantID, participantCtrl.Participant.GetParticipantId())
	}
	if output.CampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", output.CampaignID)
	}
	if output.CharacterID != "character-456" {
		t.Fatalf("expected character id character-456, got %q", output.CharacterID)
	}
	if output.Controller != participantID {
		t.Fatalf("expected controller %q, got %q", participantID, output.Controller)
	}
}

// TestCharacterControlSetHandlerCaseInsensitiveGM ensures GM controller accepts case-insensitive input.
func TestCharacterControlSetHandlerCaseInsensitiveGM(t *testing.T) {
	client := &fakeCampaignClient{setDefaultControlResponse: &campaignv1.SetDefaultControlResponse{
		CampaignId:  "camp-123",
		CharacterId: "character-456",
		Controller: &campaignv1.CharacterController{
			Controller: &campaignv1.CharacterController_Gm{
				Gm: &campaignv1.GmController{},
			},
		},
	}}
	for _, input := range []string{"GM", "gm", "Gm", "gM"} {
		t.Run(input, func(t *testing.T) {
			_, output, err := domain.CharacterControlSetHandler(client)(
				context.Background(),
				&mcp.CallToolRequest{},
				domain.CharacterControlSetInput{
					CampaignID:  "camp-123",
					CharacterID: "character-456",
					Controller:  input,
				},
			)
			if err != nil {
				t.Fatalf("expected no error for %q, got %v", input, err)
			}
			if output.Controller != "GM" {
				t.Fatalf("expected controller GM, got %q", output.Controller)
			}
		})
	}
}

// TestCharacterControlSetHandlerRejectsEmptyResponse ensures nil responses are rejected.
func TestCharacterControlSetHandlerRejectsEmptyResponse(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CharacterControlSetHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CharacterControlSetInput{
		CampaignID:  "camp-123",
		CharacterID: "character-456",
		Controller:  "GM",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCharacterControlSetHandlerRejectsEmptyController ensures empty controller is rejected.
func TestCharacterControlSetHandlerRejectsEmptyController(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CharacterControlSetHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.CharacterControlSetInput{
		CampaignID:  "camp-123",
		CharacterID: "character-456",
		Controller:  "",
	})
	if err == nil {
		t.Fatal("expected error for empty controller")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestSessionStartHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestSessionStartHandlerReturnsClientError(t *testing.T) {
	client := &fakeSessionClient{err: errors.New("boom")}
	handler := domain.SessionStartHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SessionStartInput{
		CampaignID: "camp-123",
		Name:       "Test Session",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestSessionStartHandlerMapsRequestAndResponse ensures inputs and outputs map consistently.
func TestSessionStartHandlerMapsRequestAndResponse(t *testing.T) {
	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	client := &fakeSessionClient{startSessionResponse: &sessionv1.StartSessionResponse{
		Session: &sessionv1.Session{
			Id:         "sess-456",
			CampaignId: "camp-123",
			Name:       "Test Session",
			Status:     sessionv1.SessionStatus_ACTIVE,
			StartedAt:  timestamppb.New(now),
			UpdatedAt:  timestamppb.New(now.Add(time.Hour)),
		},
	}}
	result, output, err := domain.SessionStartHandler(client)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.SessionStartInput{
			CampaignID: "camp-123",
			Name:       "Test Session",
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastRequest.GetCampaignId() != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", client.lastRequest.GetCampaignId())
	}
	if client.lastRequest.GetName() != "Test Session" {
		t.Fatalf("expected name Test Session, got %q", client.lastRequest.GetName())
	}
	if output.ID != "sess-456" {
		t.Fatalf("expected id sess-456, got %q", output.ID)
	}
	if output.CampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", output.CampaignID)
	}
	if output.Name != "Test Session" {
		t.Fatalf("expected name Test Session, got %q", output.Name)
	}
	if output.Status != "ACTIVE" {
		t.Fatalf("expected status ACTIVE, got %q", output.Status)
	}
	if output.StartedAt != now.Format(time.RFC3339) {
		t.Fatalf("expected started_at %q, got %q", now.Format(time.RFC3339), output.StartedAt)
	}
	if output.UpdatedAt != now.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("expected updated_at %q, got %q", now.Add(time.Hour).Format(time.RFC3339), output.UpdatedAt)
	}
}

// TestSessionStartHandlerOptionalName ensures optional name field works.
func TestSessionStartHandlerOptionalName(t *testing.T) {
	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	client := &fakeSessionClient{startSessionResponse: &sessionv1.StartSessionResponse{
		Session: &sessionv1.Session{
			Id:         "sess-789",
			CampaignId: "camp-123",
			Name:       "",
			Status:     sessionv1.SessionStatus_ACTIVE,
			StartedAt:  timestamppb.New(now),
			UpdatedAt:  timestamppb.New(now),
		},
	}}
	result, output, err := domain.SessionStartHandler(client)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.SessionStartInput{
			CampaignID: "camp-123",
			// Name omitted
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if client.lastRequest == nil {
		t.Fatal("expected gRPC request")
	}
	if client.lastRequest.GetName() != "" {
		t.Fatalf("expected empty name when omitted, got %q", client.lastRequest.GetName())
	}
	if output.Name != "" {
		t.Fatalf("expected empty name, got %q", output.Name)
	}
	if output.Status != "ACTIVE" {
		t.Fatalf("expected status ACTIVE, got %q", output.Status)
	}
}

// TestSessionStartHandlerRejectsEmptyResponse ensures nil responses are rejected.
func TestSessionStartHandlerRejectsEmptyResponse(t *testing.T) {
	client := &fakeSessionClient{}
	handler := domain.SessionStartHandler(client)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SessionStartInput{
		CampaignID: "camp-123",
		Name:       "Test Session",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestSessionStartHandlerMapsEndedAt ensures ended_at is mapped when present.
func TestSessionStartHandlerMapsEndedAt(t *testing.T) {
	now := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	endedAt := now.Add(2 * time.Hour)
	client := &fakeSessionClient{startSessionResponse: &sessionv1.StartSessionResponse{
		Session: &sessionv1.Session{
			Id:         "sess-999",
			CampaignId: "camp-123",
			Name:       "Ended Session",
			Status:     sessionv1.SessionStatus_ENDED,
			StartedAt:  timestamppb.New(now),
			UpdatedAt:  timestamppb.New(endedAt),
			EndedAt:    timestamppb.New(endedAt),
		},
	}}
	result, output, err := domain.SessionStartHandler(client)(
		context.Background(),
		&mcp.CallToolRequest{},
		domain.SessionStartInput{
			CampaignID: "camp-123",
			Name:       "Ended Session",
		},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if output.EndedAt != endedAt.Format(time.RFC3339) {
		t.Fatalf("expected ended_at %q, got %q", endedAt.Format(time.RFC3339), output.EndedAt)
	}
	if output.Status != "ENDED" {
		t.Fatalf("expected status ENDED, got %q", output.Status)
	}
}

// TestParticipantListResourceHandlerMapsResponse ensures JSON payload is formatted correctly.
func TestParticipantListResourceHandlerMapsResponse(t *testing.T) {
	now := time.Date(2026, 1, 23, 13, 0, 0, 0, time.UTC)
	campaignID := "camp-456"
	client := &fakeCampaignClient{listParticipantsResponse: &campaignv1.ListParticipantsResponse{
		Participants: []*campaignv1.Participant{{
			Id:          "part-1",
			CampaignId:  campaignID,
			DisplayName: "Test Player",
			Role:        campaignv1.ParticipantRole_PLAYER,
			Controller:  campaignv1.Controller_CONTROLLER_HUMAN,
			CreatedAt:   timestamppb.New(now),
			UpdatedAt:   timestamppb.New(now.Add(time.Hour)),
		}, {
			Id:          "part-2",
			CampaignId:  campaignID,
			DisplayName: "Test GM",
			Role:        campaignv1.ParticipantRole_GM,
			Controller:  campaignv1.Controller_CONTROLLER_AI,
			CreatedAt:   timestamppb.New(now),
			UpdatedAt:   timestamppb.New(now),
		}},
	}}

	handler := domain.ParticipantListResourceHandler(client)
	resourceURI := "campaign://" + campaignID + "/participants"
	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: resourceURI},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || len(result.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %v", result)
	}
	if client.lastListParticipantsRequest == nil {
		t.Fatal("expected list participants request")
	}
	if client.lastListParticipantsRequest.GetCampaignId() != campaignID {
		t.Fatalf("expected campaign id %q, got %q", campaignID, client.lastListParticipantsRequest.GetCampaignId())
	}
	if client.lastListParticipantsRequest.GetPageSize() != 10 {
		t.Fatalf("expected page size 10, got %d", client.lastListParticipantsRequest.GetPageSize())
	}

	var payload struct {
		Participants []struct {
			ID          string `json:"id"`
			CampaignID  string `json:"campaign_id"`
			DisplayName string `json:"display_name"`
			Role        string `json:"role"`
			Controller  string `json:"controller"`
			CreatedAt   string `json:"created_at"`
			UpdatedAt   string `json:"updated_at"`
		} `json:"participants"`
	}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(payload.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(payload.Participants))
	}
	if payload.Participants[0].ID != "part-1" {
		t.Fatalf("expected first participant id part-1, got %q", payload.Participants[0].ID)
	}
	if payload.Participants[0].Role != "PLAYER" {
		t.Fatalf("expected first participant role PLAYER, got %q", payload.Participants[0].Role)
	}
	if payload.Participants[1].ID != "part-2" {
		t.Fatalf("expected second participant id part-2, got %q", payload.Participants[1].ID)
	}
	if payload.Participants[1].Role != "GM" {
		t.Fatalf("expected second participant role GM, got %q", payload.Participants[1].Role)
	}
	if payload.Participants[0].CreatedAt != now.Format(time.RFC3339) {
		t.Fatalf("expected created_at %q, got %q", now.Format(time.RFC3339), payload.Participants[0].CreatedAt)
	}
	if result.Contents[0].URI != resourceURI {
		t.Fatalf("expected resource URI %q, got %q", resourceURI, result.Contents[0].URI)
	}
}

// TestParticipantListResourceHandlerRejectsPlaceholder ensures placeholder campaign ID is rejected.
func TestParticipantListResourceHandlerRejectsPlaceholder(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.ParticipantListResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "campaign://_/participants"},
	})
	if err == nil {
		t.Fatal("expected error for placeholder campaign ID")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestParticipantListResourceHandlerReturnsClientError ensures list errors are returned.
func TestParticipantListResourceHandlerReturnsClientError(t *testing.T) {
	client := &fakeCampaignClient{listParticipantsErr: errors.New("boom")}
	handler := domain.ParticipantListResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "campaign://camp-123/participants"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCharacterListResourceHandlerMapsResponse ensures JSON payload is formatted correctly.
func TestCharacterListResourceHandlerMapsResponse(t *testing.T) {
	now := time.Date(2026, 1, 23, 13, 0, 0, 0, time.UTC)
	campaignID := "camp-789"
	client := &fakeCampaignClient{listCharactersResponse: &campaignv1.ListCharactersResponse{
		Characters: []*campaignv1.Character{{
			Id:         "character-1",
			CampaignId: campaignID,
			Name:       "Test PC",
			Kind:       campaignv1.CharacterKind_PC,
			Notes:      "A brave warrior",
			CreatedAt:  timestamppb.New(now),
			UpdatedAt:  timestamppb.New(now.Add(time.Hour)),
		}, {
			Id:         "character-2",
			CampaignId: campaignID,
			Name:       "Test NPC",
			Kind:       campaignv1.CharacterKind_NPC,
			Notes:      "A helpful merchant",
			CreatedAt:  timestamppb.New(now),
			UpdatedAt:  timestamppb.New(now),
		}},
	}}

	handler := domain.CharacterListResourceHandler(client)
	resourceURI := "campaign://" + campaignID + "/characters"
	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: resourceURI},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || len(result.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %v", result)
	}
	if client.lastListCharactersRequest == nil {
		t.Fatal("expected list characters request")
	}
	if client.lastListCharactersRequest.GetCampaignId() != campaignID {
		t.Fatalf("expected campaign id %q, got %q", campaignID, client.lastListCharactersRequest.GetCampaignId())
	}
	if client.lastListCharactersRequest.GetPageSize() != 10 {
		t.Fatalf("expected page size 10, got %d", client.lastListCharactersRequest.GetPageSize())
	}

	var payload struct {
		Characters []struct {
			ID         string `json:"id"`
			CampaignID string `json:"campaign_id"`
			Name       string `json:"name"`
			Kind       string `json:"kind"`
			Notes      string `json:"notes"`
			CreatedAt  string `json:"created_at"`
			UpdatedAt  string `json:"updated_at"`
		} `json:"characters"`
	}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(payload.Characters) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(payload.Characters))
	}
	if payload.Characters[0].ID != "character-1" {
		t.Fatalf("expected first character id character-1, got %q", payload.Characters[0].ID)
	}
	if payload.Characters[0].Kind != "PC" {
		t.Fatalf("expected first character kind PC, got %q", payload.Characters[0].Kind)
	}
	if payload.Characters[1].ID != "character-2" {
		t.Fatalf("expected second character id character-2, got %q", payload.Characters[1].ID)
	}
	if payload.Characters[1].Kind != "NPC" {
		t.Fatalf("expected second character kind NPC, got %q", payload.Characters[1].Kind)
	}
	if payload.Characters[0].CreatedAt != now.Format(time.RFC3339) {
		t.Fatalf("expected created_at %q, got %q", now.Format(time.RFC3339), payload.Characters[0].CreatedAt)
	}
	if result.Contents[0].URI != resourceURI {
		t.Fatalf("expected resource URI %q, got %q", resourceURI, result.Contents[0].URI)
	}
}

// TestCharacterListResourceHandlerRejectsPlaceholder ensures placeholder campaign ID is rejected.
func TestCharacterListResourceHandlerRejectsPlaceholder(t *testing.T) {
	client := &fakeCampaignClient{}
	handler := domain.CharacterListResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "campaign://_/characters"},
	})
	if err == nil {
		t.Fatal("expected error for placeholder campaign ID")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestCharacterListResourceHandlerReturnsClientError ensures list errors are returned.
func TestCharacterListResourceHandlerReturnsClientError(t *testing.T) {
	client := &fakeCampaignClient{listCharactersErr: errors.New("boom")}
	handler := domain.CharacterListResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "campaign://camp-123/characters"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestSessionListResourceHandlerMapsResponse ensures JSON payload is formatted correctly.
func TestSessionListResourceHandlerMapsResponse(t *testing.T) {
	now := time.Date(2026, 1, 23, 13, 0, 0, 0, time.UTC)
	endedAt := now.Add(2 * time.Hour)
	campaignID := "camp-999"
	client := &fakeSessionClient{listSessionsResponse: &sessionv1.ListSessionsResponse{
		Sessions: []*sessionv1.Session{{
			Id:         "sess-1",
			CampaignId: campaignID,
			Name:       "Session One",
			Status:     sessionv1.SessionStatus_ACTIVE,
			StartedAt:  timestamppb.New(now),
			UpdatedAt:  timestamppb.New(now.Add(time.Hour)),
		}, {
			Id:         "sess-2",
			CampaignId: campaignID,
			Name:       "Session Two",
			Status:     sessionv1.SessionStatus_ENDED,
			StartedAt:  timestamppb.New(now),
			UpdatedAt:  timestamppb.New(endedAt),
			EndedAt:    timestamppb.New(endedAt),
		}},
	}}

	handler := domain.SessionListResourceHandler(client)
	resourceURI := "campaign://" + campaignID + "/sessions"
	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: resourceURI},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || len(result.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %v", result)
	}
	if client.lastListSessionsRequest == nil {
		t.Fatal("expected list sessions request")
	}
	if client.lastListSessionsRequest.GetCampaignId() != campaignID {
		t.Fatalf("expected campaign id %q, got %q", campaignID, client.lastListSessionsRequest.GetCampaignId())
	}
	if client.lastListSessionsRequest.GetPageSize() != 10 {
		t.Fatalf("expected page size 10, got %d", client.lastListSessionsRequest.GetPageSize())
	}

	var payload struct {
		Sessions []struct {
			ID         string `json:"id"`
			CampaignID string `json:"campaign_id"`
			Name       string `json:"name"`
			Status     string `json:"status"`
			StartedAt  string `json:"started_at"`
			UpdatedAt  string `json:"updated_at"`
			EndedAt    string `json:"ended_at,omitempty"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(payload.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(payload.Sessions))
	}
	if payload.Sessions[0].ID != "sess-1" {
		t.Fatalf("expected first session id sess-1, got %q", payload.Sessions[0].ID)
	}
	if payload.Sessions[0].Status != "ACTIVE" {
		t.Fatalf("expected first session status ACTIVE, got %q", payload.Sessions[0].Status)
	}
	if payload.Sessions[1].ID != "sess-2" {
		t.Fatalf("expected second session id sess-2, got %q", payload.Sessions[1].ID)
	}
	if payload.Sessions[1].Status != "ENDED" {
		t.Fatalf("expected second session status ENDED, got %q", payload.Sessions[1].Status)
	}
	if payload.Sessions[0].StartedAt != now.Format(time.RFC3339) {
		t.Fatalf("expected started_at %q, got %q", now.Format(time.RFC3339), payload.Sessions[0].StartedAt)
	}
	if payload.Sessions[1].EndedAt != endedAt.Format(time.RFC3339) {
		t.Fatalf("expected ended_at %q, got %q", endedAt.Format(time.RFC3339), payload.Sessions[1].EndedAt)
	}
	if result.Contents[0].URI != resourceURI {
		t.Fatalf("expected resource URI %q, got %q", resourceURI, result.Contents[0].URI)
	}
}

// TestSessionListResourceHandlerRejectsPlaceholder ensures placeholder campaign ID is rejected.
func TestSessionListResourceHandlerRejectsPlaceholder(t *testing.T) {
	client := &fakeSessionClient{}
	handler := domain.SessionListResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "campaign://_/sessions"},
	})
	if err == nil {
		t.Fatal("expected error for placeholder campaign ID")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestSessionListResourceHandlerReturnsClientError ensures list errors are returned.
func TestSessionListResourceHandlerReturnsClientError(t *testing.T) {
	client := &fakeSessionClient{listSessionsErr: errors.New("boom")}
	handler := domain.SessionListResourceHandler(client)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "campaign://camp-123/sessions"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestSetContextHandlerReturnsClientError ensures gRPC errors are returned as tool errors.
func TestSetContextHandlerReturnsClientError(t *testing.T) {
	campaignClient := &fakeCampaignClient{getCampaignErr: errors.New("boom")}
	sessionClient := &fakeSessionClient{}
	server := &Server{}
	handler := domain.SetContextHandler(campaignClient, sessionClient, server.setContext, server.getContext)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID: "camp-123",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestSetContextHandlerRejectsEmptyCampaignID validates empty campaign_id is rejected.
func TestSetContextHandlerRejectsEmptyCampaignID(t *testing.T) {
	campaignClient := &fakeCampaignClient{}
	sessionClient := &fakeSessionClient{}
	server := &Server{}
	handler := domain.SetContextHandler(campaignClient, sessionClient, server.setContext, server.getContext)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID: "",
	})
	if err == nil {
		t.Fatal("expected error for empty campaign_id")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if campaignClient.lastGetCampaignRequest != nil {
		t.Fatal("expected no GetCampaign call for empty campaign_id")
	}
}

// TestSetContextHandlerRejectsNonExistentCampaign validates non-existent campaign returns error.
func TestSetContextHandlerRejectsNonExistentCampaign(t *testing.T) {
	campaignClient := &fakeCampaignClient{
		getCampaignErr: status.Error(codes.NotFound, "campaign not found"),
	}
	sessionClient := &fakeSessionClient{}
	server := &Server{}
	handler := domain.SetContextHandler(campaignClient, sessionClient, server.setContext, server.getContext)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID: "camp-123",
	})
	if err == nil {
		t.Fatal("expected error for non-existent campaign")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if campaignClient.lastGetCampaignRequest == nil {
		t.Fatal("expected GetCampaign call")
	}
	if campaignClient.lastGetCampaignRequest.GetCampaignId() != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", campaignClient.lastGetCampaignRequest.GetCampaignId())
	}
}

// TestSetContextHandlerTreatsWhitespaceOnlySessionIDAsOmitted validates whitespace-only session_id is treated as omitted.
func TestSetContextHandlerTreatsWhitespaceOnlySessionIDAsOmitted(t *testing.T) {
	campaignClient := &fakeCampaignClient{
		getCampaignResponse: &campaignv1.GetCampaignResponse{
			Campaign: &campaignv1.Campaign{Id: "camp-123"},
		},
	}
	sessionClient := &fakeSessionClient{}
	server := &Server{}
	handler := domain.SetContextHandler(campaignClient, sessionClient, server.setContext, server.getContext)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID: "camp-123",
		SessionID:  "   ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if output.Context.SessionID != "" {
		t.Fatalf("expected empty session_id after trim, got %q", output.Context.SessionID)
	}
	if sessionClient.lastGetSessionRequest != nil {
		t.Fatal("expected no GetSession call for whitespace-only session_id")
	}
}

// TestSetContextHandlerRejectsNonExistentSession validates non-existent session returns error.
func TestSetContextHandlerRejectsNonExistentSession(t *testing.T) {
	campaignClient := &fakeCampaignClient{
		getCampaignResponse: &campaignv1.GetCampaignResponse{
			Campaign: &campaignv1.Campaign{Id: "camp-123"},
		},
	}
	sessionClient := &fakeSessionClient{
		getSessionErr: status.Error(codes.NotFound, "session not found"),
	}
	server := &Server{}
	handler := domain.SetContextHandler(campaignClient, sessionClient, server.setContext, server.getContext)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID: "camp-123",
		SessionID:  "sess-456",
	})
	if err == nil {
		t.Fatal("expected error for non-existent session")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if sessionClient.lastGetSessionRequest == nil {
		t.Fatal("expected GetSession call")
	}
	if sessionClient.lastGetSessionRequest.GetCampaignId() != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", sessionClient.lastGetSessionRequest.GetCampaignId())
	}
	if sessionClient.lastGetSessionRequest.GetSessionId() != "sess-456" {
		t.Fatalf("expected session id sess-456, got %q", sessionClient.lastGetSessionRequest.GetSessionId())
	}
}

// TestSetContextHandlerRejectsSessionFromDifferentCampaign validates session belonging to different campaign returns error.
func TestSetContextHandlerRejectsSessionFromDifferentCampaign(t *testing.T) {
	campaignClient := &fakeCampaignClient{
		getCampaignResponse: &campaignv1.GetCampaignResponse{
			Campaign: &campaignv1.Campaign{Id: "camp-123"},
		},
	}
	sessionClient := &fakeSessionClient{
		getSessionErr: status.Error(codes.InvalidArgument, "session not found or does not belong to campaign"),
	}
	server := &Server{}
	handler := domain.SetContextHandler(campaignClient, sessionClient, server.setContext, server.getContext)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID: "camp-123",
		SessionID:  "sess-456",
	})
	if err == nil {
		t.Fatal("expected error for session from different campaign")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestSetContextHandlerTreatsWhitespaceOnlyParticipantIDAsOmitted validates whitespace-only participant_id is treated as omitted.
func TestSetContextHandlerTreatsWhitespaceOnlyParticipantIDAsOmitted(t *testing.T) {
	campaignClient := &fakeCampaignClient{
		getCampaignResponse: &campaignv1.GetCampaignResponse{
			Campaign: &campaignv1.Campaign{Id: "camp-123"},
		},
	}
	sessionClient := &fakeSessionClient{}
	server := &Server{}
	handler := domain.SetContextHandler(campaignClient, sessionClient, server.setContext, server.getContext)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID:    "camp-123",
		ParticipantID: "   ",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if output.Context.ParticipantID != "" {
		t.Fatalf("expected empty participant_id after trim, got %q", output.Context.ParticipantID)
	}
	if campaignClient.lastGetParticipantRequest != nil {
		t.Fatal("expected no GetParticipant call for whitespace-only participant_id")
	}
}

// TestSetContextHandlerRejectsNonExistentParticipant validates non-existent participant returns error.
func TestSetContextHandlerRejectsNonExistentParticipant(t *testing.T) {
	campaignClient := &fakeCampaignClient{
		getCampaignResponse: &campaignv1.GetCampaignResponse{
			Campaign: &campaignv1.Campaign{Id: "camp-123"},
		},
		getParticipantErr: status.Error(codes.NotFound, "participant not found"),
	}
	sessionClient := &fakeSessionClient{}
	server := &Server{}
	handler := domain.SetContextHandler(campaignClient, sessionClient, server.setContext, server.getContext)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID:    "camp-123",
		ParticipantID: "part-456",
	})
	if err == nil {
		t.Fatal("expected error for non-existent participant")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if campaignClient.lastGetParticipantRequest == nil {
		t.Fatal("expected GetParticipant call")
	}
	if campaignClient.lastGetParticipantRequest.GetCampaignId() != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", campaignClient.lastGetParticipantRequest.GetCampaignId())
	}
	if campaignClient.lastGetParticipantRequest.GetParticipantId() != "part-456" {
		t.Fatalf("expected participant id part-456, got %q", campaignClient.lastGetParticipantRequest.GetParticipantId())
	}
}

// TestSetContextHandlerRejectsParticipantFromDifferentCampaign validates participant belonging to different campaign returns error.
func TestSetContextHandlerRejectsParticipantFromDifferentCampaign(t *testing.T) {
	campaignClient := &fakeCampaignClient{
		getCampaignResponse: &campaignv1.GetCampaignResponse{
			Campaign: &campaignv1.Campaign{Id: "camp-123"},
		},
		getParticipantErr: status.Error(codes.InvalidArgument, "participant not found or does not belong to campaign"),
	}
	sessionClient := &fakeSessionClient{}
	server := &Server{}
	handler := domain.SetContextHandler(campaignClient, sessionClient, server.setContext, server.getContext)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID:    "camp-123",
		ParticipantID: "part-456",
	})
	if err == nil {
		t.Fatal("expected error for participant from different campaign")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestSetContextHandlerMapsRequestAndResponse ensures context is properly set and returned with all fields.
func TestSetContextHandlerMapsRequestAndResponse(t *testing.T) {
	campaignClient := &fakeCampaignClient{
		getCampaignResponse: &campaignv1.GetCampaignResponse{
			Campaign: &campaignv1.Campaign{Id: "camp-123"},
		},
		getParticipantResponse: &campaignv1.GetParticipantResponse{
			Participant: &campaignv1.Participant{
				Id:         "part-456",
				CampaignId: "camp-123",
			},
		},
	}
	sessionClient := &fakeSessionClient{
		getSessionResponse: &sessionv1.GetSessionResponse{
			Session: &sessionv1.Session{
				Id:         "sess-789",
				CampaignId: "camp-123",
			},
		},
	}
	server := &Server{}
	handler := domain.SetContextHandler(campaignClient, sessionClient, server.setContext, server.getContext)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID:    "camp-123",
		SessionID:     "sess-789",
		ParticipantID: "part-456",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if output.Context.CampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", output.Context.CampaignID)
	}
	if output.Context.SessionID != "sess-789" {
		t.Fatalf("expected session id sess-789, got %q", output.Context.SessionID)
	}
	if output.Context.ParticipantID != "part-456" {
		t.Fatalf("expected participant id part-456, got %q", output.Context.ParticipantID)
	}
	// Verify context was set on server
	currentCtx := server.getContext()
	if currentCtx.CampaignID != "camp-123" {
		t.Fatalf("expected server context campaign id camp-123, got %q", currentCtx.CampaignID)
	}
	if currentCtx.SessionID != "sess-789" {
		t.Fatalf("expected server context session id sess-789, got %q", currentCtx.SessionID)
	}
	if currentCtx.ParticipantID != "part-456" {
		t.Fatalf("expected server context participant id part-456, got %q", currentCtx.ParticipantID)
	}
}

// TestSetContextHandlerOptionalFields tests that optional session_id and participant_id can be omitted.
func TestSetContextHandlerOptionalFields(t *testing.T) {
	campaignClient := &fakeCampaignClient{
		getCampaignResponse: &campaignv1.GetCampaignResponse{
			Campaign: &campaignv1.Campaign{Id: "camp-123"},
		},
	}
	sessionClient := &fakeSessionClient{}
	server := &Server{}
	handler := domain.SetContextHandler(campaignClient, sessionClient, server.setContext, server.getContext)

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID: "camp-123",
		// SessionID and ParticipantID omitted
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	requireToolMetadata(t, result)
	if output.Context.CampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", output.Context.CampaignID)
	}
	if output.Context.SessionID != "" {
		t.Fatalf("expected empty session_id, got %q", output.Context.SessionID)
	}
	if output.Context.ParticipantID != "" {
		t.Fatalf("expected empty participant_id, got %q", output.Context.ParticipantID)
	}
	// Verify context was set on server
	currentCtx := server.getContext()
	if currentCtx.CampaignID != "camp-123" {
		t.Fatalf("expected server context campaign id camp-123, got %q", currentCtx.CampaignID)
	}
	if currentCtx.SessionID != "" {
		t.Fatalf("expected server context session id empty, got %q", currentCtx.SessionID)
	}
	if currentCtx.ParticipantID != "" {
		t.Fatalf("expected server context participant id empty, got %q", currentCtx.ParticipantID)
	}
}

// TestSetContextHandlerClearsOptionalFields tests that omitting optional fields clears them from context.
func TestSetContextHandlerClearsOptionalFields(t *testing.T) {
	campaignClient := &fakeCampaignClient{
		getCampaignResponse: &campaignv1.GetCampaignResponse{
			Campaign: &campaignv1.Campaign{Id: "camp-123"},
		},
		getParticipantResponse: &campaignv1.GetParticipantResponse{
			Participant: &campaignv1.Participant{
				Id:         "part-456",
				CampaignId: "camp-123",
			},
		},
	}
	sessionClient := &fakeSessionClient{
		getSessionResponse: &sessionv1.GetSessionResponse{
			Session: &sessionv1.Session{
				Id:         "sess-789",
				CampaignId: "camp-123",
			},
		},
	}
	server := &Server{}
	handler := domain.SetContextHandler(campaignClient, sessionClient, server.setContext, server.getContext)

	// First set context with all fields
	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID:    "camp-123",
		SessionID:     "sess-789",
		ParticipantID: "part-456",
	})
	if err != nil {
		t.Fatalf("first set context: %v", err)
	}

	// Verify initial context has all fields
	initialCtx := server.getContext()
	if initialCtx.SessionID == "" || initialCtx.ParticipantID == "" {
		t.Fatal("expected initial context to have session and participant")
	}

	// Now set context with only campaign_id (omitting session and participant)
	_, output, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID: "camp-123",
		// SessionID and ParticipantID omitted
	})
	if err != nil {
		t.Fatalf("second set context: %v", err)
	}
	if output.Context.SessionID != "" {
		t.Fatalf("expected session_id to be cleared, got %q", output.Context.SessionID)
	}
	if output.Context.ParticipantID != "" {
		t.Fatalf("expected participant_id to be cleared, got %q", output.Context.ParticipantID)
	}
	// Verify context was cleared on server
	currentCtx := server.getContext()
	if currentCtx.SessionID != "" {
		t.Fatalf("expected server context session id to be cleared, got %q", currentCtx.SessionID)
	}
	if currentCtx.ParticipantID != "" {
		t.Fatalf("expected server context participant id to be cleared, got %q", currentCtx.ParticipantID)
	}
}

// TestSetContextHandlerGetSetContextIntegration tests getContext/setContext integration.
func TestSetContextHandlerGetSetContextIntegration(t *testing.T) {
	campaignClient := &fakeCampaignClient{
		getCampaignResponse: &campaignv1.GetCampaignResponse{
			Campaign: &campaignv1.Campaign{Id: "camp-123"},
		},
		getParticipantResponse: &campaignv1.GetParticipantResponse{
			Participant: &campaignv1.Participant{
				Id:         "part-456",
				CampaignId: "camp-123",
			},
		},
	}
	sessionClient := &fakeSessionClient{
		getSessionResponse: &sessionv1.GetSessionResponse{
			Session: &sessionv1.Session{
				Id:         "sess-789",
				CampaignId: "camp-123",
			},
		},
	}
	server := &Server{}
	handler := domain.SetContextHandler(campaignClient, sessionClient, server.setContext, server.getContext)

	// Set context
	_, output1, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID:    "camp-123",
		SessionID:     "sess-789",
		ParticipantID: "part-456",
	})
	if err != nil {
		t.Fatalf("set context: %v", err)
	}

	// Verify output matches server context
	currentCtx := server.getContext()
	if output1.Context.CampaignID != currentCtx.CampaignID {
		t.Fatalf("output campaign id %q != server context %q", output1.Context.CampaignID, currentCtx.CampaignID)
	}
	if output1.Context.SessionID != currentCtx.SessionID {
		t.Fatalf("output session id %q != server context %q", output1.Context.SessionID, currentCtx.SessionID)
	}
	if output1.Context.ParticipantID != currentCtx.ParticipantID {
		t.Fatalf("output participant id %q != server context %q", output1.Context.ParticipantID, currentCtx.ParticipantID)
	}

	// Set context again with different values
	_, output2, err := handler(context.Background(), &mcp.CallToolRequest{}, domain.SetContextInput{
		CampaignID: "camp-123",
		// Only campaign, no session or participant
	})
	if err != nil {
		t.Fatalf("set context again: %v", err)
	}

	// Verify output matches updated server context
	currentCtx2 := server.getContext()
	if output2.Context.CampaignID != currentCtx2.CampaignID {
		t.Fatalf("output2 campaign id %q != server context %q", output2.Context.CampaignID, currentCtx2.CampaignID)
	}
	if output2.Context.SessionID != currentCtx2.SessionID {
		t.Fatalf("output2 session id %q != server context %q", output2.Context.SessionID, currentCtx2.SessionID)
	}
	if output2.Context.ParticipantID != currentCtx2.ParticipantID {
		t.Fatalf("output2 participant id %q != server context %q", output2.Context.ParticipantID, currentCtx2.ParticipantID)
	}
}

// TestContextResourceHandlerRejectsNilGetter ensures nil getContextFunc is rejected.
func TestContextResourceHandlerRejectsNilGetter(t *testing.T) {
	handler := domain.ContextResourceHandler(nil)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "context://current"},
	})
	if err == nil {
		t.Fatal("expected error for nil getContextFunc")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
}

// TestContextResourceHandlerRejectsInvalidURI ensures invalid URI is rejected.
func TestContextResourceHandlerRejectsInvalidURI(t *testing.T) {
	server := &Server{}
	handler := domain.ContextResourceHandler(server.getContext)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "context://invalid"},
	})
	if err == nil {
		t.Fatal("expected error for invalid URI")
	}
	if result != nil {
		t.Fatal("expected nil result on error")
	}
	if !strings.Contains(err.Error(), "context://current") {
		t.Fatalf("expected error to mention context://current, got %q", err.Error())
	}
}

// TestContextResourceHandlerReturnsEmptyContext ensures empty context returns all null fields.
func TestContextResourceHandlerReturnsEmptyContext(t *testing.T) {
	server := &Server{}
	handler := domain.ContextResourceHandler(server.getContext)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "context://current"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || len(result.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %v", result)
	}

	var payload domain.ContextResourcePayload
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal JSON: %v", err)
	}

	if payload.Context.CampaignID != nil {
		t.Fatalf("expected null campaign_id, got %v", payload.Context.CampaignID)
	}
	if payload.Context.SessionID != nil {
		t.Fatalf("expected null session_id, got %v", payload.Context.SessionID)
	}
	if payload.Context.ParticipantID != nil {
		t.Fatalf("expected null participant_id, got %v", payload.Context.ParticipantID)
	}
}

// TestContextResourceHandlerReturnsAllFields ensures all fields are returned when set.
func TestContextResourceHandlerReturnsAllFields(t *testing.T) {
	server := &Server{}
	server.setContext(domain.Context{
		CampaignID:    "camp-123",
		SessionID:     "sess-456",
		ParticipantID: "part-789",
	})
	handler := domain.ContextResourceHandler(server.getContext)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "context://current"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || len(result.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %v", result)
	}

	var payload domain.ContextResourcePayload
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal JSON: %v", err)
	}

	if payload.Context.CampaignID == nil || *payload.Context.CampaignID != "camp-123" {
		t.Fatalf("expected campaign_id camp-123, got %v", payload.Context.CampaignID)
	}
	if payload.Context.SessionID == nil || *payload.Context.SessionID != "sess-456" {
		t.Fatalf("expected session_id sess-456, got %v", payload.Context.SessionID)
	}
	if payload.Context.ParticipantID == nil || *payload.Context.ParticipantID != "part-789" {
		t.Fatalf("expected participant_id part-789, got %v", payload.Context.ParticipantID)
	}
}

// TestContextResourceHandlerReturnsPartialFields ensures partial fields return null for unset values.
func TestContextResourceHandlerReturnsPartialFields(t *testing.T) {
	server := &Server{}
	server.setContext(domain.Context{
		CampaignID: "camp-123",
		// SessionID and ParticipantID are empty
	})
	handler := domain.ContextResourceHandler(server.getContext)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "context://current"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || len(result.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %v", result)
	}

	var payload domain.ContextResourcePayload
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal JSON: %v", err)
	}

	if payload.Context.CampaignID == nil || *payload.Context.CampaignID != "camp-123" {
		t.Fatalf("expected campaign_id camp-123, got %v", payload.Context.CampaignID)
	}
	if payload.Context.SessionID != nil {
		t.Fatalf("expected null session_id, got %v", payload.Context.SessionID)
	}
	if payload.Context.ParticipantID != nil {
		t.Fatalf("expected null participant_id, got %v", payload.Context.ParticipantID)
	}
}

// TestContextResourceHandlerUsesDefaultURI ensures default URI is used when not provided.
func TestContextResourceHandlerUsesDefaultURI(t *testing.T) {
	server := &Server{}
	handler := domain.ContextResourceHandler(server.getContext)

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result == nil || len(result.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %v", result)
	}
	if result.Contents[0].URI != "context://current" {
		t.Fatalf("expected URI context://current, got %q", result.Contents[0].URI)
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
