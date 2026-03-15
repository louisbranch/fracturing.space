package transcript

import "context"

// MessageActor captures the participant identity attached to one human message.
type MessageActor struct {
	ParticipantID string
	Name          string
}

// Message is one persisted human transcript row for the active-play surface.
type Message struct {
	MessageID       string
	CampaignID      string
	SessionID       string
	SequenceID      int64
	SentAt          string
	Actor           MessageActor
	Body            string
	ClientMessageID string
}

// Store persists and replays play-owned human transcript messages.
type Store interface {
	LatestSequence(ctx context.Context, campaignID string, sessionID string) (int64, error)
	AppendMessage(ctx context.Context, campaignID string, sessionID string, actor MessageActor, body string, clientMessageID string) (Message, bool, error)
	HistoryAfter(ctx context.Context, campaignID string, sessionID string, afterSequenceID int64) ([]Message, error)
	HistoryBefore(ctx context.Context, campaignID string, sessionID string, beforeSequenceID int64, limit int) ([]Message, error)
}
