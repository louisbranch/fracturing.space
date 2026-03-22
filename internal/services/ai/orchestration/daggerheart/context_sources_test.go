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
		"daggerheart://rules/version":                                `{"system":"Daggerheart","module":"duality","rules_version":"1.0","dice_model":"2d12","total_formula":"hope+fear+modifier","crit_rule":"doubles","difficulty_rule":"total >= difficulty","outcomes":["CRITICAL_SUCCESS","SUCCESS_WITH_HOPE","SUCCESS_WITH_FEAR","FAILURE"]}`,
		"campaign://camp-1/interaction":                              `{"active_scene":{"scene_id":"scene-1"}}`,
		"campaign://camp-1/sessions/sess-1/scenes":                   `{"scenes":[{"scene_id":"scene-1","character_ids":["char-1"]}]}`,
		"campaign://camp-1/characters/char-1/sheet":                  `{"character":{"id":"char-1","name":"Aria"},"daggerheart":{"level":1,"class":{"name":"Guardian"},"subclass":{"name":"Stalwart"},"heritage":{"ancestry":"Human","community":"Highborne"},"traits":{"agility":2,"strength":1},"resources":{"hp":10,"hp_max":10,"hope":3,"hope_max":6,"stress":2,"armor":1,"life_state":"ALIVE"},"equipment":{"primary_weapon":{"name":"Longsword","trait":"Strength","damage_dice":"1d10"},"active_armor":{"name":"Gambeson Armor"}},"domain_cards":[{"name":"Shield Wall","domain":"Valor"}],"active_class_features":[{"name":"Hold the Line"}],"active_subclass_features":[{"name":"Bulwark"}],"conditions":[{"label":"Vulnerable"}]}}`,
		"daggerheart://campaign/camp-1/sessions/sess-1/combat_board": `{"gm_fear":3,"session_id":"sess-1","scene_id":"scene-1","spotlight":{"type":"CHARACTER","character_id":"char-1"},"countdowns":[{"id":"cd-1","name":"Breach","kind":"CONSEQUENCE","current":2,"max":4,"direction":"INCREASE"}],"adversaries":[{"id":"adv-1","name":"Bandit","scene_id":"scene-1","hp":5,"spotlight_count":1}]}`,
		"daggerheart://campaign/camp-1/snapshot":                     `{"gm_fear":3,"consecutive_short_rests":0,"characters":[{"character_id":"char-1","hp":10,"hope":3,"hope_max":6,"stress":2,"armor":1,"life_state":"ALIVE"}]}`,
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
	if len(sections) != 4 {
		t.Fatalf("got %d sections, want 4", len(sections))
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

	capabilities, ok := ids["daggerheart_active_character_capabilities"]
	if !ok {
		t.Fatal("missing daggerheart_active_character_capabilities section")
	}
	if capabilities.Priority != 225 {
		t.Fatalf("capabilities Priority = %d, want 225", capabilities.Priority)
	}
	if !strings.Contains(capabilities.Content, "Shield Wall") {
		t.Fatalf("capabilities content missing domain card: %q", capabilities.Content)
	}
	if !strings.Contains(capabilities.Content, "Longsword") {
		t.Fatalf("capabilities content missing weapon: %q", capabilities.Content)
	}

	board, ok := ids["daggerheart_combat_board"]
	if !ok {
		t.Fatal("missing daggerheart_combat_board section")
	}
	if board.Priority != 240 {
		t.Fatalf("combat board Priority = %d, want 240", board.Priority)
	}
	if !strings.Contains(board.Content, "Bandit") {
		t.Fatalf("combat board content missing adversary: %q", board.Content)
	}
	if !strings.Contains(board.Content, "CHARACTER") {
		t.Fatalf("combat board content missing spotlight type: %q", board.Content)
	}
	if !strings.Contains(board.Content, "Breach") {
		t.Fatalf("combat board content missing countdown: %q", board.Content)
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
