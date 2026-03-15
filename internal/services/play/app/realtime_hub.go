package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	"golang.org/x/net/websocket"
)

type realtimeHub struct {
	server  *Server
	runtime realtimeRuntime

	mu     sync.Mutex
	rooms  map[string]*campaignRoom
	closed bool
}

func newRealtimeHub(server *Server) *realtimeHub {
	return newRealtimeHubWithRuntime(server, defaultRealtimeRuntime())
}

// newRealtimeHubWithRuntime injects runtime hooks for deterministic realtime
// tests while keeping production callers on the default clock and timers.
func newRealtimeHubWithRuntime(server *Server, runtime realtimeRuntime) *realtimeHub {
	return &realtimeHub{
		server:  server,
		runtime: runtime.normalize(),
		rooms:   map[string]*campaignRoom{},
	}
}

func (h *realtimeHub) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h == nil || h.server == nil {
			http.Error(w, "realtime unavailable", http.StatusServiceUnavailable)
			return
		}
		userID, err := h.server.resolvePlayUserID(r.Context(), r)
		if err != nil {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		websocket.Handler(func(conn *websocket.Conn) {
			h.handleWSConn(conn, userID)
		}).ServeHTTP(w, r)
	})
}

func (h *realtimeHub) Close() {
	if h == nil {
		return
	}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	rooms := make([]*campaignRoom, 0, len(h.rooms))
	for _, room := range h.rooms {
		rooms = append(rooms, room)
	}
	h.rooms = map[string]*campaignRoom{}
	h.mu.Unlock()
	for _, room := range rooms {
		room.cancel()
	}
}

func (h *realtimeHub) broadcastCurrent(campaignID string) {
	if h == nil {
		return
	}
	room := h.roomIfExists(strings.TrimSpace(campaignID))
	if room == nil {
		return
	}
	room.broadcastCurrent()
}

func (h *realtimeHub) handleWSConn(conn *websocket.Conn, userID string) {
	defer func() { _ = conn.Close() }()

	decoder := json.NewDecoder(conn)
	session := &realtimeSession{
		userID: strings.TrimSpace(userID),
		peer:   &wsPeer{encoder: json.NewEncoder(conn)},
	}
	defer h.unregisterSession(session)

	windowStart := h.runtime.nowTime()
	framesInWindow := 0
	decodeErrors := 0

	for {
		var frame wsFrame
		if err := decoder.Decode(&frame); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			decodeErrors++
			_ = session.peer.writeError(frame.RequestID, "invalid_argument", "invalid frame payload", nil)
			if decodeErrors >= maxDecodeErrorsPerConn {
				return
			}
			continue
		}
		if len(frame.Payload) > maxFramePayloadBytes {
			_ = session.peer.writeError(frame.RequestID, "invalid_argument", "payload too large", nil)
			continue
		}
		now := h.runtime.nowTime()
		if now.Sub(windowStart) >= time.Second {
			windowStart = now
			framesInWindow = 0
		}
		framesInWindow++
		if framesInWindow > maxFramesPerSecond {
			_ = session.peer.writeError(frame.RequestID, "resource_exhausted", "rate limit exceeded", nil)
			return
		}
		switch frame.Type {
		case "play.connect":
			h.handleConnect(conn.Request().Context(), session, frame)
		case "play.chat.send":
			h.handleChatSend(conn.Request().Context(), session, frame)
		case "play.chat.typing":
			h.handleTyping(session, frame, "play.chat.typing")
		case "play.draft.typing":
			h.handleTyping(session, frame, "play.draft.typing")
		case "play.ping":
			_ = session.peer.writeFrame(wsFrame{
				Type:      "play.pong",
				RequestID: frame.RequestID,
				Payload:   mustJSON(playprotocol.Pong{Timestamp: h.runtime.nowTime().Format(time.RFC3339Nano)}),
			})
		default:
			_ = session.peer.writeError(frame.RequestID, "invalid_argument", "unsupported frame type", nil)
		}
	}
}

func (h *realtimeHub) handleConnect(ctx context.Context, session *realtimeSession, frame wsFrame) {
	var payload playprotocol.ConnectRequest
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		_ = session.peer.writeError(frame.RequestID, "invalid_argument", "invalid connect payload", nil)
		return
	}
	campaignID := strings.TrimSpace(payload.CampaignID)
	if campaignID == "" {
		_ = session.peer.writeError(frame.RequestID, "invalid_argument", "campaign_id is required", nil)
		return
	}
	room := h.room(campaignID)
	app := h.server.application()
	state, err := app.interactionState(ctx, playRequest{
		campaignRequest: campaignRequest{CampaignID: campaignID},
		UserID:          session.userID,
	})
	if err != nil {
		_ = session.peer.writeError(frame.RequestID, "unavailable", "failed to load interaction state", nil)
		return
	}
	session.attach(room, state)
	room.add(session)

	snapshot, err := app.roomSnapshotFromState(ctx, campaignID, state, room.latestGameSequence())
	if err != nil {
		_ = session.peer.writeError(frame.RequestID, "unavailable", "failed to build play snapshot", nil)
		return
	}
	_ = session.peer.writeFrame(wsFrame{
		Type:      "play.ready",
		RequestID: frame.RequestID,
		Payload:   mustJSON(snapshot),
	})

	if sessionID := session.activeSession(); sessionID != "" && payload.LastChatSeq < snapshot.Chat.LatestSequenceID {
		messages, err := app.incrementalChatMessages(ctx, transcript.Scope{CampaignID: campaignID, SessionID: sessionID}, payload.LastChatSeq)
		if err != nil {
			_ = session.peer.writeFrame(wsFrame{Type: "play.resync", Payload: mustJSON(map[string]string{"reason": "chat history drifted; reload required"})})
			return
		}
		for _, message := range messages {
			_ = session.peer.writeFrame(wsFrame{
				Type:    "play.chat.message",
				Payload: mustJSON(playprotocol.ChatMessageEnvelope{Message: message}),
			})
		}
	}
}

func (h *realtimeHub) handleChatSend(ctx context.Context, session *realtimeSession, frame wsFrame) {
	var payload playprotocol.ChatSendRequest
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		_ = session.peer.writeError(frame.RequestID, "invalid_argument", "invalid chat payload", nil)
		return
	}
	body := strings.TrimSpace(payload.Body)
	if body == "" {
		_ = session.peer.writeError(frame.RequestID, "invalid_argument", "body is required", nil)
		return
	}
	if len([]rune(body)) > maxMessageBodyRunes {
		_ = session.peer.writeError(frame.RequestID, "invalid_argument", "body is too long", nil)
		return
	}
	clientMessageID := strings.TrimSpace(payload.ClientMessageID)
	if len(clientMessageID) > maxClientMessageIDLen {
		_ = session.peer.writeError(frame.RequestID, "invalid_argument", "client_message_id is too long", nil)
		return
	}

	campaignID, sessionID, participantID, participantName, ok := session.chatIdentity()
	if !ok {
		_ = session.peer.writeError(frame.RequestID, "failed_precondition", "join an active session before sending chat", nil)
		return
	}
	result, err := h.server.transcripts.AppendMessage(ctx, transcript.AppendRequest{
		Scope: transcript.Scope{
			CampaignID: campaignID,
			SessionID:  sessionID,
		},
		Actor: transcript.MessageActor{
			ParticipantID: participantID,
			Name:          participantName,
		},
		Body:            body,
		ClientMessageID: clientMessageID,
	})
	if err != nil {
		_ = session.peer.writeError(frame.RequestID, "unavailable", "failed to persist chat message", nil)
		return
	}
	room := session.currentRoom()
	if room == nil {
		return
	}
	room.broadcastFrame(wsFrame{
		Type:      "play.chat.message",
		RequestID: frame.RequestID,
		Payload:   mustJSON(playprotocol.ChatMessageEnvelope{Message: playprotocol.TranscriptMessage(result.Message)}),
	})
}

func (h *realtimeHub) handleTyping(session *realtimeSession, frame wsFrame, frameType string) {
	var payload typingPayload
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		_ = session.peer.writeError(frame.RequestID, "invalid_argument", "invalid typing payload", nil)
		return
	}
	room := session.currentRoom()
	if room == nil {
		_ = session.peer.writeError(frame.RequestID, "failed_precondition", "join a campaign before sending typing", nil)
		return
	}
	_, sessionID, participantID, participantName, ok := session.chatIdentity()
	if !ok {
		_ = session.peer.writeError(frame.RequestID, "failed_precondition", "participant identity unavailable", nil)
		return
	}
	room.broadcastFrame(wsFrame{Type: frameType, Payload: mustJSON(playprotocol.TypingEvent{
		SessionID:     sessionID,
		ParticipantID: participantID,
		Name:          participantName,
		Active:        payload.Active,
	})})
	session.resetTypingTimer(frameType, payload.Active)
}

func (h *realtimeHub) room(campaignID string) *campaignRoom {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil
	}
	if room := h.rooms[campaignID]; room != nil {
		return room
	}
	ctx, cancel := context.WithCancel(context.Background())
	room := &campaignRoom{
		hub:        h,
		campaignID: campaignID,
		ctx:        ctx,
		cancel:     cancel,
		sessions:   map[*realtimeSession]struct{}{},
	}
	h.rooms[campaignID] = room
	go room.runProjectionSubscription()
	return room
}

func (h *realtimeHub) roomIfExists(campaignID string) *campaignRoom {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.rooms[campaignID]
}

func (h *realtimeHub) unregisterSession(session *realtimeSession) {
	if session == nil {
		return
	}
	session.mu.Lock()
	room := session.room
	if session.chatTypingTimer != nil {
		session.chatTypingTimer.Stop()
	}
	if session.draftTypingTimer != nil {
		session.draftTypingTimer.Stop()
	}
	session.room = nil
	session.mu.Unlock()
	if room != nil {
		room.remove(session)
	}
}
