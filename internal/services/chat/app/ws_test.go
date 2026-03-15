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

	"golang.org/x/net/websocket"
)

type wsTestFrame struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type wsTestConn struct {
	conn *websocket.Conn
	enc  *json.Encoder
	dec  *json.Decoder
}

type fakeWSAuthorizer struct {
	userID       string
	tokenToUser  map[string]string
	authErr      error
	resolveErr   error
	welcomeByKey map[string]joinWelcome
}

func (f fakeWSAuthorizer) Authenticate(_ context.Context, accessToken string) (string, error) {
	if f.authErr != nil {
		return "", f.authErr
	}
	if f.tokenToUser != nil {
		if userID := strings.TrimSpace(f.tokenToUser[strings.TrimSpace(accessToken)]); userID != "" {
			return userID, nil
		}
	}
	if strings.TrimSpace(f.userID) == "" {
		return "", errors.New("missing user id")
	}
	return strings.TrimSpace(f.userID), nil
}

func (f fakeWSAuthorizer) ResolveJoinWelcome(_ context.Context, campaignID string, sessionID string, userID string) (joinWelcome, error) {
	if f.resolveErr != nil {
		return joinWelcome{}, f.resolveErr
	}
	if welcome, ok := f.welcomeByKey[campaignID+"::"+sessionID+"::"+userID]; ok {
		return welcome, nil
	}
	return joinWelcome{
		ParticipantID:   "part-" + userID,
		ParticipantName: "User " + userID,
		CampaignName:    campaignID,
		SessionID:       sessionID,
		SessionName:     sessionID,
	}, nil
}

func openWSConn(t *testing.T, handler http.Handler, cookies ...string) *wsTestConn {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	cfg, err := websocket.NewConfig(wsURL, srv.URL)
	if err != nil {
		t.Fatalf("websocket.NewConfig() error = %v", err)
	}
	if len(cookies) > 0 {
		cfg.Header = http.Header{}
		for _, cookie := range cookies {
			cfg.Header.Add("Cookie", cookie)
		}
	}
	conn, err := websocket.DialConfig(cfg)
	if err != nil {
		t.Fatalf("websocket.DialConfig() error = %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	return &wsTestConn{
		conn: conn,
		enc:  json.NewEncoder(conn),
		dec:  json.NewDecoder(conn),
	}
}

func (c *wsTestConn) writeFrame(t *testing.T, frame map[string]any) {
	t.Helper()
	if err := c.enc.Encode(frame); err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
}

func (c *wsTestConn) readFrame(t *testing.T) wsTestFrame {
	t.Helper()
	var frame wsTestFrame
	if err := c.dec.Decode(&frame); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	return frame
}

func TestWebSocketJoinRequiresSessionID(t *testing.T) {
	t.Parallel()

	conn := openWSConn(t, NewHandler())
	conn.writeFrame(t, map[string]any{
		"type": "chat.join",
		"payload": map[string]any{
			"campaign_id": "camp-1",
		},
	})

	frame := conn.readFrame(t)
	if frame.Type != "chat.error" {
		t.Fatalf("frame.Type = %q, want chat.error", frame.Type)
	}
	var payload wsErrorEnvelope
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Error.Code != "INVALID_ARGUMENT" || payload.Error.Message != "session_id is required" {
		t.Fatalf("unexpected error payload: %+v", payload.Error)
	}
}

func TestWebSocketSendBeforeJoinReturnsForbidden(t *testing.T) {
	t.Parallel()

	conn := openWSConn(t, NewHandler())
	conn.writeFrame(t, map[string]any{
		"type": "chat.send",
		"payload": map[string]any{
			"client_message_id": "cli-1",
			"body":              "hello",
		},
	})

	frame := conn.readFrame(t)
	if frame.Type != "chat.error" {
		t.Fatalf("frame.Type = %q, want chat.error", frame.Type)
	}
	var payload wsErrorEnvelope
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Error.Code != "FORBIDDEN" {
		t.Fatalf("payload.Error.Code = %q, want FORBIDDEN", payload.Error.Code)
	}
}

func TestWebSocketJoinAndBroadcastWithinSessionRoom(t *testing.T) {
	t.Parallel()

	hub := newRoomHub()
	handler := newHandler(fakeWSAuthorizer{
		tokenToUser: map[string]string{
			"web_session:tok-a": "user-a",
			"web_session:tok-b": "user-b",
			"web_session:tok-c": "user-c",
		},
		welcomeByKey: map[string]joinWelcome{
			"camp-1::sess-1::user-a": {ParticipantID: "part-a", ParticipantName: "Ari", CampaignName: "Guildhouse", SessionID: "sess-1", SessionName: "One"},
			"camp-1::sess-1::user-b": {ParticipantID: "part-b", ParticipantName: "Bo", CampaignName: "Guildhouse", SessionID: "sess-1", SessionName: "One"},
			"camp-1::sess-2::user-c": {ParticipantID: "part-c", ParticipantName: "Cy", CampaignName: "Guildhouse", SessionID: "sess-2", SessionName: "Two"},
		},
	}, true, hub)
	conn1 := openWSConn(t, handler, "web_session=tok-a")
	conn2 := openWSConn(t, handler, "web_session=tok-b")
	conn3 := openWSConn(t, handler, "web_session=tok-c")
	conn1.writeFrame(t, map[string]any{"type": "chat.join", "payload": map[string]any{"campaign_id": "camp-1", "session_id": "sess-1"}})
	conn2.writeFrame(t, map[string]any{"type": "chat.join", "payload": map[string]any{"campaign_id": "camp-1", "session_id": "sess-1"}})
	conn3.writeFrame(t, map[string]any{"type": "chat.join", "payload": map[string]any{"campaign_id": "camp-1", "session_id": "sess-2"}})
	_ = conn1.readFrame(t)
	_ = conn2.readFrame(t)
	_ = conn3.readFrame(t)

	conn1.writeFrame(t, map[string]any{
		"type": "chat.send",
		"payload": map[string]any{
			"client_message_id": "cli-1",
			"body":              "hello table",
		},
	})

	ack := conn1.readFrame(t)
	if ack.Type != "chat.ack" {
		t.Fatalf("ack.Type = %q, want chat.ack", ack.Type)
	}
	msgA := conn1.readFrame(t)
	msgB := conn2.readFrame(t)
	for _, frame := range []wsTestFrame{msgA, msgB} {
		if frame.Type != "chat.message" {
			t.Fatalf("frame.Type = %q, want chat.message", frame.Type)
		}
	}

	_ = conn3.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	var isolated wsTestFrame
	if err := conn3.dec.Decode(&isolated); err == nil {
		t.Fatalf("unexpected frame for isolated session: %+v", isolated)
	}
}

func TestWebSocketHistoryBeforeReturnsRoomHistory(t *testing.T) {
	t.Parallel()

	handler := newHandler(nil, false, newRoomHub())
	conn := openWSConn(t, handler)
	conn.writeFrame(t, map[string]any{"type": "chat.join", "payload": map[string]any{"campaign_id": "camp-1", "session_id": "sess-1"}})
	_ = conn.readFrame(t)
	conn.writeFrame(t, map[string]any{"type": "chat.send", "payload": map[string]any{"client_message_id": "cli-1", "body": "one"}})
	_ = conn.readFrame(t)
	_ = conn.readFrame(t)
	conn.writeFrame(t, map[string]any{"type": "chat.send", "payload": map[string]any{"client_message_id": "cli-2", "body": "two"}})
	_ = conn.readFrame(t)
	_ = conn.readFrame(t)

	conn.writeFrame(t, map[string]any{"type": "chat.history.before", "request_id": "req-1", "payload": map[string]any{"before_sequence_id": 3, "limit": 10}})
	first := conn.readFrame(t)
	second := conn.readFrame(t)
	ack := conn.readFrame(t)
	if first.Type != "chat.message" || second.Type != "chat.message" || ack.Type != "chat.ack" {
		t.Fatalf("unexpected history frame sequence: %q %q %q", first.Type, second.Type, ack.Type)
	}
}

func TestWebSocketJoinReplaysMessagesAfterLastSequenceID(t *testing.T) {
	t.Parallel()

	handler := newHandler(nil, false, newRoomHub())
	author := openWSConn(t, handler)
	author.writeFrame(t, map[string]any{"type": "chat.join", "payload": map[string]any{"campaign_id": "camp-1", "session_id": "sess-1"}})
	_ = author.readFrame(t)
	author.writeFrame(t, map[string]any{"type": "chat.send", "payload": map[string]any{"client_message_id": "cli-1", "body": "one"}})
	_ = author.readFrame(t)
	_ = author.readFrame(t)
	author.writeFrame(t, map[string]any{"type": "chat.send", "payload": map[string]any{"client_message_id": "cli-2", "body": "two"}})
	_ = author.readFrame(t)
	_ = author.readFrame(t)

	rejoin := openWSConn(t, handler)
	rejoin.writeFrame(t, map[string]any{
		"type": "chat.join",
		"payload": map[string]any{
			"campaign_id":      "camp-1",
			"session_id":       "sess-1",
			"last_sequence_id": 1,
		},
	})

	joined := rejoin.readFrame(t)
	if joined.Type != "chat.joined" {
		t.Fatalf("joined.Type = %q, want chat.joined", joined.Type)
	}
	replayed := rejoin.readFrame(t)
	if replayed.Type != "chat.message" {
		t.Fatalf("replayed.Type = %q, want chat.message", replayed.Type)
	}
	var payload messageEnvelope
	if err := json.Unmarshal(replayed.Payload, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Message.SequenceID != 2 || payload.Message.Body != "two" {
		t.Fatalf("replayed message = %+v, want seq=2 body=two", payload.Message)
	}
}

func TestWebSocketControlFrameIsUnsupported(t *testing.T) {
	t.Parallel()

	conn := openWSConn(t, NewHandler())
	conn.writeFrame(t, map[string]any{
		"type": "chat.control",
		"payload": map[string]any{
			"action": "gate.open",
		},
	})

	frame := conn.readFrame(t)
	if frame.Type != "chat.error" {
		t.Fatalf("frame.Type = %q, want chat.error", frame.Type)
	}
	var payload wsErrorEnvelope
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Error.Message != "unsupported frame type" {
		t.Fatalf("payload.Error.Message = %q, want unsupported frame type", payload.Error.Message)
	}
}
