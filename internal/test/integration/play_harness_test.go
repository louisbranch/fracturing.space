//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	"golang.org/x/net/websocket"
)

type playFrame struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type playSendFrame struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Payload   any    `json:"payload,omitempty"`
}

func readPlayFrame(t *testing.T, conn *websocket.Conn) playFrame {
	t.Helper()

	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("set websocket deadline: %v", err)
	}
	var frame playFrame
	if err := websocket.JSON.Receive(conn, &frame); err != nil {
		t.Fatalf("read websocket frame: %v", err)
	}
	return frame
}

func waitForPlayFrame(t *testing.T, conn *websocket.Conn, wantType string) playFrame {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		frame := readPlayFrame(t, conn)
		if strings.TrimSpace(frame.Type) == wantType {
			return frame
		}
	}
	t.Fatalf("timed out waiting for websocket frame type %q", wantType)
	return playFrame{}
}

func waitForPlayInteractionUpdate(
	t *testing.T,
	conn *websocket.Conn,
	match func(playprotocol.RoomSnapshot) bool,
) playprotocol.RoomSnapshot {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		frame := readPlayFrame(t, conn)
		if strings.TrimSpace(frame.Type) != "play.interaction.updated" {
			continue
		}
		snapshot := decodePlayPayload[playprotocol.RoomSnapshot](t, frame.Payload)
		if match(snapshot) {
			return snapshot
		}
	}
	t.Fatal("timed out waiting for matching play interaction update")
	return playprotocol.RoomSnapshot{}
}

func decodePlayPayload[T any](t *testing.T, payload json.RawMessage) T {
	t.Helper()

	var value T
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatalf("decode play payload: %v", err)
	}
	return value
}

func sessionCookieHeader(sessionID string) http.Header {
	header := http.Header{}
	header.Set("Cookie", fmt.Sprintf("play_session=%s", strings.TrimSpace(sessionID)))
	return header
}
