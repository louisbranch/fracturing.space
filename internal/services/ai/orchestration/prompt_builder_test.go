package orchestration

import (
	"context"
	"strings"
	"testing"
)

func TestPromptBuilderUsesBootstrapModeWithoutActiveScene(t *testing.T) {
	sess := &fakeSession{resources: baseSessionResources("gm-1", "")}
	sess.resources["campaign://camp-1/artifacts/memory.md"] = "Session memory."
	sess.resources["campaign://camp-1/sessions/sess-1/scenes"] = `{"scenes":[]}`

	prompt, err := newDegradedPromptBuilder().Build(context.Background(), sess, PromptInput{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !strings.Contains(prompt, "Bootstrap mode: there is no active scene yet.") {
		t.Fatalf("prompt missing bootstrap instructions: %q", prompt)
	}
}

func TestPromptBuilderUsesConfiguredInstructions(t *testing.T) {
	sess := &fakeSession{resources: baseSessionResources("gm-1", "scene-1")}
	sess.resources["campaign://camp-1/artifacts/memory.md"] = ""

	reg := fullSourceRegistry()
	pb := NewPromptBuilder(PromptBuilderConfig{
		Collector: reg,
		Renderer: NewBriefPromptRenderer(BriefPromptRendererConfig{
			Policy: PromptRenderPolicy{
				Instructions: PromptInstructions{
					Skills:              "# Custom Skills\nBe awesome.",
					InteractionContract: "# Custom Interaction\nCommit everything.",
				},
				ClosingInstruction: "Return narrated GM output once you have enough information.",
			},
		}),
	})

	prompt, err := pb.Build(context.Background(), sess, PromptInput{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !strings.Contains(prompt, "Custom Skills") {
		t.Fatalf("prompt missing custom skills")
	}
	if !strings.Contains(prompt, "Custom Interaction") {
		t.Fatalf("prompt missing custom interaction contract")
	}
}

type fakeBriefCollector struct {
	brief  SessionBrief
	err    error
	inputs []PromptInput
}

func (f *fakeBriefCollector) CollectBrief(_ context.Context, _ Session, input PromptInput) (SessionBrief, error) {
	f.inputs = append(f.inputs, input)
	if f.err != nil {
		return SessionBrief{}, f.err
	}
	return f.brief, nil
}

type fakePromptRenderer struct {
	prompt string
	briefs []SessionBrief
	inputs []PromptInput
}

func (f *fakePromptRenderer) Render(brief SessionBrief, input PromptInput) string {
	f.briefs = append(f.briefs, brief)
	f.inputs = append(f.inputs, input)
	return f.prompt
}

func TestPromptBuilderDelegatesToCollectorAndRenderer(t *testing.T) {
	collector := &fakeBriefCollector{
		brief: SessionBrief{
			Sections: []BriefSection{{ID: "campaign", Content: "Campaign data"}},
		},
	}
	renderer := &fakePromptRenderer{prompt: "Rendered prompt"}
	builder := NewPromptBuilder(PromptBuilderConfig{
		Collector: collector,
		Renderer:  renderer,
	})

	prompt, err := builder.Build(context.Background(), &fakeSession{}, PromptInput{
		CampaignID:    "camp-1",
		SessionID:     "sess-1",
		ParticipantID: "gm-1",
		TurnInput:     "Advance the scene.",
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if prompt != "Rendered prompt" {
		t.Fatalf("prompt = %q", prompt)
	}
	if len(collector.inputs) != 1 || collector.inputs[0].TurnInput != "Advance the scene." {
		t.Fatalf("collector inputs = %#v", collector.inputs)
	}
	if len(renderer.briefs) != 1 || renderer.briefs[0].Sections[0].ID != "campaign" {
		t.Fatalf("renderer briefs = %#v", renderer.briefs)
	}
}

func TestPromptBuilderMergesContextSourceSections(t *testing.T) {
	sess := &fakeSession{resources: baseSessionResources("gm-1", "scene-1")}
	sess.resources["campaign://camp-1/artifacts/memory.md"] = ""

	reg := fullSourceRegistry()
	reg.Register(ContextSourceFunc(func(_ context.Context, _ Session, _ PromptInput) (BriefContribution, error) {
		return SectionContribution(BriefSection{
			ID:       "custom_source",
			Priority: 250,
			Label:    "Custom Source",
			Content:  "Extra context from a game system.",
		}), nil
	}))

	pb := NewPromptBuilder(PromptBuilderConfig{
		Collector: reg,
		Renderer: NewBriefPromptRenderer(BriefPromptRendererConfig{
			Policy: DefaultPromptRenderPolicy(),
		}),
	})

	prompt, err := pb.Build(context.Background(), sess, PromptInput{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !strings.Contains(prompt, "Extra context from a game system.") {
		t.Fatalf("prompt missing context source section")
	}
}

func TestPromptBuilderActiveSceneMode(t *testing.T) {
	sess := &fakeSession{resources: baseSessionResources("gm-1", "scene-1")}
	sess.resources["campaign://camp-1/artifacts/memory.md"] = ""

	prompt, err := newDegradedPromptBuilder().Build(context.Background(), sess, PromptInput{
		CampaignID:    "camp-1",
		SessionID:     "sess-1",
		ParticipantID: "gm-1",
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if !strings.Contains(prompt, "Active scene mode") {
		t.Fatalf("prompt missing active scene mode")
	}
	if strings.Contains(prompt, "Bootstrap mode") {
		t.Fatalf("prompt should not have bootstrap mode in active scene")
	}
}

func TestPromptBuilderRejectsMalformedInteractionState(t *testing.T) {
	sess := &fakeSession{resources: baseSessionResources("gm-1", "scene-1")}
	sess.resources["campaign://camp-1/interaction"] = "{not json"

	_, err := newDegradedPromptBuilder().Build(context.Background(), sess, PromptInput{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBriefPromptRendererUsesExplicitClosingInstruction(t *testing.T) {
	renderer := NewBriefPromptRenderer(BriefPromptRendererConfig{
		Policy: PromptRenderPolicy{
			ClosingInstruction: "Stop after the authoritative GM commit.",
		},
	})

	prompt := renderer.Render(SessionBrief{}, PromptInput{})
	if !strings.Contains(prompt, "Stop after the authoritative GM commit.") {
		t.Fatalf("prompt missing explicit closing instruction: %q", prompt)
	}
}

// fullSourceRegistry builds a registry with the core prompt context sources
// for tests that construct a PromptBuilder via NewPromptBuilder. Game-system
// sources (e.g. Daggerheart) are tested in their own subpackages.
func fullSourceRegistry() *ContextSourceRegistry {
	reg := NewContextSourceRegistry()
	for _, src := range CoreContextSources() {
		reg.Register(src)
	}
	return reg
}
