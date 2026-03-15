package app

import (
	"strings"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
)

const (
	playRealtimeProtocolVersion = 1
	defaultChatHistoryLimit     = 50
	typingTTL                   = 3 * time.Second
)

type playBootstrap struct {
	CampaignID       string                    `json:"campaign_id"`
	Viewer           *gamev1.InteractionViewer `json:"viewer,omitempty"`
	System           playSystem                `json:"system"`
	InteractionState *gamev1.InteractionState  `json:"interaction_state"`
	Chat             playChatSnapshot          `json:"chat"`
	Realtime         playRealtimeConfig        `json:"realtime"`
}

type playSystem struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Name    string `json:"name"`
}

type playRealtimeConfig struct {
	URL             string `json:"url"`
	ProtocolVersion int    `json:"protocol_version"`
}

type playChatSnapshot struct {
	SessionID        string            `json:"session_id"`
	LatestSequenceID int64             `json:"latest_sequence_id"`
	Messages         []playChatMessage `json:"messages"`
	HistoryURL       string            `json:"history_url"`
}

type playChatMessage struct {
	MessageID       string        `json:"message_id"`
	CampaignID      string        `json:"campaign_id"`
	SessionID       string        `json:"session_id"`
	SequenceID      int64         `json:"sequence_id"`
	SentAt          string        `json:"sent_at"`
	Actor           playChatActor `json:"actor"`
	Body            string        `json:"body"`
	ClientMessageID string        `json:"client_message_id,omitempty"`
}

type playChatActor struct {
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
}

type playHistoryResponse struct {
	SessionID string            `json:"session_id"`
	Messages  []playChatMessage `json:"messages"`
}

type playRoomSnapshot struct {
	InteractionState *gamev1.InteractionState `json:"interaction_state"`
	Chat             playChatSnapshot         `json:"chat"`
	LatestGameSeq    uint64                   `json:"latest_game_sequence"`
}

type playWSFrame struct {
	Type      string      `json:"type"`
	RequestID string      `json:"request_id,omitempty"`
	Payload   interface{} `json:"payload,omitempty"`
}

type playWSRawFrame struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id,omitempty"`
	Payload   []byte `json:"payload,omitempty"`
}

type playWSErrorEnvelope struct {
	Error playWSError `json:"error"`
}

type playWSError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

type playWSChatMessageEnvelope struct {
	Message playChatMessage `json:"message"`
}

type playWSTypingPayload struct {
	SessionID     string `json:"session_id,omitempty"`
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
	Active        bool   `json:"active"`
}

type playWSConnectPayload struct {
	CampaignID  string `json:"campaign_id"`
	LastGameSeq uint64 `json:"last_game_seq,omitempty"`
	LastChatSeq int64  `json:"last_chat_seq,omitempty"`
}

type playWSChatSendPayload struct {
	ClientMessageID string `json:"client_message_id"`
	Body            string `json:"body"`
}

type playWSPongPayload struct {
	Timestamp string `json:"timestamp,omitempty"`
}

func transcriptMessageToPayload(message transcript.Message) playChatMessage {
	return playChatMessage{
		MessageID:       strings.TrimSpace(message.MessageID),
		CampaignID:      strings.TrimSpace(message.CampaignID),
		SessionID:       strings.TrimSpace(message.SessionID),
		SequenceID:      message.SequenceID,
		SentAt:          strings.TrimSpace(message.SentAt),
		Actor:           playChatActor{ParticipantID: strings.TrimSpace(message.Actor.ParticipantID), Name: strings.TrimSpace(message.Actor.Name)},
		Body:            strings.TrimSpace(message.Body),
		ClientMessageID: strings.TrimSpace(message.ClientMessageID),
	}
}

func transcriptMessagesToPayload(messages []transcript.Message) []playChatMessage {
	values := make([]playChatMessage, 0, len(messages))
	for _, message := range messages {
		values = append(values, transcriptMessageToPayload(message))
	}
	return values
}
