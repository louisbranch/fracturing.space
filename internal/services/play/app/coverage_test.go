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
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	gogrpc "google.golang.org/grpc"
	gogrpccodes "google.golang.org/grpc/codes"
	gogrpcmetadata "google.golang.org/grpc/metadata"
	gogrpcstatus "google.golang.org/grpc/status"
)

func TestHandleChatHistoryVariants(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		interaction := newRecordingInteractionClient(playTestState())
		transcripts := &scriptTranscriptStore{
			latest: 7,
			before: []transcript.Message{{
				MessageID:  "m1",
				CampaignID: "c1",
				SessionID:  "s1",
				SequenceID: 4,
				SentAt:     "2026-03-13T12:00:00Z",
				Actor: transcript.MessageActor{
					ParticipantID: "p1",
					Name:          "Avery",
				},
				Body:            "Hello",
				ClientMessageID: "cm-1",
			}},
		}
		server := newAuthedPlayServer(interaction, transcripts)

		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/chat/history?before_seq=9&limit=2", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleChatHistory(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var payload playHistoryResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode history response: %v", err)
		}
		if payload.SessionID != "s1" {
			t.Fatalf("session_id = %q, want %q", payload.SessionID, "s1")
		}
		if len(payload.Messages) != 1 || payload.Messages[0].MessageID != "m1" {
			t.Fatalf("messages = %#v", payload.Messages)
		}
		if transcripts.beforeArgs.before != 9 {
			t.Fatalf("before_seq = %d, want %d", transcripts.beforeArgs.before, 9)
		}
		if transcripts.beforeArgs.limit != 2 {
			t.Fatalf("limit = %d, want %d", transcripts.beforeArgs.limit, 2)
		}
	})

	t.Run("invalid before sequence", func(t *testing.T) {
		t.Parallel()

		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/chat/history?before_seq=oops", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleChatHistory(rr, req)

		assertJSONError(t, rr, http.StatusBadRequest, "invalid before_seq")
	})

	t.Run("invalid limit", func(t *testing.T) {
		t.Parallel()

		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/chat/history?limit=oops", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleChatHistory(rr, req)

		assertJSONError(t, rr, http.StatusBadRequest, "invalid limit")
	})

	t.Run("missing active session returns empty payload", func(t *testing.T) {
		t.Parallel()

		state := playTestState()
		state.ActiveSession = nil
		server := newAuthedPlayServer(newRecordingInteractionClient(state), &scriptTranscriptStore{})
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/api/campaigns/c1/chat/history", nil)
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleChatHistory(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
		}
		var payload playHistoryResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("decode history response: %v", err)
		}
		if payload.SessionID != "" || len(payload.Messages) != 0 {
			t.Fatalf("payload = %#v", payload)
		}
	})
}

func TestInteractionMutationHandlersProxyRequests(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		body       string
		wantMethod string
		call       func(*Server, http.ResponseWriter, *http.Request)
	}{
		{name: "set active scene", body: `{}`, wantMethod: "SetActiveScene", call: (*Server).handleSetActiveScene},
		{name: "start scene player phase", body: `{}`, wantMethod: "StartScenePlayerPhase", call: (*Server).handleStartScenePlayerPhase},
		{name: "submit scene player post", body: `{}`, wantMethod: "SubmitScenePlayerPost", call: (*Server).handleSubmitScenePlayerPost},
		{name: "yield scene player phase", body: `{}`, wantMethod: "YieldScenePlayerPhase", call: (*Server).handleYieldScenePlayerPhase},
		{name: "unyield scene player phase", body: `{}`, wantMethod: "UnyieldScenePlayerPhase", call: (*Server).handleUnyieldScenePlayerPhase},
		{name: "end scene player phase", body: `{}`, wantMethod: "EndScenePlayerPhase", call: (*Server).handleEndScenePlayerPhase},
		{name: "commit scene gm output", body: `{}`, wantMethod: "CommitSceneGMOutput", call: (*Server).handleCommitSceneGMOutput},
		{name: "accept scene player phase", body: `{}`, wantMethod: "AcceptScenePlayerPhase", call: (*Server).handleAcceptScenePlayerPhase},
		{name: "request scene player revisions", body: `{}`, wantMethod: "RequestScenePlayerRevisions", call: (*Server).handleRequestScenePlayerRevisions},
		{name: "pause session for ooc", body: `{}`, wantMethod: "PauseSessionForOOC", call: (*Server).handlePauseSessionForOOC},
		{name: "post session ooc", body: `{}`, wantMethod: "PostSessionOOC", call: (*Server).handlePostSessionOOC},
		{name: "mark ooc ready", wantMethod: "MarkOOCReadyToResume", call: (*Server).handleMarkOOCReadyToResume},
		{name: "clear ooc ready", wantMethod: "ClearOOCReadyToResume", call: (*Server).handleClearOOCReadyToResume},
		{name: "resume from ooc", wantMethod: "ResumeFromOOC", call: (*Server).handleResumeFromOOC},
		{name: "set gm authority", body: `{}`, wantMethod: "SetSessionGMAuthority", call: (*Server).handleSetSessionGMAuthority},
		{name: "retry ai gm turn", body: `{}`, wantMethod: "RetryAIGMTurn", call: (*Server).handleRetryAIGMTurn},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			interaction := newRecordingInteractionClient(playTestState())
			transcripts := &scriptTranscriptStore{latest: 11}
			server := newAuthedPlayServer(interaction, transcripts)

			req := httptest.NewRequest(http.MethodPost, "http://play.example.com/api/campaigns/c1/interaction/test", strings.NewReader(tc.body))
			req.SetPathValue("campaignID", "c1")
			req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
			rr := httptest.NewRecorder()

			tc.call(server, rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d body=%s", rr.Code, http.StatusOK, rr.Body.String())
			}
			if interaction.lastMethod != tc.wantMethod {
				t.Fatalf("method = %q, want %q", interaction.lastMethod, tc.wantMethod)
			}
			if interaction.lastCampaignID != "c1" {
				t.Fatalf("campaign_id = %q, want %q", interaction.lastCampaignID, "c1")
			}

			var payload playRoomSnapshot
			if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode interaction response: %v", err)
			}
			if payload.Chat.LatestSequenceID != 11 {
				t.Fatalf("latest_sequence_id = %d, want %d", payload.Chat.LatestSequenceID, 11)
			}
		})
	}
}

func TestInteractionMutationRejectsInvalidJSONAndAuthFailures(t *testing.T) {
	t.Parallel()

	t.Run("invalid json body", func(t *testing.T) {
		t.Parallel()

		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		req := httptest.NewRequest(http.MethodPost, "http://play.example.com/api/campaigns/c1/interaction/set-active-scene", strings.NewReader(`{"unknown":true}`))
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleSetActiveScene(rr, req)

		assertJSONError(t, rr, http.StatusBadRequest, "invalid json body")
	})

	t.Run("missing play session", func(t *testing.T) {
		t.Parallel()

		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		req := httptest.NewRequest(http.MethodPost, "http://play.example.com/api/campaigns/c1/interaction/set-active-scene", strings.NewReader(`{}`))
		req.SetPathValue("campaignID", "c1")
		rr := httptest.NewRecorder()

		server.handleSetActiveScene(rr, req)

		assertJSONError(t, rr, http.StatusUnauthorized, "authentication required")
	})

	t.Run("upstream invalid argument", func(t *testing.T) {
		t.Parallel()

		interaction := newRecordingInteractionClient(playTestState())
		interaction.mutationErr = gogrpcstatus.Error(gogrpccodes.InvalidArgument, "bad scene")
		server := newAuthedPlayServer(interaction, &scriptTranscriptStore{})
		req := httptest.NewRequest(http.MethodPost, "http://play.example.com/api/campaigns/c1/interaction/set-active-scene", strings.NewReader(`{}`))
		req.SetPathValue("campaignID", "c1")
		req.AddCookie(&http.Cookie{Name: playSessionCookieName, Value: "ps-1"})
		rr := httptest.NewRecorder()

		server.handleSetActiveScene(rr, req)

		assertJSONError(t, rr, http.StatusBadRequest, "bad scene")
	})
}

func TestNewHandlerRegistersHealthAndRealtimeRoutes(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	handler, err := server.newHandler(Config{LaunchGrant: testPlayLaunchGrantConfig(t)})
	if err != nil {
		t.Fatalf("newHandler() error = %v", err)
	}

	upReq := httptest.NewRequest(http.MethodGet, "http://play.example.com/up", nil)
	upRR := httptest.NewRecorder()
	handler.ServeHTTP(upRR, upReq)
	if upRR.Code != http.StatusOK || strings.TrimSpace(upRR.Body.String()) != "OK" {
		t.Fatalf("/up = %d %q", upRR.Code, upRR.Body.String())
	}

	realtimeReq := httptest.NewRequest(http.MethodGet, "http://play.example.com/realtime", nil)
	realtimeRR := httptest.NewRecorder()
	handler.ServeHTTP(realtimeRR, realtimeReq)
	if realtimeRR.Code != http.StatusUnauthorized {
		t.Fatalf("/realtime status = %d, want %d", realtimeRR.Code, http.StatusUnauthorized)
	}
}

func TestPlayHelpers(t *testing.T) {
	t.Parallel()

	t.Run("load system metadata", func(t *testing.T) {
		t.Parallel()

		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		server.campaign = fakePlayCampaignClient{response: &gamev1.GetCampaignResponse{
			Campaign: &gamev1.Campaign{Id: "c1", System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
		}}
		server.system = fakePlaySystemClient{response: &gamev1.GetGameSystemResponse{
			System: &gamev1.GameSystemInfo{Id: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, Name: "Daggerheart", Version: "v1"},
		}}

		system, err := server.loadSystemMetadata(context.Background(), "c1", "user-1")
		if err != nil {
			t.Fatalf("loadSystemMetadata() error = %v", err)
		}
		if system.ID != "daggerheart" || system.Name != "Daggerheart" || system.Version != "v1" {
			t.Fatalf("system = %#v", system)
		}
	})

	t.Run("shell assets dev server and html", func(t *testing.T) {
		t.Parallel()

		assets, err := loadShellAssets("http://localhost:5173/")
		if err != nil {
			t.Fatalf("loadShellAssets() error = %v", err)
		}
		if assets.devServerURL != "http://localhost:5173" {
			t.Fatalf("dev_server_url = %q", assets.devServerURL)
		}
		html, err := assets.renderHTML(shellRenderInput{
			CampaignID:    "c1",
			BootstrapPath: "/api/campaigns/c1/bootstrap",
			RealtimePath:  "/realtime",
			BackURL:       "/app/campaigns/c1/game",
		})
		if err != nil {
			t.Fatalf("renderHTML() error = %v", err)
		}
		if !strings.Contains(string(html), "<div id=\"root\"></div>") {
			t.Fatalf("rendered html missing root: %s", html)
		}
	})

	t.Run("rpc errors map to http status", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			err    error
			status int
		}{
			{err: gogrpcstatus.Error(gogrpccodes.InvalidArgument, "bad"), status: http.StatusBadRequest},
			{err: gogrpcstatus.Error(gogrpccodes.PermissionDenied, "bad"), status: http.StatusForbidden},
			{err: gogrpcstatus.Error(gogrpccodes.NotFound, "bad"), status: http.StatusNotFound},
			{err: gogrpcstatus.Error(gogrpccodes.FailedPrecondition, "bad"), status: http.StatusConflict},
			{err: gogrpcstatus.Error(gogrpccodes.Unauthenticated, "bad"), status: http.StatusUnauthorized},
			{err: errors.New("boom"), status: http.StatusBadGateway},
		}
		for _, tc := range cases {
			rr := httptest.NewRecorder()
			writeRPCError(rr, tc.err)
			if rr.Code != tc.status {
				t.Fatalf("status = %d, want %d for %v", rr.Code, tc.status, tc.err)
			}
		}
	})

	t.Run("parse helpers and path helpers", func(t *testing.T) {
		t.Parallel()

		if value, err := parseInt64(" 42 "); err != nil || value != 42 {
			t.Fatalf("parseInt64() = %d, %v", value, err)
		}
		if value, err := parseInt(" 7 "); err != nil || value != 7 {
			t.Fatalf("parseInt() = %d, %v", value, err)
		}
		if got := gameSystemIDString(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART); got != "daggerheart" {
			t.Fatalf("gameSystemIDString() = %q", got)
		}
		if got := pathForCampaignAPI("c1", "chat/history"); got != "/api/campaigns/c1/chat/history" {
			t.Fatalf("pathForCampaignAPI() = %q", got)
		}
		req := httptest.NewRequest(http.MethodGet, "http://play.example.com/campaigns/c1?launch=token&foo=bar", nil)
		if got := stripLaunchGrant(req); got != "http://play.example.com/campaigns/c1?foo=bar" {
			t.Fatalf("stripLaunchGrant() = %q", got)
		}
		if loggerOrDefault(nil) == nil {
			t.Fatal("loggerOrDefault(nil) returned nil")
		}
	})

	t.Run("listen and serve guards", func(t *testing.T) {
		t.Parallel()

		if err := (*Server)(nil).ListenAndServe(context.Background()); err == nil {
			t.Fatal("ListenAndServe(nil) error = nil, want non-nil")
		}
		server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
		if err := server.ListenAndServe(nil); err == nil {
			t.Fatal("ListenAndServe(nil context) error = nil, want non-nil")
		}
	})
}

func TestRealtimeConnectTypingAndChatSend(t *testing.T) {
	t.Parallel()

	interaction := newRecordingInteractionClient(playTestState())
	transcripts := &scriptTranscriptStore{
		latest: 2,
		after: []transcript.Message{{
			MessageID:  "m-after",
			CampaignID: "c1",
			SessionID:  "s1",
			SequenceID: 2,
			SentAt:     "2026-03-13T12:00:00Z",
			Actor: transcript.MessageActor{
				ParticipantID: "p1",
				Name:          "Avery",
			},
			Body: "Earlier",
		}},
		appendMessage: transcript.Message{
			MessageID:  "m-new",
			CampaignID: "c1",
			SessionID:  "s1",
			SequenceID: 3,
			SentAt:     "2026-03-13T12:01:00Z",
			Actor: transcript.MessageActor{
				ParticipantID: "p1",
				Name:          "Avery",
			},
			Body:            "Fresh message",
			ClientMessageID: "cm-2",
		},
	}
	server := newAuthedPlayServer(interaction, transcripts)
	server.events = fakeEventClient{stream: &fakeCampaignUpdateStream{}}
	hub := newRealtimeHub(server)
	server.realtime = hub

	var buffer bytes.Buffer
	session := &realtimeSession{
		userID: "user-1",
		peer:   &wsPeer{encoder: json.NewEncoder(&buffer)},
	}

	connectPayload := mustJSON(playWSConnectPayload{CampaignID: "c1", LastChatSeq: 0})
	hub.handleConnect(context.Background(), session, wsFrame{Type: "play.connect", RequestID: "req-1", Payload: connectPayload})

	frames := drainWSFrames(t, &buffer)
	if len(frames) < 2 {
		t.Fatalf("connect frames = %#v, want ready + chat message", frames)
	}
	if frames[0].Type != "play.ready" {
		t.Fatalf("first frame type = %q, want %q", frames[0].Type, "play.ready")
	}
	if frames[1].Type != "play.chat.message" {
		t.Fatalf("second frame type = %q, want %q", frames[1].Type, "play.chat.message")
	}

	hub.handleTyping(session, wsFrame{
		Type:      "play.chat.typing",
		RequestID: "req-2",
		Payload:   mustJSON(typingPayload{Active: true}),
	}, "play.chat.typing")
	typingFrames := drainWSFrames(t, &buffer)
	if len(typingFrames) != 1 || typingFrames[0].Type != "play.chat.typing" {
		t.Fatalf("typing frames = %#v", typingFrames)
	}

	hub.handleChatSend(context.Background(), session, wsFrame{
		Type:      "play.chat.send",
		RequestID: "req-3",
		Payload:   mustJSON(playWSChatSendPayload{Body: "Fresh message", ClientMessageID: "cm-2"}),
	})
	chatFrames := drainWSFrames(t, &buffer)
	if len(chatFrames) != 1 || chatFrames[0].Type != "play.chat.message" {
		t.Fatalf("chat frames = %#v", chatFrames)
	}
	if transcripts.appendArgs.body != "Fresh message" {
		t.Fatalf("append body = %q, want %q", transcripts.appendArgs.body, "Fresh message")
	}

	hub.Close()
}

func TestRealtimeChatSendRequiresActiveSession(t *testing.T) {
	t.Parallel()

	state := playTestState()
	state.ActiveSession = nil
	interaction := newRecordingInteractionClient(state)
	transcripts := &scriptTranscriptStore{}
	server := newAuthedPlayServer(interaction, transcripts)
	server.events = fakeEventClient{stream: &fakeCampaignUpdateStream{}}
	hub := newRealtimeHub(server)
	server.realtime = hub

	var buffer bytes.Buffer
	session := &realtimeSession{
		userID: "user-1",
		peer:   &wsPeer{encoder: json.NewEncoder(&buffer)},
	}

	hub.handleConnect(context.Background(), session, wsFrame{
		Type:      "play.connect",
		RequestID: "req-1",
		Payload:   mustJSON(playWSConnectPayload{CampaignID: "c1"}),
	})
	_ = drainWSFrames(t, &buffer)

	hub.handleChatSend(context.Background(), session, wsFrame{
		Type:      "play.chat.send",
		RequestID: "req-2",
		Payload:   mustJSON(playWSChatSendPayload{Body: "Fresh message", ClientMessageID: "cm-2"}),
	})

	frames := drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.error" {
		t.Fatalf("chat frames = %#v", frames)
	}
	var payload playWSErrorEnvelope
	if err := json.Unmarshal(frames[0].Payload, &payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Error.Message != "join an active session before sending chat" {
		t.Fatalf("error message = %q", payload.Error.Message)
	}
	if transcripts.appendArgs.sessionID != "" || transcripts.appendArgs.body != "" {
		t.Fatalf("append args = %#v, want zero value", transcripts.appendArgs)
	}
}

func TestRealtimeRoomLifecycleAndBroadcasts(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{latest: 5})
	hub := newRealtimeHub(server)
	server.realtime = hub
	room := &campaignRoom{
		hub:        hub,
		campaignID: "c1",
		ctx:        context.Background(),
		cancel:     func() {},
		sessions:   map[*realtimeSession]struct{}{},
	}
	hub.rooms["c1"] = room

	var buffer bytes.Buffer
	session := &realtimeSession{
		userID: "user-1",
		peer:   &wsPeer{encoder: json.NewEncoder(&buffer)},
	}
	session.attach(room, playTestState())
	room.add(session)
	room.setLatestGameSequence(9)

	if got := room.latestGameSequence(); got != 9 {
		t.Fatalf("latestGameSequence() = %d, want %d", got, 9)
	}
	if got := len(room.sessionsSnapshot()); got != 1 {
		t.Fatalf("sessionsSnapshot() len = %d, want %d", got, 1)
	}
	if got := hub.roomIfExists("c1"); got != room {
		t.Fatalf("roomIfExists() = %#v, want %#v", got, room)
	}
	if got := session.currentRoom(); got != room {
		t.Fatalf("currentRoom() = %#v, want %#v", got, room)
	}
	if got := session.activeSession(); got != "s1" {
		t.Fatalf("activeSession() = %q, want %q", got, "s1")
	}
	if campaignID, sessionID, participantID, participantName, ok := session.chatIdentity(); !ok || campaignID != "c1" || sessionID != "s1" || participantID != "p1" || participantName != "Avery" {
		t.Fatalf("chatIdentity() = (%q, %q, %q, %q, %v)", campaignID, sessionID, participantID, participantName, ok)
	}

	hub.broadcastCurrent("c1")
	frames := drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.interaction.updated" {
		t.Fatalf("broadcastCurrent frames = %#v", frames)
	}

	room.broadcastFrame(wsFrame{Type: "play.ping", Payload: mustJSON(map[string]any{"ok": true})})
	frames = drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.ping" {
		t.Fatalf("broadcastFrame frames = %#v", frames)
	}

	session.resetTypingTimer("play.chat.typing", true)
	time.Sleep(typingTTL + 200*time.Millisecond)
	frames = drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.chat.typing" {
		t.Fatalf("typing expiry frames = %#v", frames)
	}

	hub.unregisterSession(session)
	if got := len(room.sessionsSnapshot()); got != 0 {
		t.Fatalf("sessions after unregister = %d, want %d", got, 0)
	}
}

func TestRealtimeSessionChatIdentityRequiresActiveSession(t *testing.T) {
	t.Parallel()

	room := &campaignRoom{campaignID: "c1"}
	session := &realtimeSession{
		room:            room,
		campaignID:      "c1",
		participantID:   "p1",
		participantName: "Avery",
	}

	if campaignID, sessionID, participantID, participantName, ok := session.chatIdentity(); ok {
		t.Fatalf("chatIdentity() = (%q, %q, %q, %q, %v), want inactive session rejection", campaignID, sessionID, participantID, participantName, ok)
	}
}

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
		transcripts:     transcripts,
		shellAssets:     shellAssets{devServerURL: "http://localhost:5173"},
		realtime:        nil,
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
		campaignID string
		sessionID  string
		before     int64
		limit      int
	}
	appendArgs struct {
		campaignID      string
		sessionID       string
		actor           transcript.MessageActor
		body            string
		clientMessageID string
	}
}

func (s *scriptTranscriptStore) LatestSequence(context.Context, string, string) (int64, error) {
	return s.latest, s.latestErr
}

func (s *scriptTranscriptStore) AppendMessage(_ context.Context, campaignID string, sessionID string, actor transcript.MessageActor, body string, clientMessageID string) (transcript.Message, bool, error) {
	s.appendArgs.campaignID = campaignID
	s.appendArgs.sessionID = sessionID
	s.appendArgs.actor = actor
	s.appendArgs.body = body
	s.appendArgs.clientMessageID = clientMessageID
	return s.appendMessage, false, s.appendErr
}

func (s *scriptTranscriptStore) HistoryAfter(context.Context, string, string, int64) ([]transcript.Message, error) {
	return s.after, s.afterErr
}

func (s *scriptTranscriptStore) HistoryBefore(_ context.Context, campaignID string, sessionID string, before int64, limit int) ([]transcript.Message, error) {
	s.beforeArgs.campaignID = campaignID
	s.beforeArgs.sessionID = sessionID
	s.beforeArgs.before = before
	s.beforeArgs.limit = limit
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
