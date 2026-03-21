package orchestration

import (
	"context"
	"strings"
)

// SessionBriefCollector gathers the typed session brief used for one prompt.
type SessionBriefCollector interface {
	CollectBrief(ctx context.Context, sess Session, input PromptInput) (SessionBrief, error)
}

// PromptInstructions contains the static instruction content injected into the
// rendered campaign turn prompt.
type PromptInstructions struct {
	Skills              string
	InteractionContract string
}

// PromptRenderer renders one prompt from a collected brief plus prompt input.
type PromptRenderer interface {
	Render(brief SessionBrief, input PromptInput) string
}

// PromptRenderPolicy declares the configurable rendering behavior for one
// prompt assembly pass.
type PromptRenderPolicy struct {
	Instructions       PromptInstructions
	ClosingInstruction string
	MaxTokens          int
}

// BriefPromptRendererConfig declares the rendering policy for prompt assembly.
type BriefPromptRendererConfig struct {
	Policy PromptRenderPolicy
}

type briefPromptRenderer struct {
	policy    PromptRenderPolicy
	assembler BriefAssembler
}

// DefaultPromptRenderPolicy returns the canonical rendering policy for
// campaign-turn prompts before any instruction files are applied.
func DefaultPromptRenderPolicy() PromptRenderPolicy {
	return PromptRenderPolicy{
		ClosingInstruction: "Return narrated GM output once you have enough information.",
	}
}

// NewBriefPromptRenderer builds the default prompt renderer over one
// instruction set and one budget policy.
func NewBriefPromptRenderer(cfg BriefPromptRendererConfig) PromptRenderer {
	policy := cfg.Policy
	if strings.TrimSpace(policy.ClosingInstruction) == "" {
		policy.ClosingInstruction = DefaultPromptRenderPolicy().ClosingInstruction
	}
	return briefPromptRenderer{
		policy:    policy,
		assembler: BriefAssembler{MaxTokens: policy.MaxTokens},
	}
}

func (r briefPromptRenderer) Render(brief SessionBrief, input PromptInput) string {
	sections := append(r.buildIntrinsicSections(brief, input), brief.Sections...)
	return r.assembler.Assemble(sections)
}

// buildIntrinsicSections constructs the config-derived sections that are not
// resource reads: skills, interaction_contract, authority, turn_input, closing.
func (r briefPromptRenderer) buildIntrinsicSections(brief SessionBrief, input PromptInput) []BriefSection {
	var sections []BriefSection

	if text := strings.TrimSpace(r.policy.Instructions.Skills); text != "" {
		sections = append(sections, BriefSection{
			ID:       "skills",
			Priority: 100,
			Label:    "Skills",
			Content:  text,
			Required: true,
		})
	}

	sections = append(sections, BriefSection{
		ID:       "interaction_contract",
		Priority: 100,
		Label:    "Interaction contract",
		Content:  interactionContractText(r.policy.Instructions),
		Required: true,
	})

	sections = append(sections, BriefSection{
		ID:       "authority",
		Priority: 100,
		Label:    "Authority",
		Content:  buildAuthorityText(brief.Bootstrap(), input),
		Required: true,
	})

	if text := strings.TrimSpace(input.TurnInput); text != "" {
		sections = append(sections, BriefSection{
			ID:       "turn_input",
			Priority: 100,
			Label:    "Turn input",
			Content:  text,
			Required: true,
		})
	}

	sections = append(sections, BriefSection{
		ID:       "closing",
		Priority: 400,
		Label:    "",
		Content:  r.policy.ClosingInstruction,
	})

	return sections
}

func interactionContractText(instructions PromptInstructions) string {
	if text := strings.TrimSpace(instructions.InteractionContract); text != "" {
		return text
	}
	return strings.Join([]string{
		"You are the AI GM for this campaign turn. You manage narration and authoritative game-state changes together.",
		"Keep in-character narration and out-of-character coordination separate.",
		"Use interaction_scene_gm_output_commit for authoritative in-character narration.",
		"Use interaction_ooc_* tools for out-of-character rules guidance, coordination, pauses, and resumptions.",
		"Use system_reference_search and system_reference_read before improvising Daggerheart rules or mechanics.",
		"Use tools for authoritative state changes; do not rely on free-form narration to mutate game state.",
	}, "\n")
}

func buildAuthorityText(bootstrap bool, input PromptInput) string {
	var b strings.Builder
	b.WriteString("Campaign, session, and participant authority are fixed for this turn.")
	if pid := strings.TrimSpace(input.ParticipantID); pid != "" {
		b.WriteString("\nFixed participant authority:\n")
		b.WriteString(pid)
	}
	if bootstrap {
		b.WriteString("\n\nBootstrap mode: there is no active scene yet.\n")
		b.WriteString("You are responsible for creating or choosing the opening scene from campaign, participant, and character context, setting it active, and committing authoritative GM output.\n")
		b.WriteString("If there are no suitable scenes yet, create one that fits the campaign theme and the player characters.\n")
		b.WriteString("After the scene is active and narrated, start the first player phase when the acting characters are clear.")
	} else {
		b.WriteString("\n\nActive scene mode: continue the session from the current interaction state and use tools for authoritative changes.")
	}
	return b.String()
}
