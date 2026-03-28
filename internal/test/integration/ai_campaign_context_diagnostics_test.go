//go:build integration

package integration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	evalsupport "github.com/louisbranch/fracturing.space/internal/test/aieval"
)

type aiGMTraceEntry struct {
	Sequence   int       `json:"sequence"`
	Kind       string    `json:"kind"`
	ResponseID string    `json:"response_id,omitempty"`
	CallID     string    `json:"call_id,omitempty"`
	ToolName   string    `json:"tool_name,omitempty"`
	Payload    string    `json:"payload,omitempty"`
	IsError    bool      `json:"is_error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type aiGMScenarioDiagnostics struct {
	RunError         string           `json:"run_error,omitempty"`
	FailureKind      string           `json:"failure_kind,omitempty"`
	FailureSummary   string           `json:"failure_summary,omitempty"`
	FailureReason    string           `json:"failure_reason,omitempty"`
	CollectionErrors []string         `json:"collection_errors,omitempty"`
	TraceEntries     []aiGMTraceEntry `json:"trace_entries,omitempty"`
}

type aiGMTraceRecorder struct {
	mu      sync.Mutex
	entries []aiGMTraceEntry
}

func (r *aiGMTraceRecorder) RecordProviderStep(_ context.Context, output orchestration.ProviderOutput) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if text := strings.TrimSpace(output.OutputText); text != "" {
		r.entries = append(r.entries, aiGMTraceEntry{
			Sequence:   len(r.entries) + 1,
			Kind:       "model_response",
			ResponseID: strings.TrimSpace(output.ConversationID),
			Payload:    text,
			CreatedAt:  now,
		})
	}
	for _, call := range output.ToolCalls {
		r.entries = append(r.entries, aiGMTraceEntry{
			Sequence:   len(r.entries) + 1,
			Kind:       "tool_call",
			ResponseID: strings.TrimSpace(output.ConversationID),
			CallID:     strings.TrimSpace(call.CallID),
			ToolName:   strings.TrimSpace(call.Name),
			Payload:    strings.TrimSpace(call.Arguments),
			CreatedAt:  now,
		})
	}
}

func (r *aiGMTraceRecorder) RecordToolResult(_ context.Context, call orchestration.ProviderToolCall, result orchestration.ProviderToolResult) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries = append(r.entries, aiGMTraceEntry{
		Sequence:  len(r.entries) + 1,
		Kind:      "tool_result",
		CallID:    strings.TrimSpace(result.CallID),
		ToolName:  strings.TrimSpace(call.Name),
		Payload:   strings.TrimSpace(result.Output),
		IsError:   result.IsError,
		CreatedAt: time.Now().UTC(),
	})
}

func (r *aiGMTraceRecorder) Snapshot() []aiGMTraceEntry {
	if r == nil {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]aiGMTraceEntry, len(r.entries))
	copy(out, r.entries)
	return out
}

func buildScenarioDiagnostics(err error, trace *aiGMTraceRecorder, collectionErrors []string) *aiGMScenarioDiagnostics {
	if err == nil && len(collectionErrors) == 0 && (trace == nil || len(trace.Snapshot()) == 0) {
		return nil
	}
	kind, summary, reason, _ := classifyScenarioFailure(err, trace)
	if err == nil && len(collectionErrors) > 0 {
		kind = "artifact_capture_error"
		summary = compactDiagnosticText(collectionErrors[0])
		reason = strings.Join(collectionErrors, "; ")
	}
	return &aiGMScenarioDiagnostics{
		RunError:         errorString(err),
		FailureKind:      kind,
		FailureSummary:   summary,
		FailureReason:    reason,
		CollectionErrors: append([]string(nil), collectionErrors...),
		TraceEntries:     trace.Snapshot(),
	}
}

func classifyScenarioFailure(err error, trace *aiGMTraceRecorder) (string, string, string, string) {
	if err == nil {
		return "", "", "", ""
	}

	if entry, ok := latestToolError(trace); ok {
		reason := strings.TrimSpace(entry.Payload)
		summary := fmt.Sprintf("tool %s failed: %s", entry.ToolName, compactDiagnosticText(reason))
		return "tool_execution_error", summary, reason, evalsupport.MetricStatusInvalid
	}

	reason := strings.TrimSpace(err.Error())
	switch {
	case errors.Is(err, orchestration.ErrNarrationNotCommitted):
		return "turn_control_error", "model returned narration without committing GM output", reason, evalsupport.MetricStatusFail
	case errors.Is(err, orchestration.ErrStepLimit):
		return "turn_control_error", "model exhausted the allowed orchestration step limit", reason, evalsupport.MetricStatusFail
	case strings.Contains(strings.ToLower(reason), "must open the next player phase"):
		return "phase_reopen", "model finished the turn without reopening the player phase", reason, evalsupport.MetricStatusFail
	case strings.Contains(strings.ToLower(reason), "provider returned no tool calls or output"):
		return "provider_error", "provider returned neither tool calls nor final output", reason, evalsupport.MetricStatusInvalid
	default:
		return "harness_error", compactDiagnosticText(reason), reason, evalsupport.MetricStatusInvalid
	}
}

func latestToolError(trace *aiGMTraceRecorder) (aiGMTraceEntry, bool) {
	entries := trace.Snapshot()
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].Kind == "tool_result" && entries[i].IsError {
			return entries[i], true
		}
	}
	return aiGMTraceEntry{}, false
}

func compactDiagnosticText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	line := strings.TrimSpace(strings.Split(value, "\n")[0])
	if len(line) <= 160 {
		return line
	}
	return strings.TrimSpace(line[:160]) + "..."
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return strings.TrimSpace(err.Error())
}

func maybeArtifactContent(ctx context.Context, client aiv1.CampaignArtifactServiceClient, campaignID, path string) (string, bool, error) {
	resp, err := client.GetCampaignArtifact(ctx, &aiv1.GetCampaignArtifactRequest{
		CampaignId: campaignID,
		Path:       path,
	})
	if err != nil {
		return "", false, err
	}
	if resp.GetArtifact() == nil {
		return "", false, fmt.Errorf("artifact %q is missing", path)
	}
	return strings.TrimSpace(resp.GetArtifact().GetContent()), resp.GetArtifact().GetReadOnly(), nil
}

func maybeScenes(ctx context.Context, client gamev1.SceneServiceClient, campaignID, sessionID string) ([]*gamev1.Scene, error) {
	resp, err := client.ListScenes(ctx, &gamev1.ListScenesRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		PageSize:   10,
	})
	if err != nil {
		return nil, err
	}
	return resp.GetScenes(), nil
}

func maybeInteractionState(ctx context.Context, client gamev1.InteractionServiceClient, campaignID string) (*gamev1.InteractionState, error) {
	resp, err := client.GetInteractionState(ctx, &gamev1.GetInteractionStateRequest{
		CampaignId: campaignID,
	})
	if err != nil {
		return nil, err
	}
	return resp.GetState(), nil
}

func maybeCharacterState(ctx context.Context, client gamev1.SnapshotServiceClient, campaignID, characterID string) (*pb.DaggerheartCharacterState, error) {
	snapshot, err := client.GetSnapshot(ctx, &gamev1.GetSnapshotRequest{CampaignId: campaignID})
	if err != nil {
		return nil, err
	}
	if snapshot.GetSnapshot() == nil {
		return nil, fmt.Errorf("snapshot is missing")
	}
	for _, state := range snapshot.GetSnapshot().GetCharacterStates() {
		if state.GetCharacterId() != characterID {
			continue
		}
		if state.GetDaggerheart() == nil {
			return nil, fmt.Errorf("daggerheart state is missing for %s", characterID)
		}
		return state.GetDaggerheart(), nil
	}
	return nil, fmt.Errorf("character state not found: %s", characterID)
}
