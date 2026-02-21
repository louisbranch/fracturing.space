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

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"golang.org/x/net/websocket"
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
		SequenceID int64  `json:"sequence_id"`
		Body       string `json:"body"`
	} `json:"message"`
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

func TestWebSocketJoinActiveSessionFailureReturnsFailedPrecondition(t *testing.T) {
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
	if !strings.Contains(string(got.Payload), "FAILED_PRECONDITION") {
		t.Fatalf("error payload = %s, expected FAILED_PRECONDITION", string(got.Payload))
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
	handler := newHandler(nil, false, nil, func(campaignID string) {
		released <- campaignID
	})

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
	handler := newHandler(nil, false, nil, func(campaignID string) {
		released <- campaignID
	})
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
