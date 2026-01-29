package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	sessionv1 "github.com/louisbranch/duality-engine/api/gen/go/session/v1"
	"github.com/louisbranch/duality-engine/internal/grpcmeta"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// SessionStartInput represents the MCP tool input for starting a session.
type SessionStartInput struct {
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string `json:"name,omitempty" jsonschema:"optional free-form name for the session"`
}

// SessionStartResult represents the MCP tool output for starting a session.
type SessionStartResult struct {
	ID         string `json:"id" jsonschema:"session identifier"`
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string `json:"name" jsonschema:"session name"`
	Status     string `json:"status" jsonschema:"session status (ACTIVE, ENDED)"`
	StartedAt  string `json:"started_at" jsonschema:"RFC3339 timestamp when session was started"`
	UpdatedAt  string `json:"updated_at" jsonschema:"RFC3339 timestamp when session was last updated"`
	EndedAt    string `json:"ended_at,omitempty" jsonschema:"RFC3339 timestamp when session ended, if applicable"`
}

// SessionStartTool defines the MCP tool schema for starting a session.
func SessionStartTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "session_start",
		Description: "Starts a new session for a campaign. Enforces at most one ACTIVE session per campaign.",
	}
}

// SessionStartHandler executes a session start request.
func SessionStartHandler(client sessionv1.SessionServiceClient) mcp.ToolHandlerFor[SessionStartInput, SessionStartResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SessionStartInput) (*mcp.CallToolResult, SessionStartResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, SessionStartResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, SessionStartResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD

		response, err := client.StartSession(callCtx, &sessionv1.StartSessionRequest{
			CampaignId: input.CampaignID,
			Name:       input.Name,
		}, grpc.Header(&header))
		if err != nil {
			return nil, SessionStartResult{}, fmt.Errorf("session start failed: %w", err)
		}
		if response == nil || response.Session == nil {
			return nil, SessionStartResult{}, fmt.Errorf("session start response is missing")
		}

		result := SessionStartResult{
			ID:         response.Session.GetId(),
			CampaignID: response.Session.GetCampaignId(),
			Name:       response.Session.GetName(),
			Status:     sessionStatusToString(response.Session.GetStatus()),
			StartedAt:  formatTimestamp(response.Session.GetStartedAt()),
			UpdatedAt:  formatTimestamp(response.Session.GetUpdatedAt()),
		}

		if response.Session.GetEndedAt() != nil {
			result.EndedAt = formatTimestamp(response.Session.GetEndedAt())
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// SessionEndInput represents the MCP tool input for ending a session.
type SessionEndInput struct {
	CampaignID string `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	SessionID  string `json:"session_id,omitempty" jsonschema:"session identifier (defaults to context)"`
}

// SessionEndResult represents the MCP tool output for ending a session.
type SessionEndResult struct {
	ID         string `json:"id" jsonschema:"session identifier"`
	CampaignID string `json:"campaign_id" jsonschema:"campaign identifier"`
	Name       string `json:"name" jsonschema:"session name"`
	Status     string `json:"status" jsonschema:"session status (ACTIVE, ENDED)"`
	StartedAt  string `json:"started_at" jsonschema:"RFC3339 timestamp when session was started"`
	UpdatedAt  string `json:"updated_at" jsonschema:"RFC3339 timestamp when session was last updated"`
	EndedAt    string `json:"ended_at,omitempty" jsonschema:"RFC3339 timestamp when session ended, if applicable"`
}

// SessionEndTool defines the MCP tool schema for ending a session.
func SessionEndTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "session_end",
		Description: "Ends a session for a campaign and clears the active session pointer.",
	}
}

// SessionEndHandler executes a session end request.
func SessionEndHandler(client sessionv1.SessionServiceClient, getContext func() Context) mcp.ToolHandlerFor[SessionEndInput, SessionEndResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SessionEndInput) (*mcp.CallToolResult, SessionEndResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, SessionEndResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		mcpCtx := Context{}
		if getContext != nil {
			mcpCtx = getContext()
		}

		campaignID := input.CampaignID
		if campaignID == "" {
			campaignID = mcpCtx.CampaignID
		}
		sessionID := input.SessionID
		if sessionID == "" {
			sessionID = mcpCtx.SessionID
		}
		if campaignID == "" {
			return nil, SessionEndResult{}, fmt.Errorf("campaign_id is required")
		}
		if sessionID == "" {
			return nil, SessionEndResult{}, fmt.Errorf("session_id is required")
		}

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, SessionEndResult{}, fmt.Errorf("create request metadata: %w", err)
		}

		var header metadata.MD
		response, err := client.EndSession(callCtx, &sessionv1.EndSessionRequest{
			CampaignId: campaignID,
			SessionId:  sessionID,
		}, grpc.Header(&header))
		if err != nil {
			return nil, SessionEndResult{}, fmt.Errorf("session end failed: %w", err)
		}
		if response == nil || response.Session == nil {
			return nil, SessionEndResult{}, fmt.Errorf("session end response is missing")
		}

		result := SessionEndResult{
			ID:         response.Session.GetId(),
			CampaignID: response.Session.GetCampaignId(),
			Name:       response.Session.GetName(),
			Status:     sessionStatusToString(response.Session.GetStatus()),
			StartedAt:  formatTimestamp(response.Session.GetStartedAt()),
			UpdatedAt:  formatTimestamp(response.Session.GetUpdatedAt()),
		}

		if response.Session.GetEndedAt() != nil {
			result.EndedAt = formatTimestamp(response.Session.GetEndedAt())
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// SessionActionRollModifier represents a modifier for a session action roll.
type SessionActionRollModifier struct {
	Source string `json:"source" jsonschema:"modifier source label"`
	Value  int    `json:"value" jsonschema:"modifier value"`
}

// SessionActionRollInput represents the MCP tool input for a session action roll.
type SessionActionRollInput struct {
	CampaignID  string                      `json:"campaign_id,omitempty" jsonschema:"campaign identifier (defaults to context)"`
	SessionID   string                      `json:"session_id,omitempty" jsonschema:"session identifier (defaults to context)"`
	CharacterID string                      `json:"character_id" jsonschema:"character identifier"`
	Trait       string                      `json:"trait" jsonschema:"trait being rolled"`
	Difficulty  int                         `json:"difficulty" jsonschema:"difficulty target"`
	Modifiers   []SessionActionRollModifier `json:"modifiers,omitempty" jsonschema:"optional roll modifiers"`
}

// SessionActionRollResult represents the MCP tool output for a session action roll.
type SessionActionRollResult struct {
	HopeDie    int    `json:"hope_die" jsonschema:"hope die result"`
	FearDie    int    `json:"fear_die" jsonschema:"fear die result"`
	Total      int    `json:"total" jsonschema:"sum of dice and modifiers"`
	Difficulty int    `json:"difficulty" jsonschema:"difficulty target"`
	Success    bool   `json:"success" jsonschema:"whether total meets difficulty"`
	Flavor     string `json:"flavor" jsonschema:"HOPE or FEAR"`
	Crit       bool   `json:"crit" jsonschema:"whether the roll is a critical success"`
}

// SessionRollOutcomeApplyInput represents the MCP tool input for applying roll outcomes.
type SessionRollOutcomeApplyInput struct {
	SessionID string   `json:"session_id,omitempty" jsonschema:"session identifier (defaults to context)"`
	RollSeq   uint64   `json:"roll_seq" jsonschema:"roll sequence number to apply"`
	Targets   []string `json:"targets,omitempty" jsonschema:"optional target character ids"`
}

// SessionRollOutcomeApplyCharacterState represents updated character state output.
type SessionRollOutcomeApplyCharacterState struct {
	CharacterID string `json:"character_id" jsonschema:"character identifier"`
	Hope        int    `json:"hope" jsonschema:"updated hope"`
	Stress      int    `json:"stress" jsonschema:"updated stress"`
	HP          int    `json:"hp" jsonschema:"updated hp"`
}

// SessionRollOutcomeApplyUpdated represents updated outcome state output.
type SessionRollOutcomeApplyUpdated struct {
	CharacterStates []SessionRollOutcomeApplyCharacterState `json:"character_states" jsonschema:"updated character states"`
	GMFear          *int                                    `json:"gm_fear,omitempty" jsonschema:"updated gm fear"`
}

// SessionRollOutcomeApplyResult represents the MCP tool output for applying roll outcomes.
type SessionRollOutcomeApplyResult struct {
	RollSeq              uint64                         `json:"roll_seq" jsonschema:"roll sequence number applied"`
	RequiresComplication bool                           `json:"requires_complication" jsonschema:"whether a complication is required"`
	Updated              SessionRollOutcomeApplyUpdated `json:"updated" jsonschema:"updated state"`
}

// SessionActionRollTool defines the MCP tool schema for session action rolls.
func SessionActionRollTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "session_action_roll",
		Description: "Rolls Duality dice for a session and appends session events",
	}
}

// SessionRollOutcomeApplyTool defines the MCP tool schema for applying roll outcomes.
func SessionRollOutcomeApplyTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "session_roll_outcome_apply",
		Description: "Applies mandatory roll outcome effects and appends session events",
	}
}

// SessionActionRollHandler executes a session action roll request.
func SessionActionRollHandler(client sessionv1.SessionServiceClient, getContext func() Context) mcp.ToolHandlerFor[SessionActionRollInput, SessionActionRollResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SessionActionRollInput) (*mcp.CallToolResult, SessionActionRollResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, SessionActionRollResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		mcpCtx := Context{}
		if getContext != nil {
			mcpCtx = getContext()
		}

		campaignID := input.CampaignID
		if campaignID == "" {
			campaignID = mcpCtx.CampaignID
		}
		sessionID := input.SessionID
		if sessionID == "" {
			sessionID = mcpCtx.SessionID
		}
		if campaignID == "" {
			return nil, SessionActionRollResult{}, fmt.Errorf("campaign_id is required")
		}
		if sessionID == "" {
			return nil, SessionActionRollResult{}, fmt.Errorf("session_id is required")
		}

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, SessionActionRollResult{}, fmt.Errorf("create request metadata: %w", err)
		}
		if mcpCtx.ParticipantID != "" {
			callCtx = metadata.AppendToOutgoingContext(callCtx, grpcmeta.ParticipantIDHeader, mcpCtx.ParticipantID)
		}

		modifiers := make([]*sessionv1.ActionRollModifier, 0, len(input.Modifiers))
		for _, modifier := range input.Modifiers {
			modifiers = append(modifiers, &sessionv1.ActionRollModifier{
				Source: modifier.Source,
				Value:  int32(modifier.Value),
			})
		}

		var header metadata.MD
		response, err := client.SessionActionRoll(callCtx, &sessionv1.SessionActionRollRequest{
			CampaignId:  campaignID,
			SessionId:   sessionID,
			CharacterId: input.CharacterID,
			Trait:       input.Trait,
			Difficulty:  int32(input.Difficulty),
			Modifiers:   modifiers,
		}, grpc.Header(&header))
		if err != nil {
			return nil, SessionActionRollResult{}, fmt.Errorf("session action roll failed: %w", err)
		}
		if response == nil {
			return nil, SessionActionRollResult{}, fmt.Errorf("session action roll response is missing")
		}

		result := SessionActionRollResult{
			HopeDie:    int(response.GetHopeDie()),
			FearDie:    int(response.GetFearDie()),
			Total:      int(response.GetTotal()),
			Difficulty: int(response.GetDifficulty()),
			Success:    response.GetSuccess(),
			Flavor:     response.GetFlavor(),
			Crit:       response.GetCrit(),
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// SessionRollOutcomeApplyHandler executes a roll outcome apply request.
func SessionRollOutcomeApplyHandler(client sessionv1.SessionServiceClient, getContext func() Context) mcp.ToolHandlerFor[SessionRollOutcomeApplyInput, SessionRollOutcomeApplyResult] {
	return func(ctx context.Context, _ *mcp.CallToolRequest, input SessionRollOutcomeApplyInput) (*mcp.CallToolResult, SessionRollOutcomeApplyResult, error) {
		invocationID, err := NewInvocationID()
		if err != nil {
			return nil, SessionRollOutcomeApplyResult{}, fmt.Errorf("generate invocation id: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		mcpCtx := Context{}
		if getContext != nil {
			mcpCtx = getContext()
		}

		sessionID := input.SessionID
		if sessionID == "" {
			sessionID = mcpCtx.SessionID
		}
		if sessionID == "" {
			return nil, SessionRollOutcomeApplyResult{}, fmt.Errorf("session_id is required")
		}

		callCtx, callMeta, err := NewOutgoingContext(runCtx, invocationID)
		if err != nil {
			return nil, SessionRollOutcomeApplyResult{}, fmt.Errorf("create request metadata: %w", err)
		}
		if mcpCtx.ParticipantID != "" {
			callCtx = metadata.AppendToOutgoingContext(callCtx, grpcmeta.ParticipantIDHeader, mcpCtx.ParticipantID)
		}

		var header metadata.MD
		response, err := client.ApplyRollOutcome(callCtx, &sessionv1.ApplyRollOutcomeRequest{
			SessionId: sessionID,
			RollSeq:   input.RollSeq,
			Targets:   input.Targets,
		}, grpc.Header(&header))
		if err != nil {
			return nil, SessionRollOutcomeApplyResult{}, fmt.Errorf("apply roll outcome failed: %w", err)
		}
		if response == nil || response.Updated == nil {
			return nil, SessionRollOutcomeApplyResult{}, fmt.Errorf("apply roll outcome response is missing")
		}

		updatedStates := make([]SessionRollOutcomeApplyCharacterState, 0, len(response.Updated.GetCharacterStates()))
		for _, state := range response.Updated.GetCharacterStates() {
			updatedStates = append(updatedStates, SessionRollOutcomeApplyCharacterState{
				CharacterID: state.GetCharacterId(),
				Hope:        int(state.GetHope()),
				Stress:      int(state.GetStress()),
				HP:          int(state.GetHp()),
			})
		}

		updated := SessionRollOutcomeApplyUpdated{CharacterStates: updatedStates}
		if response.Updated.GmFear != nil {
			gmFear := int(response.Updated.GetGmFear())
			updated.GMFear = &gmFear
		}

		result := SessionRollOutcomeApplyResult{
			RollSeq:              response.GetRollSeq(),
			RequiresComplication: response.GetRequiresComplication(),
			Updated:              updated,
		}

		responseMeta := MergeResponseMetadata(callMeta, header)
		return CallToolResultWithMetadata(responseMeta), result, nil
	}
}

// sessionStatusToString converts a protobuf SessionStatus to a string representation.
func sessionStatusToString(status sessionv1.SessionStatus) string {
	switch status {
	case sessionv1.SessionStatus_ACTIVE:
		return "ACTIVE"
	case sessionv1.SessionStatus_ENDED:
		return "ENDED"
	case sessionv1.SessionStatus_STATUS_UNSPECIFIED:
		return "UNSPECIFIED"
	default:
		return "UNSPECIFIED"
	}
}

// SessionListEntry represents a readable session entry.
type SessionListEntry struct {
	ID         string `json:"id"`
	CampaignID string `json:"campaign_id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	StartedAt  string `json:"started_at"`
	UpdatedAt  string `json:"updated_at"`
	EndedAt    string `json:"ended_at,omitempty"`
}

// SessionListPayload represents the MCP resource payload for session listings.
type SessionListPayload struct {
	Sessions []SessionListEntry `json:"sessions"`
}

// SessionEventEntry represents a readable session event.
type SessionEventEntry struct {
	SessionID     string `json:"session_id"`
	Seq           uint64 `json:"seq"`
	Timestamp     string `json:"ts"`
	Type          string `json:"type"`
	RequestID     string `json:"request_id"`
	InvocationID  string `json:"invocation_id"`
	ParticipantID string `json:"participant_id,omitempty"`
	CharacterID   string `json:"character_id,omitempty"`
	PayloadJSON   string `json:"payload_json"`
}

// SessionEventsPayload represents the MCP resource payload for session events.
type SessionEventsPayload struct {
	Events []SessionEventEntry `json:"events"`
}

// SessionListResourceTemplate defines the MCP resource template for session listings.
func SessionListResourceTemplate() *mcp.ResourceTemplate {
	return &mcp.ResourceTemplate{
		Name:        "session_list",
		Title:       "Sessions",
		Description: "Readable listing of sessions for a campaign. URI format: campaign://{campaign_id}/sessions",
		MIMEType:    "application/json",
		URITemplate: "campaign://{campaign_id}/sessions",
	}
}

// SessionListResourceHandler returns a readable session listing resource.
func SessionListResourceHandler(client sessionv1.SessionServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("session list client is not configured")
		}

		if req == nil || req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("campaign ID is required; use URI format campaign://{campaign_id}/sessions")
		}
		uri := req.Params.URI

		// Parse campaign_id from URI: expected format is campaign://{campaign_id}/sessions.
		campaignID, err := parseCampaignIDFromSessionURI(uri)
		if err != nil {
			return nil, fmt.Errorf("parse campaign ID from URI: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		payload := SessionListPayload{}
		response, err := client.ListSessions(callCtx, &sessionv1.ListSessionsRequest{
			CampaignId: campaignID,
			PageSize:   10,
		})
		if err != nil {
			return nil, fmt.Errorf("session list failed: %w", err)
		}
		if response == nil {
			return nil, fmt.Errorf("session list response is missing")
		}

		for _, session := range response.GetSessions() {
			entry := SessionListEntry{
				ID:         session.GetId(),
				CampaignID: session.GetCampaignId(),
				Name:       session.GetName(),
				Status:     sessionStatusToString(session.GetStatus()),
				StartedAt:  formatTimestamp(session.GetStartedAt()),
				UpdatedAt:  formatTimestamp(session.GetUpdatedAt()),
			}
			if session.GetEndedAt() != nil {
				entry.EndedAt = formatTimestamp(session.GetEndedAt())
			}
			payload.Sessions = append(payload.Sessions, entry)
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal session list: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "application/json",
					Text:     string(data),
				},
			},
		}, nil
	}
}

// SessionEventsResourceTemplate defines the MCP resource template for session event listings.
func SessionEventsResourceTemplate() *mcp.ResourceTemplate {
	return &mcp.ResourceTemplate{
		Name:        "session_events",
		Title:       "Session Events",
		Description: "Readable listing of session events. URI format: session://{session_id}/events",
		MIMEType:    "application/json",
		URITemplate: "session://{session_id}/events",
	}
}

// SessionEventsResourceHandler returns a readable session events listing resource.
func SessionEventsResourceHandler(client sessionv1.SessionServiceClient) mcp.ResourceHandler {
	return func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		if client == nil {
			return nil, fmt.Errorf("session events client is not configured")
		}

		if req == nil || req.Params == nil || req.Params.URI == "" {
			return nil, fmt.Errorf("session ID is required; use URI format session://{session_id}/events")
		}
		uri := req.Params.URI

		sessionID, err := parseSessionIDFromSessionEventsURI(uri)
		if err != nil {
			return nil, fmt.Errorf("parse session ID from URI: %w", err)
		}

		runCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		callCtx, _, err := NewOutgoingContext(runCtx, "")
		if err != nil {
			return nil, fmt.Errorf("create request metadata: %w", err)
		}

		response, err := client.SessionEventsList(callCtx, &sessionv1.SessionEventsListRequest{
			SessionId: sessionID,
			Limit:     50,
		})
		if err != nil {
			return nil, fmt.Errorf("session events list failed: %w", err)
		}
		if response == nil {
			return nil, fmt.Errorf("session events list response is missing")
		}

		events := response.GetEvents()
		payload := SessionEventsPayload{Events: make([]SessionEventEntry, 0, len(events))}
		for i := len(events); i > 0; i-- {
			event := events[i-1]
			payload.Events = append(payload.Events, SessionEventEntry{
				SessionID:     event.GetSessionId(),
				Seq:           event.GetSeq(),
				Timestamp:     formatTimestamp(event.GetTs()),
				Type:          event.GetType().String(),
				RequestID:     event.GetRequestId(),
				InvocationID:  event.GetInvocationId(),
				ParticipantID: event.GetParticipantId(),
				CharacterID:   event.GetCharacterId(),
				PayloadJSON:   string(event.GetPayloadJson()),
			})
		}

		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal session events: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "application/json",
					Text:     string(data),
				},
			},
		}, nil
	}
}

// parseCampaignIDFromSessionURI extracts the campaign ID from a URI of the form campaign://{campaign_id}/sessions.
// It parses URIs of the expected format but requires an actual campaign ID.
func parseCampaignIDFromSessionURI(uri string) (string, error) {
	return parseCampaignIDFromResourceURI(uri, "sessions")
}

// parseSessionIDFromSessionEventsURI extracts the session ID from a URI of the form session://{session_id}/events.
// It parses URIs of the expected format but requires an actual session ID.
func parseSessionIDFromSessionEventsURI(uri string) (string, error) {
	return parseSessionIDFromResourceURI(uri, "events")
}
