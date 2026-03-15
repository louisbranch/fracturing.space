package orchestration

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
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
}

func (f *fakeProvider) Run(_ context.Context, input ProviderInput) (ProviderOutput, error) {
	f.calls = append(f.calls, input)
	if len(f.steps) == 0 {
		return ProviderOutput{}, errors.New("missing provider step")
	}
	step := f.steps[0]
	f.steps = f.steps[1:]
	return step, nil
}

func TestRunnerRunsToolLoopWithCuratedTools(t *testing.T) {
	sess := &fakeSession{
		tools: []Tool{
			{Name: "set_context"},
			{Name: "campaign"},
			{Name: "campaign_end"},
			{Name: "scene_create"},
			{Name: "interaction_active_scene_set"},
			{Name: "interaction_scene_gm_output_commit"},
			{Name: "roll_dice"},
		},
		resources: map[string]string{
			"campaign://camp-1/artifacts/skills.md":    "# GM Skills\nUse tools.",
			"campaign://camp-1/artifacts/memory.md":    "Remember the lighthouse omen.",
			"context://current":                        `{"context":{"campaign_id":"camp-1","session_id":"sess-1","participant_id":"gm-1"}}`,
			"campaign://camp-1":                        `{"campaign":{"id":"camp-1","name":"Ashes","theme_prompt":"Ruined empire"}}`,
			"campaign://camp-1/participants":           `{"participants":[{"id":"gm-1","role":"GM"},{"id":"p-1","role":"PLAYER"}]}`,
			"campaign://camp-1/characters":             `{"characters":[{"id":"char-1","name":"Theron"}]}`,
			"campaign://camp-1/sessions":               `{"sessions":[{"id":"sess-1","status":"ACTIVE"}]}`,
			"campaign://camp-1/sessions/sess-1/scenes": `{"scenes":[]}`,
			"campaign://camp-1/interaction":            `{"campaign_id":"camp-1","active_session":{"session_id":"sess-1"},"active_scene":{"scene_id":"scene-1"}}`,
		},
		results: map[string]ToolResult{
			"set_context":                        {Output: `{"context":{"campaign_id":"camp-1","session_id":"sess-1","participant_id":"gm-1"}}`},
			"campaign":                           {Output: `{"id":"camp-1","name":"Ashes"}`},
			"interaction_scene_gm_output_commit": {Output: `{"campaign_id":"camp-1","active_scene":{"scene_id":"scene-1"}}`},
		},
	}
	provider := &fakeProvider{
		steps: []ProviderOutput{
			{
				ConversationID: "resp-1",
				ToolCalls: []ProviderToolCall{
					{
						CallID:    "call-1",
						Name:      "campaign",
						Arguments: `{"campaign_id":"camp-1"}`,
					},
					{
						CallID:    "call-2",
						Name:      "interaction_scene_gm_output_commit",
						Arguments: `{"scene_id":"scene-1","text":"The GM describes the ruined city."}`,
					},
				},
			},
			{
				ConversationID: "resp-2",
				OutputText:     "The GM describes the ruined city.",
			},
		},
	}

	res, err := NewRunner(&fakeDialer{sess: sess}, 4).Run(context.Background(), Input{
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
	if !reflect.DeepEqual(sess.calls, []string{"set_context", "campaign", "interaction_scene_gm_output_commit"}) {
		t.Fatalf("tool calls = %#v", sess.calls)
	}
	if got := sess.args["set_context"]; !reflect.DeepEqual(got, map[string]any{
		"campaign_id":    "camp-1",
		"session_id":     "sess-1",
		"participant_id": "gm-1",
	}) {
		t.Fatalf("set_context args = %#v", got)
	}
	if len(provider.calls) != 2 {
		t.Fatalf("provider calls = %d", len(provider.calls))
	}
	if got := toolNames(provider.calls[0].Tools); !reflect.DeepEqual(got, []string{"campaign", "scene_create", "interaction_active_scene_set", "interaction_scene_gm_output_commit", "roll_dice"}) {
		t.Fatalf("filtered tools = %#v", got)
	}
	if provider.calls[1].ConversationID != "resp-1" {
		t.Fatalf("conversation id = %q", provider.calls[1].ConversationID)
	}
	if len(provider.calls[1].Results) != 2 || provider.calls[1].Results[1].CallID != "call-2" {
		t.Fatalf("tool results = %#v", provider.calls[1].Results)
	}
}

func TestRunnerRejectsFinalOutputWithoutNarrationCommit(t *testing.T) {
	sess := &fakeSession{
		tools: []Tool{
			{Name: "set_context"},
			{Name: "campaign"},
		},
		resources: map[string]string{
			"campaign://camp-1/artifacts/skills.md":    "# GM Skills\nUse tools.",
			"campaign://camp-1/artifacts/memory.md":    "",
			"context://current":                        `{"context":{"campaign_id":"camp-1","session_id":"sess-1","participant_id":"gm-1"}}`,
			"campaign://camp-1":                        `{"campaign":{"id":"camp-1"}}`,
			"campaign://camp-1/participants":           `{"participants":[]}`,
			"campaign://camp-1/characters":             `{"characters":[]}`,
			"campaign://camp-1/sessions":               `{"sessions":[{"id":"sess-1","status":"ACTIVE"}]}`,
			"campaign://camp-1/sessions/sess-1/scenes": `{"scenes":[]}`,
			"campaign://camp-1/interaction":            `{"campaign_id":"camp-1","active_session":{"session_id":"sess-1"},"active_scene":{"scene_id":"scene-1"}}`,
		},
		results: map[string]ToolResult{
			"set_context": {Output: `{"context":{"campaign_id":"camp-1","session_id":"sess-1","participant_id":"gm-1"}}`},
		},
	}
	provider := &fakeProvider{
		steps: []ProviderOutput{
			{
				ConversationID: "resp-1",
				OutputText:     "Narration without an authoritative write.",
			},
			{
				ConversationID: "resp-2",
				OutputText:     "Still no commit.",
			},
		},
	}

	_, err := NewRunner(&fakeDialer{sess: sess}, 2).Run(context.Background(), Input{
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
	if len(provider.calls) != 2 {
		t.Fatalf("provider calls = %d", len(provider.calls))
	}
	if !strings.Contains(provider.calls[1].FollowUpPrompt, "interaction_scene_gm_output_commit") {
		t.Fatalf("follow-up prompt = %q", provider.calls[1].FollowUpPrompt)
	}
}

func TestRunnerRejectsToolCallsOutsideCuratedAllowlist(t *testing.T) {
	sess := &fakeSession{
		tools: []Tool{
			{Name: "set_context"},
			{Name: "scene_create"},
			{Name: "interaction_scene_gm_output_commit"},
		},
		resources: map[string]string{
			"campaign://camp-1/artifacts/skills.md":    "# GM Skills\nUse tools.",
			"campaign://camp-1/artifacts/memory.md":    "",
			"context://current":                        `{"context":{"campaign_id":"camp-1","session_id":"sess-1","participant_id":"gm-1"}}`,
			"campaign://camp-1":                        `{"campaign":{"id":"camp-1"}}`,
			"campaign://camp-1/participants":           `{"participants":[]}`,
			"campaign://camp-1/characters":             `{"characters":[]}`,
			"campaign://camp-1/sessions":               `{"sessions":[{"id":"sess-1","status":"ACTIVE"}]}`,
			"campaign://camp-1/sessions/sess-1/scenes": `{"scenes":[]}`,
			"campaign://camp-1/interaction":            `{"campaign_id":"camp-1","active_session":{"session_id":"sess-1"},"active_scene":{"scene_id":"scene-1"}}`,
		},
		results: map[string]ToolResult{
			"set_context":                        {Output: `{"context":{"campaign_id":"camp-1","session_id":"sess-1","participant_id":"gm-1"}}`},
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
			{
				ConversationID: "resp-2",
				OutputText:     "The scene opens.",
			},
		},
	}

	res, err := NewRunner(&fakeDialer{sess: sess}, 4).Run(context.Background(), Input{
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
	if reflect.DeepEqual(sess.calls, []string{"set_context", "campaign_create", "interaction_scene_gm_output_commit"}) {
		t.Fatalf("unexpected disallowed tool execution: %#v", sess.calls)
	}
	if !reflect.DeepEqual(sess.calls, []string{"set_context", "interaction_scene_gm_output_commit"}) {
		t.Fatalf("tool calls = %#v", sess.calls)
	}
	if len(provider.calls) != 2 || len(provider.calls[1].Results) != 2 {
		t.Fatalf("provider calls = %#v", provider.calls)
	}
	if !provider.calls[1].Results[0].IsError || !strings.Contains(provider.calls[1].Results[0].Output, "not allowed") {
		t.Fatalf("tool result = %#v", provider.calls[1].Results[0])
	}
}

func TestBuildPromptUsesBootstrapModeWithoutActiveScene(t *testing.T) {
	sess := &fakeSession{
		resources: map[string]string{
			"campaign://camp-1/artifacts/skills.md":    "# GM Skills\nUse tools.",
			"campaign://camp-1/artifacts/memory.md":    "Session memory.",
			"context://current":                        `{"context":{"campaign_id":"camp-1","session_id":"sess-1","participant_id":"gm-1"}}`,
			"campaign://camp-1":                        `{"campaign":{"id":"camp-1","name":"Ashes","theme_prompt":"Ruined empire"}}`,
			"campaign://camp-1/participants":           `{"participants":[{"id":"gm-1","role":"GM"},{"id":"p-1","role":"PLAYER"}]}`,
			"campaign://camp-1/characters":             `{"characters":[{"id":"char-1","name":"Theron","notes":"Former sentinel"}]}`,
			"campaign://camp-1/sessions":               `{"sessions":[{"id":"sess-1","status":"ACTIVE"}]}`,
			"campaign://camp-1/sessions/sess-1/scenes": `{"scenes":[]}`,
			"campaign://camp-1/interaction":            `{"campaign_id":"camp-1","active_session":{"session_id":"sess-1"},"active_scene":{"scene_id":""}}`,
		},
	}

	prompt, err := buildPrompt(context.Background(), sess, Input{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
	})
	if err != nil {
		t.Fatalf("buildPrompt() error = %v", err)
	}
	if !strings.Contains(prompt, "Bootstrap mode: there is no active scene yet.") {
		t.Fatalf("prompt missing bootstrap instructions: %q", prompt)
	}
	if !strings.Contains(prompt, "Scenes:\n{\"scenes\":[]}") && !strings.Contains(prompt, "\"scenes\":[]") {
		t.Fatalf("prompt missing scenes section: %q", prompt)
	}
}

func TestRunnerBootstrapAllowsCreateActivateCommitSequence(t *testing.T) {
	sess := &fakeSession{
		tools: []Tool{
			{Name: "set_context"},
			{Name: "scene_create"},
			{Name: "interaction_active_scene_set"},
			{Name: "interaction_scene_gm_output_commit"},
		},
		resources: map[string]string{
			"campaign://camp-1/artifacts/skills.md":    "# GM Skills\nUse tools.",
			"campaign://camp-1/artifacts/memory.md":    "",
			"context://current":                        `{"context":{"campaign_id":"camp-1","session_id":"sess-1","participant_id":"gm-ai"}}`,
			"campaign://camp-1":                        `{"campaign":{"id":"camp-1"}}`,
			"campaign://camp-1/participants":           `{"participants":[{"id":"gm-ai","role":"GM"}]}`,
			"campaign://camp-1/characters":             `{"characters":[{"id":"char-1","name":"Theron"}]}`,
			"campaign://camp-1/sessions":               `{"sessions":[{"id":"sess-1","status":"ACTIVE"}]}`,
			"campaign://camp-1/sessions/sess-1/scenes": `{"scenes":[]}`,
			"campaign://camp-1/interaction":            `{"campaign_id":"camp-1","active_session":{"session_id":"sess-1"},"active_scene":{"scene_id":""}}`,
		},
		results: map[string]ToolResult{
			"set_context":                        {Output: `{"context":{"campaign_id":"camp-1","session_id":"sess-1","participant_id":"gm-ai"}}`},
			"scene_create":                       {Output: `{"scene_id":"scene-1","campaign_id":"camp-1","session_id":"sess-1"}`},
			"interaction_active_scene_set":       {Output: `{"campaign_id":"camp-1","active_scene":{"scene_id":"scene-1"}}`},
			"interaction_scene_gm_output_commit": {Output: `{"campaign_id":"camp-1","active_scene":{"scene_id":"scene-1"}}`},
		},
	}
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
			{
				ConversationID: "resp-2",
				OutputText:     "The scene opens in fog.",
			},
		},
	}

	res, err := NewRunner(&fakeDialer{sess: sess}, 4).Run(context.Background(), Input{
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
	if !reflect.DeepEqual(sess.calls, []string{"set_context", "scene_create", "interaction_active_scene_set", "interaction_scene_gm_output_commit"}) {
		t.Fatalf("tool calls = %#v", sess.calls)
	}
}

func TestRunnerPromptsProviderToCommitDraftNarration(t *testing.T) {
	sess := &fakeSession{
		tools: []Tool{
			{Name: "set_context"},
			{Name: "interaction_scene_gm_output_commit"},
		},
		resources: map[string]string{
			"campaign://camp-1/artifacts/skills.md":    "# GM Skills\nUse tools.",
			"campaign://camp-1/artifacts/memory.md":    "",
			"context://current":                        `{"context":{"campaign_id":"camp-1","session_id":"sess-1","participant_id":"gm-ai"}}`,
			"campaign://camp-1":                        `{"campaign":{"id":"camp-1"}}`,
			"campaign://camp-1/participants":           `{"participants":[{"id":"gm-ai","role":"GM"}]}`,
			"campaign://camp-1/characters":             `{"characters":[]}`,
			"campaign://camp-1/sessions":               `{"sessions":[{"id":"sess-1","status":"ACTIVE"}]}`,
			"campaign://camp-1/sessions/sess-1/scenes": `{"scenes":[{"scene_id":"scene-1"}]}`,
			"campaign://camp-1/interaction":            `{"campaign_id":"camp-1","active_session":{"session_id":"sess-1"},"active_scene":{"scene_id":"scene-1"}}`,
		},
		results: map[string]ToolResult{
			"set_context":                        {Output: `{"context":{"campaign_id":"camp-1","session_id":"sess-1","participant_id":"gm-ai"}}`},
			"interaction_scene_gm_output_commit": {Output: `{"campaign_id":"camp-1","active_scene":{"scene_id":"scene-1"}}`},
		},
	}
	provider := &fakeProvider{
		steps: []ProviderOutput{
			{
				ConversationID: "resp-1",
				OutputText:     "Fog gathers at the pier.",
			},
			{
				ConversationID: "resp-2",
				ToolCalls: []ProviderToolCall{
					{CallID: "call-1", Name: "interaction_scene_gm_output_commit", Arguments: `{"scene_id":"scene-1","text":"Fog gathers at the pier."}`},
				},
			},
			{
				ConversationID: "resp-3",
				OutputText:     "Fog gathers at the pier.",
			},
		},
	}

	res, err := NewRunner(&fakeDialer{sess: sess}, 4).Run(context.Background(), Input{
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

func toolNames(tools []Tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	return names
}
