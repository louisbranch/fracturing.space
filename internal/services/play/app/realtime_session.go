package app

import (
	"encoding/json"
	"strings"
	"sync"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
)

// realtimeSession tracks one websocket connection's campaign identity and
// typing lifecycle so hub and room code stay focused on orchestration.
type realtimeSession struct {
	userID string
	peer   *wsPeer

	mu               sync.Mutex
	room             *campaignRoom
	campaignID       string
	participantID    string
	participantName  string
	activeSessionID  string
	chatTypingTimer  realtimeTimer
	draftTypingTimer realtimeTimer
}

type wsPeer struct {
	mu      sync.Mutex
	encoder *json.Encoder
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
	var timer *realtimeTimer
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
	*timer = room.hub.runtime.newTimer(room.hub.runtime.typingTTL, func() {
		room.broadcastFrame(wsFrame{Type: frameType, Payload: mustJSON(playprotocol.TypingEvent{
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
		Payload: mustJSON(playprotocol.ErrorEnvelope{Error: playprotocol.ErrorPayload{
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
