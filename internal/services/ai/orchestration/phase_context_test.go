package orchestration

import (
	"strings"
	"testing"
)

func TestPhaseResourceName(t *testing.T) {
	tests := []struct {
		name string
		mode InteractionTurnMode
		want string
	}{
		{name: "bootstrap", mode: InteractionTurnModeBootstrap, want: "scene-bootstrap"},
		{name: "review", mode: InteractionTurnModeReviewResolution, want: "action-review"},
		{name: "active scene", mode: InteractionTurnModeActiveScene, want: "scene-play"},
		{name: "ooc", mode: InteractionTurnModeOOCOpen, want: "scene-play"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := PhaseResourceName(tc.mode); got != tc.want {
				t.Fatalf("PhaseResourceName(%q) = %q, want %q", tc.mode, got, tc.want)
			}
		})
	}
}

func TestBuildPhaseGuide(t *testing.T) {
	tests := []struct {
		name string
		mode InteractionTurnMode
		want []string
	}{
		{
			name: "bootstrap",
			mode: InteractionTurnModeBootstrap,
			want: []string{"Current phase: scene-bootstrap", "create or choose the next scene", "Prefer summaries and indexes"},
		},
		{
			name: "review",
			mode: InteractionTurnModeReviewResolution,
			want: []string{"Current phase: action-review", "review the submitted player action", "exact rules and sheet details"},
		},
		{
			name: "play",
			mode: InteractionTurnModeActiveScene,
			want: []string{"Current phase: scene-play", "continue the active scene", "deeper story or rules context"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := BuildPhaseGuide(tc.mode, PromptInput{ParticipantID: "gm-1"})
			for _, want := range tc.want {
				if !strings.Contains(got, want) {
					t.Fatalf("BuildPhaseGuide(%q) = %q, want substring %q", tc.mode, got, want)
				}
			}
			if !strings.Contains(got, "Fixed GM participant: gm-1") {
				t.Fatalf("BuildPhaseGuide(%q) = %q, want fixed participant", tc.mode, got)
			}
		})
	}
}

func TestBuildContextAccessMap(t *testing.T) {
	input := PromptInput{CampaignID: "camp-1", SessionID: "sess-1"}

	bootstrap := BuildContextAccessMap(InteractionTurnModeBootstrap, input)
	if !strings.Contains(bootstrap, "campaign://camp-1/sessions/sess-1/scenes") {
		t.Fatalf("bootstrap access map = %q", bootstrap)
	}
	if !strings.Contains(bootstrap, "campaign://camp-1/sessions -> prior session continuity") {
		t.Fatalf("bootstrap access map = %q, want session continuity hint", bootstrap)
	}

	review := BuildContextAccessMap(InteractionTurnModeReviewResolution, input)
	if !strings.Contains(review, "system_reference_search/read") {
		t.Fatalf("review access map = %q, want mechanics lookup hint", review)
	}

	play := BuildContextAccessMap(InteractionTurnModeActiveScene, input)
	if !strings.Contains(play, "campaign://camp-1 -> campaign theme") {
		t.Fatalf("play access map = %q, want campaign theme hint", play)
	}
	if !strings.Contains(play, "read the acting character sheet") {
		t.Fatalf("play access map = %q, want sheet hint", play)
	}
}

func TestBuildContextAccessMapOmitsEmptyCampaignID(t *testing.T) {
	if got := BuildContextAccessMap(InteractionTurnModeActiveScene, PromptInput{}); got != "" {
		t.Fatalf("BuildContextAccessMap() = %q, want empty", got)
	}
}

func TestBuildStoryContextIndex(t *testing.T) {
	story := strings.Join([]string{
		"# Act I",
		"The Black Lantern warns of a debt collected at dawn.",
		"## Harbor",
		"The bells toll through the fog.",
	}, "\n")

	got := BuildStoryContextIndex("camp-1", story)
	for _, want := range []string{
		"Use this story index before reading full story.md.",
		"Deep source: campaign://camp-1/artifacts/story.md",
		"Act I",
		"Harbor",
		"The Black Lantern warns of a debt collected at dawn.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("BuildStoryContextIndex() = %q, want substring %q", got, want)
		}
	}
}

func TestBuildStoryContextIndexOmitsBlankStory(t *testing.T) {
	if got := BuildStoryContextIndex("camp-1", "   "); got != "" {
		t.Fatalf("BuildStoryContextIndex() = %q, want empty", got)
	}
}

func TestNormalizeStorySnippetTruncatesLongLines(t *testing.T) {
	input := strings.Repeat("word ", 60)
	got := normalizeStorySnippet(input)
	if len(got) >= len(strings.Join(strings.Fields(input), " ")) {
		t.Fatalf("normalizeStorySnippet() = %q, want truncation", got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("normalizeStorySnippet() = %q, want ellipsis", got)
	}
}
