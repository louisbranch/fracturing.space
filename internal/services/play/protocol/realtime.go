package protocol

type ErrorEnvelope struct {
	Error ErrorPayload `json:"error"`
}

type ErrorPayload struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
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

type Pong struct {
	Timestamp string `json:"timestamp,omitempty"`
}
