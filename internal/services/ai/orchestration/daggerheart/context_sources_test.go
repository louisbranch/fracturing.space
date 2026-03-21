package daggerheart_test

import (
	"context"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	dh "github.com/louisbranch/fracturing.space/internal/services/ai/orchestration/daggerheart"
)

type fakeSession struct {
	resources map[string]string
}

func (f *fakeSession) ListTools(context.Context) ([]orchestration.Tool, error) { return nil, nil }
func (f *fakeSession) CallTool(context.Context, string, any) (orchestration.ToolResult, error) {
	return orchestration.ToolResult{}, nil
}
func (f *fakeSession) ReadResource(_ context.Context, uri string) (string, error) {
	v, ok := f.resources[uri]
	if !ok {
		return "", nil
	}
	return v, nil
}
func (f *fakeSession) Close() error { return nil }

func TestContextSourcesProduceExpectedSections(t *testing.T) {
	sess := &fakeSession{resources: map[string]string{
		"daggerheart://rules/version":            `{"system":"Daggerheart","module":"duality","rules_version":"1.0","dice_model":"2d12","total_formula":"hope+fear+modifier","crit_rule":"doubles","difficulty_rule":"total >= difficulty","outcomes":["CRITICAL_SUCCESS","SUCCESS_WITH_HOPE","SUCCESS_WITH_FEAR","FAILURE"]}`,
		"daggerheart://campaign/camp-1/snapshot": `{"gm_fear":3,"consecutive_short_rests":0,"characters":[{"character_id":"char-1","hp":10,"hope":3,"hope_max":6,"stress":2,"armor":1,"life_state":"ALIVE"}]}`,
	}}

	reg := orchestration.NewContextSourceRegistry()
	for _, src := range dh.ContextSources() {
		reg.Register(src)
	}

	sections, err := reg.CollectSections(context.Background(), sess, orchestration.PromptInput{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
	})
	if err != nil {
		t.Fatalf("CollectSections() error = %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("got %d sections, want 2", len(sections))
	}

	ids := make(map[string]orchestration.BriefSection, len(sections))
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
