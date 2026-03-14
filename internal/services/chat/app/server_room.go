package server

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

const aiSessionGrantRefreshLead = 5 * time.Second
const (
	chatStreamSystemLabel  = "System"
	chatStreamTableLabel   = "Table"
	chatStreamControlLabel = "Control"
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

func (s *wsSession) setCommunicationState(ctx communicationContext) {
	state := wsCommunicationState{
		participantID:    strings.TrimSpace(ctx.ParticipantID),
		defaultStreamID:  strings.TrimSpace(ctx.DefaultStreamID),
		defaultPersonaID: strings.TrimSpace(ctx.DefaultPersonaID),
		streamsByID:      make(map[string]chatStream, len(ctx.Streams)),
		personasByID:     make(map[string]chatPersona, len(ctx.Personas)),
	}
	for _, stream := range ctx.Streams {
		if streamID := strings.TrimSpace(stream.StreamID); streamID != "" {
			state.streamsByID[streamID] = stream
		}
	}
	for _, persona := range ctx.Personas {
		if personaID := strings.TrimSpace(persona.PersonaID); personaID != "" {
			state.personasByID[personaID] = persona
		}
	}
	s.mu.Lock()
	s.state = state
	s.mu.Unlock()
}

func (s *wsSession) communicationState() wsCommunicationState {
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

func (h *roomHub) roomIfExists(campaignID string) *campaignRoom {
	h.mu.Lock()
	room := h.rooms[campaignID]
	h.mu.Unlock()
	return room
}

type campaignRoom struct {
	mu                      sync.Mutex
	campaignID              string
	sessionID               string
	gmMode                  string
	aiAgentID               string
	aiAuthEpoch             uint64
	aiSessionGrant          string
	aiGrantExpiresAt        time.Time
	nextSequence            int64
	messagesByStream        map[string][]chatMessage
	messageLog              []chatMessage
	idempotencyBy           map[string]chatMessage
	idempotencyOrder        []string
	activeSessionGate       *chatSessionGate
	activeSessionSpotlight  *chatSessionSpotlight
	aiLastSubmittedSequence int64
	subscribers             map[*wsPeer]roomSubscription
}

type roomSubscription struct {
	visibleStreams map[string]struct{}
}

func newCampaignRoom(campaignID string) *campaignRoom {
	return &campaignRoom{
		campaignID:       campaignID,
		sessionID:        defaultSessionID,
		messagesByStream: make(map[string][]chatMessage),
		idempotencyBy:    make(map[string]chatMessage),
		subscribers:      make(map[*wsPeer]roomSubscription),
	}
}

func (r *campaignRoom) join(peer *wsPeer, streamIDs []string) int64 {
	r.mu.Lock()
	r.subscribers[peer] = roomSubscription{visibleStreams: streamIDSet(streamIDs)}
	latest := r.nextSequence
	r.mu.Unlock()
	return latest
}

func (r *campaignRoom) setSessionID(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	r.mu.Lock()
	if r.sessionID != sessionID {
		r.aiSessionGrant = ""
		r.aiGrantExpiresAt = time.Time{}
		r.activeSessionGate = nil
		r.activeSessionSpotlight = nil
		r.aiLastSubmittedSequence = r.nextSequence
	}
	r.sessionID = sessionID
	r.mu.Unlock()
}

func (r *campaignRoom) setAIBinding(gmMode string, aiAgentID string) {
	r.mu.Lock()
	normalizedMode := strings.ToLower(strings.TrimSpace(gmMode))
	normalizedAgentID := strings.TrimSpace(aiAgentID)
	if r.gmMode != normalizedMode || r.aiAgentID != normalizedAgentID {
		r.aiSessionGrant = ""
		r.aiGrantExpiresAt = time.Time{}
		r.aiLastSubmittedSequence = r.nextSequence
	}
	r.gmMode = normalizedMode
	r.aiAgentID = normalizedAgentID
	r.mu.Unlock()
}

func (r *campaignRoom) setControlState(gate *chatSessionGate, spotlight *chatSessionSpotlight) {
	r.mu.Lock()
	r.activeSessionGate = cloneChatSessionGate(gate)
	r.activeSessionSpotlight = cloneChatSessionSpotlight(spotlight)
	r.mu.Unlock()
}

func (r *campaignRoom) activeSessionGateState() *chatSessionGate {
	r.mu.Lock()
	defer r.mu.Unlock()
	return cloneChatSessionGate(r.activeSessionGate)
}

func (r *campaignRoom) aiRelayEnabled() bool {
	r.mu.Lock()
	enabled := r.aiAgentID != "" && (r.gmMode == "ai" || r.gmMode == "hybrid" || r.gmMode == "gm_mode_ai" || r.gmMode == "gm_mode_hybrid")
	r.mu.Unlock()
	return enabled
}

func (r *campaignRoom) aiAgentIDValue() string {
	r.mu.Lock()
	value := r.aiAgentID
	r.mu.Unlock()
	return value
}

func (r *campaignRoom) gmModeValue() string {
	r.mu.Lock()
	value := r.gmMode
	r.mu.Unlock()
	return value
}

func (r *campaignRoom) setAISessionGrant(token string, authEpoch uint64, expiresAt time.Time) {
	r.mu.Lock()
	r.aiSessionGrant = strings.TrimSpace(token)
	r.aiAuthEpoch = authEpoch
	if expiresAt.IsZero() {
		r.aiGrantExpiresAt = time.Time{}
	} else {
		r.aiGrantExpiresAt = expiresAt.UTC()
	}
	r.mu.Unlock()
}

func (r *campaignRoom) clearAISessionGrant() {
	r.mu.Lock()
	r.aiSessionGrant = ""
	r.aiGrantExpiresAt = time.Time{}
	r.mu.Unlock()
}

func (r *campaignRoom) aiSessionGrantValue() string {
	r.mu.Lock()
	value := r.aiSessionGrant
	r.mu.Unlock()
	return value
}

func (r *campaignRoom) aiRelayReady() bool {
	r.mu.Lock()
	enabled := r.aiAgentID != "" &&
		(r.gmMode == "ai" || r.gmMode == "hybrid" || r.gmMode == "gm_mode_ai" || r.gmMode == "gm_mode_hybrid") &&
		strings.TrimSpace(r.aiSessionGrant) != ""
	if enabled && !r.aiGrantExpiresAt.IsZero() {
		refreshAt := r.aiGrantExpiresAt.Add(-aiSessionGrantRefreshLead)
		if !time.Now().UTC().Before(refreshAt) {
			r.aiSessionGrant = ""
			r.aiGrantExpiresAt = time.Time{}
			enabled = false
		}
	}
	r.mu.Unlock()
	return enabled
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

func (r *campaignRoom) subscribersSnapshot() []*wsPeer {
	r.mu.Lock()
	subscribers := make([]*wsPeer, 0, len(r.subscribers))
	for subscriber := range r.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	r.mu.Unlock()
	return subscribers
}

func (r *campaignRoom) appendMessage(actor messageActor, body string, clientMessageID string, streamID string) (chatMessage, bool, []*wsPeer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.idempotencyBy[clientMessageID]; ok {
		return existing, true, nil
	}

	r.nextSequence++
	actor.ParticipantID = strings.TrimSpace(actor.ParticipantID)
	if actor.ParticipantID == "" {
		actor.ParticipantID = "participant"
	}
	if strings.TrimSpace(actor.Name) == "" {
		actor.Name = actor.ParticipantID
	}
	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		streamID = chatDefaultStreamID(r.campaignID)
	}
	msg := chatMessage{
		MessageID:       fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		CampaignID:      r.campaignID,
		SessionID:       r.sessionID,
		SequenceID:      r.nextSequence,
		SentAt:          time.Now().UTC().Format(time.RFC3339),
		Kind:            "text",
		StreamID:        streamID,
		Actor:           actor,
		Body:            body,
		ClientMessageID: clientMessageID,
	}

	r.messagesByStream[streamID] = append(r.messagesByStream[streamID], msg)
	if len(r.messagesByStream[streamID]) > maxRoomMessages {
		r.messagesByStream[streamID] = r.messagesByStream[streamID][len(r.messagesByStream[streamID])-maxRoomMessages:]
	}
	r.messageLog = append(r.messageLog, msg)
	if len(r.messageLog) > maxRoomMessages {
		r.messageLog = r.messageLog[len(r.messageLog)-maxRoomMessages:]
	}

	r.idempotencyBy[clientMessageID] = msg
	r.idempotencyOrder = append(r.idempotencyOrder, clientMessageID)
	if len(r.idempotencyOrder) > maxIdempotencyRecord {
		evict := r.idempotencyOrder[0]
		r.idempotencyOrder = r.idempotencyOrder[1:]
		delete(r.idempotencyBy, evict)
	}

	return msg, false, r.subscribersForStreamLocked(streamID)
}

func (r *campaignRoom) appendAIGMMessage(sessionID string, body string, correlationMessageID string) (chatMessage, bool, []*wsPeer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := ""
	if strings.TrimSpace(correlationMessageID) != "" {
		key = "ai:" + strings.TrimSpace(correlationMessageID)
		if existing, ok := r.idempotencyBy[key]; ok {
			return existing, true, nil
		}
	}

	r.nextSequence++
	normalizedSessionID := strings.TrimSpace(sessionID)
	if normalizedSessionID == "" {
		normalizedSessionID = r.sessionID
	}
	if normalizedSessionID == "" {
		normalizedSessionID = defaultSessionID
	}
	msg := chatMessage{
		MessageID:  fmt.Sprintf("ai_%d", time.Now().UnixNano()),
		CampaignID: r.campaignID,
		SessionID:  normalizedSessionID,
		SequenceID: r.nextSequence,
		SentAt:     time.Now().UTC().Format(time.RFC3339),
		Kind:       "ai",
		StreamID:   chatDefaultStreamID(r.campaignID),
		Actor: messageActor{
			ParticipantID: "ai_gm",
			PersonaID:     "participant:ai_gm",
			Mode:          "participant",
			Name:          "AI GM",
		},
		Body: body,
	}
	r.messagesByStream[msg.StreamID] = append(r.messagesByStream[msg.StreamID], msg)
	if len(r.messagesByStream[msg.StreamID]) > maxRoomMessages {
		r.messagesByStream[msg.StreamID] = r.messagesByStream[msg.StreamID][len(r.messagesByStream[msg.StreamID])-maxRoomMessages:]
	}
	r.messageLog = append(r.messageLog, msg)
	if len(r.messageLog) > maxRoomMessages {
		r.messageLog = r.messageLog[len(r.messageLog)-maxRoomMessages:]
	}
	if key != "" {
		r.idempotencyBy[key] = msg
		r.idempotencyOrder = append(r.idempotencyOrder, key)
		if len(r.idempotencyOrder) > maxIdempotencyRecord {
			evict := r.idempotencyOrder[0]
			r.idempotencyOrder = r.idempotencyOrder[1:]
			delete(r.idempotencyBy, evict)
		}
	}
	return msg, false, r.subscribersForStreamLocked(msg.StreamID)
}

type aiTurnSubmission struct {
	body                 string
	correlationMessageID string
	highestSequenceID    int64
}

func (r *campaignRoom) pendingAITurnSubmission(handoffReason string) (aiTurnSubmission, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	selected := make([]chatMessage, 0, maxAITurnMessages)
	var correlationMessageID string
	var highestSequenceID int64
	for _, msg := range r.messageLog {
		if msg.SequenceID <= r.aiLastSubmittedSequence {
			continue
		}
		if msg.Kind != "text" {
			continue
		}
		if msg.StreamID == chatSystemStreamID(r.campaignID) || msg.StreamID == chatControlStreamID(r.campaignID) {
			continue
		}
		selected = append(selected, msg)
		if len(selected) > maxAITurnMessages {
			selected = selected[len(selected)-maxAITurnMessages:]
		}
		correlationMessageID = msg.MessageID
		if msg.SequenceID > highestSequenceID {
			highestSequenceID = msg.SequenceID
		}
	}

	reason := strings.TrimSpace(handoffReason)
	if len(selected) == 0 && reason == "" {
		return aiTurnSubmission{}, false
	}

	var builder strings.Builder
	builder.WriteString("GM handoff requested.\n\n")
	if len(selected) > 0 {
		builder.WriteString("Recent participant transcript:\n")
		for _, msg := range selected {
			line := fmt.Sprintf("[%s] %s: %s\n", aiTurnStreamLabel(r.campaignID, msg.StreamID), aiTurnActorLabel(msg.Actor), msg.Body)
			if builder.Len()+len(line) > maxAITurnBodyBytes {
				break
			}
			builder.WriteString(line)
		}
	}
	if reason != "" {
		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString("Handoff reason:\n")
		remaining := maxAITurnBodyBytes - builder.Len()
		if remaining > 0 && len(reason) > remaining {
			reason = reason[:remaining]
		}
		builder.WriteString(reason)
	}

	body := strings.TrimSpace(builder.String())
	if body == "" {
		return aiTurnSubmission{}, false
	}
	return aiTurnSubmission{
		body:                 body,
		correlationMessageID: correlationMessageID,
		highestSequenceID:    highestSequenceID,
	}, true
}

func (r *campaignRoom) markAITurnSubmitted(highestSequenceID int64) {
	if highestSequenceID <= 0 {
		return
	}
	r.mu.Lock()
	if highestSequenceID > r.aiLastSubmittedSequence {
		r.aiLastSubmittedSequence = highestSequenceID
	}
	r.mu.Unlock()
}

func (r *campaignRoom) historyBefore(streamID string, beforeSequenceID int64, limit int) []chatMessage {
	r.mu.Lock()
	defer r.mu.Unlock()

	streamID = strings.TrimSpace(streamID)
	if streamID == "" {
		streamID = chatDefaultStreamID(r.campaignID)
	}

	history := make([]chatMessage, 0, limit)
	for _, msg := range r.messagesByStream[streamID] {
		if msg.SequenceID < beforeSequenceID {
			history = append(history, msg)
		}
	}
	if len(history) > limit {
		history = history[len(history)-limit:]
	}
	return history
}

func (r *campaignRoom) subscribersForStreamLocked(streamID string) []*wsPeer {
	subscribers := make([]*wsPeer, 0, len(r.subscribers))
	for subscriber, subscription := range r.subscribers {
		if _, ok := subscription.visibleStreams[streamID]; ok {
			subscribers = append(subscribers, subscriber)
		}
	}
	return subscribers
}

func streamIDSet(streamIDs []string) map[string]struct{} {
	set := make(map[string]struct{}, len(streamIDs))
	for _, streamID := range streamIDs {
		normalized := strings.TrimSpace(streamID)
		if normalized == "" {
			continue
		}
		set[normalized] = struct{}{}
	}
	return set
}

func chatSystemStreamID(campaignID string) string {
	return "campaign:" + campaignID + ":system"
}

func chatDefaultStreamID(campaignID string) string {
	return "campaign:" + campaignID + ":table"
}

func chatControlStreamID(campaignID string) string {
	return "campaign:" + campaignID + ":control"
}

func aiTurnStreamLabel(campaignID string, streamID string) string {
	switch streamID {
	case chatDefaultStreamID(campaignID):
		return "table"
	case chatSystemStreamID(campaignID):
		return "system"
	case chatControlStreamID(campaignID):
		return "control"
	default:
		return strings.TrimSpace(streamID)
	}
}

func aiTurnActorLabel(actor messageActor) string {
	name := strings.TrimSpace(actor.Name)
	if name == "" {
		name = strings.TrimSpace(actor.ParticipantID)
	}
	if strings.TrimSpace(actor.Mode) == "character" {
		return name + " (character)"
	}
	return name
}

func cloneChatSessionGate(gate *chatSessionGate) *chatSessionGate {
	if gate == nil {
		return nil
	}
	cloned := &chatSessionGate{
		GateID:   gate.GateID,
		GateType: gate.GateType,
		Status:   gate.Status,
		Reason:   gate.Reason,
	}
	if len(gate.Metadata) > 0 {
		cloned.Metadata = make(map[string]any, len(gate.Metadata))
		for key, value := range gate.Metadata {
			cloned.Metadata[key] = value
		}
	}
	if len(gate.Progress) > 0 {
		cloned.Progress = make(map[string]any, len(gate.Progress))
		for key, value := range gate.Progress {
			cloned.Progress[key] = value
		}
	}
	return cloned
}

func cloneChatSessionSpotlight(spotlight *chatSessionSpotlight) *chatSessionSpotlight {
	if spotlight == nil {
		return nil
	}
	return &chatSessionSpotlight{
		Type:        spotlight.Type,
		CharacterID: spotlight.CharacterID,
	}
}
