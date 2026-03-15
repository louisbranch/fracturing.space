package orchestration

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	providerpkg "github.com/louisbranch/fracturing.space/internal/services/ai/provider"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

type fakeDialer struct {
	sess Session
	err  error
}

func (f *fakeDialer) Dial(context.Context) (Session, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.sess, nil
}

type fakeSession struct {
	tools     []Tool
	resources map[string]string
	results   map[string]ToolResult
	calls     []string
	args      map[string]any
}

func (f *fakeSession) ListTools(context.Context) ([]Tool, error) {
	return append([]Tool(nil), f.tools...), nil
}

func (f *fakeSession) CallTool(_ context.Context, name string, args any) (ToolResult, error) {
	f.calls = append(f.calls, name)
	if f.args == nil {
		f.args = map[string]any{}
	}
	f.args[name] = args
	res, ok := f.results[name]
	if !ok {
		return ToolResult{}, errors.New("missing tool result")
	}
	return res, nil
}

func (f *fakeSession) ReadResource(_ context.Context, uri string) (string, error) {
	value, ok := f.resources[uri]
	if !ok {
		return "", errors.New("missing resource")
	}
	return value, nil
}

func (f *fakeSession) Close() error { return nil }

type fakeProvider struct {
	steps []ProviderOutput
	calls []ProviderInput
	run   func(context.Context, ProviderInput) (ProviderOutput, error)
}

func (f *fakeProvider) Run(ctx context.Context, input ProviderInput) (ProviderOutput, error) {
	f.calls = append(f.calls, input)
	if f.run != nil {
		return f.run(ctx, input)
	}
	if len(f.steps) == 0 {
		return ProviderOutput{}, errors.New("missing provider step")
	}
	step := f.steps[0]
	f.steps = f.steps[1:]
	return step, nil
}

type fakePromptBuilder struct {
	prompt string
	err    error
}

func (f fakePromptBuilder) Build(context.Context, Session, Input) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return f.prompt, nil
}

func newTestRunner(dialer Dialer, maxSteps int) CampaignTurnRunner {
	return NewRunner(RunnerConfig{
		Dialer:   dialer,
		MaxSteps: maxSteps,
	})
}

func baseSessionResources(participantID string, activeSceneID string) map[string]string {
	return map[string]string{
		"campaign://camp-1/artifacts/skills.md":    "# GM Skills\nUse tools.",
		"campaign://camp-1/artifacts/memory.md":    "",
		"context://current":                        `{"context":{"campaign_id":"camp-1","session_id":"sess-1","participant_id":"` + participantID + `"}}`,
		"campaign://camp-1":                        `{"campaign":{"id":"camp-1","name":"Ashes","theme_prompt":"Ruined empire"}}`,
		"campaign://camp-1/participants":           `{"participants":[{"id":"` + participantID + `","role":"GM"},{"id":"p-1","role":"PLAYER"}]}`,
		"campaign://camp-1/characters":             `{"characters":[{"id":"char-1","name":"Theron"}]}`,
		"campaign://camp-1/sessions":               `{"sessions":[{"id":"sess-1","status":"ACTIVE"}]}`,
		"campaign://camp-1/sessions/sess-1/scenes": `{"scenes":[{"scene_id":"scene-1"}]}`,
		"campaign://camp-1/interaction":            `{"campaign_id":"camp-1","active_session":{"session_id":"sess-1"},"active_scene":{"scene_id":"` + activeSceneID + `"}}`,
	}
}

func TestRunnerRunsToolLoopWithCuratedTools(t *testing.T) {
	sess := &fakeSession{
		tools: []Tool{
			{Name: "scene_create"},
			{Name: "interaction_active_scene_set"},
			{Name: "interaction_scene_gm_output_commit"},
			{Name: "roll_dice"},
		},
		resources: baseSessionResources("gm-1", "scene-1"),
		results: map[string]ToolResult{
			"interaction_scene_gm_output_commit": {Output: `{"campaign_id":"camp-1","active_scene":{"scene_id":"scene-1"}}`},
		},
	}
	sess.resources["campaign://camp-1/artifacts/memory.md"] = "Remember the lighthouse omen."
	sess.resources["campaign://camp-1/sessions/sess-1/scenes"] = `{"scenes":[]}`

	provider := &fakeProvider{
		steps: []ProviderOutput{
			{
				ConversationID: "resp-1",
				Usage:          providerpkg.Usage{InputTokens: 10, OutputTokens: 4, ReasoningTokens: 1, TotalTokens: 14},
				ToolCalls: []ProviderToolCall{{
					CallID:    "call-1",
					Name:      "interaction_scene_gm_output_commit",
					Arguments: `{"scene_id":"scene-1","text":"The GM describes the ruined city."}`,
				}},
			},
			{
				ConversationID: "resp-2",
				OutputText:     "The GM describes the ruined city.",
				Usage:          providerpkg.Usage{InputTokens: 6, OutputTokens: 8, ReasoningTokens: 2, TotalTokens: 14},
			},
		},
	}

	res, err := newTestRunner(&fakeDialer{sess: sess}, 4).Run(context.Background(), Input{
		CampaignID:       "camp-1",
		SessionID:        "sess-1",
		ParticipantID:    "gm-1",
		Input:            "Advance the scene.",
		Model:            "gpt-4.1-mini",
		Instructions:     "Be concise.",
		CredentialSecret: "sk-1",
		Provider:         provider,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.OutputText != "The GM describes the ruined city." {
		t.Fatalf("output = %q", res.OutputText)
	}
	if !reflect.DeepEqual(sess.calls, []string{"interaction_scene_gm_output_commit"}) {
		t.Fatalf("tool calls = %#v", sess.calls)
	}
	if got := toolNames(provider.calls[0].Tools); !reflect.DeepEqual(got, []string{"scene_create", "interaction_active_scene_set", "interaction_scene_gm_output_commit", "roll_dice"}) {
		t.Fatalf("filtered tools = %#v", got)
	}
	if provider.calls[1].ConversationID != "resp-1" {
		t.Fatalf("conversation id = %q", provider.calls[1].ConversationID)
	}
	if len(provider.calls[1].Results) != 1 || provider.calls[1].Results[0].CallID != "call-1" {
		t.Fatalf("tool results = %#v", provider.calls[1].Results)
	}
	if res.Usage != (providerpkg.Usage{InputTokens: 16, OutputTokens: 12, ReasoningTokens: 3, TotalTokens: 28}) {
		t.Fatalf("usage = %#v", res.Usage)
	}
}

func TestRunnerRejectsFinalOutputWithoutNarrationCommit(t *testing.T) {
	sess := &fakeSession{
		tools:     []Tool{{Name: "campaign"}},
		resources: baseSessionResources("gm-1", "scene-1"),
	}
	provider := &fakeProvider{
		steps: []ProviderOutput{
			{ConversationID: "resp-1", OutputText: "Narration without an authoritative write."},
			{ConversationID: "resp-2", OutputText: "Still no commit."},
		},
	}

	_, err := newTestRunner(&fakeDialer{sess: sess}, 2).Run(context.Background(), Input{
		CampaignID:       "camp-1",
		SessionID:        "sess-1",
		ParticipantID:    "gm-1",
		Model:            "gpt-4.1-mini",
		CredentialSecret: "sk-1",
		Provider:         provider,
	})
	if !errors.Is(err, ErrNarrationNotCommitted) {
		t.Fatalf("err = %v, want %v", err, ErrNarrationNotCommitted)
	}
	if !strings.Contains(provider.calls[1].FollowUpPrompt, "interaction_scene_gm_output_commit") {
		t.Fatalf("follow-up prompt = %q", provider.calls[1].FollowUpPrompt)
	}
}

func TestRunnerRejectsToolCallsOutsideCuratedAllowlist(t *testing.T) {
	sess := &fakeSession{
		tools: []Tool{
			{Name: "scene_create"},
			{Name: "interaction_scene_gm_output_commit"},
		},
		resources: baseSessionResources("gm-1", "scene-1"),
		results: map[string]ToolResult{
			"interaction_scene_gm_output_commit": {Output: `{"campaign_id":"camp-1","active_scene":{"scene_id":"scene-1"}}`},
		},
	}
	provider := &fakeProvider{
		steps: []ProviderOutput{
			{
				ConversationID: "resp-1",
				ToolCalls: []ProviderToolCall{
					{CallID: "call-1", Name: "campaign_create", Arguments: `{"name":"Nope"}`},
					{CallID: "call-2", Name: "interaction_scene_gm_output_commit", Arguments: `{"scene_id":"scene-1","text":"The scene opens."}`},
				},
			},
			{ConversationID: "resp-2", OutputText: "The scene opens."},
		},
	}

	res, err := newTestRunner(&fakeDialer{sess: sess}, 4).Run(context.Background(), Input{
		CampaignID:       "camp-1",
		SessionID:        "sess-1",
		ParticipantID:    "gm-1",
		Model:            "gpt-4.1-mini",
		CredentialSecret: "sk-1",
		Provider:         provider,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.OutputText != "The scene opens." {
		t.Fatalf("output = %q", res.OutputText)
	}
	if !reflect.DeepEqual(sess.calls, []string{"interaction_scene_gm_output_commit"}) {
		t.Fatalf("tool calls = %#v", sess.calls)
	}
	if !provider.calls[1].Results[0].IsError || !strings.Contains(provider.calls[1].Results[0].Output, "not allowed") {
		t.Fatalf("tool result = %#v", provider.calls[1].Results[0])
	}
}

func TestRunnerBootstrapAllowsCreateActivateCommitSequence(t *testing.T) {
	sess := &fakeSession{
		tools: []Tool{
			{Name: "scene_create"},
			{Name: "interaction_active_scene_set"},
			{Name: "interaction_scene_gm_output_commit"},
		},
		resources: baseSessionResources("gm-ai", ""),
		results: map[string]ToolResult{
			"scene_create":                       {Output: `{"scene_id":"scene-1","campaign_id":"camp-1","session_id":"sess-1"}`},
			"interaction_active_scene_set":       {Output: `{"campaign_id":"camp-1","active_scene":{"scene_id":"scene-1"}}`},
			"interaction_scene_gm_output_commit": {Output: `{"campaign_id":"camp-1","active_scene":{"scene_id":"scene-1"}}`},
		},
	}
	sess.resources["campaign://camp-1/sessions/sess-1/scenes"] = `{"scenes":[]}`

	provider := &fakeProvider{
		steps: []ProviderOutput{
			{
				ConversationID: "resp-1",
				ToolCalls: []ProviderToolCall{
					{CallID: "call-1", Name: "scene_create", Arguments: `{"name":"Opening","description":"Night fog","character_ids":["char-1"]}`},
					{CallID: "call-2", Name: "interaction_active_scene_set", Arguments: `{"scene_id":"scene-1"}`},
					{CallID: "call-3", Name: "interaction_scene_gm_output_commit", Arguments: `{"scene_id":"scene-1","text":"The scene opens in fog."}`},
				},
			},
			{ConversationID: "resp-2", OutputText: "The scene opens in fog."},
		},
	}

	res, err := newTestRunner(&fakeDialer{sess: sess}, 4).Run(context.Background(), Input{
		CampaignID:       "camp-1",
		SessionID:        "sess-1",
		ParticipantID:    "gm-ai",
		Model:            "gpt-4.1-mini",
		CredentialSecret: "sk-1",
		Provider:         provider,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.OutputText != "The scene opens in fog." {
		t.Fatalf("output = %q", res.OutputText)
	}
	if !reflect.DeepEqual(sess.calls, []string{"scene_create", "interaction_active_scene_set", "interaction_scene_gm_output_commit"}) {
		t.Fatalf("tool calls = %#v", sess.calls)
	}
}

func TestRunnerPromptsProviderToCommitDraftNarration(t *testing.T) {
	sess := &fakeSession{
		tools:     []Tool{{Name: "interaction_scene_gm_output_commit"}},
		resources: baseSessionResources("gm-ai", "scene-1"),
		results: map[string]ToolResult{
			"interaction_scene_gm_output_commit": {Output: `{"campaign_id":"camp-1","active_scene":{"scene_id":"scene-1"}}`},
		},
	}
	provider := &fakeProvider{
		steps: []ProviderOutput{
			{ConversationID: "resp-1", OutputText: "Fog gathers at the pier."},
			{
				ConversationID: "resp-2",
				ToolCalls: []ProviderToolCall{{
					CallID:    "call-1",
					Name:      "interaction_scene_gm_output_commit",
					Arguments: `{"scene_id":"scene-1","text":"Fog gathers at the pier."}`,
				}},
			},
			{ConversationID: "resp-3", OutputText: "Fog gathers at the pier."},
		},
	}

	res, err := newTestRunner(&fakeDialer{sess: sess}, 4).Run(context.Background(), Input{
		CampaignID:       "camp-1",
		SessionID:        "sess-1",
		ParticipantID:    "gm-ai",
		Model:            "gpt-4.1-mini",
		CredentialSecret: "sk-1",
		Provider:         provider,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.OutputText != "Fog gathers at the pier." {
		t.Fatalf("output = %q", res.OutputText)
	}
	if len(provider.calls) < 2 || !strings.Contains(provider.calls[1].FollowUpPrompt, "Fog gathers at the pier.") {
		t.Fatalf("follow-up prompt = %#v", provider.calls)
	}
}

func TestRunnerAppliesToolResultBudget(t *testing.T) {
	sess := &fakeSession{
		tools:     []Tool{{Name: "interaction_scene_gm_output_commit"}},
		resources: baseSessionResources("gm-ai", "scene-1"),
		results: map[string]ToolResult{
			"interaction_scene_gm_output_commit": {Output: strings.Repeat("x", 256)},
		},
	}
	provider := &fakeProvider{
		steps: []ProviderOutput{
			{
				ConversationID: "resp-1",
				ToolCalls: []ProviderToolCall{{
					CallID:    "call-1",
					Name:      "interaction_scene_gm_output_commit",
					Arguments: `{"scene_id":"scene-1","text":"Budget test."}`,
				}},
			},
			{ConversationID: "resp-2", OutputText: "Budget test."},
		},
	}

	res, err := NewRunner(RunnerConfig{
		Dialer:             &fakeDialer{sess: sess},
		MaxSteps:           4,
		ToolResultMaxBytes: 96,
	}).Run(context.Background(), Input{
		CampaignID:       "camp-1",
		SessionID:        "sess-1",
		ParticipantID:    "gm-ai",
		Model:            "gpt-4.1-mini",
		CredentialSecret: "sk-1",
		Provider:         provider,
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.OutputText != "Budget test." {
		t.Fatalf("output = %q", res.OutputText)
	}
	got := provider.calls[1].Results[0].Output
	if len(got) > 96 {
		t.Fatalf("tool result len = %d, want <= 96", len(got))
	}
	if !strings.Contains(got, "truncated by AI orchestration tool-result budget") {
		t.Fatalf("tool result = %q", got)
	}
}

func TestRunnerHonorsTurnTimeout(t *testing.T) {
	sess := &fakeSession{
		tools:     []Tool{{Name: "interaction_scene_gm_output_commit"}},
		resources: baseSessionResources("gm-ai", "scene-1"),
	}
	provider := &fakeProvider{
		run: func(ctx context.Context, input ProviderInput) (ProviderOutput, error) {
			<-ctx.Done()
			return ProviderOutput{}, ctx.Err()
		},
	}

	_, err := NewRunner(RunnerConfig{
		Dialer:      &fakeDialer{sess: sess},
		MaxSteps:    4,
		TurnTimeout: 5 * time.Millisecond,
	}).Run(context.Background(), Input{
		CampaignID:       "camp-1",
		SessionID:        "sess-1",
		ParticipantID:    "gm-ai",
		Model:            "gpt-4.1-mini",
		CredentialSecret: "sk-1",
		Provider:         provider,
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context deadline exceeded", err)
	}
	if got := apperrors.GetCode(err); got != apperrors.CodeAIOrchestrationTimedOut {
		t.Fatalf("error code = %v, want %v", got, apperrors.CodeAIOrchestrationTimedOut)
	}
}

func TestRunnerWrapsPromptBuildFailures(t *testing.T) {
	sess := &fakeSession{tools: []Tool{{Name: "interaction_scene_gm_output_commit"}}}
	_, err := NewRunner(RunnerConfig{
		Dialer:        &fakeDialer{sess: sess},
		PromptBuilder: fakePromptBuilder{err: errors.New("boom")},
		MaxSteps:      4,
	}).Run(context.Background(), Input{
		CampaignID:       "camp-1",
		SessionID:        "sess-1",
		ParticipantID:    "gm-ai",
		Model:            "gpt-4.1-mini",
		CredentialSecret: "sk-1",
		Provider:         &fakeProvider{},
	})
	if got := apperrors.GetCode(err); got != apperrors.CodeAIOrchestrationPromptBuildFailed {
		t.Fatalf("error code = %v, want %v", got, apperrors.CodeAIOrchestrationPromptBuildFailed)
	}
}

func TestRunnerEmitsSpansForRunAndToolCalls(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer func() {
		otel.SetTracerProvider(prev)
		_ = tp.Shutdown(context.Background())
	}()

	sess := &fakeSession{
		tools:     []Tool{{Name: "interaction_scene_gm_output_commit"}},
		resources: baseSessionResources("gm-ai", "scene-1"),
		results: map[string]ToolResult{
			"interaction_scene_gm_output_commit": {Output: `{"ok":true}`},
		},
	}
	provider := &fakeProvider{
		steps: []ProviderOutput{
			{
				ConversationID: "resp-1",
				ToolCalls: []ProviderToolCall{{
					CallID:    "call-1",
					Name:      "interaction_scene_gm_output_commit",
					Arguments: `{"scene_id":"scene-1","text":"Span test."}`,
				}},
			},
			{ConversationID: "resp-2", OutputText: "Span test."},
		},
	}

	if _, err := newTestRunner(&fakeDialer{sess: sess}, 4).Run(context.Background(), Input{
		CampaignID:       "camp-1",
		SessionID:        "sess-1",
		ParticipantID:    "gm-ai",
		Model:            "gpt-4.1-mini",
		CredentialSecret: "sk-1",
		Provider:         provider,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}

	names := make([]string, 0, len(exporter.GetSpans()))
	for _, span := range exporter.GetSpans() {
		names = append(names, span.Name)
	}
	if !containsSpanName(names, "ai.orchestration.run") {
		t.Fatalf("span names = %#v", names)
	}
	if !containsSpanName(names, "ai.orchestration.provider_step") {
		t.Fatalf("span names = %#v", names)
	}
	if !containsSpanName(names, "ai.orchestration.build_prompt") {
		t.Fatalf("span names = %#v", names)
	}
}

func toolNames(tools []Tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	return names
}

func containsSpanName(names []string, want string) bool {
	for _, name := range names {
		if name == want {
			return true
		}
	}
	return false
}
