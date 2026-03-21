package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
)

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

	connectPayload := mustJSON(playprotocol.ConnectRequest{CampaignID: "c1", LastChatSeq: 0})
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
		Payload:   mustJSON(playprotocol.ChatSendRequest{Body: "Fresh message", ClientMessageID: "cm-2"}),
	})
	chatFrames := drainWSFrames(t, &buffer)
	if len(chatFrames) != 1 || chatFrames[0].Type != "play.chat.message" {
		t.Fatalf("chat frames = %#v", chatFrames)
	}
	if transcripts.appendArgs.request.Body != "Fresh message" {
		t.Fatalf("append body = %q, want %q", transcripts.appendArgs.request.Body, "Fresh message")
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
		Payload:   mustJSON(playprotocol.ConnectRequest{CampaignID: "c1"}),
	})
	_ = drainWSFrames(t, &buffer)

	hub.handleChatSend(context.Background(), session, wsFrame{
		Type:      "play.chat.send",
		RequestID: "req-2",
		Payload:   mustJSON(playprotocol.ChatSendRequest{Body: "Fresh message", ClientMessageID: "cm-2"}),
	})

	frames := drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.error" {
		t.Fatalf("chat frames = %#v", frames)
	}
	var payload playprotocol.ErrorEnvelope
	if err := json.Unmarshal(frames[0].Payload, &payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Error.Message != "join an active session before sending chat" {
		t.Fatalf("error message = %q", payload.Error.Message)
	}
	if transcripts.appendArgs.request.Scope.SessionID != "" || transcripts.appendArgs.request.Body != "" {
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
	session.attach(room, playprotocol.InteractionStateFromGameState(playTestState()))
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
	time.Sleep(defaultTypingTTL + 200*time.Millisecond)
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

func TestNewHandlerRegistersHealthAndRealtimeRoutes(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	handler, err := server.newHandler(testPlayLaunchGrantConfig(t))
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
