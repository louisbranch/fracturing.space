package protocol

import (
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
)

const RealtimeProtocolVersion = 1

type Bootstrap struct {
	CampaignID       string                    `json:"campaign_id"`
	Viewer           *gamev1.InteractionViewer `json:"viewer,omitempty"`
	System           System                    `json:"system"`
	InteractionState *gamev1.InteractionState  `json:"interaction_state"`
	Chat             ChatSnapshot              `json:"chat"`
	Realtime         RealtimeConfig            `json:"realtime"`
}

type System struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Name    string `json:"name"`
}

type RealtimeConfig struct {
	URL             string `json:"url"`
	ProtocolVersion int    `json:"protocol_version"`
}

type ChatSnapshot struct {
	SessionID        string        `json:"session_id"`
	LatestSequenceID int64         `json:"latest_sequence_id"`
	Messages         []ChatMessage `json:"messages"`
	HistoryURL       string        `json:"history_url"`
}

type ChatMessage struct {
	MessageID       string    `json:"message_id"`
	CampaignID      string    `json:"campaign_id"`
	SessionID       string    `json:"session_id"`
	SequenceID      int64     `json:"sequence_id"`
	SentAt          string    `json:"sent_at"`
	Actor           ChatActor `json:"actor"`
	Body            string    `json:"body"`
	ClientMessageID string    `json:"client_message_id,omitempty"`
}

type ChatActor struct {
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
}

type HistoryResponse struct {
	SessionID string        `json:"session_id"`
	Messages  []ChatMessage `json:"messages"`
}

type RoomSnapshot struct {
	InteractionState *gamev1.InteractionState `json:"interaction_state"`
	Chat             ChatSnapshot             `json:"chat"`
	LatestGameSeq    uint64                   `json:"latest_game_sequence"`
}

type WSFrame struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Payload   any    `json:"payload,omitempty"`
}

type WSRawFrame struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Payload   []byte `json:"payload,omitempty"`
}

type ErrorEnvelope struct {
	Error ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

type ChatMessageEnvelope struct {
	Message ChatMessage `json:"message"`
}

type TypingEvent struct {
	SessionID     string `json:"session_id,omitempty"`
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
	Active        bool   `json:"active"`
}

type ConnectRequest struct {
	CampaignID  string `json:"campaign_id"`
	LastGameSeq uint64 `json:"last_game_seq,omitempty"`
	LastChatSeq int64  `json:"last_chat_seq,omitempty"`
}

type ChatSendRequest struct {
	ClientMessageID string `json:"client_message_id"`
	Body            string `json:"body"`
}

type Pong struct {
	Timestamp string `json:"timestamp,omitempty"`
}

func TranscriptMessage(message transcript.Message) ChatMessage {
	return ChatMessage{
		MessageID:       strings.TrimSpace(message.MessageID),
		CampaignID:      strings.TrimSpace(message.CampaignID),
		SessionID:       strings.TrimSpace(message.SessionID),
		SequenceID:      message.SequenceID,
		SentAt:          strings.TrimSpace(message.SentAt),
		Actor:           ChatActor{ParticipantID: strings.TrimSpace(message.Actor.ParticipantID), Name: strings.TrimSpace(message.Actor.Name)},
		Body:            strings.TrimSpace(message.Body),
		ClientMessageID: strings.TrimSpace(message.ClientMessageID),
	}
}

func TranscriptMessages(messages []transcript.Message) []ChatMessage {
	values := make([]ChatMessage, 0, len(messages))
	for _, message := range messages {
		values = append(values, TranscriptMessage(message))
	}
	return values
}
