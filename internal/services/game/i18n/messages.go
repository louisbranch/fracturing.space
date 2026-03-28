package i18n

import _ "github.com/louisbranch/fracturing.space/internal/platform/i18n/catalog"

// MessageContract pairs a stable localization key with the English fallback
// that callers should use when the catalog cannot satisfy the request.
type MessageContract struct {
	Key      string
	Fallback string
}

var (
	// ParticipantDefaultUnknownName is the translated fallback for users with no
	// available name.
	ParticipantDefaultUnknownName = MessageContract{
		Key:      "game.participant.default_unknown_name",
		Fallback: "Mysterious Person",
	}
	// ParticipantDefaultAIName is the translated fallback for AI participants.
	ParticipantDefaultAIName = MessageContract{
		Key:      "game.participant.default_ai_name",
		Fallback: "Narrator",
	}
	// SessionDefaultName is the translated fallback for auto-named sessions.
	SessionDefaultName = MessageContract{
		Key:      "game.session.default_name",
		Fallback: "Session %d",
	}
)

var messageContracts = []MessageContract{
	ParticipantDefaultUnknownName,
	ParticipantDefaultAIName,
	SessionDefaultName,
}

// Contracts returns the canonical game-localized message contracts. New
// game-scoped localized copy should be added here so package tests enforce
// catalog coverage automatically.
func Contracts() []MessageContract {
	return append([]MessageContract(nil), messageContracts...)
}
