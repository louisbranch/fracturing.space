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

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	"golang.org/x/net/websocket"
)

const (
	maxFramePayloadBytes   = 32 * 1024
	maxFramesPerSecond     = 50
	maxDecodeErrorsPerConn = 3
	maxMessageBodyRunes    = 12000
	maxClientMessageIDLen  = 128
)

type wsFrame struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type typingPayload struct {
	Active bool `json:"active"`
}

type realtimeHub struct {
	server *Server

	mu     sync.Mutex
	rooms  map[string]*campaignRoom
	closed bool
}

type campaignRoom struct {
	hub        *realtimeHub
	campaignID string

	ctx    context.Context
	cancel context.CancelFunc

	mu          sync.Mutex
	sessions    map[*realtimeSession]struct{}
	lastGameSeq uint64
}

type realtimeSession struct {
	userID string
	peer   *wsPeer

	mu               sync.Mutex
	room             *campaignRoom
	campaignID       string
	participantID    string
	participantName  string
	activeSessionID  string
	chatTypingTimer  *time.Timer
	draftTypingTimer *time.Timer
}

type wsPeer struct {
	mu      sync.Mutex
	encoder *json.Encoder
}

func newRealtimeHub(server *Server) *realtimeHub {
	return &realtimeHub{
		server: server,
		rooms:  map[string]*campaignRoom{},
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

	windowStart := time.Now()
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
		now := time.Now()
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
				Payload:   mustJSON(playWSPongPayload{Timestamp: time.Now().UTC().Format(time.RFC3339Nano)}),
			})
		default:
			_ = session.peer.writeError(frame.RequestID, "invalid_argument", "unsupported frame type", nil)
		}
	}
}

func (h *realtimeHub) handleConnect(ctx context.Context, session *realtimeSession, frame wsFrame) {
	var payload playWSConnectPayload
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
	state, err := h.server.loadInteractionState(ctx, campaignID, session.userID)
	if err != nil {
		_ = session.peer.writeError(frame.RequestID, "unavailable", "failed to load interaction state", nil)
		return
	}
	session.attach(room, state)
	room.add(session)

	snapshot, err := h.server.buildRoomSnapshot(ctx, campaignID, state, room.latestGameSequence())
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
		messages, err := h.server.buildIncrementalChatMessages(ctx, campaignID, sessionID, payload.LastChatSeq)
		if err != nil {
			_ = session.peer.writeFrame(wsFrame{Type: "play.resync", Payload: mustJSON(map[string]string{"reason": "chat history drifted; reload required"})})
			return
		}
		for _, message := range messages {
			_ = session.peer.writeFrame(wsFrame{
				Type:    "play.chat.message",
				Payload: mustJSON(playWSChatMessageEnvelope{Message: message}),
			})
		}
	}
}

func (h *realtimeHub) handleChatSend(ctx context.Context, session *realtimeSession, frame wsFrame) {
	var payload playWSChatSendPayload
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
	message, _, err := h.server.transcripts.AppendMessage(ctx, campaignID, sessionID, transcript.MessageActor{
		ParticipantID: participantID,
		Name:          participantName,
	}, body, clientMessageID)
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
		Payload:   mustJSON(playWSChatMessageEnvelope{Message: transcriptMessageToPayload(message)}),
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
	room.broadcastFrame(wsFrame{Type: frameType, Payload: mustJSON(playWSTypingPayload{
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

func (r *campaignRoom) runProjectionSubscription() {
	for {
		stream, err := r.hub.server.events.SubscribeCampaignUpdates(r.ctx, &gamev1.SubscribeCampaignUpdatesRequest{
			CampaignId:       r.campaignID,
			Kinds:            []gamev1.CampaignUpdateKind{gamev1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_PROJECTION_APPLIED},
			ProjectionScopes: []string{"campaign_sessions", "campaign_scenes"},
		})
		if err != nil {
			if !sleepUntilRetry(r.ctx, time.Second) {
				return
			}
			continue
		}
		for {
			update, recvErr := stream.Recv()
			if recvErr != nil {
				if errors.Is(recvErr, io.EOF) {
					break
				}
				if !sleepUntilRetry(r.ctx, time.Second) {
					return
				}
				break
			}
			if update == nil || update.GetProjectionApplied() == nil {
				continue
			}
			r.setLatestGameSequence(update.GetSeq())
			r.broadcastCurrent()
		}
	}
}

func (r *campaignRoom) add(session *realtimeSession) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session] = struct{}{}
}

func (r *campaignRoom) remove(session *realtimeSession) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.sessions, session)
	if len(r.sessions) != 0 {
		return
	}
	r.cancel()
	r.hub.mu.Lock()
	delete(r.hub.rooms, r.campaignID)
	r.hub.mu.Unlock()
}

func (r *campaignRoom) setLatestGameSequence(seq uint64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if seq > r.lastGameSeq {
		r.lastGameSeq = seq
	}
}

func (r *campaignRoom) latestGameSequence() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.lastGameSeq
}

func (r *campaignRoom) sessionsSnapshot() []*realtimeSession {
	r.mu.Lock()
	defer r.mu.Unlock()
	values := make([]*realtimeSession, 0, len(r.sessions))
	for session := range r.sessions {
		values = append(values, session)
	}
	return values
}

func (r *campaignRoom) broadcastCurrent() {
	for _, session := range r.sessionsSnapshot() {
		state, err := r.hub.server.loadInteractionState(r.ctx, r.campaignID, session.userID)
		if err != nil {
			_ = session.peer.writeFrame(wsFrame{Type: "play.resync", Payload: mustJSON(map[string]string{"reason": "interaction state changed; reload required"})})
			continue
		}
		session.attach(r, state)
		snapshot, err := r.hub.server.buildRoomSnapshot(r.ctx, r.campaignID, state, r.latestGameSequence())
		if err != nil {
			_ = session.peer.writeFrame(wsFrame{Type: "play.resync", Payload: mustJSON(map[string]string{"reason": "interaction state changed; reload required"})})
			continue
		}
		_ = session.peer.writeFrame(wsFrame{Type: "play.interaction.updated", Payload: mustJSON(snapshot)})
	}
}

func (r *campaignRoom) broadcastFrame(frame wsFrame) {
	for _, session := range r.sessionsSnapshot() {
		_ = session.peer.writeFrame(frame)
	}
}

func (s *realtimeSession) attach(room *campaignRoom, state *gamev1.InteractionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.room = room
	s.campaignID = strings.TrimSpace(room.campaignID)
	s.participantID = strings.TrimSpace(state.GetViewer().GetParticipantId())
	s.participantName = strings.TrimSpace(state.GetViewer().GetName())
	s.activeSessionID = strings.TrimSpace(state.GetActiveSession().GetSessionId())
}

func (s *realtimeSession) currentRoom() *campaignRoom {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.room
}

func (s *realtimeSession) activeSession() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activeSessionID
}

func (s *realtimeSession) chatIdentity() (string, string, string, string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.room == nil || s.campaignID == "" || s.activeSessionID == "" || s.participantID == "" {
		return "", "", "", "", false
	}
	return s.campaignID, s.activeSessionID, s.participantID, s.participantName, true
}

func (s *realtimeSession) resetTypingTimer(frameType string, active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var timer **time.Timer
	switch frameType {
	case "play.chat.typing":
		timer = &s.chatTypingTimer
	default:
		timer = &s.draftTypingTimer
	}
	if *timer != nil {
		(*timer).Stop()
		*timer = nil
	}
	if !active || s.room == nil {
		return
	}
	room := s.room
	sessionID := s.activeSessionID
	participantID := s.participantID
	participantName := s.participantName
	*timer = time.AfterFunc(typingTTL, func() {
		room.broadcastFrame(wsFrame{Type: frameType, Payload: mustJSON(playWSTypingPayload{
			SessionID:     sessionID,
			ParticipantID: participantID,
			Name:          participantName,
			Active:        false,
		})})
	})
}

func (p *wsPeer) writeFrame(frame wsFrame) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.encoder.Encode(frame)
}

func (p *wsPeer) writeError(requestID string, code string, message string, details map[string]any) error {
	return p.writeFrame(wsFrame{
		Type:      "play.error",
		RequestID: requestID,
		Payload: mustJSON(playWSErrorEnvelope{Error: playWSError{
			Code:    code,
			Message: message,
			Details: details,
		}}),
	})
}

func mustJSON(value any) json.RawMessage {
	data, _ := json.Marshal(value)
	return data
}

func sleepUntilRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
