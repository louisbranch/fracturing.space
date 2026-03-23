package orchestration

import (
	"strings"
)

// PromptInstructions contains the static instruction content injected into the
// rendered campaign turn prompt.
type PromptInstructions struct {
	Skills              string
	InteractionContract string
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

	if r.policy.Instructions.Skills != "" {
		sections = append(sections, BriefSection{
			ID:       "skills",
			Priority: 100,
			Label:    "Skills",
			Content:  r.policy.Instructions.Skills,
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
		Content:  buildAuthorityText(brief.TurnMode(), input),
		Required: true,
	})

	if input.TurnInput != "" {
		sections = append(sections, BriefSection{
			ID:       "turn_input",
			Priority: 100,
			Label:    "Turn input",
			Content:  input.TurnInput,
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
	if instructions.InteractionContract != "" {
		return instructions.InteractionContract
	}
	return strings.Join([]string{
		"You are the AI GM for this campaign turn. You manage narration and authoritative game-state changes together.",
		"You author one structured GM interaction at a time.",
		"Each interaction is an ordered set of beats.",
		"A beat is a coherent GM move or information unit, not a paragraph container.",
		"Keep related prose in one beat even when it spans multiple paragraphs.",
		"Start a new beat only when the GM function changes or the information context materially shifts; repeated beat types are for distinct units, not extra paragraphs.",
		"Keep in-character narration and out-of-character coordination separate.",
		"Before narrating a claimed capability or permissive fiction, verify that it fits the established scene and the acting character's real capabilities; if it does not, clarify or move to OOC instead of narrating a false permission.",
		"The GM authors NPC dialogue and world responses; prompt beats ask only what the player character does, says, chooses, or commits to next.",
		"Use fiction beats to establish the situation, resolution beats only after real adjudication, consequence beats to return adjudicated results to the fiction, guidance beats to clarify what is actionable next, and prompt beats as the player-facing handoff when players should act next.",
		"Do not split narration and player handoff into separate frame artifacts.",
		"Use interaction_record_scene_gm_interaction for standalone in-character narration when framing a fresh beat outside GM review.",
		"Use interaction_resolve_scene_player_review when the scene is waiting on GM review.",
		"Use interaction_session_ooc_resolve when OOC has resumed but players are still blocked pending interaction resolution.",
		"Once interaction_open_scene_player_phase, interaction_resolve_scene_player_review opening the next player phase, or interaction_session_ooc_resolve replacing or resuming a player phase succeeds, the GM turn is complete unless OOC or GM review still needs resolution.",
		"Use interaction_open_session_ooc, interaction_post_session_ooc, interaction_mark_ooc_ready_to_resume, interaction_clear_ooc_ready_to_resume, and interaction_session_ooc_resolve for out-of-character rules guidance, coordination, pauses, and resumptions.",
		"Use system_reference_search and system_reference_read only when exact Daggerheart wording or procedure choice is unclear.",
		"Use tools for authoritative state changes; do not rely on free-form narration to mutate game state.",
	}, "\n")
}

func buildAuthorityText(mode InteractionTurnMode, input PromptInput) string {
	var b strings.Builder
	b.WriteString("Campaign, session, and participant authority are fixed for this turn.")
	if input.ParticipantID != "" {
		b.WriteString("\nFixed participant authority:\n")
		b.WriteString(input.ParticipantID)
	}
	switch mode {
	case InteractionTurnModeBootstrap:
		b.WriteString("\n\nBootstrap mode: there is no active scene yet.\n")
		b.WriteString("You are responsible for creating or choosing the opening scene from campaign, participant, and character context and committing authoritative GM output.\n")
		b.WriteString("scene_create activates a new scene by default; use interaction_activate_scene only when switching to an existing scene.\n")
		b.WriteString("If there are no suitable scenes yet, create one that fits the campaign theme and the player characters.\n")
		b.WriteString("After the opening scene is active, commit one opening interaction built from ordered beats. Keep related setup inside one beat unless the interaction job or information context materially changes. Start with fiction, and when players should act next, end that interaction with a prompt beat before opening the first player phase.\n")
		b.WriteString("After interaction_open_scene_player_phase succeeds, return final text instead of making another GM interaction call.\n")
		b.WriteString("Pause for OOC instead only if table coordination is required.")
	case InteractionTurnModeReviewResolution:
		b.WriteString("\n\nReview-resolution mode: players have yielded and the scene is waiting on GM review.\n")
		b.WriteString("Use interaction_resolve_scene_player_review to commit one interaction that reflects the adjudicated outcome.\n")
		b.WriteString("If the submitted move is character-specific or capability-sensitive, read the acting character sheet before you adjudicate.\n")
		b.WriteString("When the submission already declares a Hope spend, named experience, or clear weapon-driven attempt to subdue, incapacitate, or otherwise force the issue, treat that as enough commitment to adjudicate rather than merely accepting it in fiction.\n")
		b.WriteString("When a consequential move needs adjudication, use the authoritative state-mutating mechanics tool rather than a preview roll or explanation-only tool.\n")
		b.WriteString("Adjudicate the submitted player action when it already contains enough intent; do not bounce a clear move back just to ask for a trait choice or restate a mechanic you can assign yourself.\n")
		b.WriteString("Do not research before the sheet and the obvious mechanics path when the move is already recognizable.\n")
		b.WriteString("If players should act next, end that interaction with a prompt beat and open the next player phase in the same call.\n")
		b.WriteString("If open_next_player_phase or request_revisions succeeds, return final text instead of making another GM interaction call.\n")
		b.WriteString("If you are sending slots back for revision, use guidance beats for what must change and keep participant-specific revision reasons in the tool payload.\n")
		b.WriteString("Do not leave the interaction in silent GM control.")
	case InteractionTurnModeOOCOpen:
		b.WriteString("\n\nOOC-open mode: the session is paused for out-of-character discussion.\n")
		b.WriteString("Use interaction_open_session_ooc, interaction_post_session_ooc, interaction_mark_ooc_ready_to_resume, and interaction_clear_ooc_ready_to_resume while the table is still coordinating.\n")
		b.WriteString("When the table is ready to continue, use interaction_session_ooc_resolve to close the pause and either resume the interrupted phase, return to GM control, or replace it with a newly opened player phase.\n")
		b.WriteString("If you replace the interrupted phase, commit one interaction that re-anchors the fiction and ends with a prompt beat for the replacement player phase.\n")
		b.WriteString("After interaction_session_ooc_resolve succeeds, return final text instead of making another GM interaction call.\n")
	case InteractionTurnModeOOCCloseResolution:
		b.WriteString("\n\nPost-OOC resolution mode: out-of-character discussion has resumed, but players are still blocked until you resolve the interrupted interaction.\n")
		b.WriteString("Use interaction_session_ooc_resolve to resume the interrupted phase, return to GM control, or replace it with a newly opened player phase.\n")
		b.WriteString("If you replace the interrupted phase, commit one interaction that re-anchors the fiction and ends with a prompt beat for the replacement player phase.\n")
		b.WriteString("After the interrupted player phase is resumed or replaced successfully, return final text instead of making another GM interaction call.\n")
		b.WriteString("If the interruption landed during GM review, interaction_resolve_scene_player_review is also valid.")
	default:
		b.WriteString("\n\nActive scene mode: continue the session from the current interaction state and use tools for authoritative changes.")
		b.WriteString("\nWhen mechanics were resolved this turn, place resolution and consequence beats before any new player-facing prompt beat.")
		b.WriteString("\nWhen no mechanic was resolved, keep the interaction in fiction and guidance rather than inventing resolution or consequence beats.")
		b.WriteString("\nKeep related prose in one beat even across multiple paragraphs; split into another beat only when the interaction function or information context materially changes.")
		b.WriteString("\nCheck character capability before accepting equipment- or feature-sensitive declarations; stance or gear alone may justify a sheet read even before a roll.")
		b.WriteString("\nPrompt beats must ask for player-character action or commitment, not NPC dialogue or world-outcome authorship.")
		b.WriteString("\nWhen handing control back to players, commit one interaction built from ordered beats first and end it with a prompt beat, then call interaction_open_scene_player_phase with explicit acting character_ids.")
		b.WriteString("\nAfter the next player phase is open for players, return final text instead of making another GM interaction call.")
		b.WriteString("\nDo not author separate frame text for the player handoff.")
	}
	return b.String()
}
