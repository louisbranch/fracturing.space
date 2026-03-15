package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

func newWSSession(userID string, peer *wsPeer) *wsSession {
	return &wsSession{
		userID: userID,
		peer:   peer,
	}
}

func (s *wsSession) setRoom(next *sessionRoom) *sessionRoom {
	s.mu.Lock()
	previous := s.room
	s.room = next
	s.mu.Unlock()
	return previous
}

func (s *wsSession) currentRoom() *sessionRoom {
	s.mu.Lock()
	room := s.room
	s.mu.Unlock()
	return room
}

func (s *wsSession) setJoinState(welcome joinWelcome) {
	state := wsJoinState{
		participantID:   strings.TrimSpace(welcome.ParticipantID),
		participantName: strings.TrimSpace(welcome.ParticipantName),
	}
	if state.participantID == "" {
		state.participantID = strings.TrimSpace(s.userID)
	}
	if state.participantName == "" {
		state.participantName = state.participantID
	}
	s.mu.Lock()
	s.state = state
	s.mu.Unlock()
}

func (s *wsSession) joinState() wsJoinState {
	s.mu.Lock()
	state := s.state
	s.mu.Unlock()
	return state
}

type wsPeer struct {
	mu      sync.Mutex
	encoder *json.Encoder
}

func newWSPeer(encoder *json.Encoder) *wsPeer {
	return &wsPeer{encoder: encoder}
}

func (p *wsPeer) writeFrame(frame wsFrame) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.encoder.Encode(frame)
}

type transcriptStore interface {
	LatestSequence(campaignID string, sessionID string) int64
	AppendMessage(campaignID string, sessionID string, actor messageActor, body string, clientMessageID string) (chatMessage, bool)
	HistoryAfter(campaignID string, sessionID string, afterSequenceID int64) []chatMessage
	HistoryBefore(campaignID string, sessionID string, beforeSequenceID int64, limit int) []chatMessage
}

type roomHub struct {
	mu    sync.Mutex
	rooms map[string]*sessionRoom
	store transcriptStore
}

func newRoomHub() *roomHub {
	return &roomHub{
		rooms: make(map[string]*sessionRoom),
		store: newInMemoryTranscriptStore(),
	}
}

func (h *roomHub) room(campaignID string, sessionID string) *sessionRoom {
	key := roomKey(campaignID, sessionID)

	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[key]
	if ok {
		return room
	}

	room = newSessionRoom(campaignID, sessionID, h.store)
	h.rooms[key] = room
	return room
}

type sessionRoom struct {
	mu          sync.Mutex
	campaignID  string
	sessionID   string
	store       transcriptStore
	subscribers map[*wsPeer]*wsSession
}

func newSessionRoom(campaignID string, sessionID string, store transcriptStore) *sessionRoom {
	return &sessionRoom{
		campaignID:  strings.TrimSpace(campaignID),
		sessionID:   strings.TrimSpace(sessionID),
		store:       store,
		subscribers: make(map[*wsPeer]*wsSession),
	}
}

func (r *sessionRoom) join(session *wsSession) int64 {
	if session == nil || session.peer == nil {
		return 0
	}
	r.mu.Lock()
	r.subscribers[session.peer] = session
	r.mu.Unlock()
	return r.latestSequenceID()
}

func (r *sessionRoom) joinWithHistory(session *wsSession, afterSequenceID int64) (int64, []chatMessage) {
	if r == nil || r.store == nil {
		return 0, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	latest := r.store.LatestSequence(r.campaignID, r.sessionID)
	history := r.store.HistoryAfter(r.campaignID, r.sessionID, afterSequenceID)
	if session != nil && session.peer != nil {
		r.subscribers[session.peer] = session
	}
	return latest, history
}

func (r *sessionRoom) leave(session *wsSession) bool {
	if session == nil || session.peer == nil {
		return false
	}
	r.mu.Lock()
	delete(r.subscribers, session.peer)
	empty := len(r.subscribers) == 0
	r.mu.Unlock()
	return empty
}

func (r *sessionRoom) subscribersSnapshot() []*wsPeer {
	r.mu.Lock()
	subscribers := make([]*wsPeer, 0, len(r.subscribers))
	for subscriber := range r.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	r.mu.Unlock()
	return subscribers
}

func (r *sessionRoom) latestSequenceID() int64 {
	if r == nil || r.store == nil {
		return 0
	}
	return r.store.LatestSequence(r.campaignID, r.sessionID)
}

func (r *sessionRoom) appendMessage(actor messageActor, body string, clientMessageID string) (chatMessage, bool, []*wsPeer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	msg, duplicate := r.store.AppendMessage(r.campaignID, r.sessionID, actor, body, clientMessageID)
	if duplicate {
		return msg, true, nil
	}
	subscribers := make([]*wsPeer, 0, len(r.subscribers))
	for subscriber := range r.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	return msg, false, subscribers
}

func (r *sessionRoom) historyBefore(beforeSequenceID int64, limit int) []chatMessage {
	if r == nil || r.store == nil {
		return nil
	}
	return r.store.HistoryBefore(r.campaignID, r.sessionID, beforeSequenceID, limit)
}

func (r *sessionRoom) historyAfter(afterSequenceID int64) []chatMessage {
	if r == nil || r.store == nil {
		return nil
	}
	return r.store.HistoryAfter(r.campaignID, r.sessionID, afterSequenceID)
}

type inMemoryTranscriptStore struct {
	mu    sync.Mutex
	rooms map[string]*inMemoryTranscriptRoom
}

type inMemoryTranscriptRoom struct {
	nextSequence     int64
	messageLog       []chatMessage
	idempotencyBy    map[string]chatMessage
	idempotencyOrder []string
}

func newInMemoryTranscriptStore() *inMemoryTranscriptStore {
	return &inMemoryTranscriptStore{
		rooms: make(map[string]*inMemoryTranscriptRoom),
	}
}

func (s *inMemoryTranscriptStore) LatestSequence(campaignID string, sessionID string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.roomLocked(campaignID, sessionID).nextSequence
}

func (s *inMemoryTranscriptStore) AppendMessage(campaignID string, sessionID string, actor messageActor, body string, clientMessageID string) (chatMessage, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	room := s.roomLocked(campaignID, sessionID)
	if existing, ok := room.idempotencyBy[clientMessageID]; ok {
		return existing, true
	}

	room.nextSequence++
	actor.ParticipantID = strings.TrimSpace(actor.ParticipantID)
	if actor.ParticipantID == "" {
		actor.ParticipantID = "participant"
	}
	actor.Name = strings.TrimSpace(actor.Name)
	if actor.Name == "" {
		actor.Name = actor.ParticipantID
	}

	msg := chatMessage{
		MessageID:       fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		CampaignID:      strings.TrimSpace(campaignID),
		SessionID:       strings.TrimSpace(sessionID),
		SequenceID:      room.nextSequence,
		SentAt:          time.Now().UTC().Format(time.RFC3339),
		Actor:           actor,
		Body:            strings.TrimSpace(body),
		ClientMessageID: strings.TrimSpace(clientMessageID),
	}

	room.messageLog = append(room.messageLog, msg)
	if len(room.messageLog) > maxRoomMessages {
		room.messageLog = room.messageLog[len(room.messageLog)-maxRoomMessages:]
	}

	room.idempotencyBy[msg.ClientMessageID] = msg
	room.idempotencyOrder = append(room.idempotencyOrder, msg.ClientMessageID)
	if len(room.idempotencyOrder) > maxIdempotencyRecord {
		evict := room.idempotencyOrder[0]
		room.idempotencyOrder = room.idempotencyOrder[1:]
		delete(room.idempotencyBy, evict)
	}

	return msg, false
}

func (s *inMemoryTranscriptStore) HistoryBefore(campaignID string, sessionID string, beforeSequenceID int64, limit int) []chatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	room := s.roomLocked(campaignID, sessionID)
	history := make([]chatMessage, 0, limit)
	for _, msg := range room.messageLog {
		if msg.SequenceID < beforeSequenceID {
			history = append(history, msg)
		}
	}
	if len(history) > limit {
		history = history[len(history)-limit:]
	}
	return history
}

func (s *inMemoryTranscriptStore) HistoryAfter(campaignID string, sessionID string, afterSequenceID int64) []chatMessage {
	s.mu.Lock()
	defer s.mu.Unlock()

	room := s.roomLocked(campaignID, sessionID)
	history := make([]chatMessage, 0, len(room.messageLog))
	for _, msg := range room.messageLog {
		if msg.SequenceID > afterSequenceID {
			history = append(history, msg)
		}
	}
	return history
}

func (s *inMemoryTranscriptStore) roomLocked(campaignID string, sessionID string) *inMemoryTranscriptRoom {
	key := roomKey(campaignID, sessionID)
	room, ok := s.rooms[key]
	if ok {
		return room
	}
	room = &inMemoryTranscriptRoom{
		idempotencyBy: make(map[string]chatMessage),
	}
	s.rooms[key] = room
	return room
}

func roomKey(campaignID string, sessionID string) string {
	return strings.TrimSpace(campaignID) + "::" + strings.TrimSpace(sessionID)
}
