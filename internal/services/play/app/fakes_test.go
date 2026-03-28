package app

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	gogrpc "google.golang.org/grpc"
	gogrpcmetadata "google.golang.org/grpc/metadata"
)

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

type fakePlayAuthClient struct {
	sessions         map[string]string
	createdSessionID string
	createdUserID    string
}

func (f *fakePlayAuthClient) CreateWebSession(_ context.Context, req *authv1.CreateWebSessionRequest, _ ...gogrpc.CallOption) (*authv1.CreateWebSessionResponse, error) {
	if f.sessions == nil {
		f.sessions = map[string]string{}
	}
	sessionID := strings.TrimSpace(f.createdSessionID)
	if sessionID == "" {
		sessionID = "ps-1"
	}
	f.createdUserID = strings.TrimSpace(req.GetUserId())
	f.sessions[sessionID] = f.createdUserID
	return &authv1.CreateWebSessionResponse{
		Session: &authv1.WebSession{Id: sessionID, UserId: f.createdUserID},
	}, nil
}

func (f *fakePlayAuthClient) GetWebSession(_ context.Context, req *authv1.GetWebSessionRequest, _ ...gogrpc.CallOption) (*authv1.GetWebSessionResponse, error) {
	sessionID := strings.TrimSpace(req.GetSessionId())
	userID := strings.TrimSpace(f.sessions[sessionID])
	if userID == "" {
		return nil, context.Canceled
	}
	return &authv1.GetWebSessionResponse{
		Session: &authv1.WebSession{Id: sessionID, UserId: userID},
	}, nil
}

// ---------------------------------------------------------------------------
// Interaction — stub (returns fixed/nil responses, no recording)
// ---------------------------------------------------------------------------

type stubInteractionClient struct {
	response *gamev1.GetInteractionStateResponse
	err      error
}

func (f stubInteractionClient) GetInteractionState(context.Context, *gamev1.GetInteractionStateRequest, ...gogrpc.CallOption) (*gamev1.GetInteractionStateResponse, error) {
	return f.response, f.err
}

func (f stubInteractionClient) ActivateScene(context.Context, *gamev1.ActivateSceneRequest, ...gogrpc.CallOption) (*gamev1.ActivateSceneResponse, error) {
	return nil, nil
}
func (f stubInteractionClient) OpenScenePlayerPhase(context.Context, *gamev1.OpenScenePlayerPhaseRequest, ...gogrpc.CallOption) (*gamev1.OpenScenePlayerPhaseResponse, error) {
	return nil, nil
}
func (f stubInteractionClient) SubmitScenePlayerAction(context.Context, *gamev1.SubmitScenePlayerActionRequest, ...gogrpc.CallOption) (*gamev1.SubmitScenePlayerActionResponse, error) {
	return nil, nil
}
func (f stubInteractionClient) YieldScenePlayerPhase(context.Context, *gamev1.YieldScenePlayerPhaseRequest, ...gogrpc.CallOption) (*gamev1.YieldScenePlayerPhaseResponse, error) {
	return nil, nil
}
func (f stubInteractionClient) WithdrawScenePlayerYield(context.Context, *gamev1.WithdrawScenePlayerYieldRequest, ...gogrpc.CallOption) (*gamev1.WithdrawScenePlayerYieldResponse, error) {
	return nil, nil
}
func (f stubInteractionClient) InterruptScenePlayerPhase(context.Context, *gamev1.InterruptScenePlayerPhaseRequest, ...gogrpc.CallOption) (*gamev1.InterruptScenePlayerPhaseResponse, error) {
	return nil, nil
}
func (f stubInteractionClient) RecordSceneGMInteraction(context.Context, *gamev1.RecordSceneGMInteractionRequest, ...gogrpc.CallOption) (*gamev1.RecordSceneGMInteractionResponse, error) {
	return nil, nil
}
func (f stubInteractionClient) ResolveScenePlayerReview(context.Context, *gamev1.ResolveScenePlayerReviewRequest, ...gogrpc.CallOption) (*gamev1.ResolveScenePlayerReviewResponse, error) {
	return nil, nil
}
func (f stubInteractionClient) OpenSessionOOC(context.Context, *gamev1.OpenSessionOOCRequest, ...gogrpc.CallOption) (*gamev1.OpenSessionOOCResponse, error) {
	return nil, nil
}
func (f stubInteractionClient) PostSessionOOC(context.Context, *gamev1.PostSessionOOCRequest, ...gogrpc.CallOption) (*gamev1.PostSessionOOCResponse, error) {
	return nil, nil
}
func (f stubInteractionClient) MarkOOCReadyToResume(context.Context, *gamev1.MarkOOCReadyToResumeRequest, ...gogrpc.CallOption) (*gamev1.MarkOOCReadyToResumeResponse, error) {
	return nil, nil
}
func (f stubInteractionClient) ClearOOCReadyToResume(context.Context, *gamev1.ClearOOCReadyToResumeRequest, ...gogrpc.CallOption) (*gamev1.ClearOOCReadyToResumeResponse, error) {
	return nil, nil
}
func (f stubInteractionClient) ResolveSessionOOC(context.Context, *gamev1.ResolveSessionOOCRequest, ...gogrpc.CallOption) (*gamev1.ResolveSessionOOCResponse, error) {
	return nil, nil
}
func (f stubInteractionClient) SetSessionGMAuthority(context.Context, *gamev1.SetSessionGMAuthorityRequest, ...gogrpc.CallOption) (*gamev1.SetSessionGMAuthorityResponse, error) {
	return nil, nil
}
func (f stubInteractionClient) RetryAIGMTurn(context.Context, *gamev1.RetryAIGMTurnRequest, ...gogrpc.CallOption) (*gamev1.RetryAIGMTurnResponse, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Interaction — recording (tracks calls)
// ---------------------------------------------------------------------------

type recordingInteractionClient struct {
	state          *gamev1.InteractionState
	mutationErr    error
	lastMethod     string
	lastCampaignID string
}

func newRecordingInteractionClient(state *gamev1.InteractionState) *recordingInteractionClient {
	return &recordingInteractionClient{state: state}
}

func (f *recordingInteractionClient) record(method string, campaignID string) {
	f.lastMethod = method
	f.lastCampaignID = strings.TrimSpace(campaignID)
}

func (f *recordingInteractionClient) GetInteractionState(context.Context, *gamev1.GetInteractionStateRequest, ...gogrpc.CallOption) (*gamev1.GetInteractionStateResponse, error) {
	return &gamev1.GetInteractionStateResponse{State: f.state}, nil
}

func (f *recordingInteractionClient) ActivateScene(_ context.Context, req *gamev1.ActivateSceneRequest, _ ...gogrpc.CallOption) (*gamev1.ActivateSceneResponse, error) {
	f.record("ActivateScene", req.GetCampaignId())
	return &gamev1.ActivateSceneResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) OpenScenePlayerPhase(_ context.Context, req *gamev1.OpenScenePlayerPhaseRequest, _ ...gogrpc.CallOption) (*gamev1.OpenScenePlayerPhaseResponse, error) {
	f.record("OpenScenePlayerPhase", req.GetCampaignId())
	return &gamev1.OpenScenePlayerPhaseResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) SubmitScenePlayerAction(_ context.Context, req *gamev1.SubmitScenePlayerActionRequest, _ ...gogrpc.CallOption) (*gamev1.SubmitScenePlayerActionResponse, error) {
	f.record("SubmitScenePlayerAction", req.GetCampaignId())
	return &gamev1.SubmitScenePlayerActionResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) YieldScenePlayerPhase(_ context.Context, req *gamev1.YieldScenePlayerPhaseRequest, _ ...gogrpc.CallOption) (*gamev1.YieldScenePlayerPhaseResponse, error) {
	f.record("YieldScenePlayerPhase", req.GetCampaignId())
	return &gamev1.YieldScenePlayerPhaseResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) WithdrawScenePlayerYield(_ context.Context, req *gamev1.WithdrawScenePlayerYieldRequest, _ ...gogrpc.CallOption) (*gamev1.WithdrawScenePlayerYieldResponse, error) {
	f.record("WithdrawScenePlayerYield", req.GetCampaignId())
	return &gamev1.WithdrawScenePlayerYieldResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) InterruptScenePlayerPhase(_ context.Context, req *gamev1.InterruptScenePlayerPhaseRequest, _ ...gogrpc.CallOption) (*gamev1.InterruptScenePlayerPhaseResponse, error) {
	f.record("InterruptScenePlayerPhase", req.GetCampaignId())
	return &gamev1.InterruptScenePlayerPhaseResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) RecordSceneGMInteraction(_ context.Context, req *gamev1.RecordSceneGMInteractionRequest, _ ...gogrpc.CallOption) (*gamev1.RecordSceneGMInteractionResponse, error) {
	f.record("RecordSceneGMInteraction", req.GetCampaignId())
	return &gamev1.RecordSceneGMInteractionResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) ResolveScenePlayerReview(_ context.Context, req *gamev1.ResolveScenePlayerReviewRequest, _ ...gogrpc.CallOption) (*gamev1.ResolveScenePlayerReviewResponse, error) {
	f.record("ResolveScenePlayerReview", req.GetCampaignId())
	return &gamev1.ResolveScenePlayerReviewResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) OpenSessionOOC(_ context.Context, req *gamev1.OpenSessionOOCRequest, _ ...gogrpc.CallOption) (*gamev1.OpenSessionOOCResponse, error) {
	f.record("OpenSessionOOC", req.GetCampaignId())
	return &gamev1.OpenSessionOOCResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) PostSessionOOC(_ context.Context, req *gamev1.PostSessionOOCRequest, _ ...gogrpc.CallOption) (*gamev1.PostSessionOOCResponse, error) {
	f.record("PostSessionOOC", req.GetCampaignId())
	return &gamev1.PostSessionOOCResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) MarkOOCReadyToResume(_ context.Context, req *gamev1.MarkOOCReadyToResumeRequest, _ ...gogrpc.CallOption) (*gamev1.MarkOOCReadyToResumeResponse, error) {
	f.record("MarkOOCReadyToResume", req.GetCampaignId())
	return &gamev1.MarkOOCReadyToResumeResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) ClearOOCReadyToResume(_ context.Context, req *gamev1.ClearOOCReadyToResumeRequest, _ ...gogrpc.CallOption) (*gamev1.ClearOOCReadyToResumeResponse, error) {
	f.record("ClearOOCReadyToResume", req.GetCampaignId())
	return &gamev1.ClearOOCReadyToResumeResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) ResolveSessionOOC(_ context.Context, req *gamev1.ResolveSessionOOCRequest, _ ...gogrpc.CallOption) (*gamev1.ResolveSessionOOCResponse, error) {
	f.record("ResolveSessionOOC", req.GetCampaignId())
	return &gamev1.ResolveSessionOOCResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) SetSessionGMAuthority(_ context.Context, req *gamev1.SetSessionGMAuthorityRequest, _ ...gogrpc.CallOption) (*gamev1.SetSessionGMAuthorityResponse, error) {
	f.record("SetSessionGMAuthority", req.GetCampaignId())
	return &gamev1.SetSessionGMAuthorityResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) RetryAIGMTurn(_ context.Context, req *gamev1.RetryAIGMTurnRequest, _ ...gogrpc.CallOption) (*gamev1.RetryAIGMTurnResponse, error) {
	f.record("RetryAIGMTurn", req.GetCampaignId())
	return &gamev1.RetryAIGMTurnResponse{State: f.state}, f.mutationErr
}

// ---------------------------------------------------------------------------
// Campaign
// ---------------------------------------------------------------------------

type fakePlayCampaignClient struct {
	response *gamev1.GetCampaignResponse
	err      error
}

func (f fakePlayCampaignClient) GetCampaign(context.Context, *gamev1.GetCampaignRequest, ...gogrpc.CallOption) (*gamev1.GetCampaignResponse, error) {
	return f.response, f.err
}

// ---------------------------------------------------------------------------
// System
// ---------------------------------------------------------------------------

type fakePlaySystemClient struct {
	response *gamev1.GetGameSystemResponse
	err      error
}

func (f fakePlaySystemClient) GetGameSystem(context.Context, *gamev1.GetGameSystemRequest, ...gogrpc.CallOption) (*gamev1.GetGameSystemResponse, error) {
	return f.response, f.err
}

// ---------------------------------------------------------------------------
// Participant
// ---------------------------------------------------------------------------

type fakePlayParticipantClient struct {
	response *gamev1.ListParticipantsResponse
	err      error
}

func (f fakePlayParticipantClient) ListParticipants(context.Context, *gamev1.ListParticipantsRequest, ...gogrpc.CallOption) (*gamev1.ListParticipantsResponse, error) {
	return f.response, f.err
}

type authSensitivePlayParticipantClient struct {
	response   *gamev1.ListParticipantsResponse
	lastUserID string
}

func (f *authSensitivePlayParticipantClient) ListParticipants(ctx context.Context, _ *gamev1.ListParticipantsRequest, _ ...gogrpc.CallOption) (*gamev1.ListParticipantsResponse, error) {
	userID := grpcauthctx.UserIDFromOutgoingContext(ctx)
	if userID == "" {
		return nil, errors.New("missing user metadata")
	}
	f.lastUserID = userID
	return f.response, nil
}

// ---------------------------------------------------------------------------
// Character
// ---------------------------------------------------------------------------

type fakePlayCharacterClient struct {
	listResponse  *gamev1.ListCharactersResponse
	listErr       error
	sheetResponse *gamev1.GetCharacterSheetResponse
	sheetErr      error
}

func (f fakePlayCharacterClient) ListCharacters(context.Context, *gamev1.ListCharactersRequest, ...gogrpc.CallOption) (*gamev1.ListCharactersResponse, error) {
	return f.listResponse, f.listErr
}

func (f fakePlayCharacterClient) GetCharacterSheet(context.Context, *gamev1.GetCharacterSheetRequest, ...gogrpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
	return f.sheetResponse, f.sheetErr
}

type authSensitivePlayCharacterClient struct {
	listResponse  *gamev1.ListCharactersResponse
	sheetResponse *gamev1.GetCharacterSheetResponse
	lastUserID    string
}

func (f *authSensitivePlayCharacterClient) ListCharacters(ctx context.Context, _ *gamev1.ListCharactersRequest, _ ...gogrpc.CallOption) (*gamev1.ListCharactersResponse, error) {
	userID := grpcauthctx.UserIDFromOutgoingContext(ctx)
	if userID == "" {
		return nil, errors.New("missing user metadata")
	}
	f.lastUserID = userID
	return f.listResponse, nil
}

func (f *authSensitivePlayCharacterClient) GetCharacterSheet(ctx context.Context, _ *gamev1.GetCharacterSheetRequest, _ ...gogrpc.CallOption) (*gamev1.GetCharacterSheetResponse, error) {
	userID := grpcauthctx.UserIDFromOutgoingContext(ctx)
	if userID == "" {
		return nil, errors.New("missing user metadata")
	}
	f.lastUserID = userID
	return f.sheetResponse, nil
}

// ---------------------------------------------------------------------------
// Transcript — stub (returns zero values)
// ---------------------------------------------------------------------------

type stubTranscriptStore struct{}

func (f *stubTranscriptStore) LatestSequence(context.Context, transcript.Scope) (int64, error) {
	return 0, nil
}

func (f *stubTranscriptStore) AppendMessage(context.Context, transcript.AppendRequest) (transcript.AppendResult, error) {
	return transcript.AppendResult{}, nil
}

func (f *stubTranscriptStore) HistoryAfter(context.Context, transcript.HistoryAfterQuery) ([]transcript.Message, error) {
	return nil, nil
}

func (f *stubTranscriptStore) HistoryBefore(context.Context, transcript.HistoryBeforeQuery) ([]transcript.Message, error) {
	return nil, nil
}

func (f *stubTranscriptStore) Close() error {
	return nil
}

// ---------------------------------------------------------------------------
// Transcript — scripted (configurable returns)
// ---------------------------------------------------------------------------

type scriptTranscriptStore struct {
	latest        int64
	latestErr     error
	before        []transcript.Message
	beforeErr     error
	after         []transcript.Message
	afterErr      error
	appendMessage transcript.Message
	appendErr     error
	beforeArgs    struct {
		scope  transcript.Scope
		before int64
		limit  int
	}
	appendArgs struct {
		request transcript.AppendRequest
	}
}

func (s *scriptTranscriptStore) LatestSequence(context.Context, transcript.Scope) (int64, error) {
	return s.latest, s.latestErr
}

func (s *scriptTranscriptStore) AppendMessage(_ context.Context, req transcript.AppendRequest) (transcript.AppendResult, error) {
	s.appendArgs.request = req
	return transcript.AppendResult{Message: s.appendMessage}, s.appendErr
}

func (s *scriptTranscriptStore) HistoryAfter(context.Context, transcript.HistoryAfterQuery) ([]transcript.Message, error) {
	return s.after, s.afterErr
}

func (s *scriptTranscriptStore) HistoryBefore(_ context.Context, query transcript.HistoryBeforeQuery) ([]transcript.Message, error) {
	s.beforeArgs.scope = query.Scope
	s.beforeArgs.before = query.BeforeSequenceID
	s.beforeArgs.limit = query.Limit
	return s.before, s.beforeErr
}

func (s *scriptTranscriptStore) Close() error {
	return nil
}

// ---------------------------------------------------------------------------
// AI Debug
// ---------------------------------------------------------------------------

type fakePlayAIDebugClient struct {
	mu sync.Mutex

	listResp *aiv1.ListCampaignDebugTurnsResponse
	listErr  error
	listReq  *aiv1.ListCampaignDebugTurnsRequest

	getResp *aiv1.GetCampaignDebugTurnResponse
	getErr  error
	getReq  *aiv1.GetCampaignDebugTurnRequest

	subscribeStream gogrpc.ServerStreamingClient[aiv1.CampaignDebugTurnUpdate]
	subscribeErr    error
	subscribeReq    *aiv1.SubscribeCampaignDebugUpdatesRequest
	subscribeUserID string
	subscribeCh     chan struct{}
}

func (f *fakePlayAIDebugClient) ListCampaignDebugTurns(_ context.Context, req *aiv1.ListCampaignDebugTurnsRequest, _ ...gogrpc.CallOption) (*aiv1.ListCampaignDebugTurnsResponse, error) {
	f.mu.Lock()
	f.listReq = req
	resp := f.listResp
	err := f.listErr
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if resp != nil {
		return resp, nil
	}
	return &aiv1.ListCampaignDebugTurnsResponse{}, nil
}

func (f *fakePlayAIDebugClient) GetCampaignDebugTurn(_ context.Context, req *aiv1.GetCampaignDebugTurnRequest, _ ...gogrpc.CallOption) (*aiv1.GetCampaignDebugTurnResponse, error) {
	f.mu.Lock()
	f.getReq = req
	resp := f.getResp
	err := f.getErr
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}
	if resp != nil {
		return resp, nil
	}
	return &aiv1.GetCampaignDebugTurnResponse{}, nil
}

func (f *fakePlayAIDebugClient) SubscribeCampaignDebugUpdates(ctx context.Context, req *aiv1.SubscribeCampaignDebugUpdatesRequest, _ ...gogrpc.CallOption) (gogrpc.ServerStreamingClient[aiv1.CampaignDebugTurnUpdate], error) {
	f.mu.Lock()
	f.subscribeUserID = grpcauthctx.UserIDFromOutgoingContext(ctx)
	if req != nil {
		cloned := *req
		f.subscribeReq = &cloned
	}
	subscribeCh := f.subscribeCh
	stream := f.subscribeStream
	err := f.subscribeErr
	f.mu.Unlock()
	if subscribeCh != nil {
		select {
		case subscribeCh <- struct{}{}:
		default:
		}
	}
	if streamState, ok := stream.(*fakeCampaignDebugUpdateStream); ok && streamState.ctx == nil {
		streamState.ctx = ctx
	}
	if err != nil {
		return nil, err
	}
	if stream != nil {
		return stream, nil
	}
	return &fakeCampaignDebugUpdateStream{ctx: ctx}, nil
}

func (f *fakePlayAIDebugClient) awaitSubscribe(t *testing.T) {
	t.Helper()
	if f.subscribeCh == nil {
		t.Fatal("subscribeCh is nil")
	}
	select {
	case <-f.subscribeCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SubscribeCampaignDebugUpdates")
	}
}

// ---------------------------------------------------------------------------
// Events
// ---------------------------------------------------------------------------

type fakeEventClient struct {
	mu          sync.Mutex
	stream      gogrpc.ServerStreamingClient[gamev1.CampaignUpdate]
	err         error
	lastUserID  string
	lastRequest *gamev1.SubscribeCampaignUpdatesRequest
	subscribeCh chan struct{}
}

func (f *fakeEventClient) SubscribeCampaignUpdates(ctx context.Context, req *gamev1.SubscribeCampaignUpdatesRequest, _ ...gogrpc.CallOption) (gogrpc.ServerStreamingClient[gamev1.CampaignUpdate], error) {
	f.mu.Lock()
	f.lastUserID = grpcauthctx.UserIDFromOutgoingContext(ctx)
	if req != nil {
		cloned := *req
		cloned.Kinds = append([]gamev1.CampaignUpdateKind(nil), req.GetKinds()...)
		cloned.ProjectionScopes = append([]string(nil), req.GetProjectionScopes()...)
		f.lastRequest = &cloned
	}
	subscribeCh := f.subscribeCh
	stream := f.stream
	err := f.err
	f.mu.Unlock()
	if subscribeCh != nil {
		select {
		case subscribeCh <- struct{}{}:
		default:
		}
	}
	if streamState, ok := stream.(*fakeCampaignUpdateStream); ok && streamState.ctx == nil {
		streamState.ctx = ctx
	}
	if err != nil {
		return nil, err
	}
	if stream != nil {
		return stream, nil
	}
	return &fakeCampaignUpdateStream{ctx: ctx}, nil
}

func (f *fakeEventClient) awaitSubscribe(t *testing.T) {
	t.Helper()
	if f.subscribeCh == nil {
		t.Fatal("subscribeCh is nil")
	}
	select {
	case <-f.subscribeCh:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for SubscribeCampaignUpdates")
	}
}

// ---------------------------------------------------------------------------
// Campaign update stream
// ---------------------------------------------------------------------------

type fakeCampaignUpdateStream struct {
	ctx     context.Context
	updates chan *gamev1.CampaignUpdate
	recvErr error
}

func (f *fakeCampaignUpdateStream) Recv() (*gamev1.CampaignUpdate, error) {
	if f.recvErr != nil {
		return nil, f.recvErr
	}
	if f.updates != nil {
		select {
		case update, ok := <-f.updates:
			if !ok {
				return nil, io.EOF
			}
			return update, nil
		case <-f.Context().Done():
			return nil, f.Context().Err()
		}
	}
	<-f.Context().Done()
	return nil, f.Context().Err()
}
func (f *fakeCampaignUpdateStream) Header() (gogrpcmetadata.MD, error) { return nil, nil }
func (f *fakeCampaignUpdateStream) Trailer() gogrpcmetadata.MD         { return nil }
func (f *fakeCampaignUpdateStream) CloseSend() error                   { return nil }
func (f *fakeCampaignUpdateStream) Context() context.Context {
	if f.ctx == nil {
		return context.Background()
	}
	return f.ctx
}
func (f *fakeCampaignUpdateStream) SendMsg(any) error { return nil }
func (f *fakeCampaignUpdateStream) RecvMsg(any) error { return nil }

// ---------------------------------------------------------------------------
// Campaign debug update stream
// ---------------------------------------------------------------------------

type fakeCampaignDebugUpdateStream struct {
	ctx     context.Context
	updates chan *aiv1.CampaignDebugTurnUpdate
	recvErr error
}

func (f *fakeCampaignDebugUpdateStream) Recv() (*aiv1.CampaignDebugTurnUpdate, error) {
	if f.recvErr != nil {
		return nil, f.recvErr
	}
	if f.updates != nil {
		select {
		case update, ok := <-f.updates:
			if !ok {
				return nil, io.EOF
			}
			return update, nil
		case <-f.Context().Done():
			return nil, f.Context().Err()
		}
	}
	<-f.Context().Done()
	return nil, f.Context().Err()
}
func (f *fakeCampaignDebugUpdateStream) Header() (gogrpcmetadata.MD, error) { return nil, nil }
func (f *fakeCampaignDebugUpdateStream) Trailer() gogrpcmetadata.MD         { return nil }
func (f *fakeCampaignDebugUpdateStream) CloseSend() error                   { return nil }
func (f *fakeCampaignDebugUpdateStream) Context() context.Context {
	if f.ctx == nil {
		return context.Background()
	}
	return f.ctx
}
func (f *fakeCampaignDebugUpdateStream) SendMsg(any) error { return nil }
func (f *fakeCampaignDebugUpdateStream) RecvMsg(any) error { return nil }

// ---------------------------------------------------------------------------
// Daggerheart content
// ---------------------------------------------------------------------------

type fakeDaggerheartContentClient struct {
	mu        sync.Mutex
	responses map[string]*daggerheartv1.DaggerheartDomainCard
	errByID   map[string]error
	requests  []*daggerheartv1.GetDaggerheartDomainCardRequest
}

func (f *fakeDaggerheartContentClient) GetDomainCard(_ context.Context, req *daggerheartv1.GetDaggerheartDomainCardRequest, _ ...gogrpc.CallOption) (*daggerheartv1.GetDaggerheartDomainCardResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if req != nil {
		copied := *req
		f.requests = append(f.requests, &copied)
	}
	cardID := strings.TrimSpace(req.GetId())
	if err := f.errByID[cardID]; err != nil {
		return nil, err
	}
	return &daggerheartv1.GetDaggerheartDomainCardResponse{
		DomainCard: f.responses[cardID],
	}, nil
}
