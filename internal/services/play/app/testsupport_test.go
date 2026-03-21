package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	gogrpc "google.golang.org/grpc"
	gogrpcmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func playTestState() *gamev1.InteractionState {
	return &gamev1.InteractionState{
		CampaignId:   "c1",
		CampaignName: "The Guildhouse",
		Viewer:       &gamev1.InteractionViewer{ParticipantId: "p1", Name: "Avery", Role: gamev1.ParticipantRole_PLAYER},
		ActiveSession: &gamev1.InteractionSession{
			SessionId: "s1",
			Name:      "Session One",
		},
	}
}

func hubDepsFromServer(s *Server) realtimeHubDeps {
	return realtimeHubDeps{
		resolveUserID: s.resolvePlayUserID,
		application:   s.application,
		transcripts:   s.transcripts,
		events:        s.events,
	}
}

func newAuthedPlayServer(interaction *recordingInteractionClient, transcripts *scriptTranscriptStore) *Server {
	server := &Server{
		auth:            &fakePlayAuthClient{sessions: map[string]string{"ps-1": "user-1"}},
		interaction:     interaction,
		campaign:        fakePlayCampaignClient{response: &gamev1.GetCampaignResponse{}},
		system:          fakePlaySystemClient{response: &gamev1.GetGameSystemResponse{}},
		participants:    fakePlayParticipantClient{response: &gamev1.ListParticipantsResponse{}},
		characters:      fakePlayCharacterClient{listResponse: &gamev1.ListCharactersResponse{}},
		transcripts:     transcripts,
		shellAssets:     shellAssets{devServerURL: "http://localhost:5173"},
		httpServer:      &http.Server{},
		webFallbackPort: "8080",
	}
	server.realtime = newRealtimeHub(server)
	return server
}

func assertJSONError(t *testing.T, rr *httptest.ResponseRecorder, wantStatus int, wantMessage string) {
	t.Helper()
	if rr.Code != wantStatus {
		t.Fatalf("status = %d, want %d body=%s", rr.Code, wantStatus, rr.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if got := strings.TrimSpace(payload["error"].(string)); got != wantMessage {
		t.Fatalf("error = %q, want %q", got, wantMessage)
	}
}

func drainWSFrames(t *testing.T, buffer *bytes.Buffer) []wsFrame {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(buffer.Bytes()))
	frames := []wsFrame{}
	for {
		var frame wsFrame
		if err := decoder.Decode(&frame); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatalf("decode ws frame: %v", err)
		}
		frames = append(frames, frame)
	}
	buffer.Reset()
	return frames
}

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

func enrichedParticipantResponse() *gamev1.ListParticipantsResponse {
	return &gamev1.ListParticipantsResponse{
		Participants: []*gamev1.Participant{
			{Id: "p1", Name: "Avery", Role: gamev1.ParticipantRole_PLAYER},
			{Id: "p2", Name: "Guide", Role: gamev1.ParticipantRole_GM},
		},
	}
}

func enrichedCharacterResponse() *gamev1.ListCharactersResponse {
	return &gamev1.ListCharactersResponse{
		Characters: []*gamev1.Character{
			{
				Id:            "char-1",
				CampaignId:    "c1",
				Name:          "Lark",
				Kind:          gamev1.CharacterKind_PC,
				ParticipantId: &wrapperspb.StringValue{Value: "p1"},
			},
		},
	}
}

func enrichedCharacterSheetResponse() *gamev1.GetCharacterSheetResponse {
	return &gamev1.GetCharacterSheetResponse{
		Character: enrichedCharacterResponse().GetCharacters()[0],
		Profile: &gamev1.CharacterProfile{
			CampaignId:  "c1",
			CharacterId: "char-1",
			SystemProfile: &gamev1.CharacterProfile_Daggerheart{
				Daggerheart: &daggerheartv1.DaggerheartProfile{
					Level: 1,
					HpMax: 10,
					Heritage: &daggerheartv1.DaggerheartHeritageSelection{
						AncestryName:  "Human",
						CommunityName: "Slyborne",
					},
					ActiveClassFeatures: []*daggerheartv1.DaggerheartActiveClassFeature{
						{
							Name:        "Rogue's Dodge",
							Description: "Spend 3 Hope to gain +2 Evasion until an attack succeeds against you.",
							HopeFeature: true,
						},
						{
							Name:        "Sneak Attack",
							Description: "When you have advantage on a melee attack, deal an extra 1d8 damage.",
						},
					},
					PrimaryWeapon: &daggerheartv1.DaggerheartSheetWeaponSummary{
						Name:       "Sword",
						Trait:      "Finesse",
						Range:      "melee",
						DamageDice: "1d8",
						DamageType: "physical",
						Feature:    "Versatile",
					},
					ActiveArmor: &daggerheartv1.DaggerheartSheetArmorSummary{
						Name:      "Leather",
						BaseScore: 2,
						Feature:   "Quiet",
					},
				},
			},
		},
	}
}
