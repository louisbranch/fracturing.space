package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"github.com/louisbranch/fracturing.space/internal/services/shared/mcpbridge"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const defaultMaxSteps = 8
const defaultTurnTimeout = 2 * time.Minute
const defaultToolResultMaxBytes = 32 * 1024

type promptBuildTrace struct {
	retrieved   []RetrievedContext
	diagnostics PromptDiagnostics
}

func (t *promptBuildTrace) RecordRetrievedContexts(contexts []RetrievedContext) {
	if len(contexts) == 0 {
		return
	}
	t.retrieved = append(t.retrieved, contexts...)
}

func (t *promptBuildTrace) RecordPromptContextPolicy(policy PromptContextPolicy) {
	t.diagnostics.ContextPolicy = policy
}

func (t *promptBuildTrace) RecordPromptAugmentation(diagnostics PromptAugmentationDiagnostics) {
	if diagnostics.Attempted {
		t.diagnostics.Augmentation.Attempted = true
	}
	if mode := strings.TrimSpace(diagnostics.Mode); mode != "" {
		t.diagnostics.Augmentation.Mode = mode
	}
	if diagnostics.SearchAttempted {
		t.diagnostics.Augmentation.SearchAttempted = true
	}
	if diagnostics.ResourceHits > 0 || diagnostics.SearchAttempted {
		t.diagnostics.Augmentation.ResourceHits = diagnostics.ResourceHits
	}
	if diagnostics.MemoryHits > 0 || diagnostics.SearchAttempted {
		t.diagnostics.Augmentation.MemoryHits = diagnostics.MemoryHits
	}
	if len(diagnostics.MirroredTargets) > 0 {
		seen := map[string]struct{}{}
		for _, item := range t.diagnostics.Augmentation.MirroredTargets {
			seen[item] = struct{}{}
		}
		for _, item := range diagnostics.MirroredTargets {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			t.diagnostics.Augmentation.MirroredTargets = append(t.diagnostics.Augmentation.MirroredTargets, item)
		}
	}
	if diagnostics.Degraded {
		t.diagnostics.Augmentation.Degraded = true
	}
	if reason := strings.TrimSpace(diagnostics.DegradationReason); reason != "" {
		t.diagnostics.Augmentation.DegradationReason = reason
	}
}

type runner struct {
	dialer             Dialer
	promptBuilder      PromptBuilder
	turnPolicy         TurnPolicy
	toolPolicy         ToolPolicy
	max                int
	turnTimeout        time.Duration
	toolResultMaxBytes int
	commitToolName     string
}

// NewRunner builds a campaign-turn runner over one dialer and one explicit
// runtime policy.
func NewRunner(cfg RunnerConfig) CampaignTurnRunner {
	maxSteps := cfg.MaxSteps
	if maxSteps <= 0 {
		maxSteps = defaultMaxSteps
	}
	turnTimeout := cfg.TurnTimeout
	if turnTimeout <= 0 {
		turnTimeout = defaultTurnTimeout
	}
	toolResultMaxBytes := cfg.ToolResultMaxBytes
	if toolResultMaxBytes <= 0 {
		toolResultMaxBytes = defaultToolResultMaxBytes
	}
	promptBuilder := cfg.PromptBuilder
	if promptBuilder == nil {
		promptBuilder = newDegradedPromptBuilder()
	}
	turnPolicy := cfg.TurnPolicy
	if turnPolicy == nil {
		turnPolicy = NewInteractionTurnPolicy()
	}
	toolPolicy := cfg.ToolPolicy
	if toolPolicy == nil {
		toolPolicy = AllowAllToolPolicy()
	}
	commitToolName := strings.TrimSpace(cfg.CommitToolName)
	if commitToolName == "" {
		commitToolName = DefaultCommitToolName
	}
	return &runner{
		dialer:             cfg.Dialer,
		promptBuilder:      promptBuilder,
		turnPolicy:         turnPolicy,
		toolPolicy:         toolPolicy,
		max:                maxSteps,
		turnTimeout:        turnTimeout,
		toolResultMaxBytes: toolResultMaxBytes,
		commitToolName:     commitToolName,
	}
}

// Run executes one provider turn.
func (r *runner) Run(ctx context.Context, input Input) (Result, error) {
	ctx, span := orchestrationTracer().Start(ctx, "ai.orchestration.run")
	defer span.End()
	if r == nil || r.dialer == nil {
		err := errRunnerUnavailable()
		recordSpanError(span, err)
		return Result{}, err
	}
	if r.promptBuilder == nil {
		err := errPromptBuilderUnavailable()
		recordSpanError(span, err)
		return Result{}, err
	}
	if input.Provider == nil {
		err := errInvalidInput("campaign turn provider is required")
		recordSpanError(span, err)
		return Result{}, err
	}
	if input.CampaignID == "" || input.SessionID == "" || input.ParticipantID == "" {
		err := errInvalidInput("campaign, session, and participant are required")
		recordSpanError(span, err)
		return Result{}, err
	}
	if input.Model == "" {
		err := errInvalidInput("model is required")
		recordSpanError(span, err)
		return Result{}, err
	}
	if input.AuthToken == "" {
		err := errInvalidInput("auth token is required")
		recordSpanError(span, err)
		return Result{}, err
	}
	span.SetAttributes(
		attribute.String("ai.orchestration.campaign_id", input.CampaignID),
		attribute.String("ai.orchestration.session_id", input.SessionID),
		attribute.String("ai.orchestration.participant_id", input.ParticipantID),
		attribute.String("ai.orchestration.model", input.Model),
		attribute.Int("ai.orchestration.max_steps", r.max),
		attribute.Int64("ai.orchestration.turn_timeout_ms", r.turnTimeout.Milliseconds()),
		attribute.Int("ai.orchestration.tool_result_max_bytes", r.toolResultMaxBytes),
	)
	if r.turnTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, r.turnTimeout)
		defer cancel()
	}

	ctx = mcpbridge.WithSessionContext(ctx, mcpbridge.SessionContext{
		CampaignID:    input.CampaignID,
		SessionID:     input.SessionID,
		ParticipantID: input.ParticipantID,
	})

	sess, err := r.dialer.Dial(ctx)
	if err != nil {
		err = errExecution(fmt.Errorf("dial mcp: %w", err))
		recordSpanError(span, err)
		return Result{}, err
	}
	defer sess.Close()

	tools, err := sess.ListTools(ctx)
	if err != nil {
		err = errExecution(fmt.Errorf("list mcp tools: %w", err))
		recordSpanError(span, err)
		return Result{}, err
	}
	allowedTools := filterTools(tools, r.toolPolicy)
	span.SetAttributes(attribute.Int("ai.orchestration.allowed_tool_count", len(allowedTools)))
	allowedToolNames := make(map[string]struct{}, len(allowedTools))
	for _, tool := range allowedTools {
		allowedToolNames[tool.Name] = struct{}{}
	}
	promptCtx, promptSpan := orchestrationTracer().Start(ctx, "ai.orchestration.build_prompt")
	promptTrace := &promptBuildTrace{}
	promptCtx = WithPromptBuildTraceRecorder(promptCtx, promptTrace)
	prompt, err := r.promptBuilder.Build(promptCtx, sess, PromptInput{
		CampaignID:    input.CampaignID,
		SessionID:     input.SessionID,
		ParticipantID: input.ParticipantID,
		TurnInput:     input.Input,
	})
	if err != nil {
		err = errPromptBuild(err)
		recordSpanError(promptSpan, err)
		recordSpanError(span, err)
		promptSpan.End()
		return Result{}, err
	}
	promptSpan.SetAttributes(attribute.Int("ai.orchestration.prompt_bytes", len(prompt)))
	promptSpan.End()

	var convo string
	turnController := r.turnPolicy.Controller(r.commitToolName)
	var results []ProviderToolResult
	commitReminderUsed := false
	completionReminderUsed := false
	playerPhaseReminderUsed := false
	var followUpPrompt string
	var usage provider.Usage

	for i := 0; i < r.max; i++ {
		stepCtx, stepSpan := orchestrationTracer().Start(ctx, "ai.orchestration.provider_step")
		stepSpan.SetAttributes(
			attribute.Int("ai.orchestration.step_index", i+1),
			attribute.Bool("ai.orchestration.has_followup_prompt", followUpPrompt != ""),
			attribute.Int("ai.orchestration.result_count", len(results)),
		)
		step, err := input.Provider.Run(stepCtx, ProviderInput{
			Model:           input.Model,
			ReasoningEffort: input.ReasoningEffort,
			Instructions:    input.Instructions,
			Prompt:          prompt,
			FollowUpPrompt:  followUpPrompt,
			AuthToken:       input.AuthToken,
			Tools:           allowedTools,
			ConversationID:  convo,
			Results:         results,
		})
		if err != nil {
			err = errExecution(fmt.Errorf("invoke provider: %w", err))
			recordSpanError(stepSpan, err)
			recordSpanError(span, err)
			stepSpan.End()
			return Result{}, err
		}
		stepSpan.SetAttributes(
			attribute.String("ai.orchestration.conversation_id", step.ConversationID),
			attribute.Int("ai.orchestration.tool_call_count", len(step.ToolCalls)),
			attribute.Bool("ai.orchestration.has_output_text", step.OutputText != ""),
		)
		stepSpan.End()
		convo = step.ConversationID
		usage = usage.Add(step.Usage)
		if input.TraceRecorder != nil {
			input.TraceRecorder.RecordProviderStep(ctx, step)
		}
		followUpPrompt = ""
		if len(step.ToolCalls) == 0 {
			text := step.OutputText
			if text == "" {
				err := errExecution(fmt.Errorf("provider returned no tool calls or output"))
				recordSpanError(span, err)
				return Result{}, err
			}
			if !turnController.HasCommittedOrResolvedInteraction() {
				if !commitReminderUsed && i+1 < r.max {
					commitReminderUsed = true
					results = nil
					followUpPrompt = turnController.BuildCommitReminder(text)
					span.AddEvent("ai.orchestration.commit_reminder_requested")
					continue
				}
				recordSpanError(span, ErrNarrationNotCommitted)
				return Result{}, ErrNarrationNotCommitted
			}
			if !turnController.ReadyForCompletion() {
				if !completionReminderUsed && i+1 < r.max {
					completionReminderUsed = true
					results = nil
					followUpPrompt = turnController.BuildTurnCompletionReminder(text)
					span.AddEvent("ai.orchestration.turn_completion_reminder_requested")
					continue
				}
				err := errExecution(fmt.Errorf("ai gm turn must open the next player phase or pause for ooc before returning final text"))
				recordSpanError(span, err)
				return Result{}, err
			}
			if turnController.PlayerHandoffRegressed() {
				if !playerPhaseReminderUsed && i+1 < r.max {
					playerPhaseReminderUsed = true
					results = nil
					followUpPrompt = turnController.BuildPlayerPhaseStartReminder(text)
					span.AddEvent("ai.orchestration.player_phase_restart_requested")
					continue
				}
				err := errExecution(fmt.Errorf("campaign orchestration committed gm output after opening a player phase without reopening the phase for players"))
				recordSpanError(span, err)
				return Result{}, err
			}
			span.SetAttributes(attribute.Bool("ai.orchestration.committed_output", turnController.HasCommittedOrResolvedInteraction()))
			return Result{
				OutputText:        text,
				Usage:             usage,
				RetrievedContexts: append([]RetrievedContext(nil), promptTrace.retrieved...),
				PromptDiagnostics: promptTrace.diagnostics,
			}, nil
		}

		results = make([]ProviderToolResult, 0, len(step.ToolCalls))
		for _, call := range step.ToolCalls {
			if _, ok := allowedToolNames[call.Name]; !ok {
				result := ProviderToolResult{
					CallID:  call.CallID,
					Output:  fmt.Sprintf("tool %q is not allowed for campaign orchestration", call.Name),
					IsError: true,
				}
				results = append(results, result)
				if input.TraceRecorder != nil {
					input.TraceRecorder.RecordToolResult(ctx, call, result)
				}
				continue
			}
			args, err := decodeArgs(call.Arguments)
			if err != nil {
				result := ProviderToolResult{
					CallID:  call.CallID,
					Output:  fmt.Sprintf("invalid tool arguments: %v", err),
					IsError: true,
				}
				results = append(results, result)
				if input.TraceRecorder != nil {
					input.TraceRecorder.RecordToolResult(ctx, call, result)
				}
				continue
			}
			res, err := sess.CallTool(ctx, call.Name, args)
			if err != nil {
				result := ProviderToolResult{
					CallID:  call.CallID,
					Output:  fmt.Sprintf("tool call failed: %v", err),
					IsError: true,
				}
				results = append(results, result)
				if input.TraceRecorder != nil {
					input.TraceRecorder.RecordToolResult(ctx, call, result)
				}
				continue
			}
			if !res.IsError {
				turnController.ObserveSuccessfulTool(call.Name, res.Output)
			}
			outputText, truncated := truncateToolResultOutput(res.Output, r.toolResultMaxBytes)
			result := ProviderToolResult{
				CallID:  call.CallID,
				Output:  outputText,
				IsError: res.IsError,
			}
			results = append(results, result)
			if input.TraceRecorder != nil {
				input.TraceRecorder.RecordToolResult(ctx, call, result)
			}
			if truncated {
				span.AddEvent("ai.orchestration.tool_result_truncated",
					trace.WithAttributes(
						attribute.String("ai.orchestration.tool_name", call.Name),
						attribute.Int("ai.orchestration.tool_result_max_bytes", r.toolResultMaxBytes),
					),
				)
			}
		}
	}

	recordSpanError(span, ErrStepLimit)
	return Result{}, ErrStepLimit
}

func truncateToolResultOutput(text string, maxBytes int) (string, bool) {
	if maxBytes <= 0 || len(text) <= maxBytes {
		return text, false
	}
	if maxBytes <= len(toolResultBudgetSuffix) {
		return toolResultBudgetSuffix[:maxBytes], true
	}
	limit := maxBytes - len(toolResultBudgetSuffix)
	for limit > 0 && (text[limit]&0xC0) == 0x80 {
		limit--
	}
	return text[:limit] + toolResultBudgetSuffix, true
}

const toolResultBudgetSuffix = "\n\n[truncated by AI orchestration tool-result budget]"

func decodeArgs(raw string) (any, error) {
	if raw == "" {
		return map[string]any{}, nil
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, err
	}
	return value, nil
}
