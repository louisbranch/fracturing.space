package aieval

import (
	"flag"
	"strings"
	"testing"

	evalsupport "github.com/louisbranch/fracturing.space/internal/test/aieval"
)

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("aieval", flag.ContinueOnError)

	cfg, err := ParseConfig(fs, []string{"--scenario", "ai_gm_campaign_context_hope_experience"})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Model != "gpt-5.4" {
		t.Fatalf("model = %q, want gpt-5.4", cfg.Model)
	}
	if cfg.ReasoningEffort != "medium" {
		t.Fatalf("reasoning effort = %q, want medium", cfg.ReasoningEffort)
	}
	if cfg.PromptProfile != "baseline" {
		t.Fatalf("prompt profile = %q, want baseline", cfg.PromptProfile)
	}
}

func TestParseConfigRejectsUnknownScenario(t *testing.T) {
	fs := flag.NewFlagSet("aieval", flag.ContinueOnError)

	if _, err := ParseConfig(fs, []string{"--scenario", "missing"}); err == nil {
		t.Fatal("expected unknown scenario to fail parsing")
	}
}

func TestBuildCommandEnvRemovesEmptyInstructionRoot(t *testing.T) {
	env := buildCommandEnv(Config{
		Model:           "gpt-5.4",
		ReasoningEffort: "medium",
		PromptProfile:   "baseline",
	}, "case-1", "/tmp/out.json")

	for _, entry := range env {
		if entry == "FRACTURING_SPACE_AI_INSTRUCTIONS_ROOT=" {
			t.Fatal("expected empty instructions root to be removed")
		}
	}
}

func TestFirstActionableFailureLineSkipsNoise(t *testing.T) {
	raw := strings.Join([]string{
		"2026/03/24 17:17:16 INFO server listening service=ai addr=127.0.0.1:35341",
		"    ai_campaign_context_live_capture_test.go:178: live capture written to /tmp/capture.json",
		"    ai_campaign_context_live_capture_test.go:178: fixture tool coverage: missing required tool \"daggerheart_action_roll_resolve\"",
		"FAIL",
	}, "\n")

	got := firstActionableFailureLine(raw)
	want := `fixture tool coverage: missing required tool "daggerheart_action_roll_resolve"`
	if got != want {
		t.Fatalf("firstActionableFailureLine() = %q, want %q", got, want)
	}
}

func TestClassifyHarnessFailureMissingAuthoritativeRoll(t *testing.T) {
	result := evalsupport.Output{
		ToolNames: []string{"character_sheet_read", "interaction_resolve_scene_player_review"},
	}
	raw := `ai_campaign_context_live_capture_test.go:178: fixture tool coverage: missing required tool "daggerheart_action_roll_resolve"`

	kind, summary, reason, metricStatus := classifyHarnessFailure(result, raw)
	if kind != "missing_authoritative_roll" {
		t.Fatalf("kind = %q, want missing_authoritative_roll", kind)
	}
	if summary != `missing required tool "daggerheart_action_roll_resolve"` {
		t.Fatalf("summary = %q", summary)
	}
	if reason != `missing required tool "daggerheart_action_roll_resolve"` {
		t.Fatalf("reason = %q", reason)
	}
	if metricStatus != evalsupport.MetricStatusFail {
		t.Fatalf("metricStatus = %q, want fail", metricStatus)
	}
}

func TestClassifyHarnessFailureOverResearch(t *testing.T) {
	raw := `ai_campaign_context_live_capture_test.go:178: system_reference_search calls = 2, want <= 0`

	kind, summary, reason, metricStatus := classifyHarnessFailure(evalsupport.Output{}, raw)
	if kind != "over_research" {
		t.Fatalf("kind = %q, want over_research", kind)
	}
	if summary != "system_reference_search calls = 2, want <= 0" {
		t.Fatalf("summary = %q", summary)
	}
	if reason != "system_reference_search calls = 2, want <= 0" {
		t.Fatalf("reason = %q", reason)
	}
	if metricStatus != evalsupport.MetricStatusFail {
		t.Fatalf("metricStatus = %q, want fail", metricStatus)
	}
}

func TestEnrichFailedResultAddsPromptContext(t *testing.T) {
	result := evalsupport.Output{}
	enrichFailedResult(&result, Config{
		PromptProfile:    "baseline",
		InstructionsRoot: "",
	}, "case-1", "", "harness_error", "failed", "failed", evalsupport.MetricStatusInvalid)

	if result.PromptContext.Profile != "baseline" {
		t.Fatalf("prompt context profile = %q, want baseline", result.PromptContext.Profile)
	}
	if result.PromptContext.Summary == "" {
		t.Fatal("expected prompt context summary")
	}
	if result.CaseID != "case-1" {
		t.Fatalf("case id = %q, want case-1", result.CaseID)
	}
	if result.MetricStatus != evalsupport.MetricStatusInvalid {
		t.Fatalf("metric status = %q, want invalid", result.MetricStatus)
	}
}
