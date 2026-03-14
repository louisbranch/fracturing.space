package domain

// OutboxEvent is the worker-facing contract shared by leased outbox event protos.
type OutboxEvent interface {
	GetId() string
	GetEventType() string
	GetPayloadJson() string
	GetAttemptCount() int32
}

// AckOutcome is the worker's source-agnostic processing outcome.
type AckOutcome int

const (
	AckOutcomeUnknown AckOutcome = iota
	AckOutcomeSucceeded
	AckOutcomeRetry
	AckOutcomeDead
)

func (o AckOutcome) String() string {
	switch o {
	case AckOutcomeSucceeded:
		return "succeeded"
	case AckOutcomeRetry:
		return "retry"
	case AckOutcomeDead:
		return "dead"
	default:
		return "unknown"
	}
}
