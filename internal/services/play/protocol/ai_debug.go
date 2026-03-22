package protocol

import (
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
)

// AIDebugTurnsPage is the play-owned browser DTO for paginated AI turn history.
type AIDebugTurnsPage struct {
	Turns         []AIDebugTurnSummary `json:"turns"`
	NextPageToken string               `json:"next_page_token,omitempty"`
}

// AIDebugUsage records provider usage counters when the upstream reported them.
type AIDebugUsage struct {
	InputTokens     int32 `json:"input_tokens,omitempty"`
	OutputTokens    int32 `json:"output_tokens,omitempty"`
	ReasoningTokens int32 `json:"reasoning_tokens,omitempty"`
	TotalTokens     int32 `json:"total_tokens,omitempty"`
}

// AIDebugTurnSummary is the collapsed browser summary for one GM turn.
type AIDebugTurnSummary struct {
	ID            string        `json:"id"`
	TurnToken     string        `json:"turn_token,omitempty"`
	ParticipantID string        `json:"participant_id,omitempty"`
	Provider      string        `json:"provider,omitempty"`
	Model         string        `json:"model,omitempty"`
	Status        string        `json:"status,omitempty"`
	LastError     string        `json:"last_error,omitempty"`
	Usage         *AIDebugUsage `json:"usage,omitempty"`
	StartedAt     string        `json:"started_at,omitempty"`
	UpdatedAt     string        `json:"updated_at,omitempty"`
	CompletedAt   string        `json:"completed_at,omitempty"`
	EntryCount    int32         `json:"entry_count,omitempty"`
}

// AIDebugEntry is one ordered model/tool event rendered by the debug panel.
type AIDebugEntry struct {
	Sequence         int32         `json:"sequence"`
	Kind             string        `json:"kind,omitempty"`
	ToolName         string        `json:"tool_name,omitempty"`
	Payload          string        `json:"payload,omitempty"`
	PayloadTruncated bool          `json:"payload_truncated,omitempty"`
	CallID           string        `json:"call_id,omitempty"`
	ResponseID       string        `json:"response_id,omitempty"`
	IsError          bool          `json:"is_error,omitempty"`
	CreatedAt        string        `json:"created_at,omitempty"`
	Usage            *AIDebugUsage `json:"usage,omitempty"`
}

// AIDebugTurn is the full detail response for one debug turn.
type AIDebugTurn struct {
	AIDebugTurnSummary
	Entries []AIDebugEntry `json:"entries"`
}

// AIDebugTurnUpdate is the play-owned websocket payload for one live AI debug delta.
type AIDebugTurnUpdate struct {
	Turn            AIDebugTurnSummary `json:"turn"`
	AppendedEntries []AIDebugEntry     `json:"appended_entries"`
}

// AIDebugTurnsPageFromProto maps AI debug list RPCs to the play browser contract.
func AIDebugTurnsPageFromProto(resp *aiv1.ListCampaignDebugTurnsResponse) AIDebugTurnsPage {
	if resp == nil {
		return AIDebugTurnsPage{Turns: []AIDebugTurnSummary{}}
	}
	page := AIDebugTurnsPage{
		Turns:         make([]AIDebugTurnSummary, 0, len(resp.GetTurns())),
		NextPageToken: strings.TrimSpace(resp.GetNextPageToken()),
	}
	for _, turn := range resp.GetTurns() {
		page.Turns = append(page.Turns, aiDebugTurnSummaryFromProto(turn))
	}
	return page
}

// AIDebugTurnFromProto maps one AI debug turn detail RPC to the play browser contract.
func AIDebugTurnFromProto(turn *aiv1.CampaignDebugTurn) AIDebugTurn {
	if turn == nil {
		return AIDebugTurn{Entries: []AIDebugEntry{}}
	}
	result := AIDebugTurn{
		AIDebugTurnSummary: AIDebugTurnSummary{
			ID:            strings.TrimSpace(turn.GetId()),
			TurnToken:     strings.TrimSpace(turn.GetTurnToken()),
			ParticipantID: strings.TrimSpace(turn.GetParticipantId()),
			Provider:      aiProviderString(turn.GetProvider()),
			Model:         strings.TrimSpace(turn.GetModel()),
			Status:        aiDebugTurnStatusString(turn.GetStatus()),
			LastError:     strings.TrimSpace(turn.GetLastError()),
			Usage:         aiDebugUsageFromProto(turn.GetUsage()),
			StartedAt:     formatTimestamp(turn.GetStartedAt()),
			UpdatedAt:     formatTimestamp(turn.GetUpdatedAt()),
			CompletedAt:   formatTimestamp(turn.GetCompletedAt()),
		},
		Entries: make([]AIDebugEntry, 0, len(turn.GetEntries())),
	}
	for _, entry := range turn.GetEntries() {
		result.Entries = append(result.Entries, aiDebugEntryFromProto(entry))
	}
	result.EntryCount = int32(len(result.Entries))
	return result
}

// AIDebugTurnUpdateFromProto maps one AI live-update proto to the play websocket contract.
func AIDebugTurnUpdateFromProto(update *aiv1.CampaignDebugTurnUpdate) AIDebugTurnUpdate {
	result := AIDebugTurnUpdate{AppendedEntries: []AIDebugEntry{}}
	if update == nil {
		return result
	}
	result.Turn = aiDebugTurnSummaryFromProto(update.GetTurn())
	if len(update.GetAppendedEntries()) == 0 {
		return result
	}
	result.AppendedEntries = make([]AIDebugEntry, 0, len(update.GetAppendedEntries()))
	for _, entry := range update.GetAppendedEntries() {
		result.AppendedEntries = append(result.AppendedEntries, aiDebugEntryFromProto(entry))
	}
	return result
}

func aiDebugTurnSummaryFromProto(turn *aiv1.CampaignDebugTurnSummary) AIDebugTurnSummary {
	if turn == nil {
		return AIDebugTurnSummary{}
	}
	return AIDebugTurnSummary{
		ID:            strings.TrimSpace(turn.GetId()),
		TurnToken:     strings.TrimSpace(turn.GetTurnToken()),
		ParticipantID: strings.TrimSpace(turn.GetParticipantId()),
		Provider:      aiProviderString(turn.GetProvider()),
		Model:         strings.TrimSpace(turn.GetModel()),
		Status:        aiDebugTurnStatusString(turn.GetStatus()),
		LastError:     strings.TrimSpace(turn.GetLastError()),
		Usage:         aiDebugUsageFromProto(turn.GetUsage()),
		StartedAt:     formatTimestamp(turn.GetStartedAt()),
		UpdatedAt:     formatTimestamp(turn.GetUpdatedAt()),
		CompletedAt:   formatTimestamp(turn.GetCompletedAt()),
		EntryCount:    turn.GetEntryCount(),
	}
}

func aiDebugEntryFromProto(entry *aiv1.CampaignDebugEntry) AIDebugEntry {
	if entry == nil {
		return AIDebugEntry{}
	}
	return AIDebugEntry{
		Sequence:         entry.GetSequence(),
		Kind:             aiDebugEntryKindString(entry.GetKind()),
		ToolName:         strings.TrimSpace(entry.GetToolName()),
		Payload:          entry.GetPayload(),
		PayloadTruncated: entry.GetPayloadTruncated(),
		CallID:           strings.TrimSpace(entry.GetCallId()),
		ResponseID:       strings.TrimSpace(entry.GetResponseId()),
		IsError:          entry.GetIsError(),
		CreatedAt:        formatTimestamp(entry.GetCreatedAt()),
		Usage:            aiDebugUsageFromProto(entry.GetUsage()),
	}
}

func aiDebugUsageFromProto(usage *aiv1.Usage) *AIDebugUsage {
	if usage == nil {
		return nil
	}
	result := &AIDebugUsage{
		InputTokens:     usage.GetInputTokens(),
		OutputTokens:    usage.GetOutputTokens(),
		ReasoningTokens: usage.GetReasoningTokens(),
		TotalTokens:     usage.GetTotalTokens(),
	}
	if result.InputTokens == 0 && result.OutputTokens == 0 && result.ReasoningTokens == 0 && result.TotalTokens == 0 {
		return nil
	}
	return result
}

func aiProviderString(value aiv1.Provider) string {
	switch value {
	case aiv1.Provider_PROVIDER_OPENAI:
		return "openai"
	default:
		return ""
	}
}

func aiDebugTurnStatusString(value aiv1.CampaignDebugTurnStatus) string {
	switch value {
	case aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_RUNNING:
		return "running"
	case aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_SUCCEEDED:
		return "succeeded"
	case aiv1.CampaignDebugTurnStatus_CAMPAIGN_DEBUG_TURN_STATUS_FAILED:
		return "failed"
	default:
		return ""
	}
}

func aiDebugEntryKindString(value aiv1.CampaignDebugEntryKind) string {
	switch value {
	case aiv1.CampaignDebugEntryKind_CAMPAIGN_DEBUG_ENTRY_KIND_MODEL_RESPONSE:
		return "model_response"
	case aiv1.CampaignDebugEntryKind_CAMPAIGN_DEBUG_ENTRY_KIND_TOOL_CALL:
		return "tool_call"
	case aiv1.CampaignDebugEntryKind_CAMPAIGN_DEBUG_ENTRY_KIND_TOOL_RESULT:
		return "tool_result"
	default:
		return ""
	}
}
