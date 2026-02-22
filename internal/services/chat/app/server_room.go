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

func (s *wsSession) setRoom(next *campaignRoom) *campaignRoom {
	s.mu.Lock()
	previous := s.room
	s.room = next
	s.mu.Unlock()
	return previous
}

func (s *wsSession) currentRoom() *campaignRoom {
	s.mu.Lock()
	room := s.room
	s.mu.Unlock()
	return room
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

type roomHub struct {
	mu    sync.Mutex
	rooms map[string]*campaignRoom
}

func newRoomHub() *roomHub {
	return &roomHub{rooms: make(map[string]*campaignRoom)}
}

func (h *roomHub) room(campaignID string) *campaignRoom {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[campaignID]
	if ok {
		return room
	}

	room = newCampaignRoom(campaignID)
	h.rooms[campaignID] = room
	return room
}

type campaignRoom struct {
	mu               sync.Mutex
	campaignID       string
	sessionID        string
	nextSequence     int64
	messages         []chatMessage
	idempotencyBy    map[string]chatMessage
	idempotencyOrder []string
	subscribers      map[*wsPeer]struct{}
}

func newCampaignRoom(campaignID string) *campaignRoom {
	return &campaignRoom{
		campaignID:    campaignID,
		sessionID:     defaultSessionID,
		idempotencyBy: make(map[string]chatMessage),
		subscribers:   make(map[*wsPeer]struct{}),
	}
}

func (r *campaignRoom) join(peer *wsPeer) int64 {
	r.mu.Lock()
	r.subscribers[peer] = struct{}{}
	latest := r.nextSequence
	r.mu.Unlock()
	return latest
}

func (r *campaignRoom) setSessionID(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	r.mu.Lock()
	r.sessionID = sessionID
	r.mu.Unlock()
}

func (r *campaignRoom) currentSessionID() string {
	r.mu.Lock()
	id := r.sessionID
	r.mu.Unlock()
	return id
}

func (r *campaignRoom) leave(peer *wsPeer) bool {
	r.mu.Lock()
	delete(r.subscribers, peer)
	empty := len(r.subscribers) == 0
	r.mu.Unlock()
	return empty
}

func (r *campaignRoom) appendMessage(actorID string, body string, clientMessageID string) (chatMessage, bool, []*wsPeer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.idempotencyBy[clientMessageID]; ok {
		return existing, true, nil
	}

	r.nextSequence++
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		actorID = "participant"
	}
	msg := chatMessage{
		MessageID:  fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		CampaignID: r.campaignID,
		SessionID:  r.sessionID,
		SequenceID: r.nextSequence,
		SentAt:     time.Now().UTC().Format(time.RFC3339),
		Kind:       "text",
		Actor: messageActor{
			ParticipantID: actorID,
			Name:          actorID,
		},
		Body:            body,
		ClientMessageID: clientMessageID,
	}

	r.messages = append(r.messages, msg)
	if len(r.messages) > maxRoomMessages {
		r.messages = r.messages[len(r.messages)-maxRoomMessages:]
	}

	r.idempotencyBy[clientMessageID] = msg
	r.idempotencyOrder = append(r.idempotencyOrder, clientMessageID)
	if len(r.idempotencyOrder) > maxIdempotencyRecord {
		evict := r.idempotencyOrder[0]
		r.idempotencyOrder = r.idempotencyOrder[1:]
		delete(r.idempotencyBy, evict)
	}

	subscribers := make([]*wsPeer, 0, len(r.subscribers))
	for subscriber := range r.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	return msg, false, subscribers
}

func (r *campaignRoom) historyBefore(beforeSequenceID int64, limit int) []chatMessage {
	r.mu.Lock()
	defer r.mu.Unlock()

	history := make([]chatMessage, 0, limit)
	for _, msg := range r.messages {
		if msg.SequenceID < beforeSequenceID {
			history = append(history, msg)
		}
	}
	if len(history) > limit {
		history = history[len(history)-limit:]
	}
	return history
}
