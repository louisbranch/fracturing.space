package app

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	"golang.org/x/net/websocket"
)

// realtimeHubDeps captures the specific dependencies the realtime hub needs,
// breaking the circular reference on *Server so the hub is testable in
// isolation and the dependency surface is explicit.
type realtimeHubDeps struct {
	resolveUserID func(context.Context, *http.Request) (string, error)
	application   func() playApplication
	logger        *slog.Logger
	aiDebug       aiDebugClient
	transcripts   transcript.Store
	events        campaignUpdateClient
}

type realtimeHub struct {
	deps    realtimeHubDeps
	runtime realtimeRuntime

	mu     sync.Mutex
	rooms  map[string]*campaignRoom
	closed bool
}

func newRealtimeHub(server *Server) *realtimeHub {
	return newRealtimeHubWithRuntime(realtimeHubDeps{
		resolveUserID: server.resolvePlayUserID,
		application:   server.application,
		logger:        server.logger,
		aiDebug:       server.deps.AIDebug,
		transcripts:   server.deps.Transcripts,
		events:        server.deps.CampaignUpdates,
	}, defaultRealtimeRuntime())
}

// newRealtimeHubWithRuntime injects runtime hooks for deterministic realtime
// tests while keeping production callers on the default clock and timers.
func (h *realtimeHub) log() *slog.Logger {
	if h != nil && h.deps.logger != nil {
		return h.deps.logger
	}
	return slog.Default()
}

func newRealtimeHubWithRuntime(deps realtimeHubDeps, runtime realtimeRuntime) *realtimeHub {
	return &realtimeHub{
		deps:    deps,
		runtime: runtime.normalize(),
		rooms:   map[string]*campaignRoom{},
	}
}

func (h *realtimeHub) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h == nil {
			http.Error(w, "realtime unavailable", http.StatusServiceUnavailable)
			return
		}
		userID, err := h.deps.resolveUserID(r.Context(), r)
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

	// Reject oversized websocket frames at the transport level before the JSON
	// decoder allocates memory for the full message body.
	conn.MaxPayloadBytes = maxFramePayloadBytes + 4096

	decoder := json.NewDecoder(conn)
	session := &realtimeSession{
		userID: strings.TrimSpace(userID),
		peer:   &wsPeer{encoder: json.NewEncoder(conn)},
	}
	defer h.unregisterSession(session)

	rateLimiter := newWSRateLimiter(h.runtime.nowTime)
	decodeErrors := 0

	for {
		var frame wsFrame
		if err := decoder.Decode(&frame); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			decodeErrors++
			_ = session.peer.writeError(frame.RequestID, WSErrorInvalidArgument, "invalid frame payload", nil)
			if decodeErrors >= maxDecodeErrorsPerConn {
				return
			}
			continue
		}
		if len(frame.Payload) > maxFramePayloadBytes {
			_ = session.peer.writeError(frame.RequestID, WSErrorInvalidArgument, "payload too large", nil)
			continue
		}
		if !rateLimiter.allow() {
			_ = session.peer.writeError(frame.RequestID, WSErrorResourceExhausted, "rate limit exceeded", nil)
			return
		}
		switch frame.Type {
		case FrameConnect:
			h.handleConnect(conn.Request().Context(), session, frame)
		case FrameChatSend:
			h.handleChatSend(conn.Request().Context(), session, frame)
		case FrameTyping:
			h.handleTyping(session, frame)
		case FramePing:
			_ = session.peer.writeFrame(wsFrame{
				Type:      FramePong,
				RequestID: frame.RequestID,
				Payload:   mustJSON(playprotocol.Pong{Timestamp: h.runtime.nowTime().Format(time.RFC3339Nano)}),
			})
		default:
			_ = session.peer.writeError(frame.RequestID, WSErrorInvalidArgument, "unsupported frame type", nil)
		}
	}
}

func (h *realtimeHub) handleConnect(ctx context.Context, session *realtimeSession, frame wsFrame) {
	var payload playprotocol.ConnectRequest
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		_ = session.peer.writeError(frame.RequestID, WSErrorInvalidArgument, "invalid connect payload", nil)
		return
	}
	campaignID := strings.TrimSpace(payload.CampaignID)
	if campaignID == "" {
		_ = session.peer.writeError(frame.RequestID, WSErrorInvalidArgument, "campaign_id is required", nil)
		return
	}
	room := h.room(campaignID)
	if room == nil {
		_ = session.peer.writeError(frame.RequestID, WSErrorUnavailable, "service is shutting down", nil)
		return
	}
	app := h.deps.application()
	req := playRequest{
		campaignRequest: campaignRequest{CampaignID: campaignID},
		UserID:          session.userID,
	}
	state, err := app.interactionState(ctx, req)
	if err != nil {
		_ = session.peer.writeError(frame.RequestID, WSErrorUnavailable, "failed to load interaction state", nil)
		return
	}
	snapshot, err := app.roomSnapshotFromState(ctx, req, state, room.latestGameSequence())
	if err != nil {
		_ = session.peer.writeError(frame.RequestID, WSErrorUnavailable, "failed to build play snapshot", nil)
		return
	}
	session.attach(room, snapshot.InteractionState)
	room.add(session)
	_ = session.peer.writeFrame(wsFrame{
		Type:      FrameReady,
		RequestID: frame.RequestID,
		Payload:   mustJSON(snapshot),
	})
	room.ensureProjectionSubscription()
	room.reconcileAIDebugSubscription(session.activeSession())

	if sessionID := session.activeSession(); sessionID != "" && payload.LastChatSeq < snapshot.Chat.LatestSequenceID {
		messages, err := app.incrementalChatMessages(ctx, transcript.Scope{CampaignID: campaignID, SessionID: sessionID}, payload.LastChatSeq)
		if err != nil {
			_ = session.peer.writeFrame(wsFrame{Type: FrameResync, Payload: mustJSON(map[string]string{"reason": "chat history drifted; reload required"})})
			return
		}
		for _, message := range messages {
			_ = session.peer.writeFrame(wsFrame{
				Type:    FrameChatMessage,
				Payload: mustJSON(playprotocol.ChatMessageEnvelope{Message: message}),
			})
		}
	}
}

func (h *realtimeHub) handleChatSend(ctx context.Context, session *realtimeSession, frame wsFrame) {
	var payload playprotocol.ChatSendRequest
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		_ = session.peer.writeError(frame.RequestID, WSErrorInvalidArgument, "invalid chat payload", nil)
		return
	}
	body := strings.TrimSpace(payload.Body)
	if body == "" {
		_ = session.peer.writeError(frame.RequestID, WSErrorInvalidArgument, "body is required", nil)
		return
	}
	if len([]rune(body)) > maxMessageBodyRunes {
		_ = session.peer.writeError(frame.RequestID, WSErrorInvalidArgument, "body is too long", nil)
		return
	}
	clientMessageID := strings.TrimSpace(payload.ClientMessageID)
	if len(clientMessageID) > maxClientMessageIDLen {
		_ = session.peer.writeError(frame.RequestID, WSErrorInvalidArgument, "client_message_id is too long", nil)
		return
	}

	identity, ok := session.chatIdentity()
	if !ok {
		_ = session.peer.writeError(frame.RequestID, WSErrorFailedPrecondition, "join an active session before sending chat", nil)
		return
	}
	result, err := h.deps.transcripts.AppendMessage(ctx, transcript.AppendRequest{
		Scope: transcript.Scope{
			CampaignID: identity.CampaignID,
			SessionID:  identity.SessionID,
		},
		Actor: transcript.MessageActor{
			ParticipantID: identity.ParticipantID,
			Name:          identity.ParticipantName,
		},
		Body:            body,
		ClientMessageID: clientMessageID,
	})
	if err != nil {
		_ = session.peer.writeError(frame.RequestID, WSErrorUnavailable, "failed to persist chat message", nil)
		return
	}
	chatRoom := session.currentRoom()
	if chatRoom == nil {
		return
	}
	chatRoom.broadcastFrame(wsFrame{
		Type:      FrameChatMessage,
		RequestID: frame.RequestID,
		Payload:   mustJSON(playprotocol.ChatMessageEnvelope{Message: playprotocol.TranscriptMessage(result.Message)}),
	})
}

func (h *realtimeHub) handleTyping(session *realtimeSession, frame wsFrame) {
	var payload typingPayload
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		_ = session.peer.writeError(frame.RequestID, WSErrorInvalidArgument, "invalid typing payload", nil)
		return
	}
	room := session.currentRoom()
	if room == nil {
		_ = session.peer.writeError(frame.RequestID, WSErrorFailedPrecondition, "join a campaign before sending typing", nil)
		return
	}
	identity, ok := session.chatIdentity()
	if !ok {
		_ = session.peer.writeError(frame.RequestID, WSErrorFailedPrecondition, "participant identity unavailable", nil)
		return
	}
	room.broadcastFrame(wsFrame{Type: FrameTyping, Payload: mustJSON(playprotocol.TypingEvent{
		SessionID:     identity.SessionID,
		ParticipantID: identity.ParticipantID,
		Name:          identity.ParticipantName,
		Active:        payload.Active,
	})})
	session.resetTypingTimer(payload.Active)
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
	typingActive := session.typingTimer != nil
	participantID := session.participantID
	participantName := session.participantName
	sessionID := session.activeSessionID
	if session.typingTimer != nil {
		session.typingTimer.Stop()
		session.typingTimer = nil
	}
	session.room = nil
	session.mu.Unlock()
	if room != nil {
		if typingActive {
			room.broadcastFrame(wsFrame{Type: FrameTyping, Payload: mustJSON(playprotocol.TypingEvent{
				SessionID:     sessionID,
				ParticipantID: participantID,
				Name:          participantName,
				Active:        false,
			})})
		}
		room.remove(session)
	}
}
