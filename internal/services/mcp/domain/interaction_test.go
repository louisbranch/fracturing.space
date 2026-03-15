package domain

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type fakeInteractionReviewState struct {
	acceptResp           *statev1.AcceptScenePlayerPhaseResponse
	acceptErr            error
	requestRevisionsResp *statev1.RequestScenePlayerRevisionsResponse
	requestRevisionsErr  error
	lastAcceptRequest    *statev1.AcceptScenePlayerPhaseRequest
	lastRevisionsRequest *statev1.RequestScenePlayerRevisionsRequest
}

var fakeInteractionReviewStates sync.Map

func reviewStateForClient(client *fakeInteractionClient) *fakeInteractionReviewState {
	state, _ := fakeInteractionReviewStates.LoadOrStore(client, &fakeInteractionReviewState{})
	return state.(*fakeInteractionReviewState)
}

func (f *fakeInteractionClient) AcceptScenePlayerPhase(ctx context.Context, req *statev1.AcceptScenePlayerPhaseRequest, _ ...grpc.CallOption) (*statev1.AcceptScenePlayerPhaseResponse, error) {
	state := reviewStateForClient(f)
	state.lastAcceptRequest = req
	return state.acceptResp, state.acceptErr
}

func (f *fakeInteractionClient) RequestScenePlayerRevisions(ctx context.Context, req *statev1.RequestScenePlayerRevisionsRequest, _ ...grpc.CallOption) (*statev1.RequestScenePlayerRevisionsResponse, error) {
	state := reviewStateForClient(f)
	state.lastRevisionsRequest = req
	return state.requestRevisionsResp, state.requestRevisionsErr
}

func TestInteractionSetActiveSceneHandlerUsesContextAndNotifies(t *testing.T) {
	client := &fakeInteractionClient{
		setActiveSceneResp: &statev1.SetActiveSceneResponse{
			State: testInteractionState(),
		},
	}
	getContext := func() Context {
		return Context{CampaignID: "camp-1", ParticipantID: "part-1"}
	}
	var notified []string
	handler := InteractionSetActiveSceneHandler(client, getContext, func(_ context.Context, uri string) {
		notified = append(notified, uri)
	})

	result, output, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionSetActiveSceneInput{SceneID: "scene-9"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.Meta == nil {
		t.Fatal("expected metadata result")
	}
	if client.lastSetRequest == nil || client.lastSetRequest.GetCampaignId() != "camp-1" || client.lastSetRequest.GetSceneId() != "scene-9" {
		t.Fatalf("request = %#v", client.lastSetRequest)
	}
	if output.ActiveScene.SceneID != "scene-1" {
		t.Fatalf("active scene id = %q, want scene-1", output.ActiveScene.SceneID)
	}
	if len(notified) != 1 || notified[0] != "campaign://camp-1/interaction" {
		t.Fatalf("notified = %#v", notified)
	}
	md, ok := metadata.FromOutgoingContext(client.setActiveSceneCtx)
	if !ok {
		t.Fatal("expected outgoing metadata")
	}
	if got := md.Get(grpcmeta.ParticipantIDHeader); len(got) != 1 || got[0] != "part-1" {
		t.Fatalf("participant metadata = %v, want [part-1]", got)
	}
}

func TestInteractionYieldScenePlayerPhaseHandlerFallsBackToActiveScene(t *testing.T) {
	client := &fakeInteractionClient{
		getResp: &statev1.GetInteractionStateResponse{
			State: testInteractionState(),
		},
		yieldResp: &statev1.YieldScenePlayerPhaseResponse{
			State: testInteractionState(),
		},
	}
	getContext := func() Context {
		return Context{CampaignID: "camp-1", ParticipantID: "part-1"}
	}
	handler := InteractionYieldScenePlayerPhaseHandler(client, getContext, nil)

	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionScenePhaseInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.lastGetRequest == nil || client.lastGetRequest.GetCampaignId() != "camp-1" {
		t.Fatalf("get request = %#v", client.lastGetRequest)
	}
	if client.lastYieldRequest == nil || client.lastYieldRequest.GetSceneId() != "scene-1" {
		t.Fatalf("yield request = %#v, want active scene fallback", client.lastYieldRequest)
	}
}

func TestInteractionYieldScenePlayerPhaseHandlerErrorsWithoutActiveScene(t *testing.T) {
	client := &fakeInteractionClient{
		getResp: &statev1.GetInteractionStateResponse{
			State: &statev1.InteractionState{CampaignId: "camp-1"},
		},
	}
	handler := InteractionYieldScenePlayerPhaseHandler(client, func() Context { return Context{CampaignID: "camp-1"} }, nil)

	result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionScenePhaseInput{})
	if err == nil {
		t.Fatal("expected error")
	}
	if result != nil {
		t.Fatal("expected nil result")
	}
}

func TestInteractionStateResourceHandlerReturnsJSONState(t *testing.T) {
	client := &fakeInteractionClient{
		getResp: &statev1.GetInteractionStateResponse{
			State: testReviewedInteractionState(),
		},
	}
	handler := InteractionStateResourceHandler(client, func() Context {
		return Context{ParticipantID: "part-1"}
	})

	result, err := handler(context.Background(), &mcp.ReadResourceRequest{
		Params: &mcp.ReadResourceParams{URI: "campaign://camp-1/interaction"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Contents) != 1 {
		t.Fatalf("contents = %d, want 1", len(result.Contents))
	}

	var payload InteractionStateResult
	if err := json.Unmarshal([]byte(result.Contents[0].Text), &payload); err != nil {
		t.Fatalf("unmarshal interaction state: %v", err)
	}
	if payload.CampaignID != "camp-1" || payload.PlayerPhase.Status != "GM_REVIEW" {
		t.Fatalf("payload = %#v", payload)
	}
	if len(payload.PlayerPhase.Slots) != 2 {
		t.Fatalf("slots = %#v, want 2", payload.PlayerPhase.Slots)
	}
	if payload.PlayerPhase.Slots[0].ReviewStatus != "CHANGES_REQUESTED" || payload.PlayerPhase.Slots[0].ReviewReason != "Corin does not know Fireball." {
		t.Fatalf("slot review = %#v", payload.PlayerPhase.Slots[0])
	}
}

func TestInteractionMetadataBuilders(t *testing.T) {
	t.Parallel()

	resource := InteractionStateResourceTemplate()
	if resource.Name != "interaction_state" || resource.URITemplate != interactionStateResourceURITemplate {
		t.Fatalf("resource template = %#v", resource)
	}

	tests := []struct {
		name     string
		tool     *mcp.Tool
		wantName string
	}{
		{name: "set active scene", tool: InteractionSetActiveSceneTool(), wantName: "interaction_active_scene_set"},
		{name: "start phase", tool: InteractionStartScenePlayerPhaseTool(), wantName: "interaction_scene_player_phase_start"},
		{name: "submit post", tool: InteractionSubmitScenePlayerPostTool(), wantName: "interaction_scene_player_post_submit"},
		{name: "yield", tool: InteractionYieldScenePlayerPhaseTool(), wantName: "interaction_scene_player_phase_yield"},
		{name: "unyield", tool: InteractionUnyieldScenePlayerPhaseTool(), wantName: "interaction_scene_player_phase_unyield"},
		{name: "commit gm output", tool: InteractionCommitSceneGMOutputTool(), wantName: "interaction_scene_gm_output_commit"},
		{name: "accept phase", tool: InteractionAcceptScenePlayerPhaseTool(), wantName: "interaction_scene_player_phase_accept"},
		{name: "request revisions", tool: InteractionRequestScenePlayerRevisionsTool(), wantName: "interaction_scene_player_revisions_request"},
		{name: "end phase", tool: InteractionEndScenePlayerPhaseTool(), wantName: "interaction_scene_player_phase_end"},
		{name: "pause ooc", tool: InteractionPauseOOCTool(), wantName: "interaction_ooc_pause"},
		{name: "post ooc", tool: InteractionPostOOCTool(), wantName: "interaction_ooc_post"},
		{name: "mark ready", tool: InteractionMarkOOCReadyTool(), wantName: "interaction_ooc_ready_mark"},
		{name: "clear ready", tool: InteractionClearOOCReadyTool(), wantName: "interaction_ooc_ready_clear"},
		{name: "resume ooc", tool: InteractionResumeOOCTool(), wantName: "interaction_ooc_resume"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.tool.Name != tc.wantName || tc.tool.Description == "" {
				t.Fatalf("tool = %#v", tc.tool)
			}
		})
	}
}

func TestInteractionStartScenePlayerPhaseHandlerUsesExplicitScene(t *testing.T) {
	client := &fakeInteractionClient{
		startPhaseResp: &statev1.StartScenePlayerPhaseResponse{State: testInteractionState()},
	}
	handler := InteractionStartScenePlayerPhaseHandler(client, func() Context {
		return Context{CampaignID: "camp-1", ParticipantID: "gm-1"}
	}, nil)

	_, output, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionStartScenePlayerPhaseInput{
		SceneID:      "scene-9",
		FrameText:    "  What now?  ",
		CharacterIDs: []string{"char-1", "char-2"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.lastGetRequest != nil {
		t.Fatalf("unexpected active-scene lookup: %#v", client.lastGetRequest)
	}
	if client.lastStartRequest == nil || client.lastStartRequest.GetSceneId() != "scene-9" {
		t.Fatalf("start request = %#v", client.lastStartRequest)
	}
	if client.lastStartRequest.GetFrameText() != "What now?" {
		t.Fatalf("frame text = %q", client.lastStartRequest.GetFrameText())
	}
	if len(client.lastStartRequest.GetCharacterIds()) != 2 || output.ActiveScene.SceneID != "scene-1" {
		t.Fatalf("request/output mismatch: %#v %#v", client.lastStartRequest, output)
	}
}

func TestInteractionCommitSceneGMOutputHandlerUsesFallbackScene(t *testing.T) {
	client := &fakeInteractionClient{
		getResp:            &statev1.GetInteractionStateResponse{State: testInteractionState()},
		commitGMOutputResp: &statev1.CommitSceneGMOutputResponse{State: testInteractionState()},
	}
	handler := InteractionCommitSceneGMOutputHandler(client, func() Context {
		return Context{CampaignID: "camp-1", ParticipantID: "gm-ai"}
	}, nil)

	_, output, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionCommitSceneGMOutputInput{
		Text: "  The torches flicker.  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.lastGetRequest == nil || client.lastCommitRequest == nil {
		t.Fatalf("requests = %#v %#v", client.lastGetRequest, client.lastCommitRequest)
	}
	if client.lastCommitRequest.GetSceneId() != "scene-1" || client.lastCommitRequest.GetText() != "The torches flicker." {
		t.Fatalf("commit request = %#v", client.lastCommitRequest)
	}
	if output.ActiveScene.SceneID != "scene-1" {
		t.Fatalf("output = %#v", output)
	}
}

func TestInteractionSubmitScenePlayerPostHandlerUsesFallbackSceneAndYieldFlag(t *testing.T) {
	client := &fakeInteractionClient{
		getResp:        &statev1.GetInteractionStateResponse{State: testInteractionState()},
		submitPostResp: &statev1.SubmitScenePlayerPostResponse{State: testInteractionState()},
	}
	handler := InteractionSubmitScenePlayerPostHandler(client, func() Context {
		return Context{CampaignID: "camp-1", ParticipantID: "part-1"}
	}, nil)

	_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionSubmitScenePlayerPostInput{
		SummaryText:    "  I cross the bridge.  ",
		CharacterIDs:   []string{"char-1"},
		YieldAfterPost: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.lastGetRequest == nil || client.lastSubmitRequest == nil {
		t.Fatalf("requests = %#v %#v", client.lastGetRequest, client.lastSubmitRequest)
	}
	if client.lastSubmitRequest.GetSceneId() != "scene-1" || client.lastSubmitRequest.GetSummaryText() != "I cross the bridge." || !client.lastSubmitRequest.GetYieldAfterPost() {
		t.Fatalf("submit request = %#v", client.lastSubmitRequest)
	}
}

func TestInteractionUnyieldEndAndOOCHandlers(t *testing.T) {
	state := testInteractionState()
	tests := []struct {
		name   string
		run    func(*fakeInteractionClient) error
		verify func(*testing.T, *fakeInteractionClient)
	}{
		{
			name: "accept phase uses explicit scene",
			run: func(client *fakeInteractionClient) error {
				reviewStateForClient(client).acceptResp = &statev1.AcceptScenePlayerPhaseResponse{State: state}
				handler := InteractionAcceptScenePlayerPhaseHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "gm-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionAcceptScenePlayerPhaseInput{SceneID: "scene-9"})
				return err
			},
			verify: func(t *testing.T, client *fakeInteractionClient) {
				t.Helper()
				if reviewStateForClient(client).lastAcceptRequest == nil || reviewStateForClient(client).lastAcceptRequest.GetSceneId() != "scene-9" {
					t.Fatalf("accept request = %#v", reviewStateForClient(client).lastAcceptRequest)
				}
			},
		},
		{
			name: "request revisions trims reason and character ids",
			run: func(client *fakeInteractionClient) error {
				reviewStateForClient(client).requestRevisionsResp = &statev1.RequestScenePlayerRevisionsResponse{State: state}
				handler := InteractionRequestScenePlayerRevisionsHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "gm-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionRequestScenePlayerRevisionsInput{
					SceneID: "scene-9",
					Revisions: []InteractionScenePlayerRevisionInput{
						{
							ParticipantID: "part-1",
							Reason:        "  Corin does not know Fireball.  ",
							CharacterIDs:  []string{"char-1"},
						},
					},
				})
				return err
			},
			verify: func(t *testing.T, client *fakeInteractionClient) {
				t.Helper()
				request := reviewStateForClient(client).lastRevisionsRequest
				if request == nil || request.GetSceneId() != "scene-9" {
					t.Fatalf("revisions request = %#v", request)
				}
				if len(request.GetRevisions()) != 1 || request.GetRevisions()[0].GetReason() != "Corin does not know Fireball." {
					t.Fatalf("revisions payload = %#v", request.GetRevisions())
				}
			},
		},
		{
			name: "unyield uses explicit scene",
			run: func(client *fakeInteractionClient) error {
				client.unyieldResp = &statev1.UnyieldScenePlayerPhaseResponse{State: state}
				handler := InteractionUnyieldScenePlayerPhaseHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "part-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionScenePhaseInput{SceneID: "scene-9"})
				return err
			},
			verify: func(t *testing.T, client *fakeInteractionClient) {
				t.Helper()
				if client.lastUnyieldRequest == nil || client.lastUnyieldRequest.GetSceneId() != "scene-9" {
					t.Fatalf("unyield request = %#v", client.lastUnyieldRequest)
				}
			},
		},
		{
			name: "end phase trims reason",
			run: func(client *fakeInteractionClient) error {
				client.endPhaseResp = &statev1.EndScenePlayerPhaseResponse{State: state}
				handler := InteractionEndScenePlayerPhaseHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "gm-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionEndScenePlayerPhaseInput{CampaignID: "camp-2", SceneID: "scene-9", Reason: "  interrupted  "})
				return err
			},
			verify: func(t *testing.T, client *fakeInteractionClient) {
				t.Helper()
				if client.lastEndRequest == nil || client.lastEndRequest.GetCampaignId() != "camp-2" || client.lastEndRequest.GetReason() != "interrupted" {
					t.Fatalf("end request = %#v", client.lastEndRequest)
				}
			},
		},
		{
			name: "pause ooc sends reason",
			run: func(client *fakeInteractionClient) error {
				client.pauseOOCResp = &statev1.PauseSessionForOOCResponse{State: state}
				handler := InteractionPauseOOCHandler(client, func() Context { return Context{CampaignID: "camp-1"} }, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionPauseOOCInput{Reason: "  table talk  "})
				return err
			},
			verify: func(t *testing.T, client *fakeInteractionClient) {
				t.Helper()
				if client.lastPauseRequest == nil || client.lastPauseRequest.GetReason() != "table talk" {
					t.Fatalf("pause request = %#v", client.lastPauseRequest)
				}
			},
		},
		{
			name: "post ooc sends body",
			run: func(client *fakeInteractionClient) error {
				client.postOOCResp = &statev1.PostSessionOOCResponse{State: state}
				handler := InteractionPostOOCHandler(client, func() Context { return Context{CampaignID: "camp-1"} }, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionPostOOCInput{Body: "  rules question  "})
				return err
			},
			verify: func(t *testing.T, client *fakeInteractionClient) {
				t.Helper()
				if client.lastPostRequest == nil || client.lastPostRequest.GetBody() != "rules question" {
					t.Fatalf("post request = %#v", client.lastPostRequest)
				}
			},
		},
		{
			name: "mark ready",
			run: func(client *fakeInteractionClient) error {
				client.markReadyResp = &statev1.MarkOOCReadyToResumeResponse{State: state}
				handler := InteractionMarkOOCReadyHandler(client, func() Context { return Context{CampaignID: "camp-1"} }, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionPauseOOCInput{})
				return err
			},
			verify: func(t *testing.T, client *fakeInteractionClient) {
				t.Helper()
				if client.lastMarkRequest == nil || client.lastMarkRequest.GetCampaignId() != "camp-1" {
					t.Fatalf("mark request = %#v", client.lastMarkRequest)
				}
			},
		},
		{
			name: "clear ready",
			run: func(client *fakeInteractionClient) error {
				client.clearReadyResp = &statev1.ClearOOCReadyToResumeResponse{State: state}
				handler := InteractionClearOOCReadyHandler(client, func() Context { return Context{CampaignID: "camp-1"} }, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionPauseOOCInput{})
				return err
			},
			verify: func(t *testing.T, client *fakeInteractionClient) {
				t.Helper()
				if client.lastClearRequest == nil || client.lastClearRequest.GetCampaignId() != "camp-1" {
					t.Fatalf("clear request = %#v", client.lastClearRequest)
				}
			},
		},
		{
			name: "resume ooc",
			run: func(client *fakeInteractionClient) error {
				client.resumeResp = &statev1.ResumeFromOOCResponse{State: state}
				handler := InteractionResumeOOCHandler(client, func() Context { return Context{CampaignID: "camp-1"} }, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionPauseOOCInput{})
				return err
			},
			verify: func(t *testing.T, client *fakeInteractionClient) {
				t.Helper()
				if client.lastResumeRequest == nil || client.lastResumeRequest.GetCampaignId() != "camp-1" {
					t.Fatalf("resume request = %#v", client.lastResumeRequest)
				}
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := &fakeInteractionClient{}
			if err := tc.run(client); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tc.verify(t, client)
		})
	}
}

func TestInteractionHandlersRejectMissingResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		run         func(*fakeInteractionClient) error
		wantMessage string
	}{
		{
			name: "start phase",
			run: func(client *fakeInteractionClient) error {
				handler := InteractionStartScenePlayerPhaseHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "gm-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionStartScenePlayerPhaseInput{
					SceneID:      "scene-1",
					FrameText:    "frame",
					CharacterIDs: []string{"char-1"},
				})
				return err
			},
			wantMessage: "start scene player phase response is missing",
		},
		{
			name: "submit post",
			run: func(client *fakeInteractionClient) error {
				handler := InteractionSubmitScenePlayerPostHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "part-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionSubmitScenePlayerPostInput{
					SceneID:     "scene-1",
					SummaryText: "advance",
				})
				return err
			},
			wantMessage: "submit scene player post response is missing",
		},
		{
			name: "unyield",
			run: func(client *fakeInteractionClient) error {
				handler := InteractionUnyieldScenePlayerPhaseHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "part-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionScenePhaseInput{SceneID: "scene-1"})
				return err
			},
			wantMessage: "unyield scene player phase response is missing",
		},
		{
			name: "accept phase",
			run: func(client *fakeInteractionClient) error {
				handler := InteractionAcceptScenePlayerPhaseHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "gm-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionAcceptScenePlayerPhaseInput{
					SceneID: "scene-1",
				})
				return err
			},
			wantMessage: "accept scene player phase response is missing",
		},
		{
			name: "request revisions",
			run: func(client *fakeInteractionClient) error {
				handler := InteractionRequestScenePlayerRevisionsHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "gm-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionRequestScenePlayerRevisionsInput{
					SceneID: "scene-1",
					Revisions: []InteractionScenePlayerRevisionInput{
						{ParticipantID: "part-1", Reason: "wrong spell"},
					},
				})
				return err
			},
			wantMessage: "request scene player revisions response is missing",
		},
		{
			name: "end phase",
			run: func(client *fakeInteractionClient) error {
				handler := InteractionEndScenePlayerPhaseHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "gm-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionEndScenePlayerPhaseInput{
					SceneID: "scene-1",
				})
				return err
			},
			wantMessage: "end scene player phase response is missing",
		},
		{
			name: "pause ooc",
			run: func(client *fakeInteractionClient) error {
				handler := InteractionPauseOOCHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "gm-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionPauseOOCInput{})
				return err
			},
			wantMessage: "pause session for ooc response is missing",
		},
		{
			name: "post ooc",
			run: func(client *fakeInteractionClient) error {
				handler := InteractionPostOOCHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "part-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionPostOOCInput{Body: "question"})
				return err
			},
			wantMessage: "post session ooc response is missing",
		},
		{
			name: "mark ready",
			run: func(client *fakeInteractionClient) error {
				handler := InteractionMarkOOCReadyHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "part-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionPauseOOCInput{})
				return err
			},
			wantMessage: "mark ooc ready response is missing",
		},
		{
			name: "clear ready",
			run: func(client *fakeInteractionClient) error {
				handler := InteractionClearOOCReadyHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "part-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionPauseOOCInput{})
				return err
			},
			wantMessage: "clear ooc ready response is missing",
		},
		{
			name: "resume ooc",
			run: func(client *fakeInteractionClient) error {
				handler := InteractionResumeOOCHandler(client, func() Context {
					return Context{CampaignID: "camp-1", ParticipantID: "gm-1"}
				}, nil)
				_, _, err := handler(context.Background(), &mcp.CallToolRequest{}, InteractionPauseOOCInput{})
				return err
			},
			wantMessage: "resume from ooc response is missing",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := &fakeInteractionClient{}
			err := tc.run(client)
			if err == nil || !strings.Contains(err.Error(), tc.wantMessage) {
				t.Fatalf("error = %v, want message containing %q", err, tc.wantMessage)
			}
		})
	}
}

func TestInteractionToolResultAndCallContextErrors(t *testing.T) {
	t.Parallel()

	if _, _, err := interactionToolResult(context.Background(), nil, "camp-1", nil, ToolCallMetadata{}, nil); err == nil {
		t.Fatal("expected missing state error")
	}

	_, _, _, err := interactionCallContext(context.Background(), func() Context { return Context{} }, "")
	if err == nil {
		t.Fatal("expected missing campaign id error")
	}
}

func TestInteractionStateResourceHandlerRejectsMissingClientAndURI(t *testing.T) {
	t.Parallel()

	handler := InteractionStateResourceHandler(nil, func() Context { return Context{} })
	if _, err := handler(context.Background(), &mcp.ReadResourceRequest{Params: &mcp.ReadResourceParams{URI: "campaign://camp-1/interaction"}}); err == nil {
		t.Fatal("expected missing client error")
	}

	handler = InteractionStateResourceHandler(&fakeInteractionClient{}, func() Context { return Context{} })
	if _, err := handler(context.Background(), &mcp.ReadResourceRequest{}); err == nil {
		t.Fatal("expected missing uri error")
	}
}

func TestScenePhaseStatusToString(t *testing.T) {
	tests := []struct {
		name   string
		status statev1.ScenePhaseStatus
		want   string
	}{
		{name: "gm", status: statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM, want: "GM"},
		{name: "players", status: statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS, want: "PLAYERS"},
		{name: "gm review", status: statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW, want: "GM_REVIEW"},
		{name: "unspecified", status: statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_UNSPECIFIED, want: "UNSPECIFIED"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := scenePhaseStatusToString(tc.status); got != tc.want {
				t.Fatalf("scenePhaseStatusToString(%v) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func TestScenePlayerSlotReviewStatusToString(t *testing.T) {
	tests := []struct {
		name   string
		status statev1.ScenePlayerSlotReviewStatus
		want   string
	}{
		{name: "open", status: statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_OPEN, want: "OPEN"},
		{name: "under review", status: statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_UNDER_REVIEW, want: "UNDER_REVIEW"},
		{name: "accepted", status: statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_ACCEPTED, want: "ACCEPTED"},
		{name: "changes requested", status: statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_CHANGES_REQUESTED, want: "CHANGES_REQUESTED"},
		{name: "unspecified", status: statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_UNSPECIFIED, want: "UNSPECIFIED"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := scenePlayerSlotReviewStatusToString(tc.status); got != tc.want {
				t.Fatalf("scenePlayerSlotReviewStatusToString(%v) = %q, want %q", tc.status, got, tc.want)
			}
		})
	}
}

func testInteractionState() *statev1.InteractionState {
	now := time.Date(2026, 3, 12, 12, 0, 0, 0, time.UTC)
	return &statev1.InteractionState{
		CampaignId:   "camp-1",
		CampaignName: "Northreach",
		Viewer: &statev1.InteractionViewer{
			ParticipantId: "part-1",
			Name:          "Aria",
			Role:          statev1.ParticipantRole_PLAYER,
		},
		ActiveSession: &statev1.InteractionSession{
			SessionId: "sess-1",
			Name:      "Session One",
		},
		ActiveScene: &statev1.InteractionScene{
			SceneId:     "scene-1",
			SessionId:   "sess-1",
			Name:        "Bridge",
			Description: "A narrow bridge.",
			Characters: []*statev1.InteractionCharacter{
				{CharacterId: "char-1", Name: "Aria", OwnerParticipantId: "part-1"},
			},
		},
		PlayerPhase: &statev1.ScenePlayerPhase{
			PhaseId:              "phase-1",
			Status:               statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS,
			FrameText:            "What do you do?",
			ActingCharacterIds:   []string{"char-1"},
			ActingParticipantIds: []string{"part-1"},
			Slots: []*statev1.ScenePlayerSlot{
				{
					ParticipantId: "part-1",
					SummaryText:   "I cross the bridge.",
					CharacterIds:  []string{"char-1"},
					UpdatedAt:     timestamppb.New(now),
					Yielded:       true,
					ReviewStatus:  statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_OPEN,
				},
			},
		},
		Ooc: &statev1.OOCState{
			Open:                        true,
			ReadyToResumeParticipantIds: []string{"part-1"},
			Posts: []*statev1.OOCPost{
				{
					PostId:        "ooc-1",
					ParticipantId: "part-1",
					Body:          "Rules check.",
					CreatedAt:     timestamppb.New(now),
				},
			},
		},
	}
}

func testReviewedInteractionState() *statev1.InteractionState {
	state := testInteractionState()
	state.PlayerPhase.Status = statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW
	state.PlayerPhase.ActingCharacterIds = []string{"char-1", "char-2"}
	state.PlayerPhase.ActingParticipantIds = []string{"part-1", "part-2"}
	state.ActiveScene.Characters = append(state.ActiveScene.Characters, &statev1.InteractionCharacter{
		CharacterId:        "char-2",
		Name:               "Corin",
		OwnerParticipantId: "part-2",
	})
	state.PlayerPhase.Slots = []*statev1.ScenePlayerSlot{
		{
			ParticipantId:      "part-1",
			SummaryText:        "Corin casts Fireball.",
			CharacterIds:       []string{"char-1"},
			UpdatedAt:          timestamppb.New(time.Date(2026, 3, 12, 12, 1, 0, 0, time.UTC)),
			Yielded:            false,
			ReviewStatus:       statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_CHANGES_REQUESTED,
			ReviewReason:       "Corin does not know Fireball.",
			ReviewCharacterIds: []string{"char-1"},
		},
		{
			ParticipantId: "part-2",
			SummaryText:   "I hold the torch.",
			CharacterIds:  []string{"char-2"},
			UpdatedAt:     timestamppb.New(time.Date(2026, 3, 12, 12, 2, 0, 0, time.UTC)),
			Yielded:       true,
			ReviewStatus:  statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_ACCEPTED,
		},
	}
	return state
}
