package debugtrace

import (
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
)

// Status represents one persisted campaign debug turn lifecycle state.
type Status string

const (
	// StatusRunning indicates the AI GM turn is currently executing.
	StatusRunning Status = "running"
	// StatusSucceeded indicates the AI GM turn finished successfully.
	StatusSucceeded Status = "succeeded"
	// StatusFailed indicates the AI GM turn terminated with an error.
	StatusFailed Status = "failed"
)

// ParseStatus normalizes one persisted turn status string.
func ParseStatus(raw string) Status {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(StatusRunning):
		return StatusRunning
	case string(StatusSucceeded):
		return StatusSucceeded
	case string(StatusFailed):
		return StatusFailed
	default:
		return ""
	}
}

// EntryKind identifies one ordered trace event shape.
type EntryKind string

const (
	// EntryKindModelResponse records one model text response.
	EntryKindModelResponse EntryKind = "model_response"
	// EntryKindToolCall records one provider-requested tool call.
	EntryKindToolCall EntryKind = "tool_call"
	// EntryKindToolResult records one tool execution result returned to the model.
	EntryKindToolResult EntryKind = "tool_result"
)

// ParseEntryKind normalizes one persisted entry kind.
func ParseEntryKind(raw string) EntryKind {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(EntryKindModelResponse):
		return EntryKindModelResponse
	case string(EntryKindToolCall):
		return EntryKindToolCall
	case string(EntryKindToolResult):
		return EntryKindToolResult
	default:
		return ""
	}
}

// Turn is one persisted AI GM turn trace summary.
type Turn struct {
	ID            string
	CampaignID    string
	SessionID     string
	TurnToken     string
	ParticipantID string
	Provider      provider.Provider
	Model         string
	Status        Status
	LastError     string
	Usage         provider.Usage
	StartedAt     time.Time
	UpdatedAt     time.Time
	CompletedAt   *time.Time
	EntryCount    int
}

// Entry is one ordered trace event within a turn.
type Entry struct {
	TurnID           string
	Sequence         int
	Kind             EntryKind
	ToolName         string
	Payload          string
	PayloadTruncated bool
	CallID           string
	ResponseID       string
	IsError          bool
	CreatedAt        time.Time
	Usage            provider.Usage
}

// Page contains one paginated slice of turn summaries.
type Page struct {
	Turns         []Turn
	NextPageToken string
}
