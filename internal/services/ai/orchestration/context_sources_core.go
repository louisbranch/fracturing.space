package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// CoreContextSources returns the standard context sources for campaign turns.
// These read authoritative game state from the session and produce brief
// sections at the appropriate priority tiers.
func CoreContextSources() []ContextSource {
	return []ContextSource{
		ContextSourceFunc(campaignContextSource),
		ContextSourceFunc(participantsContextSource),
		ContextSourceFunc(charactersContextSource),
		ContextSourceFunc(sessionsContextSource),
		ContextSourceFunc(scenesContextSource),
		ContextSourceFunc(storyContextSource),
		ContextSourceFunc(memoryContextSource),
		ContextSourceFunc(currentContextSource),
		ContextSourceFunc(interactionStateContextSource),
	}
}

func currentContextSource(ctx context.Context, sess Session, _ PromptInput) (BriefContribution, error) {
	current, err := sess.ReadResource(ctx, "context://current")
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read mcp context: %w", err)
	}
	return SectionContribution(BriefSection{
		ID:       "current_context",
		Priority: 200,
		Label:    "Current context",
		Content:  current,
	}), nil
}

func campaignContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	campaign, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s", input.CampaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read campaign: %w", err)
	}
	return SectionContribution(BriefSection{
		ID:       "campaign",
		Priority: 200,
		Label:    "Campaign",
		Content:  campaign,
	}), nil
}

func participantsContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	participants, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/participants", input.CampaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read participants: %w", err)
	}
	return SectionContribution(BriefSection{
		ID:       "participants",
		Priority: 300,
		Label:    "Participants",
		Content:  participants,
	}), nil
}

func charactersContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	characters, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/characters", input.CampaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read characters: %w", err)
	}
	return SectionContribution(BriefSection{
		ID:       "characters",
		Priority: 300,
		Label:    "Characters",
		Content:  characters,
	}), nil
}

func sessionsContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	sessions, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/sessions", input.CampaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read sessions: %w", err)
	}
	return SectionContribution(BriefSection{
		ID:       "sessions",
		Priority: 300,
		Label:    "Sessions",
		Content:  sessions,
	}), nil
}

func scenesContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	scenes, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/sessions/%s/scenes", input.CampaignID, input.SessionID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read scenes: %w", err)
	}
	return SectionContribution(BriefSection{
		ID:       "scenes",
		Priority: 300,
		Label:    "Scenes",
		Content:  scenes,
	}), nil
}

func storyContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	story, err := readOptionalResource(ctx, sess, fmt.Sprintf("campaign://%s/artifacts/story.md", input.CampaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read story artifact: %w", err)
	}
	if strings.TrimSpace(story) != "" {
		return SectionContribution(BriefSection{
			ID:       "story",
			Priority: 300,
			Label:    "story.md",
			Content:  story,
		}), nil
	}
	return BriefContribution{}, nil
}

func memoryContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	memory, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/artifacts/memory.md", input.CampaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read memory artifact: %w", err)
	}
	if strings.TrimSpace(memory) != "" {
		return SectionContribution(BriefSection{
			ID:       "memory",
			Priority: 400,
			Label:    "memory.md",
			Content:  memory,
		}), nil
	}
	return BriefContribution{}, nil
}

func interactionStateContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	interaction, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/interaction", input.CampaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read interaction state: %w", err)
	}
	snapshot, err := decodeInteractionStateSnapshot(interaction)
	if err != nil {
		return BriefContribution{}, fmt.Errorf("decode interaction state: %w", err)
	}
	return BriefContribution{
		Sections: []BriefSection{{
			ID:       "interaction_state",
			Priority: 200,
			Label:    "Current interaction state",
			Content:  interaction,
		}},
		InteractionState: &snapshot,
	}, nil
}

// SectionContribution wraps a single BriefSection as a BriefContribution.
func SectionContribution(section BriefSection) BriefContribution {
	return BriefContribution{Sections: []BriefSection{section}}
}

func decodeInteractionStateSnapshot(raw string) (InteractionStateSnapshot, error) {
	var state struct {
		ActiveScene struct {
			SceneID string `json:"scene_id"`
		} `json:"active_scene"`
		PlayerPhase struct {
			Status string `json:"status"`
		} `json:"player_phase"`
		OOC struct {
			Open              bool `json:"open"`
			ResolutionPending bool `json:"resolution_pending"`
		} `json:"ooc"`
	}
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return InteractionStateSnapshot{}, err
	}
	return InteractionStateSnapshot{
		ActiveSceneID:        state.ActiveScene.SceneID,
		PlayerPhaseStatus:    state.PlayerPhase.Status,
		OOCOpen:              state.OOC.Open,
		OOCResolutionPending: state.OOC.ResolutionPending,
	}, nil
}
