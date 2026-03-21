package orchestration

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestContextSourceRegistryCollectsFromMultipleSources(t *testing.T) {
	reg := NewContextSourceRegistry()
	reg.Register(ContextSourceFunc(func(_ context.Context, _ Session, _ Input) ([]BriefSection, error) {
		return []BriefSection{{ID: "a", Priority: 100, Content: "alpha"}}, nil
	}))
	reg.Register(ContextSourceFunc(func(_ context.Context, _ Session, _ Input) ([]BriefSection, error) {
		return []BriefSection{{ID: "b", Priority: 200, Content: "beta"}}, nil
	}))

	sections, err := reg.CollectSections(context.Background(), nil, Input{})
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
	reg.Register(ContextSourceFunc(func(_ context.Context, _ Session, _ Input) ([]BriefSection, error) {
		return nil, errors.New("boom")
	}))
	reg.Register(ContextSourceFunc(func(_ context.Context, _ Session, _ Input) ([]BriefSection, error) {
		t.Fatal("should not be called after error")
		return nil, nil
	}))

	_, err := reg.CollectSections(context.Background(), nil, Input{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestContextSourceRegistryHandlesNil(t *testing.T) {
	var reg *ContextSourceRegistry
	sections, err := reg.CollectSections(context.Background(), nil, Input{})
	if err != nil || sections != nil {
		t.Fatalf("nil registry: sections=%v, err=%v", sections, err)
	}
}

func TestCoreContextSourcesProduceExpectedSections(t *testing.T) {
	sess := &fakeSession{resources: baseSessionResources("gm-1", "scene-1")}
	sess.resources["campaign://camp-1/artifacts/memory.md"] = "NPC notes."

	reg := NewContextSourceRegistry()
	for _, src := range CoreContextSources() {
		reg.Register(src)
	}

	sections, err := reg.CollectSections(context.Background(), sess, Input{
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

func TestStoryContextSourceOmitsEmptyStory(t *testing.T) {
	sess := &fakeSession{resources: map[string]string{}}

	sections, err := storyContextSource(context.Background(), sess, Input{CampaignID: "camp-1"})
	if err != nil {
		t.Fatalf("storyContextSource() error = %v", err)
	}
	if len(sections) != 0 {
		t.Fatalf("expected no sections for missing story, got %d", len(sections))
	}
}

func TestMemoryContextSourceOmitsEmptyMemory(t *testing.T) {
	sess := &fakeSession{resources: map[string]string{
		"campaign://camp-1/artifacts/memory.md": "   ",
	}}

	sections, err := memoryContextSource(context.Background(), sess, Input{CampaignID: "camp-1"})
	if err != nil {
		t.Fatalf("memoryContextSource() error = %v", err)
	}
	if len(sections) != 0 {
		t.Fatalf("expected no sections for blank memory, got %d", len(sections))
	}
}

func TestDaggerheartContextSourcesProduceExpectedSections(t *testing.T) {
	sess := &fakeSession{resources: baseSessionResources("gm-1", "scene-1")}

	reg := NewContextSourceRegistry()
	for _, src := range DaggerheartContextSources() {
		reg.Register(src)
	}

	sections, err := reg.CollectSections(context.Background(), sess, Input{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
	})
	if err != nil {
		t.Fatalf("CollectSections() error = %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("got %d sections, want 2", len(sections))
	}

	ids := make(map[string]BriefSection, len(sections))
	for _, s := range sections {
		ids[s.ID] = s
	}

	rules, ok := ids["daggerheart_duality_rules"]
	if !ok {
		t.Fatal("missing daggerheart_duality_rules section")
	}
	if rules.Priority != 200 {
		t.Fatalf("duality rules Priority = %d, want 200", rules.Priority)
	}
	if !strings.Contains(rules.Content, "Daggerheart") {
		t.Fatalf("duality rules content missing system name: %q", rules.Content)
	}
	if !strings.Contains(rules.Content, "dice_model") {
		t.Fatalf("duality rules content missing dice_model: %q", rules.Content)
	}

	state, ok := ids["daggerheart_character_state"]
	if !ok {
		t.Fatal("missing daggerheart_character_state section")
	}
	if state.Priority != 250 {
		t.Fatalf("character state Priority = %d, want 250", state.Priority)
	}
	if !strings.Contains(state.Content, "gm_fear") {
		t.Fatalf("character state content missing gm_fear: %q", state.Content)
	}
}

func TestCharacterStateContextSourceContent(t *testing.T) {
	snapshot := `{"gm_fear":5,"consecutive_short_rests":1,"characters":[{"character_id":"char-1","hp":8,"hope":2,"hope_max":6,"stress":3,"armor":1,"life_state":"ALIVE","conditions":[{"label":"Vulnerable","clear_triggers":["SHORT_REST"]}],"temporary_armor":[{"source":"Shield Spell","amount":2}],"stat_modifiers":[{"target":"evasion","delta":1,"label":"Blessing"}]}]}`
	sess := &fakeSession{resources: map[string]string{
		"daggerheart://campaign/camp-1/snapshot": snapshot,
	}}

	sections, err := characterStateContextSource(context.Background(), sess, Input{CampaignID: "camp-1"})
	if err != nil {
		t.Fatalf("characterStateContextSource() error = %v", err)
	}
	if len(sections) != 1 {
		t.Fatalf("got %d sections, want 1", len(sections))
	}
	s := sections[0]
	if s.ID != "daggerheart_character_state" {
		t.Errorf("ID = %q, want %q", s.ID, "daggerheart_character_state")
	}
	if s.Priority != 250 {
		t.Errorf("Priority = %d, want 250", s.Priority)
	}
	if s.Label != "Daggerheart character state" {
		t.Errorf("Label = %q", s.Label)
	}
	// Content is pass-through from ReadResource; verify key fields present.
	for _, want := range []string{"gm_fear", "char-1", "life_state", "Vulnerable", "Shield Spell", "evasion"} {
		if !strings.Contains(s.Content, want) {
			t.Errorf("content missing %q: %q", want, s.Content)
		}
	}
}

func TestMemoryContextSourceIncludesNonEmptyMemory(t *testing.T) {
	sess := &fakeSession{resources: map[string]string{
		"campaign://camp-1/artifacts/memory.md": "## NPCs\nDark merchant at the pier.",
	}}

	sections, err := memoryContextSource(context.Background(), sess, Input{CampaignID: "camp-1"})
	if err != nil {
		t.Fatalf("memoryContextSource() error = %v", err)
	}
	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if !strings.Contains(sections[0].Content, "Dark merchant") {
		t.Fatalf("unexpected content: %q", sections[0].Content)
	}
}
