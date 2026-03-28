package app

import (
	"encoding/json"
	"strings"
	"sync"

	playprotocol "github.com/louisbranch/fracturing.space/internal/services/play/protocol"
)

// realtimeSession tracks one websocket connection's campaign identity and
// typing lifecycle so hub and room code stay focused on orchestration.
type realtimeSession struct {
	userID string
	peer   *wsPeer

	mu              sync.Mutex
	room            *campaignRoom
	campaignID      string
	participantID   string
	participantName string
	activeSessionID string
	typingTimer     realtimeTimer
}

type wsPeer struct {
	mu      sync.Mutex
	encoder *json.Encoder
}

func (s *realtimeSession) attach(room *campaignRoom, state playprotocol.InteractionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.room = room
	s.campaignID = strings.TrimSpace(room.campaignID)
	if state.Viewer != nil {
		s.participantID = strings.TrimSpace(state.Viewer.ParticipantID)
		s.participantName = strings.TrimSpace(state.Viewer.Name)
	} else {
		s.participantID = ""
		s.participantName = ""
	}
	if state.ActiveSession != nil {
		s.activeSessionID = strings.TrimSpace(state.ActiveSession.SessionID)
	} else {
		s.activeSessionID = ""
	}
}

// refreshCampaignState updates campaign-level fields (active session) during
// broadcast cycles without overwriting the per-session viewer identity that was
// established during the initial connect handshake.
func (s *realtimeSession) refreshCampaignState(state playprotocol.InteractionState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state.ActiveSession != nil {
		s.activeSessionID = strings.TrimSpace(state.ActiveSession.SessionID)
	} else {
		s.activeSessionID = ""
	}
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

// sessionChatIdentity holds the participant context needed to post a chat
// message so callers don't decipher positional return values.
type sessionChatIdentity struct {
	CampaignID      string
	SessionID       string
	ParticipantID   string
	ParticipantName string
}

func (s *realtimeSession) chatIdentity() (sessionChatIdentity, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.room == nil || s.campaignID == "" || s.activeSessionID == "" || s.participantID == "" {
		return sessionChatIdentity{}, false
	}
	return sessionChatIdentity{
		CampaignID:      s.campaignID,
		SessionID:       s.activeSessionID,
		ParticipantID:   s.participantID,
		ParticipantName: s.participantName,
	}, true
}

func (s *realtimeSession) resetTypingTimer(active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.typingTimer != nil {
		s.typingTimer.Stop()
		s.typingTimer = nil
	}
	if !active || s.room == nil {
		return
	}
	room := s.room
	sessionID := s.activeSessionID
	participantID := s.participantID
	participantName := s.participantName
	s.typingTimer = room.hub.runtime.newTimer(room.hub.runtime.typingTTL, func() {
		room.broadcastFrame(wsFrame{Type: FrameTyping, Payload: mustJSON(playprotocol.TypingEvent{
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
		Type:      FrameError,
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
