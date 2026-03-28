package orchestration

import (
	"fmt"
	"strings"
)

const (
	phaseResourceSceneBootstrap = "scene-bootstrap"
	phaseResourceScenePlay      = "scene-play"
	phaseResourceActionReview   = "action-review"
)

// PhaseResourceName returns the deterministic phase resource name used by the
// prompt builder and OpenViking-generated context files.
func PhaseResourceName(mode InteractionTurnMode) string {
	switch mode {
	case InteractionTurnModeBootstrap:
		return phaseResourceSceneBootstrap
	case InteractionTurnModeReviewResolution:
		return phaseResourceActionReview
	default:
		return phaseResourceScenePlay
	}
}

// BuildPhaseGuide describes what the GM agent needs for the current turn
// phase before it consults deeper context.
func BuildPhaseGuide(mode InteractionTurnMode, input PromptInput) string {
	var b strings.Builder
	b.WriteString("Current phase: ")
	b.WriteString(PhaseResourceName(mode))
	switch mode {
	case InteractionTurnModeBootstrap:
		b.WriteString("\nGoal: create or choose the next scene and open the first player-facing interaction.")
		b.WriteString("\nKnow first: the GM/narrator role contract, the current session handoff point, the campaign plan, and the characters who are about to be involved.")
		b.WriteString("\nPrefer summaries and indexes before reading full source artifacts.")
	case InteractionTurnModeReviewResolution:
		b.WriteString("\nGoal: review the submitted player action, adjudicate it, and resolve the scene back into play.")
		b.WriteString("\nKnow first: the submitted intent, active scene state, relevant character state, and the mechanics path that actually governs the move.")
		b.WriteString("\nUse exact rules and sheet details only when the quick context is not enough to adjudicate safely.")
	default:
		b.WriteString("\nGoal: continue the active scene and hand play back with one coherent GM interaction.")
		b.WriteString("\nKnow first: the active scene frame, who is present, the immediate tension, and any capability-sensitive facts that constrain the response.")
		b.WriteString("\nUse deeper story or rules context only when the current scene cannot be advanced safely without it.")
	}
	if participantID := strings.TrimSpace(input.ParticipantID); participantID != "" {
		b.WriteString("\nFixed GM participant: ")
		b.WriteString(participantID)
	}
	return b.String()
}

// BuildContextAccessMap tells the agent which resources exist and when each is
// worth consulting for the current phase.
func BuildContextAccessMap(mode InteractionTurnMode, input PromptInput) string {
	campaignID := strings.TrimSpace(input.CampaignID)
	sessionID := strings.TrimSpace(input.SessionID)
	if campaignID == "" {
		return ""
	}

	lines := []string{
		"Consult these resources only when the current phase needs them:",
		fmt.Sprintf("- campaign://%s/interaction -> current phase, active scene, and player-review status", campaignID),
		fmt.Sprintf("- campaign://%s/sessions/%s/scenes -> current session scene list and continuity", campaignID, sessionID),
		fmt.Sprintf("- campaign://%s/characters -> campaign cast index and identities", campaignID),
		fmt.Sprintf("- campaign://%s/participants -> table participants and GM/player authority", campaignID),
		fmt.Sprintf("- campaign://%s/artifacts/story.md -> full authored campaign source; read only after the story index is insufficient", campaignID),
		fmt.Sprintf("- campaign://%s/artifacts/memory.md -> campaign reminders, unresolved threads, and recurring notes", campaignID),
	}

	switch mode {
	case InteractionTurnModeReviewResolution:
		lines = append(lines,
			"- system_reference_search/read -> exact Daggerheart wording only when the mechanics path is still unclear",
			"- read the acting character sheet before adjudicating capability-sensitive submissions",
		)
	case InteractionTurnModeBootstrap:
		lines = append(lines,
			fmt.Sprintf("- campaign://%s -> campaign theme and high-level metadata for scene framing", campaignID),
			fmt.Sprintf("- campaign://%s/sessions -> prior session continuity and current session status", campaignID),
		)
	default:
		lines = append(lines,
			fmt.Sprintf("- campaign://%s -> campaign theme and current framing constraints", campaignID),
			"- read the acting character sheet before accepting capability- or gear-sensitive declarations",
		)
	}

	return strings.Join(lines, "\n")
}

// BuildStoryContextIndex derives one compact story index from the full story
// artifact so prompt assembly can expose the plan surface without injecting the
// full authored document.
func BuildStoryContextIndex(campaignID string, story string) string {
	story = strings.TrimSpace(story)
	if story == "" {
		return ""
	}

	headings := extractStoryHeadings(story, 8)
	snippets := extractStorySnippets(story, 3)

	var b strings.Builder
	b.WriteString("Use this story index before reading full story.md.")
	if campaignID = strings.TrimSpace(campaignID); campaignID != "" {
		b.WriteString("\nDeep source: campaign://")
		b.WriteString(campaignID)
		b.WriteString("/artifacts/story.md")
	}
	if len(headings) > 0 {
		b.WriteString("\nHeadings:")
		for _, heading := range headings {
			b.WriteString("\n- ")
			b.WriteString(heading)
		}
	}
	if len(snippets) > 0 {
		b.WriteString("\nKey lines:")
		for _, snippet := range snippets {
			b.WriteString("\n- ")
			b.WriteString(snippet)
		}
	}
	return b.String()
}

func extractStoryHeadings(story string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	headings := make([]string, 0, limit)
	seen := map[string]struct{}{}
	for _, line := range strings.Split(story, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimLeft(line, "#"))
		if line == "" {
			continue
		}
		if _, ok := seen[line]; ok {
			continue
		}
		seen[line] = struct{}{}
		headings = append(headings, line)
		if len(headings) >= limit {
			break
		}
	}
	return headings
}

func extractStorySnippets(story string, limit int) []string {
	if limit <= 0 {
		return nil
	}
	lines := make([]string, 0, limit)
	for _, line := range strings.Split(story, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = normalizeStorySnippet(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
		if len(lines) >= limit {
			break
		}
	}
	return lines
}

func normalizeStorySnippet(line string) string {
	const maxLen = 180
	line = strings.Join(strings.Fields(line), " ")
	if len(line) <= maxLen {
		return line
	}
	return strings.TrimSpace(line[:maxLen-3]) + "..."
}
