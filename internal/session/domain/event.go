package domain

import "time"

// SessionEventType identifies the type of a session event.
type SessionEventType string

const (
	// SessionEventTypeSessionStarted records the start of a session.
	SessionEventTypeSessionStarted SessionEventType = "SESSION_STARTED"
	// SessionEventTypeSessionEnded records the end of a session.
	SessionEventTypeSessionEnded SessionEventType = "SESSION_ENDED"
	// SessionEventTypeNoteAdded records a GM/player note.
	SessionEventTypeNoteAdded SessionEventType = "NOTE_ADDED"
	// SessionEventTypeActionRollRequested records an action roll request.
	SessionEventTypeActionRollRequested SessionEventType = "ACTION_ROLL_REQUESTED"
	// SessionEventTypeActionRollResolved records an action roll resolution.
	SessionEventTypeActionRollResolved SessionEventType = "ACTION_ROLL_RESOLVED"
	// SessionEventTypeRequestRejected records a rejected request.
	SessionEventTypeRequestRejected SessionEventType = "REQUEST_REJECTED"
)

// SessionEvent captures an immutable session-scoped event.
type SessionEvent struct {
	SessionID     string
	Seq           uint64
	Timestamp     time.Time
	Type          SessionEventType
	RequestID     string
	InvocationID  string
	ParticipantID string
	CharacterID   string
	PayloadJSON   []byte
}

// IsValid reports whether the session event type is supported.
func (t SessionEventType) IsValid() bool {
	switch t {
	case SessionEventTypeSessionStarted,
		SessionEventTypeSessionEnded,
		SessionEventTypeNoteAdded,
		SessionEventTypeActionRollRequested,
		SessionEventTypeActionRollResolved,
		SessionEventTypeRequestRejected:
		return true
	default:
		return false
	}
}
