package orchestration

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestContextSourceRegistryCollectsFromMultipleSources(t *testing.T) {
	reg := NewContextSourceRegistry()
	reg.Register(ContextSourceFunc(func(_ context.Context, _ Session, _ PromptInput) (BriefContribution, error) {
		return SectionContribution(BriefSection{ID: "a", Priority: 100, Content: "alpha"}), nil
	}))
	reg.Register(ContextSourceFunc(func(_ context.Context, _ Session, _ PromptInput) (BriefContribution, error) {
		return SectionContribution(BriefSection{ID: "b", Priority: 200, Content: "beta"}), nil
	}))

	sections, err := reg.CollectSections(context.Background(), nil, PromptInput{})
	if err != nil {
		t.Fatalf("CollectSections() error = %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("got %d sections, want 2", len(sections))
	}
	if sections[0].ID != "a" || sections[1].ID != "b" {
		t.Fatalf("section IDs = %q, %q", sections[0].ID, sections[1].ID)
	}
}

func TestContextSourceRegistryStopsOnError(t *testing.T) {
	reg := NewContextSourceRegistry()
	reg.Register(ContextSourceFunc(func(_ context.Context, _ Session, _ PromptInput) (BriefContribution, error) {
		return BriefContribution{}, errors.New("boom")
	}))
	reg.Register(ContextSourceFunc(func(_ context.Context, _ Session, _ PromptInput) (BriefContribution, error) {
		t.Fatal("should not be called after error")
		return BriefContribution{}, nil
	}))

	_, err := reg.CollectSections(context.Background(), nil, PromptInput{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestContextSourceRegistryHandlesNil(t *testing.T) {
	var reg *ContextSourceRegistry
	sections, err := reg.CollectSections(context.Background(), nil, PromptInput{})
	if err != nil || sections != nil {
		t.Fatalf("nil registry: sections=%v, err=%v", sections, err)
	}
}

func TestContextSourceRegistryCollectsTypedInteractionState(t *testing.T) {
	sess := &fakeSession{resources: baseSessionResources("gm-1", "")}
	sess.resources["campaign://camp-1/artifacts/memory.md"] = ""

	reg := NewContextSourceRegistry()
	for _, src := range CoreContextSources() {
		reg.Register(src)
	}

	brief, err := reg.CollectBrief(context.Background(), sess, PromptInput{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
	})
	if err != nil {
		t.Fatalf("CollectBrief() error = %v", err)
	}
	if brief.InteractionState == nil {
		t.Fatal("missing typed interaction state")
	}
	if !brief.Bootstrap() {
		t.Fatal("expected bootstrap mode without an active scene")
	}
}

func TestCoreContextSourcesProduceExpectedSections(t *testing.T) {
	sess := &fakeSession{resources: baseSessionResources("gm-1", "scene-1")}
	sess.resources["campaign://camp-1/artifacts/memory.md"] = "NPC notes."

	reg := NewContextSourceRegistry()
	for _, src := range CoreContextSources() {
		reg.Register(src)
	}

	sections, err := reg.CollectSections(context.Background(), sess, PromptInput{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
	})
	if err != nil {
		t.Fatalf("CollectSections() error = %v", err)
	}

	ids := make(map[string]bool, len(sections))
	for _, s := range sections {
		ids[s.ID] = true
	}

	expected := []string{"current_context", "campaign", "participants", "characters", "sessions", "scenes", "memory", "interaction_state"}
	for _, id := range expected {
		if !ids[id] {
			t.Errorf("missing section %q", id)
		}
	}
}

func TestInteractionStateContextSourceRejectsMalformedState(t *testing.T) {
	sess := &fakeSession{resources: map[string]string{
		"campaign://camp-1/interaction": "{not json",
	}}

	_, err := interactionStateContextSource(context.Background(), sess, PromptInput{CampaignID: "camp-1"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestStoryContextSourceOmitsEmptyStory(t *testing.T) {
	sess := &fakeSession{resources: map[string]string{}}

	contribution, err := storyContextSource(context.Background(), sess, PromptInput{CampaignID: "camp-1"})
	if err != nil {
		t.Fatalf("storyContextSource() error = %v", err)
	}
	sections := contribution.Sections
	if len(sections) != 0 {
		t.Fatalf("expected no sections for missing story, got %d", len(sections))
	}
}

func TestMemoryContextSourceOmitsEmptyMemory(t *testing.T) {
	sess := &fakeSession{resources: map[string]string{
		"campaign://camp-1/artifacts/memory.md": "   ",
	}}

	contribution, err := memoryContextSource(context.Background(), sess, PromptInput{CampaignID: "camp-1"})
	if err != nil {
		t.Fatalf("memoryContextSource() error = %v", err)
	}
	sections := contribution.Sections
	if len(sections) != 0 {
		t.Fatalf("expected no sections for blank memory, got %d", len(sections))
	}
}

func TestMemoryContextSourceIncludesNonEmptyMemory(t *testing.T) {
	sess := &fakeSession{resources: map[string]string{
		"campaign://camp-1/artifacts/memory.md": "## NPCs\nDark merchant at the pier.",
	}}

	contribution, err := memoryContextSource(context.Background(), sess, PromptInput{CampaignID: "camp-1"})
	if err != nil {
		t.Fatalf("memoryContextSource() error = %v", err)
	}
	sections := contribution.Sections
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if !strings.Contains(sections[0].Content, "Dark merchant") {
		t.Fatalf("unexpected content: %q", sections[0].Content)
	}
}
