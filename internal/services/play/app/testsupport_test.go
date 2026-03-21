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
	"testing"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	gogrpc "google.golang.org/grpc"
	gogrpcmetadata "google.golang.org/grpc/metadata"
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

func (f *recordingInteractionClient) SetActiveScene(_ context.Context, req *gamev1.SetActiveSceneRequest, _ ...gogrpc.CallOption) (*gamev1.SetActiveSceneResponse, error) {
	f.record("SetActiveScene", req.GetCampaignId())
	return &gamev1.SetActiveSceneResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) StartScenePlayerPhase(_ context.Context, req *gamev1.StartScenePlayerPhaseRequest, _ ...gogrpc.CallOption) (*gamev1.StartScenePlayerPhaseResponse, error) {
	f.record("StartScenePlayerPhase", req.GetCampaignId())
	return &gamev1.StartScenePlayerPhaseResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) SubmitScenePlayerPost(_ context.Context, req *gamev1.SubmitScenePlayerPostRequest, _ ...gogrpc.CallOption) (*gamev1.SubmitScenePlayerPostResponse, error) {
	f.record("SubmitScenePlayerPost", req.GetCampaignId())
	return &gamev1.SubmitScenePlayerPostResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) YieldScenePlayerPhase(_ context.Context, req *gamev1.YieldScenePlayerPhaseRequest, _ ...gogrpc.CallOption) (*gamev1.YieldScenePlayerPhaseResponse, error) {
	f.record("YieldScenePlayerPhase", req.GetCampaignId())
	return &gamev1.YieldScenePlayerPhaseResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) UnyieldScenePlayerPhase(_ context.Context, req *gamev1.UnyieldScenePlayerPhaseRequest, _ ...gogrpc.CallOption) (*gamev1.UnyieldScenePlayerPhaseResponse, error) {
	f.record("UnyieldScenePlayerPhase", req.GetCampaignId())
	return &gamev1.UnyieldScenePlayerPhaseResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) EndScenePlayerPhase(_ context.Context, req *gamev1.EndScenePlayerPhaseRequest, _ ...gogrpc.CallOption) (*gamev1.EndScenePlayerPhaseResponse, error) {
	f.record("EndScenePlayerPhase", req.GetCampaignId())
	return &gamev1.EndScenePlayerPhaseResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) CommitSceneGMOutput(_ context.Context, req *gamev1.CommitSceneGMOutputRequest, _ ...gogrpc.CallOption) (*gamev1.CommitSceneGMOutputResponse, error) {
	f.record("CommitSceneGMOutput", req.GetCampaignId())
	return &gamev1.CommitSceneGMOutputResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) AcceptScenePlayerPhase(_ context.Context, req *gamev1.AcceptScenePlayerPhaseRequest, _ ...gogrpc.CallOption) (*gamev1.AcceptScenePlayerPhaseResponse, error) {
	f.record("AcceptScenePlayerPhase", req.GetCampaignId())
	return &gamev1.AcceptScenePlayerPhaseResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) RequestScenePlayerRevisions(_ context.Context, req *gamev1.RequestScenePlayerRevisionsRequest, _ ...gogrpc.CallOption) (*gamev1.RequestScenePlayerRevisionsResponse, error) {
	f.record("RequestScenePlayerRevisions", req.GetCampaignId())
	return &gamev1.RequestScenePlayerRevisionsResponse{State: f.state}, f.mutationErr
}

func (f *recordingInteractionClient) PauseSessionForOOC(_ context.Context, req *gamev1.PauseSessionForOOCRequest, _ ...gogrpc.CallOption) (*gamev1.PauseSessionForOOCResponse, error) {
	f.record("PauseSessionForOOC", req.GetCampaignId())
	return &gamev1.PauseSessionForOOCResponse{State: f.state}, f.mutationErr
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

func (f *recordingInteractionClient) ResumeFromOOC(_ context.Context, req *gamev1.ResumeFromOOCRequest, _ ...gogrpc.CallOption) (*gamev1.ResumeFromOOCResponse, error) {
	f.record("ResumeFromOOC", req.GetCampaignId())
	return &gamev1.ResumeFromOOCResponse{State: f.state}, f.mutationErr
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
	stream gogrpc.ServerStreamingClient[gamev1.CampaignUpdate]
	err    error
}

func (f fakeEventClient) SubscribeCampaignUpdates(context.Context, *gamev1.SubscribeCampaignUpdatesRequest, ...gogrpc.CallOption) (gogrpc.ServerStreamingClient[gamev1.CampaignUpdate], error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.stream != nil {
		return f.stream, nil
	}
	return &fakeCampaignUpdateStream{}, nil
}

type fakeCampaignUpdateStream struct{}

func (f *fakeCampaignUpdateStream) Recv() (*gamev1.CampaignUpdate, error) { return nil, io.EOF }
func (f *fakeCampaignUpdateStream) Header() (gogrpcmetadata.MD, error)    { return nil, nil }
func (f *fakeCampaignUpdateStream) Trailer() gogrpcmetadata.MD            { return nil }
func (f *fakeCampaignUpdateStream) CloseSend() error                      { return nil }
func (f *fakeCampaignUpdateStream) Context() context.Context              { return context.Background() }
func (f *fakeCampaignUpdateStream) SendMsg(any) error                     { return nil }
func (f *fakeCampaignUpdateStream) RecvMsg(any) error                     { return nil }
