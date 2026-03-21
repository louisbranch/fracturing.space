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

	prompt, err := newDefaultPromptBuilder().Build(context.Background(), sess, Input{
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

	reg := coreSourceRegistry()
	pb := NewPromptBuilder(PromptBuilderConfig{
		Skills:              "# Custom Skills\nBe awesome.",
		InteractionContract: "# Custom Interaction\nCommit everything.",
		ContextSources:      reg,
	})

	prompt, err := pb.Build(context.Background(), sess, Input{
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

func TestPromptBuilderMergesContextSourceSections(t *testing.T) {
	sess := &fakeSession{resources: baseSessionResources("gm-1", "scene-1")}
	sess.resources["campaign://camp-1/artifacts/memory.md"] = ""

	reg := coreSourceRegistry()
	reg.Register(ContextSourceFunc(func(_ context.Context, _ Session, _ Input) ([]BriefSection, error) {
		return []BriefSection{{
			ID:       "custom_source",
			Priority: 250,
			Label:    "Custom Source",
			Content:  "Extra context from a game system.",
		}}, nil
	}))

	pb := NewPromptBuilder(PromptBuilderConfig{
		ContextSources: reg,
	})

	prompt, err := pb.Build(context.Background(), sess, Input{
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

	prompt, err := newDefaultPromptBuilder().Build(context.Background(), sess, Input{
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

// coreSourceRegistry builds a registry with CoreContextSources for tests
// that construct a PromptBuilder via NewPromptBuilder.
func coreSourceRegistry() *ContextSourceRegistry {
	reg := NewContextSourceRegistry()
	for _, src := range CoreContextSources() {
		reg.Register(src)
	}
	return reg
}
