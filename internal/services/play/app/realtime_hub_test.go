package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	participants := &authSensitivePlayParticipantClient{response: enrichedParticipantResponse()}
	characters := &authSensitivePlayCharacterClient{
		listResponse:  enrichedCharacterResponse(),
		sheetResponse: enrichedCharacterSheetResponse(),
	}
	server.participants = participants
	server.characters = characters
	events := &fakeEventClient{stream: &fakeCampaignUpdateStream{}, subscribeCh: make(chan struct{}, 1)}
	server.events = events
	hub := newRealtimeHub(server)
	server.realtime = hub

	var buffer syncedFrameBuffer
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
	var ready playprotocol.RoomSnapshot
	if err := json.Unmarshal(frames[0].Payload, &ready); err != nil {
		t.Fatalf("decode ready payload: %v", err)
	}
	if len(ready.Participants) != 2 {
		t.Fatalf("ready participants = %#v, want 2 entries", ready.Participants)
	}
	if got := ready.CharacterInspectionCatalog["char-1"].System; got != "daggerheart" {
		t.Fatalf("ready character_inspection_catalog[char-1].system = %q, want %q", got, "daggerheart")
	}
	if frames[1].Type != "play.chat.message" {
		t.Fatalf("second frame type = %q, want %q", frames[1].Type, "play.chat.message")
	}
	if participants.lastUserID != "user-1" || characters.lastUserID != "user-1" {
		t.Fatalf("auth metadata = participant:%q character:%q, want user-1", participants.lastUserID, characters.lastUserID)
	}
	events.awaitSubscribe(t)
	if events.lastUserID != "user-1" {
		t.Fatalf("event auth metadata = %q, want %q", events.lastUserID, "user-1")
	}
	if events.lastRequest == nil {
		t.Fatal("SubscribeCampaignUpdates request = nil")
	}
	if events.lastRequest.GetAfterSeq() != 0 {
		t.Fatalf("SubscribeCampaignUpdates after_seq = %d, want %d", events.lastRequest.GetAfterSeq(), 0)
	}

	hub.handleTyping(session, wsFrame{
		Type:      "play.typing",
		RequestID: "req-2",
		Payload:   mustJSON(typingPayload{Active: true}),
	})
	typingFrames := drainWSFrames(t, &buffer)
	if len(typingFrames) != 1 || typingFrames[0].Type != "play.typing" {
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

func TestRealtimeConnectSubscribesToAIDebugAndBroadcastsUpdates(t *testing.T) {
	t.Parallel()

	interaction := newRecordingInteractionClient(playTestState())
	transcripts := &scriptTranscriptStore{}
	server := newAuthedPlayServer(interaction, transcripts)
	server.events = &fakeEventClient{stream: &fakeCampaignUpdateStream{}, subscribeCh: make(chan struct{}, 1)}
	aiDebug := &fakePlayAIDebugClient{
		subscribeStream: &fakeCampaignDebugUpdateStream{updates: make(chan *aiv1.CampaignDebugTurnUpdate, 1)},
		subscribeCh:     make(chan struct{}, 1),
	}
	server.aiDebug = aiDebug
	hub := newRealtimeHub(server)
	server.realtime = hub
	defer hub.Close()

	var buffer syncedFrameBuffer
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

	aiDebug.awaitSubscribe(t)
	if aiDebug.subscribeUserID != "user-1" {
		t.Fatalf("ai debug auth metadata = %q, want %q", aiDebug.subscribeUserID, "user-1")
	}
	if aiDebug.subscribeReq == nil {
		t.Fatal("SubscribeCampaignDebugUpdates request = nil")
	}
	if aiDebug.subscribeReq.GetCampaignId() != "c1" || aiDebug.subscribeReq.GetSessionId() != "s1" {
		t.Fatalf("SubscribeCampaignDebugUpdates request = %#v", aiDebug.subscribeReq)
	}

	stream := aiDebug.subscribeStream.(*fakeCampaignDebugUpdateStream)
	stream.updates <- &aiv1.CampaignDebugTurnUpdate{
		Turn: &aiv1.CampaignDebugTurnSummary{
			Id:         "turn-1",
			CampaignId: "c1",
			SessionId:  "s1",
			Status:     aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_RUNNING,
			StartedAt:  timestamppb.Now(),
			UpdatedAt:  timestamppb.Now(),
			EntryCount: 1,
		},
		AppendedEntries: []*aiv1.CampaignDebugEntry{{
			Sequence:  1,
			Kind:      aiv1.CampaignDebugEntryKind_CAMPAIGN_DEBUG_ENTRY_KIND_TOOL_CALL,
			ToolName:  "scene_create",
			Payload:   `{"name":"Harbor"}`,
			CreatedAt: timestamppb.Now(),
		}},
	}

	var frames []wsFrame
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); {
		frames = drainWSFrames(t, &buffer)
		if len(frames) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(frames) != 1 || frames[0].Type != "play.ai_debug.turn.updated" {
		t.Fatalf("ai debug frames = %#v", frames)
	}
	var payload playprotocol.AIDebugTurnUpdate
	if err := json.Unmarshal(frames[0].Payload, &payload); err != nil {
		t.Fatalf("decode ai debug update payload: %v", err)
	}
	if payload.Turn.ID != "turn-1" || len(payload.AppendedEntries) != 1 || payload.AppendedEntries[0].ToolName != "scene_create" {
		t.Fatalf("ai debug payload = %#v", payload)
	}
}

func TestRealtimeChatSendRequiresActiveSession(t *testing.T) {
	t.Parallel()

	state := playTestState()
	state.ActiveSession = nil
	interaction := newRecordingInteractionClient(state)
	transcripts := &scriptTranscriptStore{}
	server := newAuthedPlayServer(interaction, transcripts)
	server.events = &fakeEventClient{stream: &fakeCampaignUpdateStream{}}
	hub := newRealtimeHub(server)
	server.realtime = hub
	defer hub.Close()

	var buffer syncedFrameBuffer
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

func TestRealtimeTypingRequiresCampaignRoom(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	hub := newRealtimeHub(server)
	server.realtime = hub
	defer hub.Close()

	var buffer syncedFrameBuffer
	session := &realtimeSession{
		userID:          "user-1",
		peer:            &wsPeer{encoder: json.NewEncoder(&buffer)},
		campaignID:      "c1",
		participantID:   "p1",
		participantName: "Avery",
		activeSessionID: "s1",
	}

	hub.handleTyping(session, wsFrame{
		Type:      "play.typing",
		RequestID: "req-1",
		Payload:   mustJSON(typingPayload{Active: true}),
	})

	frames := drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.error" {
		t.Fatalf("typing frames = %#v", frames)
	}
	var payload playprotocol.ErrorEnvelope
	if err := json.Unmarshal(frames[0].Payload, &payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Error.Message != "join a campaign before sending typing" {
		t.Fatalf("error message = %q", payload.Error.Message)
	}
}

func TestRealtimeTypingRequiresParticipantIdentity(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	hub := newRealtimeHub(server)
	server.realtime = hub
	defer hub.Close()

	room := &campaignRoom{
		hub:        hub,
		campaignID: "c1",
		ctx:        context.Background(),
		cancel:     func() {},
		sessions:   map[*realtimeSession]struct{}{},
	}
	hub.rooms["c1"] = room

	var buffer syncedFrameBuffer
	session := &realtimeSession{
		userID:        "user-1",
		peer:          &wsPeer{encoder: json.NewEncoder(&buffer)},
		room:          room,
		campaignID:    "c1",
		participantID: "p1",
	}

	hub.handleTyping(session, wsFrame{
		Type:      "play.typing",
		RequestID: "req-1",
		Payload:   mustJSON(typingPayload{Active: true}),
	})

	frames := drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.error" {
		t.Fatalf("typing frames = %#v", frames)
	}
	var payload playprotocol.ErrorEnvelope
	if err := json.Unmarshal(frames[0].Payload, &payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Error.Message != "participant identity unavailable" {
		t.Fatalf("error message = %q", payload.Error.Message)
	}
}

func TestRealtimeTypingRejectsInvalidPayload(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{})
	hub := newRealtimeHub(server)
	server.realtime = hub
	defer hub.Close()

	var buffer syncedFrameBuffer
	session := &realtimeSession{
		userID: "user-1",
		peer:   &wsPeer{encoder: json.NewEncoder(&buffer)},
	}

	hub.handleTyping(session, wsFrame{
		Type:      "play.typing",
		RequestID: "req-1",
		Payload:   []byte(`{`),
	})

	frames := drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.error" {
		t.Fatalf("typing frames = %#v", frames)
	}
	var payload playprotocol.ErrorEnvelope
	if err := json.Unmarshal(frames[0].Payload, &payload); err != nil {
		t.Fatalf("decode error payload: %v", err)
	}
	if payload.Error.Message != "invalid typing payload" {
		t.Fatalf("error message = %q", payload.Error.Message)
	}
}

func TestRealtimeRoomLifecycleAndBroadcasts(t *testing.T) {
	t.Parallel()

	server := newAuthedPlayServer(newRecordingInteractionClient(playTestState()), &scriptTranscriptStore{latest: 5})
	participants := &authSensitivePlayParticipantClient{response: enrichedParticipantResponse()}
	characters := &authSensitivePlayCharacterClient{
		listResponse:  enrichedCharacterResponse(),
		sheetResponse: enrichedCharacterSheetResponse(),
	}
	server.participants = participants
	server.characters = characters
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

	var buffer syncedFrameBuffer
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
	if identity, ok := session.chatIdentity(); !ok || identity.CampaignID != "c1" || identity.SessionID != "s1" || identity.ParticipantID != "p1" || identity.ParticipantName != "Avery" {
		t.Fatalf("chatIdentity() = (%+v, %v)", identity, ok)
	}

	hub.broadcastCurrent("c1")
	frames := drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.interaction.updated" {
		t.Fatalf("broadcastCurrent frames = %#v", frames)
	}
	var payload playprotocol.RoomSnapshot
	if err := json.Unmarshal(frames[0].Payload, &payload); err != nil {
		t.Fatalf("decode interaction update payload: %v", err)
	}
	if len(payload.Participants) != 2 {
		t.Fatalf("broadcast participants = %#v, want 2 entries", payload.Participants)
	}
	if got := payload.CharacterInspectionCatalog["char-1"].System; got != "daggerheart" {
		t.Fatalf("broadcast character_inspection_catalog[char-1].system = %q, want %q", got, "daggerheart")
	}
	if participants.lastUserID != "user-1" || characters.lastUserID != "user-1" {
		t.Fatalf("auth metadata = participant:%q character:%q, want user-1", participants.lastUserID, characters.lastUserID)
	}

	room.broadcastFrame(wsFrame{Type: "play.ping", Payload: mustJSON(map[string]any{"ok": true})})
	frames = drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.ping" {
		t.Fatalf("broadcastFrame frames = %#v", frames)
	}

	session.resetTypingTimer(true)
	time.Sleep(defaultTypingTTL + 200*time.Millisecond)
	frames = drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.typing" {
		t.Fatalf("typing expiry frames = %#v", frames)
	}

	hub.unregisterSession(session)
	if got := len(room.sessionsSnapshot()); got != 0 {
		t.Fatalf("sessions after unregister = %d, want %d", got, 0)
	}
}

func TestRealtimeUnregisterSessionClearsTypingPresence(t *testing.T) {
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

	var buffer syncedFrameBuffer
	session := &realtimeSession{
		userID: "user-1",
		peer:   &wsPeer{encoder: json.NewEncoder(&buffer)},
	}
	session.attach(room, playprotocol.InteractionStateFromGameState(playTestState()))
	room.add(session)

	session.resetTypingTimer(true)
	_ = drainWSFrames(t, &buffer)

	hub.unregisterSession(session)

	frames := drainWSFrames(t, &buffer)
	if len(frames) != 1 || frames[0].Type != "play.typing" {
		t.Fatalf("typing frames = %#v", frames)
	}
	var payload playprotocol.TypingEvent
	if err := json.Unmarshal(frames[0].Payload, &payload); err != nil {
		t.Fatalf("decode typing payload: %v", err)
	}
	if payload.Active {
		t.Fatal("typing clear event should mark inactive")
	}
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

	if identity, ok := session.chatIdentity(); ok {
		t.Fatalf("chatIdentity() = (%+v, %v), want inactive session rejection", identity, ok)
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
