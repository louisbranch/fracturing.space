package protocol

import (
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
)

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

type ChatMessageEnvelope struct {
	Message ChatMessage `json:"message"`
}

type ChatSendRequest struct {
	ClientMessageID string `json:"client_message_id"`
	Body            string `json:"body"`
}

type RoomSnapshot struct {
	InteractionState           InteractionState               `json:"interaction_state"`
	Participants               []Participant                  `json:"participants"`
	CharacterInspectionCatalog map[string]CharacterInspection `json:"character_inspection_catalog"`
	Chat                       ChatSnapshot                   `json:"chat"`
	LatestGameSeq              uint64                         `json:"latest_game_sequence"`
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
