package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"golang.org/x/net/websocket"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type wsTestFrame struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type wsTestAckPayload struct {
	Result struct {
		Status     string `json:"status"`
		MessageID  string `json:"message_id"`
		SequenceID int64  `json:"sequence_id"`
		Count      int    `json:"count"`
	} `json:"result"`
}

type wsTestMessagePayload struct {
	Message struct {
		MessageID  string `json:"message_id"`
		SequenceID int64  `json:"sequence_id"`
		Body       string `json:"body"`
		StreamID   string `json:"stream_id"`
		Actor      struct {
			PersonaID   string `json:"persona_id"`
			CharacterID string `json:"character_id"`
			Mode        string `json:"mode"`
			Name        string `json:"name"`
		} `json:"actor"`
	} `json:"message"`
}

type wsTestJoinedPayload struct {
	CampaignID       string `json:"campaign_id"`
	SessionID        string `json:"session_id"`
	DefaultStreamID  string `json:"default_stream_id"`
	DefaultPersonaID string `json:"default_persona_id"`
	Streams          []struct {
		StreamID string `json:"stream_id"`
		Label    string `json:"label"`
	} `json:"streams"`
	Personas []struct {
		PersonaID   string `json:"persona_id"`
		DisplayName string `json:"display_name"`
	} `json:"personas"`
	ActiveSessionGate struct {
		GateID   string         `json:"gate_id"`
		GateType string         `json:"gate_type"`
		Status   string         `json:"status"`
		Progress map[string]any `json:"progress"`
	} `json:"active_session_gate"`
	ActiveSessionSpotlight struct {
		Type        string `json:"type"`
		CharacterID string `json:"character_id"`
	} `json:"active_session_spotlight"`
}

type wsTestStatePayload struct {
	CampaignID        string `json:"campaign_id"`
	SessionID         string `json:"session_id"`
	ActiveSessionGate struct {
		GateID   string         `json:"gate_id"`
		GateType string         `json:"gate_type"`
		Status   string         `json:"status"`
		Progress map[string]any `json:"progress"`
	} `json:"active_session_gate"`
}

type fakeWSAuthorizer struct {
	userID                string
	authErr               error
	participantAllowed    bool
	participantByCampaign map[string]bool
	participantErr        error
}

func (f fakeWSAuthorizer) Authenticate(_ context.Context, _ string) (string, error) {
	if f.authErr != nil {
		return "", f.authErr
	}
	if strings.TrimSpace(f.userID) == "" {
		return "", errors.New("missing user id")
	}
	return strings.TrimSpace(f.userID), nil
}

func (f fakeWSAuthorizer) IsCampaignParticipant(_ context.Context, campaignID string, _ string) (bool, error) {
	if f.participantErr != nil {
		return false, f.participantErr
	}
	if f.participantByCampaign != nil {
		return f.participantByCampaign[campaignID], nil
	}
	return f.participantAllowed, nil
}

type fakeWSWelcomeAuthorizer struct {
	userID             string
	authErr            error
	participantAllowed bool
	participantCalls   int
	resolveWelcome     joinWelcome
	resolveErr         error
}

type fakeWSCommunicationAuthorizer struct {
	tokenToUser              map[string]string
	contextByUserID          map[string]communicationContext
	participantErr           error
	participantByUser        map[string]bool
	openGateContext          communicationContext
	openGateErr              error
	resolveGateContext       communicationContext
	resolveGateErr           error
	respondGateContext       communicationContext
	respondGateErr           error
	abandonGateContext       communicationContext
	abandonGateErr           error
	requestGMHandoffContext  communicationContext
	requestGMHandoffErr      error
	resolveGMHandoffContext  communicationContext
	resolveGMHandoffErr      error
	abandonGMHandoffContext  communicationContext
	abandonGMHandoffErr      error
	lastControlParticipantID string
	lastControlAction        string
	lastControlGateType      string
}

func (f *fakeWSWelcomeAuthorizer) Authenticate(_ context.Context, _ string) (string, error) {
	if f.authErr != nil {
		return "", f.authErr
	}
	if strings.TrimSpace(f.userID) == "" {
		return "", errors.New("missing user id")
	}
	return strings.TrimSpace(f.userID), nil
}

func (f *fakeWSWelcomeAuthorizer) IsCampaignParticipant(_ context.Context, _ string, _ string) (bool, error) {
	f.participantCalls++
	return f.participantAllowed, nil
}

func (f *fakeWSWelcomeAuthorizer) ResolveJoinWelcome(_ context.Context, campaignID string, userID string) (joinWelcome, error) {
	if f.resolveErr != nil {
		return joinWelcome{}, f.resolveErr
	}
	welcome := f.resolveWelcome
	if strings.TrimSpace(welcome.ParticipantName) == "" {
		welcome.ParticipantName = userID
	}
	if strings.TrimSpace(welcome.CampaignName) == "" {
		welcome.CampaignName = campaignID
	}
	if welcome.Locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		welcome.Locale = commonv1.Locale_LOCALE_EN_US
	}
	return welcome, nil
}

func (f *fakeWSCommunicationAuthorizer) Authenticate(_ context.Context, accessToken string) (string, error) {
	userID := strings.TrimSpace(f.tokenToUser[strings.TrimSpace(accessToken)])
	if userID == "" {
		return "", errors.New("missing user id")
	}
	return userID, nil
}

func (f *fakeWSCommunicationAuthorizer) IsCampaignParticipant(_ context.Context, _ string, userID string) (bool, error) {
	if f.participantErr != nil {
		return false, f.participantErr
	}
	if f.participantByUser == nil {
		return true, nil
	}
	return f.participantByUser[userID], nil
}

func (f *fakeWSCommunicationAuthorizer) ResolveCommunicationContext(_ context.Context, campaignID string, userID string) (communicationContext, error) {
	contextState, ok := f.contextByUserID[userID]
	if !ok {
		return communicationContext{}, errCampaignParticipantRequired
	}
	if strings.TrimSpace(contextState.Welcome.CampaignName) == "" {
		contextState.Welcome.CampaignName = campaignID
	}
	if contextState.Welcome.Locale == commonv1.Locale_LOCALE_UNSPECIFIED {
		contextState.Welcome.Locale = commonv1.Locale_LOCALE_EN_US
	}
	return contextState, nil
}

func (f *fakeWSCommunicationAuthorizer) OpenCommunicationGate(_ context.Context, _ string, participantID string, gateType string, _ string, _ map[string]any) (communicationContext, error) {
	f.lastControlParticipantID = participantID
	f.lastControlAction = "gate.open"
	f.lastControlGateType = gateType
	if f.openGateErr != nil {
		return communicationContext{}, f.openGateErr
	}
	return f.openGateContext, nil
}

func (f *fakeWSCommunicationAuthorizer) ResolveCommunicationGate(_ context.Context, _ string, participantID string, _ string, _ map[string]any) (communicationContext, error) {
	f.lastControlParticipantID = participantID
	f.lastControlAction = "gate.resolve"
	if f.resolveGateErr != nil {
		return communicationContext{}, f.resolveGateErr
	}
	return f.resolveGateContext, nil
}

func (f *fakeWSCommunicationAuthorizer) RespondToCommunicationGate(_ context.Context, _ string, participantID string, decision string, _ map[string]any) (communicationContext, error) {
	f.lastControlParticipantID = participantID
	f.lastControlAction = "gate.respond"
	f.lastControlGateType = decision
	if f.respondGateErr != nil {
		return communicationContext{}, f.respondGateErr
	}
	return f.respondGateContext, nil
}

func (f *fakeWSCommunicationAuthorizer) AbandonCommunicationGate(_ context.Context, _ string, participantID string, _ string) (communicationContext, error) {
	f.lastControlParticipantID = participantID
	f.lastControlAction = "gate.abandon"
	if f.abandonGateErr != nil {
		return communicationContext{}, f.abandonGateErr
	}
	return f.abandonGateContext, nil
}

func (f *fakeWSCommunicationAuthorizer) RequestGMHandoff(_ context.Context, _ string, participantID string, _ string, _ map[string]any) (communicationContext, error) {
	f.lastControlParticipantID = participantID
	f.lastControlAction = "gm_handoff.request"
	if f.requestGMHandoffErr != nil {
		return communicationContext{}, f.requestGMHandoffErr
	}
	return f.requestGMHandoffContext, nil
}

func (f *fakeWSCommunicationAuthorizer) ResolveGMHandoff(_ context.Context, _ string, participantID string, _ string, _ map[string]any) (communicationContext, error) {
	f.lastControlParticipantID = participantID
	f.lastControlAction = "gm_handoff.resolve"
	if f.resolveGMHandoffErr != nil {
		return communicationContext{}, f.resolveGMHandoffErr
	}
	return f.resolveGMHandoffContext, nil
}

func (f *fakeWSCommunicationAuthorizer) AbandonGMHandoff(_ context.Context, _ string, participantID string, _ string) (communicationContext, error) {
	f.lastControlParticipantID = participantID
	f.lastControlAction = "gm_handoff.abandon"
	if f.abandonGMHandoffErr != nil {
		return communicationContext{}, f.abandonGMHandoffErr
	}
	return f.abandonGMHandoffContext, nil
}

func dialWS(t *testing.T, path string) *websocket.Conn {
	t.Helper()
	return dialWSWithHandler(t, NewHandler(), path, "")
}

func dialWSWithHandler(t *testing.T, handler http.Handler, path string, cookie string) *websocket.Conn {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	conn, err := dialWSWithServerURL(srv.URL, path, cookie)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
	return conn
}

func dialWSWithHandlerErr(t *testing.T, handler http.Handler, path string, cookie string) error {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	conn, err := dialWSWithServerURL(srv.URL, path, cookie)
	if conn != nil {
		_ = conn.Close()
	}
	return err
}

func dialWSWithServerURL(httpURL string, path string, cookie string) (*websocket.Conn, error) {
	wsURL := "ws" + strings.TrimPrefix(httpURL, "http") + path
	if strings.TrimSpace(cookie) == "" {
		return websocket.Dial(wsURL, "", httpURL)
	}
	cfg, err := websocket.NewConfig(wsURL, httpURL)
	if err != nil {
		return nil, err
	}
	cfg.Header = make(http.Header)
	cfg.Header.Set("Cookie", cookie)
	return websocket.DialConfig(cfg)
}

func dialWSWithExistingServer(t *testing.T, srv *httptest.Server, path string, cookie string) *websocket.Conn {
	t.Helper()
	conn, err := dialWSWithServerURL(srv.URL, path, cookie)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})
	return conn
}

func writeFrame(t *testing.T, conn *websocket.Conn, frame map[string]any) {
	t.Helper()
	if err := json.NewEncoder(conn).Encode(frame); err != nil {
		t.Fatalf("encode frame: %v", err)
	}
}

func readFrame(t *testing.T, conn *websocket.Conn) wsTestFrame {
	t.Helper()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	var got wsTestFrame
	if err := json.NewDecoder(conn).Decode(&got); err != nil {
		t.Fatalf("decode server frame: %v", err)
	}
	return got
}

func decodeAckPayload(t *testing.T, payload json.RawMessage) wsTestAckPayload {
	t.Helper()
	var ack wsTestAckPayload
	if err := json.Unmarshal(payload, &ack); err != nil {
		t.Fatalf("decode ack payload: %v", err)
	}
	return ack
}

func decodeMessagePayload(t *testing.T, payload json.RawMessage) wsTestMessagePayload {
	t.Helper()
	var msg wsTestMessagePayload
	if err := json.Unmarshal(payload, &msg); err != nil {
		t.Fatalf("decode message payload: %v", err)
	}
	return msg
}

func joinCampaign(t *testing.T, conn *websocket.Conn, campaignID string) {
	t.Helper()
	writeFrame(t, conn, map[string]any{
		"type":       "chat.join",
		"request_id": "req-join-1",
		"payload": map[string]any{
			"campaign_id":      campaignID,
			"last_sequence_id": 0,
		},
	})
	got := readFrame(t, conn)
	if got.Type != "chat.joined" {
		t.Fatalf("frame type = %q, want %q", got.Type, "chat.joined")
	}
	welcome := readFrame(t, conn)
	if welcome.Type != "chat.message" {
		t.Fatalf("frame type = %q, want %q", welcome.Type, "chat.message")
	}
}

func TestWebSocketJoinReturnsJoinedFrame(t *testing.T) {
	conn := dialWS(t, "/ws")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.join",
		"request_id": "req-join-1",
		"payload": map[string]any{
			"campaign_id":      "camp-1",
			"last_sequence_id": 0,
		},
	})

	got := readFrame(t, conn)
	if got.Type != "chat.joined" {
		t.Fatalf("frame type = %q, want %q", got.Type, "chat.joined")
	}
	if !strings.Contains(string(got.Payload), "camp-1") {
		t.Fatalf("joined payload = %s, expected campaign id", string(got.Payload))
	}
}

func TestWebSocketUnknownTypeReturnsChatError(t *testing.T) {
	conn := dialWS(t, "/ws")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.unknown",
		"request_id": "req-bad-1",
		"payload":    map[string]any{},
	})

	got := readFrame(t, conn)
	if got.Type != "chat.error" {
		t.Fatalf("frame type = %q, want %q", got.Type, "chat.error")
	}
	if !strings.Contains(string(got.Payload), "INVALID_ARGUMENT") {
		t.Fatalf("error payload = %s, expected INVALID_ARGUMENT code", string(got.Payload))
	}
}

func TestWebSocketSendBeforeJoinReturnsForbidden(t *testing.T) {
	conn := dialWS(t, "/ws")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.send",
		"request_id": "req-send-before-join",
		"payload": map[string]any{
			"client_message_id": "cli-1",
			"body":              "hello",
		},
	})

	got := readFrame(t, conn)
	if got.Type != "chat.error" {
		t.Fatalf("frame type = %q, want %q", got.Type, "chat.error")
	}
	if !strings.Contains(string(got.Payload), "FORBIDDEN") {
		t.Fatalf("error payload = %s, expected FORBIDDEN", string(got.Payload))
	}
}

func TestWebSocketSendBroadcastsWithinCampaignRoom(t *testing.T) {
	srv := httptest.NewServer(NewHandler())
	t.Cleanup(srv.Close)

	connA := dialWSWithExistingServer(t, srv, "/ws", "")
	connB := dialWSWithExistingServer(t, srv, "/ws", "")

	joinCampaign(t, connA, "camp-1")
	joinCampaign(t, connB, "camp-1")

	writeFrame(t, connA, map[string]any{
		"type":       "chat.send",
		"request_id": "req-send-1",
		"payload": map[string]any{
			"client_message_id": "cli-1",
			"body":              "hello room",
		},
	})

	ack := readFrame(t, connA)
	if ack.Type != "chat.ack" {
		t.Fatalf("sender frame type = %q, want %q", ack.Type, "chat.ack")
	}
	senderMessage := readFrame(t, connA)
	if senderMessage.Type != "chat.message" {
		t.Fatalf("sender frame type = %q, want %q", senderMessage.Type, "chat.message")
	}

	receiverMessage := readFrame(t, connB)
	if receiverMessage.Type != "chat.message" {
		t.Fatalf("receiver frame type = %q, want %q", receiverMessage.Type, "chat.message")
	}
	payload := decodeMessagePayload(t, receiverMessage.Payload)
	if payload.Message.Body != "hello room" {
		t.Fatalf("receiver message body = %q, want %q", payload.Message.Body, "hello room")
	}
}

func TestWebSocketSendIsIdempotentByClientMessageID(t *testing.T) {
	conn := dialWS(t, "/ws")
	joinCampaign(t, conn, "camp-1")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.send",
		"request_id": "req-send-1",
		"payload": map[string]any{
			"client_message_id": "cli-dup-1",
			"body":              "hello once",
		},
	})
	firstAck := readFrame(t, conn)
	if firstAck.Type != "chat.ack" {
		t.Fatalf("first frame type = %q, want %q", firstAck.Type, "chat.ack")
	}
	_ = readFrame(t, conn)

	writeFrame(t, conn, map[string]any{
		"type":       "chat.send",
		"request_id": "req-send-2",
		"payload": map[string]any{
			"client_message_id": "cli-dup-1",
			"body":              "hello twice",
		},
	})
	secondAck := readFrame(t, conn)
	if secondAck.Type != "chat.ack" {
		t.Fatalf("second frame type = %q, want %q", secondAck.Type, "chat.ack")
	}

	first := decodeAckPayload(t, firstAck.Payload)
	second := decodeAckPayload(t, secondAck.Payload)
	if first.Result.MessageID == "" {
		t.Fatal("expected first ack message_id")
	}
	if first.Result.MessageID != second.Result.MessageID {
		t.Fatalf("message_id mismatch: %q != %q", first.Result.MessageID, second.Result.MessageID)
	}
	if first.Result.SequenceID != second.Result.SequenceID {
		t.Fatalf("sequence_id mismatch: %d != %d", first.Result.SequenceID, second.Result.SequenceID)
	}
}

func TestWebSocketHistoryBeforeReturnsMessagesAndAck(t *testing.T) {
	conn := dialWS(t, "/ws")
	joinCampaign(t, conn, "camp-1")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.send",
		"request_id": "req-send-1",
		"payload": map[string]any{
			"client_message_id": "cli-1",
			"body":              "m1",
		},
	})
	_ = readFrame(t, conn)
	_ = readFrame(t, conn)

	writeFrame(t, conn, map[string]any{
		"type":       "chat.send",
		"request_id": "req-send-2",
		"payload": map[string]any{
			"client_message_id": "cli-2",
			"body":              "m2",
		},
	})
	_ = readFrame(t, conn)
	_ = readFrame(t, conn)

	writeFrame(t, conn, map[string]any{
		"type":       "chat.history.before",
		"request_id": "req-history-1",
		"payload": map[string]any{
			"before_sequence_id": 3,
			"limit":              10,
		},
	})

	m1 := readFrame(t, conn)
	m2 := readFrame(t, conn)
	ack := readFrame(t, conn)
	if m1.Type != "chat.message" || m2.Type != "chat.message" {
		t.Fatalf("expected two chat.message frames, got %q and %q", m1.Type, m2.Type)
	}
	if ack.Type != "chat.ack" {
		t.Fatalf("ack frame type = %q, want %q", ack.Type, "chat.ack")
	}
	ackPayload := decodeAckPayload(t, ack.Payload)
	if ackPayload.Result.Count != 2 {
		t.Fatalf("history ack count = %d, want 2", ackPayload.Result.Count)
	}
}

func TestWebSocketEndpointRequiresTokenWhenAuthorizerConfigured(t *testing.T) {
	err := dialWSWithHandlerErr(t, NewHandlerWithAuthorizer(fakeWSAuthorizer{userID: "user-1", participantAllowed: true}), "/ws", "")
	if err == nil {
		t.Fatal("expected websocket dial error")
	}
	if !strings.Contains(err.Error(), "bad status") {
		t.Fatalf("dial error = %v, expected bad status", err)
	}
}

func TestWebSocketEndpointAcceptsWebSessionCookieWhenAuthorizerConfigured(t *testing.T) {
	conn := dialWSWithHandler(t, NewHandlerWithAuthorizer(fakeWSAuthorizer{userID: "user-1", participantAllowed: true}), "/ws", "web_session=session-1")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.join",
		"request_id": "req-join-1",
		"payload": map[string]any{
			"campaign_id": "camp-1",
		},
	})

	got := readFrame(t, conn)
	if got.Type != "chat.joined" {
		t.Fatalf("frame type = %q, want %q", got.Type, "chat.joined")
	}
}

func TestWebSocketJoinRequiresParticipantMembership(t *testing.T) {
	authorizer := fakeWSAuthorizer{
		userID:             "user-1",
		participantAllowed: false,
	}
	conn := dialWSWithHandler(t, NewHandlerWithAuthorizer(authorizer), "/ws", "fs_token=token-1")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.join",
		"request_id": "req-join-1",
		"payload": map[string]any{
			"campaign_id": "camp-1",
		},
	})

	got := readFrame(t, conn)
	if got.Type != "chat.error" {
		t.Fatalf("frame type = %q, want %q", got.Type, "chat.error")
	}
	if !strings.Contains(string(got.Payload), "FORBIDDEN") {
		t.Fatalf("error payload = %s, expected FORBIDDEN", string(got.Payload))
	}
}

func TestWebSocketJoinMembershipLookupFailureReturnsUnavailable(t *testing.T) {
	authorizer := fakeWSAuthorizer{
		userID:         "user-1",
		participantErr: errCampaignSessionInactive,
	}
	conn := dialWSWithHandler(t, NewHandlerWithAuthorizer(authorizer), "/ws", "fs_token=token-1")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.join",
		"request_id": "req-join-1",
		"payload": map[string]any{
			"campaign_id": "camp-1",
		},
	})

	got := readFrame(t, conn)
	if got.Type != "chat.error" {
		t.Fatalf("frame type = %q, want %q", got.Type, "chat.error")
	}
	if !strings.Contains(string(got.Payload), "UNAVAILABLE") {
		t.Fatalf("error payload = %s, expected UNAVAILABLE", string(got.Payload))
	}
}

func TestWebSocketJoinSendsWelcomeSystemMessage(t *testing.T) {
	authorizer := fakeWSAuthorizer{
		userID:             "user-1",
		participantAllowed: true,
	}
	conn := dialWSWithHandler(t, NewHandlerWithAuthorizer(authorizer), "/ws", "fs_token=token-1")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.join",
		"request_id": "req-join-1",
		"payload": map[string]any{
			"campaign_id": "camp-1",
		},
	})

	joined := readFrame(t, conn)
	if joined.Type != "chat.joined" {
		t.Fatalf("frame type = %q, want %q", joined.Type, "chat.joined")
	}

	systemMessage := readFrame(t, conn)
	if systemMessage.Type != "chat.message" {
		t.Fatalf("frame type = %q, want %q", systemMessage.Type, "chat.message")
	}
	if !strings.Contains(string(systemMessage.Payload), "Welcome") {
		t.Fatalf("message payload = %s, expected Welcome text", string(systemMessage.Payload))
	}
}

func TestWebSocketJoinWithWelcomeProviderSkipsParticipantCheck(t *testing.T) {
	authorizer := &fakeWSWelcomeAuthorizer{
		userID:             "user-1",
		participantAllowed: false,
		resolveWelcome: joinWelcome{
			ParticipantName: "Ari",
			CampaignName:    "Camp One",
			SessionName:     "Session One",
			Locale:          commonv1.Locale_LOCALE_EN_US,
		},
	}
	conn := dialWSWithHandler(t, NewHandlerWithAuthorizer(authorizer), "/ws", "fs_token=token-1")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.join",
		"request_id": "req-join-1",
		"payload": map[string]any{
			"campaign_id": "camp-1",
		},
	})

	joined := readFrame(t, conn)
	if joined.Type != "chat.joined" {
		t.Fatalf("frame type = %q, want %q", joined.Type, "chat.joined")
	}
	welcome := readFrame(t, conn)
	if welcome.Type != "chat.message" {
		t.Fatalf("frame type = %q, want %q", welcome.Type, "chat.message")
	}
	if authorizer.participantCalls != 0 {
		t.Fatalf("participant checks = %d, want 0", authorizer.participantCalls)
	}
}

func TestWebSocketJoinIncludesCommunicationControlState(t *testing.T) {
	authorizer := &fakeWSCommunicationAuthorizer{
		tokenToUser: map[string]string{"token-1": "user-1"},
		contextByUserID: map[string]communicationContext{
			"user-1": {
				Welcome: joinWelcome{
					ParticipantName: "Ari",
					CampaignName:    "Camp One",
					SessionID:       "sess-1",
					SessionName:     "Session One",
					Locale:          commonv1.Locale_LOCALE_EN_US,
				},
				ParticipantID:    "part-1",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-1",
				ActiveSessionGate: &chatSessionGate{
					GateID:   "gate-1",
					Status:   "open",
					GateType: "choice",
				},
				ActiveSessionSpotlight: &chatSessionSpotlight{
					Type:        "character",
					CharacterID: "char-1",
				},
			},
		},
	}
	conn := dialWSWithHandler(t, NewHandlerWithAuthorizer(authorizer), "/ws", "fs_token=token-1")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.join",
		"request_id": "req-join-1",
		"payload": map[string]any{
			"campaign_id": "camp-1",
		},
	})

	joined := readFrame(t, conn)
	if joined.Type != "chat.joined" {
		t.Fatalf("frame type = %q, want %q", joined.Type, "chat.joined")
	}

	var payload wsTestJoinedPayload
	if err := json.Unmarshal(joined.Payload, &payload); err != nil {
		t.Fatalf("unmarshal joined payload: %v", err)
	}
	if payload.ActiveSessionGate.GateID != "gate-1" || payload.ActiveSessionGate.Status != "open" {
		t.Fatalf("unexpected active gate payload: %+v", payload.ActiveSessionGate)
	}
	if payload.ActiveSessionSpotlight.Type != "character" || payload.ActiveSessionSpotlight.CharacterID != "char-1" {
		t.Fatalf("unexpected active spotlight payload: %+v", payload.ActiveSessionSpotlight)
	}
}

func TestWebSocketControlGMHandoffBroadcastsState(t *testing.T) {
	authorizer := &fakeWSCommunicationAuthorizer{
		tokenToUser: map[string]string{
			"token-a": "user-a",
			"token-b": "user-b",
		},
		contextByUserID: map[string]communicationContext{
			"user-a": {
				Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", SessionID: "sess-1", SessionName: "Session One", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-a",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-a",
				Streams: []chatStream{
					{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", SessionID: "sess-1", Label: "System"},
					{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", SessionID: "sess-1", Label: "Table"},
					{StreamID: "campaign:camp-1:control", Kind: "control", Scope: "session", SessionID: "sess-1", Label: "Control"},
				},
				Personas: []chatPersona{
					{PersonaID: "participant:part-a", Kind: "participant", ParticipantID: "part-a", DisplayName: "A"},
				},
			},
			"user-b": {
				Welcome:          joinWelcome{ParticipantName: "B", CampaignName: "camp-1", SessionID: "sess-1", SessionName: "Session One", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-b",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-b",
				Streams: []chatStream{
					{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", SessionID: "sess-1", Label: "System"},
					{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", SessionID: "sess-1", Label: "Table"},
					{StreamID: "campaign:camp-1:control", Kind: "control", Scope: "session", SessionID: "sess-1", Label: "Control"},
				},
				Personas: []chatPersona{
					{PersonaID: "participant:part-b", Kind: "participant", ParticipantID: "part-b", DisplayName: "B"},
				},
			},
		},
		requestGMHandoffContext: communicationContext{
			Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", SessionID: "sess-1", SessionName: "Session One", Locale: commonv1.Locale_LOCALE_EN_US},
			ParticipantID:    "part-a",
			DefaultStreamID:  "campaign:camp-1:table",
			DefaultPersonaID: "participant:part-a",
			ActiveSessionGate: &chatSessionGate{
				GateID:   "gate-1",
				GateType: "gm_handoff",
				Status:   "open",
			},
		},
	}
	srv := httptest.NewServer(NewHandlerWithAuthorizer(authorizer))
	t.Cleanup(srv.Close)

	connA := dialWSWithExistingServer(t, srv, "/ws", "fs_token=token-a")
	connB := dialWSWithExistingServer(t, srv, "/ws", "fs_token=token-b")
	joinCampaign(t, connA, "camp-1")
	joinCampaign(t, connB, "camp-1")

	writeFrame(t, connA, map[string]any{
		"type":       "chat.control",
		"request_id": "req-control-1",
		"payload": map[string]any{
			"action": "gm_handoff.request",
			"reason": "party ready",
		},
	})

	ack := readFrame(t, connA)
	if ack.Type != "chat.ack" {
		t.Fatalf("ack frame type = %q, want chat.ack", ack.Type)
	}
	stateA := readFrame(t, connA)
	if stateA.Type != "chat.state" {
		t.Fatalf("sender frame type = %q, want chat.state", stateA.Type)
	}
	stateB := readFrame(t, connB)
	if stateB.Type != "chat.state" {
		t.Fatalf("subscriber frame type = %q, want chat.state", stateB.Type)
	}

	var senderState wsTestStatePayload
	if err := json.Unmarshal(stateA.Payload, &senderState); err != nil {
		t.Fatalf("decode sender state: %v", err)
	}
	if senderState.ActiveSessionGate.GateID != "gate-1" || senderState.ActiveSessionGate.GateType != "gm_handoff" {
		t.Fatalf("unexpected sender state payload: %+v", senderState.ActiveSessionGate)
	}
	if authorizer.lastControlParticipantID != "part-a" {
		t.Fatalf("control participant id = %q, want %q", authorizer.lastControlParticipantID, "part-a")
	}
	if authorizer.lastControlAction != "gm_handoff.request" {
		t.Fatalf("control action = %q, want %q", authorizer.lastControlAction, "gm_handoff.request")
	}
}

func TestWebSocketControlGateOpenBroadcastsState(t *testing.T) {
	authorizer := &fakeWSCommunicationAuthorizer{
		tokenToUser: map[string]string{
			"token-a": "user-a",
			"token-b": "user-b",
		},
		contextByUserID: map[string]communicationContext{
			"user-a": {
				Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", SessionID: "sess-1", SessionName: "Session One", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-a",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-a",
			},
			"user-b": {
				Welcome:          joinWelcome{ParticipantName: "B", CampaignName: "camp-1", SessionID: "sess-1", SessionName: "Session One", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-b",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-b",
			},
		},
		openGateContext: communicationContext{
			Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", SessionID: "sess-1", SessionName: "Session One", Locale: commonv1.Locale_LOCALE_EN_US},
			ParticipantID:    "part-a",
			DefaultStreamID:  "campaign:camp-1:table",
			DefaultPersonaID: "participant:part-a",
			ActiveSessionGate: &chatSessionGate{
				GateID:   "gate-choice-1",
				GateType: "choice",
				Status:   "open",
			},
		},
	}
	srv := httptest.NewServer(NewHandlerWithAuthorizer(authorizer))
	t.Cleanup(srv.Close)

	connA := dialWSWithExistingServer(t, srv, "/ws", "fs_token=token-a")
	connB := dialWSWithExistingServer(t, srv, "/ws", "fs_token=token-b")
	joinCampaign(t, connA, "camp-1")
	joinCampaign(t, connB, "camp-1")

	writeFrame(t, connA, map[string]any{
		"type":       "chat.control",
		"request_id": "req-control-open-gate",
		"payload": map[string]any{
			"action":    "gate.open",
			"gate_type": "choice",
			"reason":    "choose a route",
		},
	})

	ack := readFrame(t, connA)
	if ack.Type != "chat.ack" {
		t.Fatalf("ack frame type = %q, want chat.ack", ack.Type)
	}
	stateA := readFrame(t, connA)
	if stateA.Type != "chat.state" {
		t.Fatalf("sender frame type = %q, want chat.state", stateA.Type)
	}
	stateB := readFrame(t, connB)
	if stateB.Type != "chat.state" {
		t.Fatalf("subscriber frame type = %q, want chat.state", stateB.Type)
	}

	var senderState wsTestStatePayload
	if err := json.Unmarshal(stateA.Payload, &senderState); err != nil {
		t.Fatalf("decode sender state: %v", err)
	}
	if senderState.ActiveSessionGate.GateID != "gate-choice-1" || senderState.ActiveSessionGate.GateType != "choice" {
		t.Fatalf("unexpected sender state payload: %+v", senderState.ActiveSessionGate)
	}
	if authorizer.lastControlParticipantID != "part-a" {
		t.Fatalf("control participant id = %q, want %q", authorizer.lastControlParticipantID, "part-a")
	}
	if authorizer.lastControlAction != "gate.open" {
		t.Fatalf("control action = %q, want %q", authorizer.lastControlAction, "gate.open")
	}
	if authorizer.lastControlGateType != "choice" {
		t.Fatalf("control gate type = %q, want %q", authorizer.lastControlGateType, "choice")
	}
}

func TestWebSocketControlGateRespondBroadcastsState(t *testing.T) {
	authorizer := &fakeWSCommunicationAuthorizer{
		tokenToUser: map[string]string{
			"token-a": "user-a",
			"token-b": "user-b",
		},
		contextByUserID: map[string]communicationContext{
			"user-a": {
				Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", SessionID: "sess-1", SessionName: "Session One", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-a",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-a",
			},
			"user-b": {
				Welcome:          joinWelcome{ParticipantName: "B", CampaignName: "camp-1", SessionID: "sess-1", SessionName: "Session One", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-b",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-b",
			},
		},
		respondGateContext: communicationContext{
			Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", SessionID: "sess-1", SessionName: "Session One", Locale: commonv1.Locale_LOCALE_EN_US},
			ParticipantID:    "part-a",
			DefaultStreamID:  "campaign:camp-1:table",
			DefaultPersonaID: "participant:part-a",
			ActiveSessionGate: &chatSessionGate{
				GateID:   "gate-ready-1",
				GateType: "ready_check",
				Status:   "open",
				Progress: map[string]any{
					"eligible_count":  float64(2),
					"responded_count": float64(1),
					"pending_count":   float64(1),
					"all_responded":   false,
				},
			},
		},
	}
	srv := httptest.NewServer(NewHandlerWithAuthorizer(authorizer))
	t.Cleanup(srv.Close)

	connA := dialWSWithExistingServer(t, srv, "/ws", "fs_token=token-a")
	connB := dialWSWithExistingServer(t, srv, "/ws", "fs_token=token-b")
	joinCampaign(t, connA, "camp-1")
	joinCampaign(t, connB, "camp-1")

	writeFrame(t, connA, map[string]any{
		"type":       "chat.control",
		"request_id": "req-control-respond-gate",
		"payload": map[string]any{
			"action":   "gate.respond",
			"decision": "ready",
			"response": map[string]any{"note": "ready to proceed"},
		},
	})

	ack := readFrame(t, connA)
	if ack.Type != "chat.ack" {
		t.Fatalf("ack frame type = %q, want chat.ack", ack.Type)
	}
	stateA := readFrame(t, connA)
	if stateA.Type != "chat.state" {
		t.Fatalf("sender frame type = %q, want chat.state", stateA.Type)
	}
	stateB := readFrame(t, connB)
	if stateB.Type != "chat.state" {
		t.Fatalf("subscriber frame type = %q, want chat.state", stateB.Type)
	}

	var senderState wsTestStatePayload
	if err := json.Unmarshal(stateA.Payload, &senderState); err != nil {
		t.Fatalf("decode sender state: %v", err)
	}
	if senderState.ActiveSessionGate.GateID != "gate-ready-1" || senderState.ActiveSessionGate.GateType != "ready_check" {
		t.Fatalf("unexpected sender state payload: %+v", senderState.ActiveSessionGate)
	}
	if got := senderState.ActiveSessionGate.Progress["responded_count"]; got != float64(1) {
		t.Fatalf("progress responded_count = %v, want 1", got)
	}
	if authorizer.lastControlParticipantID != "part-a" {
		t.Fatalf("control participant id = %q, want %q", authorizer.lastControlParticipantID, "part-a")
	}
	if authorizer.lastControlAction != "gate.respond" {
		t.Fatalf("control action = %q, want %q", authorizer.lastControlAction, "gate.respond")
	}
	if authorizer.lastControlGateType != "ready" {
		t.Fatalf("control decision = %q, want %q", authorizer.lastControlGateType, "ready")
	}
}

func TestRefreshRoomCommunicationContextBroadcastsContextAndState(t *testing.T) {
	roomHub := newRoomHub()
	authorizer := &fakeWSCommunicationAuthorizer{
		tokenToUser: map[string]string{
			"token-a": "user-a",
			"token-b": "user-b",
		},
		contextByUserID: map[string]communicationContext{
			"user-a": {
				Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", SessionID: "sess-1", SessionName: "Session One", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-a",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-a",
				Streams: []chatStream{
					{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", SessionID: "sess-1", Label: "System"},
					{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", SessionID: "sess-1", Label: "Table"},
					{StreamID: "campaign:camp-1:control", Kind: "control", Scope: "session", SessionID: "sess-1", Label: "Control"},
				},
				Personas: []chatPersona{
					{PersonaID: "participant:part-a", Kind: "participant", ParticipantID: "part-a", DisplayName: "A"},
				},
			},
			"user-b": {
				Welcome:          joinWelcome{ParticipantName: "B", CampaignName: "camp-1", SessionID: "sess-1", SessionName: "Session One", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-b",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-b",
				Streams: []chatStream{
					{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", SessionID: "sess-1", Label: "System"},
					{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", SessionID: "sess-1", Label: "Table"},
					{StreamID: "campaign:camp-1:control", Kind: "control", Scope: "session", SessionID: "sess-1", Label: "Control"},
				},
				Personas: []chatPersona{
					{PersonaID: "participant:part-b", Kind: "participant", ParticipantID: "part-b", DisplayName: "B"},
				},
			},
		},
	}
	handler := newHandler(authorizer, true, roomHub, nil, nil, nil, nil, nil, nil)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	connA := dialWSWithExistingServer(t, srv, "/ws", "fs_token=token-a")
	connB := dialWSWithExistingServer(t, srv, "/ws", "fs_token=token-b")
	joinCampaign(t, connA, "camp-1")
	joinCampaign(t, connB, "camp-1")

	authorizer.contextByUserID["user-a"] = communicationContext{
		Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", SessionID: "sess-2", SessionName: "Session Two", Locale: commonv1.Locale_LOCALE_EN_US},
		ParticipantID:    "part-a",
		DefaultStreamID:  "campaign:camp-1:table",
		DefaultPersonaID: "participant:part-a",
		ActiveSessionGate: &chatSessionGate{
			GateID:   "gate-2",
			GateType: "ready_check",
			Status:   "open",
			Progress: map[string]any{
				"responded_count": float64(1),
			},
		},
		ActiveSessionSpotlight: &chatSessionSpotlight{
			Type:        "character",
			CharacterID: "char-7",
		},
		Streams: []chatStream{
			{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", SessionID: "sess-2", Label: "System"},
			{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", SessionID: "sess-2", Label: "Table"},
			{StreamID: "campaign:camp-1:control", Kind: "control", Scope: "session", SessionID: "sess-2", Label: "Control"},
		},
		Personas: []chatPersona{
			{PersonaID: "participant:part-a", Kind: "participant", ParticipantID: "part-a", DisplayName: "A"},
		},
	}
	authorizer.contextByUserID["user-b"] = communicationContext{
		Welcome:          joinWelcome{ParticipantName: "B", CampaignName: "camp-1", SessionID: "sess-2", SessionName: "Session Two", Locale: commonv1.Locale_LOCALE_EN_US},
		ParticipantID:    "part-b",
		DefaultStreamID:  "campaign:camp-1:table",
		DefaultPersonaID: "participant:part-b",
		ActiveSessionGate: &chatSessionGate{
			GateID:   "gate-2",
			GateType: "ready_check",
			Status:   "open",
			Progress: map[string]any{
				"responded_count": float64(1),
			},
		},
		ActiveSessionSpotlight: &chatSessionSpotlight{
			Type:        "character",
			CharacterID: "char-7",
		},
		Streams: []chatStream{
			{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", SessionID: "sess-2", Label: "System"},
			{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", SessionID: "sess-2", Label: "Table"},
			{StreamID: "campaign:camp-1:control", Kind: "control", Scope: "session", SessionID: "sess-2", Label: "Control"},
		},
		Personas: []chatPersona{
			{PersonaID: "participant:part-b", Kind: "participant", ParticipantID: "part-b", DisplayName: "B"},
			{PersonaID: "character:char-7", Kind: "character", ParticipantID: "part-b", CharacterID: "char-7", DisplayName: "Vera"},
		},
	}

	room := roomHub.roomIfExists("camp-1")
	if room == nil {
		t.Fatal("expected room for campaign")
	}
	if err := refreshRoomCommunicationContext(context.Background(), authorizer, room, nil, nil); err != nil {
		t.Fatalf("refresh room communication context: %v", err)
	}

	contextA := readFrame(t, connA)
	contextB := readFrame(t, connB)
	if contextA.Type != "chat.context" {
		t.Fatalf("connA frame type = %q, want chat.context", contextA.Type)
	}
	if contextB.Type != "chat.context" {
		t.Fatalf("connB frame type = %q, want chat.context", contextB.Type)
	}

	stateA := readFrame(t, connA)
	stateB := readFrame(t, connB)
	if stateA.Type != "chat.state" {
		t.Fatalf("connA frame type = %q, want chat.state", stateA.Type)
	}
	if stateB.Type != "chat.state" {
		t.Fatalf("connB frame type = %q, want chat.state", stateB.Type)
	}

	var payloadA wsTestJoinedPayload
	if err := json.Unmarshal(contextA.Payload, &payloadA); err != nil {
		t.Fatalf("decode connA context: %v", err)
	}
	if payloadA.SessionID != "sess-2" || payloadA.ActiveSessionGate.GateID != "gate-2" {
		t.Fatalf("unexpected connA context payload: %+v", payloadA)
	}

	var payloadB wsTestJoinedPayload
	if err := json.Unmarshal(contextB.Payload, &payloadB); err != nil {
		t.Fatalf("decode connB context: %v", err)
	}
	if len(payloadB.Personas) != 2 || payloadB.Personas[1].PersonaID != "character:char-7" {
		t.Fatalf("unexpected connB personas: %+v", payloadB.Personas)
	}

	var statePayload wsTestStatePayload
	if err := json.Unmarshal(stateA.Payload, &statePayload); err != nil {
		t.Fatalf("decode state payload: %v", err)
	}
	if statePayload.SessionID != "sess-2" || statePayload.ActiveSessionGate.GateID != "gate-2" {
		t.Fatalf("unexpected state payload: %+v", statePayload)
	}
}

func TestRefreshRoomCommunicationContextClearsSessionIDWhenSessionEnds(t *testing.T) {
	roomHub := newRoomHub()
	authorizer := &fakeWSCommunicationAuthorizer{
		tokenToUser: map[string]string{
			"token-a": "user-a",
		},
		contextByUserID: map[string]communicationContext{
			"user-a": {
				Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", SessionID: "sess-1", SessionName: "Session One", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-a",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-a",
				Streams: []chatStream{
					{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", SessionID: "sess-1", Label: "System"},
					{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", SessionID: "sess-1", Label: "Table"},
				},
				Personas: []chatPersona{
					{PersonaID: "participant:part-a", Kind: "participant", ParticipantID: "part-a", DisplayName: "A"},
				},
			},
		},
	}
	handler := newHandler(authorizer, true, roomHub, nil, nil, nil, nil, nil, nil)
	conn := dialWSWithHandler(t, handler, "/ws", "fs_token=token-a")
	joinCampaign(t, conn, "camp-1")

	authorizer.contextByUserID["user-a"] = communicationContext{
		Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", Locale: commonv1.Locale_LOCALE_EN_US},
		ParticipantID:    "part-a",
		DefaultStreamID:  "campaign:camp-1:table",
		DefaultPersonaID: "participant:part-a",
		Streams: []chatStream{
			{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "campaign", Label: "System"},
			{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "campaign", Label: "Table"},
		},
		Personas: []chatPersona{
			{PersonaID: "participant:part-a", Kind: "participant", ParticipantID: "part-a", DisplayName: "A"},
		},
	}

	room := roomHub.roomIfExists("camp-1")
	if room == nil {
		t.Fatal("expected room for campaign")
	}
	if err := refreshRoomCommunicationContext(context.Background(), authorizer, room, nil, nil); err != nil {
		t.Fatalf("refresh room communication context: %v", err)
	}

	contextFrame := readFrame(t, conn)
	if contextFrame.Type != "chat.context" {
		t.Fatalf("frame type = %q, want chat.context", contextFrame.Type)
	}
	stateFrame := readFrame(t, conn)
	if stateFrame.Type != "chat.state" {
		t.Fatalf("frame type = %q, want chat.state", stateFrame.Type)
	}

	var payload wsTestJoinedPayload
	if err := json.Unmarshal(contextFrame.Payload, &payload); err != nil {
		t.Fatalf("decode context payload: %v", err)
	}
	if payload.SessionID != "" {
		t.Fatalf("context session_id = %q, want empty", payload.SessionID)
	}

	var statePayload wsTestStatePayload
	if err := json.Unmarshal(stateFrame.Payload, &statePayload); err != nil {
		t.Fatalf("decode state payload: %v", err)
	}
	if statePayload.SessionID != "" {
		t.Fatalf("state session_id = %q, want empty", statePayload.SessionID)
	}
}

func TestRefreshRoomCommunicationContextUpdatesStreamAccess(t *testing.T) {
	roomHub := newRoomHub()
	authorizer := &fakeWSCommunicationAuthorizer{
		tokenToUser: map[string]string{
			"token-a": "user-a",
			"token-b": "user-b",
		},
		contextByUserID: map[string]communicationContext{
			"user-a": {
				Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-a",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-a",
				Streams: []chatStream{
					{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", Label: "System"},
					{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", Label: "Table"},
				},
				Personas: []chatPersona{
					{PersonaID: "participant:part-a", Kind: "participant", ParticipantID: "part-a", DisplayName: "A"},
				},
			},
			"user-b": {
				Welcome:          joinWelcome{ParticipantName: "B", CampaignName: "camp-1", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-b",
				DefaultStreamID:  "campaign:camp-1:system",
				DefaultPersonaID: "participant:part-b",
				Streams: []chatStream{
					{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", Label: "System"},
				},
				Personas: []chatPersona{
					{PersonaID: "participant:part-b", Kind: "participant", ParticipantID: "part-b", DisplayName: "B"},
				},
			},
		},
	}
	handler := newHandler(authorizer, true, roomHub, nil, nil, nil, nil, nil, nil)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	connA := dialWSWithExistingServer(t, srv, "/ws", "fs_token=token-a")
	connB := dialWSWithExistingServer(t, srv, "/ws", "fs_token=token-b")
	joinCampaign(t, connA, "camp-1")
	joinCampaign(t, connB, "camp-1")

	authorizer.contextByUserID["user-b"] = communicationContext{
		Welcome:          joinWelcome{ParticipantName: "B", CampaignName: "camp-1", Locale: commonv1.Locale_LOCALE_EN_US},
		ParticipantID:    "part-b",
		DefaultStreamID:  "campaign:camp-1:table",
		DefaultPersonaID: "participant:part-b",
		Streams: []chatStream{
			{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", Label: "System"},
			{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", Label: "Table"},
		},
		Personas: []chatPersona{
			{PersonaID: "participant:part-b", Kind: "participant", ParticipantID: "part-b", DisplayName: "B"},
			{PersonaID: "character:char-9", Kind: "character", ParticipantID: "part-b", CharacterID: "char-9", DisplayName: "Mira"},
		},
	}

	room := roomHub.roomIfExists("camp-1")
	if room == nil {
		t.Fatal("expected room for campaign")
	}
	if err := refreshRoomCommunicationContext(context.Background(), authorizer, room, nil, nil); err != nil {
		t.Fatalf("refresh room communication context: %v", err)
	}

	_ = readFrame(t, connA)
	contextB := readFrame(t, connB)
	if contextB.Type != "chat.context" {
		t.Fatalf("connB frame type = %q, want chat.context", contextB.Type)
	}
	var payloadB wsTestJoinedPayload
	if err := json.Unmarshal(contextB.Payload, &payloadB); err != nil {
		t.Fatalf("decode connB context: %v", err)
	}
	if payloadB.DefaultStreamID != "campaign:camp-1:table" || len(payloadB.Streams) != 2 {
		t.Fatalf("unexpected connB stream refresh: %+v", payloadB)
	}

	writeFrame(t, connA, map[string]any{
		"type":       "chat.send",
		"request_id": "req-send-after-refresh",
		"payload": map[string]any{
			"client_message_id": "cli-refresh-1",
			"stream_id":         "campaign:camp-1:table",
			"body":              "table after refresh",
		},
	})

	ack := readFrame(t, connA)
	if ack.Type != "chat.ack" {
		t.Fatalf("sender frame type = %q, want chat.ack", ack.Type)
	}
	_ = readFrame(t, connA)

	receiverMessage := readFrame(t, connB)
	if receiverMessage.Type != "chat.message" {
		t.Fatalf("receiver frame type = %q, want chat.message", receiverMessage.Type)
	}
	payload := decodeMessagePayload(t, receiverMessage.Payload)
	if payload.Message.Body != "table after refresh" {
		t.Fatalf("receiver message body = %q, want %q", payload.Message.Body, "table after refresh")
	}
}

func TestRefreshRoomCommunicationContextEvictsPeerOnPermissionLoss(t *testing.T) {
	roomHub := newRoomHub()
	authorizer := &fakeWSCommunicationAuthorizer{
		tokenToUser: map[string]string{
			"token-a": "user-a",
			"token-b": "user-b",
		},
		contextByUserID: map[string]communicationContext{
			"user-a": {
				Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-a",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-a",
				Streams: []chatStream{
					{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", Label: "System"},
					{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", Label: "Table"},
				},
				Personas: []chatPersona{
					{PersonaID: "participant:part-a", Kind: "participant", ParticipantID: "part-a", DisplayName: "A"},
				},
			},
			"user-b": {
				Welcome:          joinWelcome{ParticipantName: "B", CampaignName: "camp-1", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-b",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-b",
				Streams: []chatStream{
					{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", Label: "System"},
					{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", Label: "Table"},
				},
				Personas: []chatPersona{
					{PersonaID: "participant:part-b", Kind: "participant", ParticipantID: "part-b", DisplayName: "B"},
				},
			},
		},
	}
	handler := newHandler(authorizer, true, roomHub, nil, nil, nil, nil, nil, nil)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	connA := dialWSWithExistingServer(t, srv, "/ws", "fs_token=token-a")
	connB := dialWSWithExistingServer(t, srv, "/ws", "fs_token=token-b")
	joinCampaign(t, connA, "camp-1")
	joinCampaign(t, connB, "camp-1")

	delete(authorizer.contextByUserID, "user-b")

	room := roomHub.roomIfExists("camp-1")
	if room == nil {
		t.Fatal("expected room for campaign")
	}
	if err := refreshRoomCommunicationContext(context.Background(), authorizer, room, nil, nil); err != nil {
		t.Fatalf("refresh room communication context: %v", err)
	}

	contextA := readFrame(t, connA)
	if contextA.Type != "chat.context" {
		t.Fatalf("connA frame type = %q, want chat.context", contextA.Type)
	}
	eviction := readFrame(t, connB)
	if eviction.Type != "chat.error" {
		t.Fatalf("connB frame type = %q, want chat.error", eviction.Type)
	}
	if !strings.Contains(string(eviction.Payload), "FORBIDDEN") {
		t.Fatalf("eviction payload = %s, want FORBIDDEN", string(eviction.Payload))
	}

	writeFrame(t, connB, map[string]any{
		"type":       "chat.send",
		"request_id": "req-send-after-evict",
		"payload": map[string]any{
			"client_message_id": "cli-evicted-1",
			"body":              "should fail",
		},
	})

	postEviction := readFrame(t, connB)
	if postEviction.Type != "chat.error" {
		t.Fatalf("post-eviction frame type = %q, want chat.error", postEviction.Type)
	}
	if !strings.Contains(string(postEviction.Payload), "FORBIDDEN") {
		t.Fatalf("post-eviction payload = %s, want FORBIDDEN", string(postEviction.Payload))
	}
}

func TestLeaveCampaignRoomKeepsNewRoomAssignment(t *testing.T) {
	session := newWSSession("user-1", &wsPeer{})
	roomA := newCampaignRoom("camp-a")
	roomB := newCampaignRoom("camp-b")

	roomA.join(session, []string{chatDefaultStreamID("camp-a")})
	session.setRoom(roomA)
	roomB.join(session, []string{chatDefaultStreamID("camp-b")})
	session.setRoom(roomB)

	leaveCampaignRoom(roomA, session, nil, nil)

	if session.currentRoom() != roomB {
		t.Fatal("expected session to keep current room assignment")
	}
}

func TestWebSocketControlPropagatesRPCError(t *testing.T) {
	authorizer := &fakeWSCommunicationAuthorizer{
		tokenToUser: map[string]string{"token-a": "user-a"},
		contextByUserID: map[string]communicationContext{
			"user-a": {
				Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", SessionID: "sess-1", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-a",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-a",
			},
		},
		requestGMHandoffErr: status.Error(codes.FailedPrecondition, "another session gate is already open"),
	}
	conn := dialWSWithHandler(t, NewHandlerWithAuthorizer(authorizer), "/ws", "fs_token=token-a")
	joinCampaign(t, conn, "camp-1")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.control",
		"request_id": "req-control-error",
		"payload": map[string]any{
			"action": "gm_handoff.request",
		},
	})

	got := readFrame(t, conn)
	if got.Type != "chat.error" {
		t.Fatalf("frame type = %q, want chat.error", got.Type)
	}
	if !strings.Contains(string(got.Payload), "FAILED_PRECONDITION") {
		t.Fatalf("error payload = %s, expected FAILED_PRECONDITION", string(got.Payload))
	}
}

func TestWebSocketSendDoesNotSubmitAITurnImmediately(t *testing.T) {
	authorizer := &fakeWSCommunicationAuthorizer{
		tokenToUser: map[string]string{"token-a": "user-a"},
		contextByUserID: map[string]communicationContext{
			"user-a": {
				Welcome: joinWelcome{
					ParticipantName: "Ari",
					CampaignName:    "Camp One",
					SessionID:       "sess-1",
					SessionName:     "Session One",
					GmMode:          "AI",
					AIAgentID:       "agent-1",
					Locale:          commonv1.Locale_LOCALE_EN_US,
				},
				ParticipantID:    "part-a",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-a",
				Streams: []chatStream{
					{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", SessionID: "sess-1", Label: "System"},
					{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", SessionID: "sess-1", Label: "Table"},
					{StreamID: "campaign:camp-1:control", Kind: "control", Scope: "session", SessionID: "sess-1", Label: "Control"},
				},
				Personas: []chatPersona{
					{PersonaID: "participant:part-a", Kind: "participant", ParticipantID: "part-a", DisplayName: "Ari"},
				},
			},
		},
	}
	invocationClient := &testInvocationClient{
		submitFn: func(context.Context, *aiv1.SubmitCampaignTurnRequest) (*aiv1.SubmitCampaignTurnResponse, error) {
			return &aiv1.SubmitCampaignTurnResponse{TurnId: "turn-1"}, nil
		},
	}
	handler := newHandler(
		authorizer,
		true,
		nil,
		nil,
		nil,
		nil,
		nil,
		func(_ context.Context, room *campaignRoom, _ string) error {
			room.setAISessionGrant("grant-token", 1, time.Now().UTC().Add(time.Minute))
			return nil
		},
		invocationClient,
	)
	conn := dialWSWithHandler(t, handler, "/ws", "fs_token=token-a")
	joinCampaign(t, conn, "camp-1")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.send",
		"request_id": "req-send-ai-buffer",
		"payload": map[string]any{
			"client_message_id": "cli-ai-buffer-1",
			"body":              "we inspect the chamber first",
		},
	})

	ack := readFrame(t, conn)
	if ack.Type != "chat.ack" {
		t.Fatalf("ack frame type = %q, want chat.ack", ack.Type)
	}
	msg := readFrame(t, conn)
	if msg.Type != "chat.message" {
		t.Fatalf("message frame type = %q, want chat.message", msg.Type)
	}
	if invocationClient.submitCalls != 0 {
		t.Fatalf("submit calls = %d, want 0", invocationClient.submitCalls)
	}
}

func TestWebSocketControlGMHandoffRequestSubmitsBufferedTranscriptToAI(t *testing.T) {
	authorizer := &fakeWSCommunicationAuthorizer{
		tokenToUser: map[string]string{"token-a": "user-a"},
		contextByUserID: map[string]communicationContext{
			"user-a": {
				Welcome: joinWelcome{
					ParticipantName: "Ari",
					CampaignName:    "Camp One",
					SessionID:       "sess-1",
					SessionName:     "Session One",
					GmMode:          "AI",
					AIAgentID:       "agent-1",
					Locale:          commonv1.Locale_LOCALE_EN_US,
				},
				ParticipantID:    "part-a",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-a",
				Streams: []chatStream{
					{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", SessionID: "sess-1", Label: "System"},
					{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", SessionID: "sess-1", Label: "Table"},
					{StreamID: "scene:scene-1:character", Kind: "character", Scope: "scene", SessionID: "sess-1", SceneID: "scene-1", Label: "Ruined Hall"},
					{StreamID: "campaign:camp-1:control", Kind: "control", Scope: "session", SessionID: "sess-1", Label: "Control"},
				},
				Personas: []chatPersona{
					{PersonaID: "participant:part-a", Kind: "participant", ParticipantID: "part-a", DisplayName: "Ari"},
					{PersonaID: "character:char-1", Kind: "character", ParticipantID: "part-a", CharacterID: "char-1", DisplayName: "Vera"},
				},
			},
		},
		requestGMHandoffContext: communicationContext{
			Welcome: joinWelcome{
				ParticipantName: "Ari",
				CampaignName:    "Camp One",
				SessionID:       "sess-1",
				SessionName:     "Session One",
				GmMode:          "AI",
				AIAgentID:       "agent-1",
				Locale:          commonv1.Locale_LOCALE_EN_US,
			},
			ParticipantID:    "part-a",
			DefaultStreamID:  "campaign:camp-1:table",
			DefaultPersonaID: "participant:part-a",
			ActiveSessionGate: &chatSessionGate{
				GateID:   "gate-1",
				GateType: "gm_handoff",
				Status:   "open",
			},
		},
	}
	invocationClient := &testInvocationClient{
		submitFn: func(context.Context, *aiv1.SubmitCampaignTurnRequest) (*aiv1.SubmitCampaignTurnResponse, error) {
			return &aiv1.SubmitCampaignTurnResponse{TurnId: "turn-1"}, nil
		},
	}
	handler := newHandler(
		authorizer,
		true,
		nil,
		nil,
		nil,
		nil,
		nil,
		func(_ context.Context, room *campaignRoom, _ string) error {
			room.setAISessionGrant("grant-token", 1, time.Now().UTC().Add(time.Minute))
			return nil
		},
		invocationClient,
	)
	conn := dialWSWithHandler(t, handler, "/ws", "fs_token=token-a")
	joinCampaign(t, conn, "camp-1")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.send",
		"request_id": "req-send-ai-1",
		"payload": map[string]any{
			"client_message_id": "cli-ai-1",
			"body":              "we inspect the chamber first",
		},
	})
	sendAck := readFrame(t, conn)
	if sendAck.Type != "chat.ack" {
		t.Fatalf("send ack frame type = %q, want chat.ack", sendAck.Type)
	}
	firstMessage := readFrame(t, conn)
	firstPayload := decodeMessagePayload(t, firstMessage.Payload)

	writeFrame(t, conn, map[string]any{
		"type":       "chat.send",
		"request_id": "req-send-ai-2",
		"payload": map[string]any{
			"client_message_id": "cli-ai-2",
			"stream_id":         "scene:scene-1:character",
			"persona_id":        "character:char-1",
			"body":              "Vera checks the door for traps",
		},
	})
	sendAck = readFrame(t, conn)
	if sendAck.Type != "chat.ack" {
		t.Fatalf("second send ack frame type = %q, want chat.ack", sendAck.Type)
	}
	secondMessage := readFrame(t, conn)
	secondPayload := decodeMessagePayload(t, secondMessage.Payload)

	writeFrame(t, conn, map[string]any{
		"type":       "chat.control",
		"request_id": "req-control-ai",
		"payload": map[string]any{
			"action": "gm_handoff.request",
			"reason": "party is ready for a ruling",
		},
	})

	controlAck := readFrame(t, conn)
	if controlAck.Type != "chat.ack" {
		t.Fatalf("control ack frame type = %q, want chat.ack", controlAck.Type)
	}
	state := readFrame(t, conn)
	if state.Type != "chat.state" {
		t.Fatalf("state frame type = %q, want chat.state", state.Type)
	}
	if invocationClient.submitCalls != 1 {
		t.Fatalf("submit calls = %d, want 1", invocationClient.submitCalls)
	}
	req := invocationClient.submitReqs[0]
	if req.GetCampaignId() != "camp-1" || req.GetSessionId() != "sess-1" || req.GetAgentId() != "agent-1" {
		t.Fatalf("unexpected submit routing: %+v", req)
	}
	if req.GetParticipantId() != "part-a" {
		t.Fatalf("submit participant id = %q, want %q", req.GetParticipantId(), "part-a")
	}
	if req.GetSessionGrant() != "grant-token" {
		t.Fatalf("submit grant = %q, want %q", req.GetSessionGrant(), "grant-token")
	}
	if req.GetMessageId() != secondPayload.Message.MessageID {
		t.Fatalf("submit correlation message id = %q, want %q", req.GetMessageId(), secondPayload.Message.MessageID)
	}
	if !strings.Contains(req.GetBody(), firstPayload.Message.Body) {
		t.Fatalf("submit body = %q, expected first message content", req.GetBody())
	}
	if !strings.Contains(req.GetBody(), secondPayload.Message.Body) {
		t.Fatalf("submit body = %q, expected second message content", req.GetBody())
	}
	if !strings.Contains(req.GetBody(), "party is ready for a ruling") {
		t.Fatalf("submit body = %q, expected handoff reason", req.GetBody())
	}
}

func TestLocalizedJoinWelcomeBodyUsesCampaignLocale(t *testing.T) {
	body := localizedJoinWelcomeBody(joinWelcome{
		ParticipantName: "Ari",
		CampaignName:    "Campanha Um",
		SessionName:     "Sessao Um",
		Locale:          commonv1.Locale_LOCALE_PT_BR,
	})
	if !strings.Contains(body, "Bem-vindo") {
		t.Fatalf("body = %q, expected Portuguese welcome", body)
	}
}

func TestLocalizedJoinWelcomeBodyOmitsSessionWhenUnavailable(t *testing.T) {
	body := localizedJoinWelcomeBody(joinWelcome{
		ParticipantName: "Ari",
		CampaignName:    "Campaign One",
		Locale:          commonv1.Locale_LOCALE_EN_US,
	})
	if strings.Contains(body, "Session") {
		t.Fatalf("body = %q, expected no session text", body)
	}
	if !strings.Contains(body, "Campaign Campaign One") {
		t.Fatalf("body = %q, expected campaign text", body)
	}
}

func TestWebSocketDisconnectReleasesCampaignUpdateSubscriptionWhenRoomEmpty(t *testing.T) {
	released := make(chan string, 1)
	handler := newHandler(
		nil,
		false,
		nil,
		nil,
		func(campaignID string) {
			released <- campaignID
		},
		nil,
		nil,
		nil,
		nil,
	)

	conn := dialWSWithHandler(t, handler, "/ws", "")
	joinCampaign(t, conn, "camp-1")
	_ = conn.Close()

	select {
	case campaignID := <-released:
		if campaignID != "camp-1" {
			t.Fatalf("released campaign id = %q, want %q", campaignID, "camp-1")
		}
	case <-time.After(time.Second):
		t.Fatal("expected campaign update subscription release")
	}
}

func TestWebSocketDisconnectDoesNotReleaseUntilLastSubscriberLeaves(t *testing.T) {
	released := make(chan string, 2)
	handler := newHandler(
		nil,
		false,
		nil,
		nil,
		func(campaignID string) {
			released <- campaignID
		},
		nil,
		nil,
		nil,
		nil,
	)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	connA := dialWSWithExistingServer(t, srv, "/ws", "")
	connB := dialWSWithExistingServer(t, srv, "/ws", "")
	joinCampaign(t, connA, "camp-1")
	joinCampaign(t, connB, "camp-1")

	_ = connA.Close()
	select {
	case campaignID := <-released:
		t.Fatalf("unexpected release while room still active: %q", campaignID)
	case <-time.After(200 * time.Millisecond):
	}

	_ = connB.Close()
	select {
	case campaignID := <-released:
		if campaignID != "camp-1" {
			t.Fatalf("released campaign id = %q, want %q", campaignID, "camp-1")
		}
	case <-time.After(time.Second):
		t.Fatal("expected release after last subscriber leaves")
	}
}

func TestWebSocketSendOnlyBroadcastsToSubscribersWithStreamAccess(t *testing.T) {
	authorizer := &fakeWSCommunicationAuthorizer{
		tokenToUser: map[string]string{
			"token-a": "user-a",
			"token-b": "user-b",
		},
		contextByUserID: map[string]communicationContext{
			"user-a": {
				Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-a",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-a",
				Streams: []chatStream{
					{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", Label: "System"},
					{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", Label: "Table"},
				},
				Personas: []chatPersona{
					{PersonaID: "participant:part-a", Kind: "participant", ParticipantID: "part-a", DisplayName: "A"},
				},
			},
			"user-b": {
				Welcome:          joinWelcome{ParticipantName: "B", CampaignName: "camp-1", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-b",
				DefaultStreamID:  "campaign:camp-1:system",
				DefaultPersonaID: "participant:part-b",
				Streams: []chatStream{
					{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", Label: "System"},
				},
				Personas: []chatPersona{
					{PersonaID: "participant:part-b", Kind: "participant", ParticipantID: "part-b", DisplayName: "B"},
				},
			},
		},
	}
	srv := httptest.NewServer(NewHandlerWithAuthorizer(authorizer))
	t.Cleanup(srv.Close)

	connA := dialWSWithExistingServer(t, srv, "/ws", "fs_token=token-a")
	connB := dialWSWithExistingServer(t, srv, "/ws", "fs_token=token-b")
	joinCampaign(t, connA, "camp-1")
	joinCampaign(t, connB, "camp-1")

	writeFrame(t, connA, map[string]any{
		"type":       "chat.send",
		"request_id": "req-send-stream",
		"payload": map[string]any{
			"client_message_id": "cli-stream-1",
			"stream_id":         "campaign:camp-1:table",
			"body":              "table-only",
		},
	})

	ack := readFrame(t, connA)
	if ack.Type != "chat.ack" {
		t.Fatalf("sender frame type = %q, want chat.ack", ack.Type)
	}
	senderMessage := readFrame(t, connA)
	if senderMessage.Type != "chat.message" {
		t.Fatalf("sender frame type = %q, want chat.message", senderMessage.Type)
	}

	_ = connB.SetDeadline(time.Now().Add(250 * time.Millisecond))
	var got wsTestFrame
	err := json.NewDecoder(connB).Decode(&got)
	if err == nil {
		t.Fatalf("unexpected frame for subscriber without stream access: %+v", got)
	}
}

func TestWebSocketSendUsesRequestedPersonaWhenAvailable(t *testing.T) {
	authorizer := &fakeWSCommunicationAuthorizer{
		tokenToUser: map[string]string{
			"token-a": "user-a",
		},
		contextByUserID: map[string]communicationContext{
			"user-a": {
				Welcome:          joinWelcome{ParticipantName: "A", CampaignName: "camp-1", Locale: commonv1.Locale_LOCALE_EN_US},
				ParticipantID:    "part-a",
				DefaultStreamID:  "campaign:camp-1:table",
				DefaultPersonaID: "participant:part-a",
				Streams: []chatStream{
					{StreamID: "campaign:camp-1:system", Kind: "system", Scope: "session", Label: "System"},
					{StreamID: "campaign:camp-1:table", Kind: "table", Scope: "session", Label: "Table"},
				},
				Personas: []chatPersona{
					{PersonaID: "participant:part-a", Kind: "participant", ParticipantID: "part-a", DisplayName: "A"},
					{PersonaID: "character:char-1", Kind: "character", ParticipantID: "part-a", CharacterID: "char-1", DisplayName: "Vera"},
				},
			},
		},
	}
	conn := dialWSWithHandler(t, NewHandlerWithAuthorizer(authorizer), "/ws", "fs_token=token-a")
	joinCampaign(t, conn, "camp-1")

	writeFrame(t, conn, map[string]any{
		"type":       "chat.send",
		"request_id": "req-send-persona",
		"payload": map[string]any{
			"client_message_id": "cli-persona-1",
			"persona_id":        "character:char-1",
			"body":              "speaking in character",
		},
	})

	ack := readFrame(t, conn)
	if ack.Type != "chat.ack" {
		t.Fatalf("ack frame type = %q, want chat.ack", ack.Type)
	}
	messageFrame := readFrame(t, conn)
	payload := decodeMessagePayload(t, messageFrame.Payload)
	if payload.Message.Actor.PersonaID != "character:char-1" {
		t.Fatalf("persona id = %q, want %q", payload.Message.Actor.PersonaID, "character:char-1")
	}
	if payload.Message.Actor.CharacterID != "char-1" {
		t.Fatalf("character id = %q, want %q", payload.Message.Actor.CharacterID, "char-1")
	}
	if payload.Message.Actor.Mode != "character" {
		t.Fatalf("actor mode = %q, want character", payload.Message.Actor.Mode)
	}
	if payload.Message.Actor.Name != "Vera" {
		t.Fatalf("actor name = %q, want Vera", payload.Message.Actor.Name)
	}
}
