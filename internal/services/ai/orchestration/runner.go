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

type runner struct {
	dialer             Dialer
	promptBuilder      PromptBuilder
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
		toolPolicy:         toolPolicy,
		max:                maxSteps,
		turnTimeout:        turnTimeout,
		toolResultMaxBytes: toolResultMaxBytes,
		commitToolName:     commitToolName,
	}
}

// Run executes one provider turn.
func (r *runner) Run(ctx context.Context, input Input) (Result, error) {
	ctx, span := tracer.Start(ctx, "ai.orchestration.run")
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
	if strings.TrimSpace(input.CampaignID) == "" || strings.TrimSpace(input.SessionID) == "" || strings.TrimSpace(input.ParticipantID) == "" {
		err := errInvalidInput("campaign, session, and participant are required")
		recordSpanError(span, err)
		return Result{}, err
	}
	if strings.TrimSpace(input.Model) == "" {
		err := errInvalidInput("model is required")
		recordSpanError(span, err)
		return Result{}, err
	}
	if strings.TrimSpace(input.CredentialSecret) == "" {
		err := errInvalidInput("credential secret is required")
		recordSpanError(span, err)
		return Result{}, err
	}
	span.SetAttributes(
		attribute.String("ai.orchestration.campaign_id", strings.TrimSpace(input.CampaignID)),
		attribute.String("ai.orchestration.session_id", strings.TrimSpace(input.SessionID)),
		attribute.String("ai.orchestration.participant_id", strings.TrimSpace(input.ParticipantID)),
		attribute.String("ai.orchestration.model", strings.TrimSpace(input.Model)),
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
	promptCtx, promptSpan := tracer.Start(ctx, "ai.orchestration.build_prompt")
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
	committed := false
	var results []ProviderToolResult
	commitReminderUsed := false
	var followUpPrompt string
	var usage provider.Usage

	for i := 0; i < r.max; i++ {
		stepCtx, stepSpan := tracer.Start(ctx, "ai.orchestration.provider_step")
		stepSpan.SetAttributes(
			attribute.Int("ai.orchestration.step_index", i+1),
			attribute.Bool("ai.orchestration.has_followup_prompt", strings.TrimSpace(followUpPrompt) != ""),
			attribute.Int("ai.orchestration.result_count", len(results)),
		)
		step, err := input.Provider.Run(stepCtx, ProviderInput{
			Model:            input.Model,
			ReasoningEffort:  input.ReasoningEffort,
			Instructions:     input.Instructions,
			Prompt:           prompt,
			FollowUpPrompt:   followUpPrompt,
			CredentialSecret: input.CredentialSecret,
			Tools:            allowedTools,
			ConversationID:   convo,
			Results:          results,
		})
		if err != nil {
			err = errExecution(fmt.Errorf("invoke provider: %w", err))
			recordSpanError(stepSpan, err)
			recordSpanError(span, err)
			stepSpan.End()
			return Result{}, err
		}
		stepSpan.SetAttributes(
			attribute.String("ai.orchestration.conversation_id", strings.TrimSpace(step.ConversationID)),
			attribute.Int("ai.orchestration.tool_call_count", len(step.ToolCalls)),
			attribute.Bool("ai.orchestration.has_output_text", strings.TrimSpace(step.OutputText) != ""),
		)
		stepSpan.End()
		convo = strings.TrimSpace(step.ConversationID)
		usage = usage.Add(step.Usage)
		followUpPrompt = ""
		if len(step.ToolCalls) == 0 {
			text := strings.TrimSpace(step.OutputText)
			if text == "" {
				err := errExecution(fmt.Errorf("provider returned no tool calls or output"))
				recordSpanError(span, err)
				return Result{}, err
			}
			if !committed {
				if !commitReminderUsed && i+1 < r.max {
					commitReminderUsed = true
					results = nil
					followUpPrompt = buildCommitReminder(text)
					span.AddEvent("ai.orchestration.commit_reminder_requested")
					continue
				}
				recordSpanError(span, ErrNarrationNotCommitted)
				return Result{}, ErrNarrationNotCommitted
			}
			span.SetAttributes(attribute.Bool("ai.orchestration.committed_output", committed))
			return Result{OutputText: text, Usage: usage}, nil
		}

		results = make([]ProviderToolResult, 0, len(step.ToolCalls))
		for _, call := range step.ToolCalls {
			if _, ok := allowedToolNames[strings.TrimSpace(call.Name)]; !ok {
				results = append(results, ProviderToolResult{
					CallID:  call.CallID,
					Output:  fmt.Sprintf("tool %q is not allowed for campaign orchestration", strings.TrimSpace(call.Name)),
					IsError: true,
				})
				continue
			}
			args, err := decodeArgs(call.Arguments)
			if err != nil {
				results = append(results, ProviderToolResult{
					CallID:  call.CallID,
					Output:  fmt.Sprintf("invalid tool arguments: %v", err),
					IsError: true,
				})
				continue
			}
			res, err := sess.CallTool(ctx, call.Name, args)
			if err != nil {
				results = append(results, ProviderToolResult{
					CallID:  call.CallID,
					Output:  fmt.Sprintf("tool call failed: %v", err),
					IsError: true,
				})
				continue
			}
			if call.Name == r.commitToolName && !res.IsError {
				committed = true
			}
			outputText, truncated := truncateToolResultOutput(res.Output, r.toolResultMaxBytes)
			results = append(results, ProviderToolResult{
				CallID:  call.CallID,
				Output:  outputText,
				IsError: res.IsError,
			})
			if truncated {
				span.AddEvent("ai.orchestration.tool_result_truncated",
					trace.WithAttributes(
						attribute.String("ai.orchestration.tool_name", strings.TrimSpace(call.Name)),
						attribute.Int("ai.orchestration.tool_result_max_bytes", r.toolResultMaxBytes),
					),
				)
			}
		}
	}

	recordSpanError(span, ErrStepLimit)
	return Result{}, ErrStepLimit
}

func buildCommitReminder(text string) string {
	text = strings.TrimSpace(text)
	var b strings.Builder
	b.WriteString("You returned narration without calling interaction_scene_gm_output_commit.\n")
	b.WriteString("Convert that draft into an authoritative tool call before returning final text.\n")
	b.WriteString("If there is no active scene, set one active first.\n")
	if text != "" {
		b.WriteString("Use this draft narration as the commit text unless you need a small correction:\n")
		b.WriteString(text)
	}
	return b.String()
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
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, err
	}
	return value, nil
}
