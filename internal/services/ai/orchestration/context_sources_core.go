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
	return sectionContribution(BriefSection{
		ID:       "current_context",
		Priority: 200,
		Label:    "Current context",
		Content:  current,
	}), nil
}

func campaignContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	campaignID := strings.TrimSpace(input.CampaignID)
	campaign, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s", campaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read campaign: %w", err)
	}
	return sectionContribution(BriefSection{
		ID:       "campaign",
		Priority: 200,
		Label:    "Campaign",
		Content:  campaign,
	}), nil
}

func participantsContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	campaignID := strings.TrimSpace(input.CampaignID)
	participants, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/participants", campaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read participants: %w", err)
	}
	return sectionContribution(BriefSection{
		ID:       "participants",
		Priority: 300,
		Label:    "Participants",
		Content:  participants,
	}), nil
}

func charactersContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	campaignID := strings.TrimSpace(input.CampaignID)
	characters, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/characters", campaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read characters: %w", err)
	}
	return sectionContribution(BriefSection{
		ID:       "characters",
		Priority: 300,
		Label:    "Characters",
		Content:  characters,
	}), nil
}

func sessionsContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	campaignID := strings.TrimSpace(input.CampaignID)
	sessions, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/sessions", campaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read sessions: %w", err)
	}
	return sectionContribution(BriefSection{
		ID:       "sessions",
		Priority: 300,
		Label:    "Sessions",
		Content:  sessions,
	}), nil
}

func scenesContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	campaignID := strings.TrimSpace(input.CampaignID)
	sessionID := strings.TrimSpace(input.SessionID)
	scenes, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/sessions/%s/scenes", campaignID, sessionID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read scenes: %w", err)
	}
	return sectionContribution(BriefSection{
		ID:       "scenes",
		Priority: 300,
		Label:    "Scenes",
		Content:  scenes,
	}), nil
}

func storyContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	campaignID := strings.TrimSpace(input.CampaignID)
	story, err := readOptionalResource(ctx, sess, fmt.Sprintf("campaign://%s/artifacts/story.md", campaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read story artifact: %w", err)
	}
	if text := strings.TrimSpace(story); text != "" {
		return sectionContribution(BriefSection{
			ID:       "story",
			Priority: 300,
			Label:    "story.md",
			Content:  text,
		}), nil
	}
	return BriefContribution{}, nil
}

func memoryContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	campaignID := strings.TrimSpace(input.CampaignID)
	memory, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/artifacts/memory.md", campaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read memory artifact: %w", err)
	}
	if text := strings.TrimSpace(memory); text != "" {
		return sectionContribution(BriefSection{
			ID:       "memory",
			Priority: 400,
			Label:    "memory.md",
			Content:  text,
		}), nil
	}
	return BriefContribution{}, nil
}

func interactionStateContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	campaignID := strings.TrimSpace(input.CampaignID)
	interaction, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/interaction", campaignID))
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

func sectionContribution(section BriefSection) BriefContribution {
	return BriefContribution{Sections: []BriefSection{section}}
}

func decodeInteractionStateSnapshot(raw string) (InteractionStateSnapshot, error) {
	var state struct {
		ActiveScene struct {
			SceneID string `json:"scene_id"`
		} `json:"active_scene"`
	}
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return InteractionStateSnapshot{}, err
	}
	return InteractionStateSnapshot{ActiveSceneID: state.ActiveScene.SceneID}, nil
}
