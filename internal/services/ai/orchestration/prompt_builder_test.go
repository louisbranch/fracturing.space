package orchestration

import (
	"context"
	"strings"
	"testing"
)

func TestInteractionStateSnapshotTurnModePrefersOOCCloseResolution(t *testing.T) {
	snapshot := InteractionStateSnapshot{
		ActiveSceneID:        "scene-1",
		PlayerPhaseStatus:    "gm_review",
		OOCResolutionPending: true,
	}

	if got := snapshot.TurnMode(); got != InteractionTurnModeOOCCloseResolution {
		t.Fatalf("TurnMode() = %q, want %q", got, InteractionTurnModeOOCCloseResolution)
	}
}

func TestInteractionStateSnapshotTurnModePrefersOOCOpenBeforeReview(t *testing.T) {
	snapshot := InteractionStateSnapshot{
		ActiveSceneID:     "scene-1",
		PlayerPhaseStatus: "GM_REVIEW",
		OOCOpen:           true,
	}

	if got := snapshot.TurnMode(); got != InteractionTurnModeOOCOpen {
		t.Fatalf("TurnMode() = %q, want %q", got, InteractionTurnModeOOCOpen)
	}
}

func TestInteractionStateSnapshotTurnModeTreatsReviewStatusCaseInsensitively(t *testing.T) {
	snapshot := InteractionStateSnapshot{
		ActiveSceneID:     "scene-1",
		PlayerPhaseStatus: "GM_REVIEW",
	}

	if got := snapshot.TurnMode(); got != InteractionTurnModeReviewResolution {
		t.Fatalf("TurnMode() = %q, want %q", got, InteractionTurnModeReviewResolution)
	}
}

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
	if !strings.Contains(prompt, "Each interaction is an ordered set of beats.") {
		t.Fatalf("prompt missing beat guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "A beat is a coherent GM move or information unit, not a paragraph container.") {
		t.Fatalf("prompt missing beat granularity guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "end that interaction with a prompt beat before opening the first player phase") {
		t.Fatalf("prompt missing bootstrap prompt-beat guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "scene_create activates a new scene by default") {
		t.Fatalf("prompt missing bootstrap default-active guidance: %q", prompt)
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
	if !strings.Contains(prompt, "resolution and consequence beats before any new player-facing prompt beat") {
		t.Fatalf("prompt missing active-scene beat ordering guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "When no mechanic was resolved, keep the interaction in fiction and guidance rather than inventing resolution or consequence beats.") {
		t.Fatalf("prompt missing fiction-only beat guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "Keep related prose in one beat even across multiple paragraphs") {
		t.Fatalf("prompt missing active-scene beat granularity guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "Prompt beats must ask for player-character action or commitment, not NPC dialogue or world-outcome authorship.") {
		t.Fatalf("prompt missing narrator-authority guidance: %q", prompt)
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

func TestBriefPromptRendererDefaultsClosingInstructionWhenBlank(t *testing.T) {
	renderer := NewBriefPromptRenderer(BriefPromptRendererConfig{
		Policy: PromptRenderPolicy{},
	})

	prompt := renderer.Render(SessionBrief{}, PromptInput{})
	if !strings.Contains(prompt, DefaultPromptRenderPolicy().ClosingInstruction) {
		t.Fatalf("prompt missing default closing instruction: %q", prompt)
	}
}

func TestBriefPromptRendererFallbackInteractionContractIncludesBeatModel(t *testing.T) {
	renderer := NewBriefPromptRenderer(BriefPromptRendererConfig{
		Policy: DefaultPromptRenderPolicy(),
	})

	prompt := renderer.Render(SessionBrief{}, PromptInput{})
	if !strings.Contains(prompt, "You author one structured GM interaction at a time.") {
		t.Fatalf("prompt missing structured interaction fallback: %q", prompt)
	}
	if !strings.Contains(prompt, "A beat is a coherent GM move or information unit, not a paragraph container.") {
		t.Fatalf("prompt missing beat granularity fallback guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "The GM authors NPC dialogue and world responses; prompt beats ask only what the player character does, says, chooses, or commits to next.") {
		t.Fatalf("prompt missing narrator-authority fallback guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "Do not split narration and player handoff into separate frame artifacts.") {
		t.Fatalf("prompt missing frame-artifact fallback guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "interaction_open_session_ooc, interaction_post_session_ooc, interaction_mark_ooc_ready_to_resume, interaction_clear_ooc_ready_to_resume, and interaction_session_ooc_resolve") {
		t.Fatalf("prompt missing explicit OOC tool family guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "Once interaction_open_scene_player_phase") {
		t.Fatalf("prompt missing turn-completion guidance: %q", prompt)
	}
}

func TestBriefPromptRendererReviewResolutionModeIncludesBeatGuidance(t *testing.T) {
	renderer := NewBriefPromptRenderer(BriefPromptRendererConfig{
		Policy: DefaultPromptRenderPolicy(),
	})

	prompt := renderer.Render(SessionBrief{
		InteractionState: &InteractionStateSnapshot{
			ActiveSceneID:     "scene-1",
			PlayerPhaseStatus: "gm_review",
		},
	}, PromptInput{})
	if !strings.Contains(prompt, "end that interaction with a prompt beat and open the next player phase in the same call") {
		t.Fatalf("prompt missing review-resolution beat guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "If open_next_player_phase or request_revisions succeeds, return final text") {
		t.Fatalf("prompt missing review-resolution completion guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "keep participant-specific revision reasons in the tool payload") {
		t.Fatalf("prompt missing review-resolution revision guidance: %q", prompt)
	}
}

func TestBriefPromptRendererOOCCloseModeIncludesBeatGuidance(t *testing.T) {
	renderer := NewBriefPromptRenderer(BriefPromptRendererConfig{
		Policy: DefaultPromptRenderPolicy(),
	})

	prompt := renderer.Render(SessionBrief{
		InteractionState: &InteractionStateSnapshot{
			ActiveSceneID:        "scene-1",
			OOCResolutionPending: true,
		},
	}, PromptInput{})
	if !strings.Contains(prompt, "replace it with a newly opened player phase") {
		t.Fatalf("prompt missing OOC resume interaction guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "After the interrupted player phase is resumed or replaced successfully, return final text") {
		t.Fatalf("prompt missing OOC resume completion guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "ends with a prompt beat for the replacement player phase") {
		t.Fatalf("prompt missing OOC resume prompt-beat guidance: %q", prompt)
	}
}

func TestBriefPromptRendererOOCOpenModeIncludesResolutionGuidance(t *testing.T) {
	renderer := NewBriefPromptRenderer(BriefPromptRendererConfig{
		Policy: DefaultPromptRenderPolicy(),
	})

	prompt := renderer.Render(SessionBrief{
		InteractionState: &InteractionStateSnapshot{
			ActiveSceneID: "scene-1",
			OOCOpen:       true,
		},
	}, PromptInput{})
	if !strings.Contains(prompt, "OOC-open mode: the session is paused for out-of-character discussion.") {
		t.Fatalf("prompt missing OOC-open mode guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "use interaction_session_ooc_resolve to close the pause") {
		t.Fatalf("prompt missing OOC-open resolution guidance: %q", prompt)
	}
	if !strings.Contains(prompt, "ends with a prompt beat for the replacement player phase") {
		t.Fatalf("prompt missing OOC-open prompt-beat guidance: %q", prompt)
	}
}

// fullSourceRegistry builds a registry with the core prompt context sources
// for tests that construct a PromptBuilder via NewPromptBuilder. Game-system
// sources (e.g. Daggerheart) are tested in their own subpackages.
func fullSourceRegistry() *ContextSourceRegistry {
	return NewCoreContextSourceRegistry()
}
