package orchestration

import (
	"encoding/json"
	"fmt"
	"strings"

	"context"
)

// PromptBuilderConfig holds the dependencies for building campaign turn prompts.
type PromptBuilderConfig struct {
	// Skills is the composed skills document (core + system).
	// When empty, the builder omits the skills section.
	Skills string

	// InteractionContract is the tool channel discipline text.
	// When empty, a minimal inline fallback is used.
	InteractionContract string

	// MaxTokens is the approximate token budget for the assembled brief.
	// Zero means unlimited.
	MaxTokens int

	// ContextSources is the registry of context sources that contribute
	// BriefSections. The builder collects all sections from the registry
	// and merges them with the intrinsic (config-derived) sections.
	ContextSources *ContextSourceRegistry
}

type defaultPromptBuilder struct {
	skills              string
	interactionContract string
	assembler           BriefAssembler
	contextSources      *ContextSourceRegistry
}

// newDefaultPromptBuilder creates a degraded-mode prompt builder with core
// context sources but no pre-loaded instruction content.
func newDefaultPromptBuilder() PromptBuilder {
	reg := NewContextSourceRegistry()
	for _, src := range CoreContextSources() {
		reg.Register(src)
	}
	return &defaultPromptBuilder{
		contextSources: reg,
	}
}

// NewPromptBuilder creates a prompt builder with explicit instruction content
// and token budget.
func NewPromptBuilder(cfg PromptBuilderConfig) PromptBuilder {
	return &defaultPromptBuilder{
		skills:              cfg.Skills,
		interactionContract: cfg.InteractionContract,
		assembler:           BriefAssembler{MaxTokens: cfg.MaxTokens},
		contextSources:      cfg.ContextSources,
	}
}

func (pb *defaultPromptBuilder) Build(ctx context.Context, sess Session, input Input) (string, error) {
	// Collect data-fetching sections from the registry.
	collected, err := pb.contextSources.CollectSections(ctx, sess, input)
	if err != nil {
		return "", fmt.Errorf("collect context sources: %w", err)
	}

	// Detect bootstrap mode from the interaction_state section.
	bootstrap := detectBootstrap(collected)

	// Build config-derived intrinsic sections.
	intrinsic := pb.buildIntrinsicSections(bootstrap, input)

	// Merge intrinsic + collected and assemble.
	sections := append(intrinsic, collected...)
	return pb.assembler.Assemble(sections), nil
}

// detectBootstrap scans collected sections for the interaction_state section
// and checks whether the active scene ID is empty (bootstrap mode).
func detectBootstrap(sections []BriefSection) bool {
	for _, s := range sections {
		if s.ID != "interaction_state" {
			continue
		}
		var state struct {
			ActiveScene struct {
				SceneID string `json:"scene_id"`
			} `json:"active_scene"`
		}
		if err := json.Unmarshal([]byte(s.Content), &state); err != nil {
			return false
		}
		return strings.TrimSpace(state.ActiveScene.SceneID) == ""
	}
	return false
}

// buildIntrinsicSections constructs the config-derived sections that are not
// resource reads: skills, interaction_contract, authority, turn_input, closing.
func (pb *defaultPromptBuilder) buildIntrinsicSections(bootstrap bool, input Input) []BriefSection {
	var sections []BriefSection

	// Priority 100: Critical
	if text := strings.TrimSpace(pb.skills); text != "" {
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
		Content:  pb.interactionContractText(),
		Required: true,
	})

	sections = append(sections, BriefSection{
		ID:       "authority",
		Priority: 100,
		Label:    "Authority",
		Content:  buildAuthorityText(bootstrap, input),
		Required: true,
	})

	if text := strings.TrimSpace(input.Input); text != "" {
		sections = append(sections, BriefSection{
			ID:       "turn_input",
			Priority: 100,
			Label:    "Turn input",
			Content:  text,
			Required: true,
		})
	}

	// Priority 400: Supplemental — closing instruction
	sections = append(sections, BriefSection{
		ID:       "closing",
		Priority: 400,
		Label:    "",
		Content:  "Return narrated GM output once you have enough information.",
	})

	return sections
}

func (pb *defaultPromptBuilder) interactionContractText() string {
	if text := strings.TrimSpace(pb.interactionContract); text != "" {
		return text
	}
	// Inline fallback when no instruction file is loaded.
	return strings.Join([]string{
		"You are the AI GM for this campaign turn. You manage narration and authoritative game-state changes together.",
		"Keep in-character narration and out-of-character coordination separate.",
		"Use interaction_scene_gm_output_commit for authoritative in-character narration.",
		"Use interaction_ooc_* tools for out-of-character rules guidance, coordination, pauses, and resumptions.",
		"Use system_reference_search and system_reference_read before improvising Daggerheart rules or mechanics.",
		"Use tools for authoritative state changes; do not rely on free-form narration to mutate game state.",
	}, "\n")
}

func buildAuthorityText(bootstrap bool, input Input) string {
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

func readOptionalResource(ctx context.Context, sess Session, uri string) (string, error) {
	value, err := sess.ReadResource(ctx, uri)
	if err != nil {
		errText := strings.ToLower(err.Error())
		if strings.Contains(errText, "not found") || strings.Contains(errText, "missing resource") {
			return "", nil
		}
		return "", err
	}
	return value, nil
}
