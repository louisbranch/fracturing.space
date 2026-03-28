package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// CoreContextSourceConfig controls which optional core prompt sections are
// always-on during campaign-turn prompt assembly.
type CoreContextSourceConfig struct {
	IncludeStory      bool
	IncludeStoryIndex bool
	IncludeMemory     bool
}

// CoreContextSources returns the standard context sources for campaign turns.
// These read authoritative game state from the session and produce brief
// sections at the appropriate priority tiers.
func CoreContextSources() []ContextSource {
	return CoreContextSourcesWithConfig(CoreContextSourceConfig{
		IncludeStory:      true,
		IncludeStoryIndex: false,
		IncludeMemory:     true,
	})
}

// CoreContextSourcesWithConfig returns the standard context sources for
// campaign turns with explicit control over optional artifact sections.
func CoreContextSourcesWithConfig(cfg CoreContextSourceConfig) []ContextSource {
	sources := []ContextSource{
		ContextSourceFunc(campaignContextSource),
		ContextSourceFunc(participantsContextSource),
		ContextSourceFunc(charactersContextSource),
		ContextSourceFunc(latestSessionRecapContextSource),
		ContextSourceFunc(sessionsContextSource),
		ContextSourceFunc(scenesContextSource),
		ContextSourceFunc(currentContextSource),
		ContextSourceFunc(interactionStateContextSource),
	}
	if cfg.IncludeStoryIndex {
		sources = append(sources, ContextSourceFunc(storyIndexContextSource))
	}
	if cfg.IncludeStory {
		sources = append(sources, ContextSourceFunc(storyContextSource))
	}
	if cfg.IncludeMemory {
		sources = append(sources, ContextSourceFunc(memoryContextSource))
	}
	return sources
}

// NewCoreContextSourceRegistry builds the always-on collector used by prompt
// builders before any game-system-specific sources are appended.
func NewCoreContextSourceRegistry() *ContextSourceRegistry {
	return NewCoreContextSourceRegistryWithConfig(CoreContextSourceConfig{
		IncludeStory:      true,
		IncludeStoryIndex: false,
		IncludeMemory:     true,
	})
}

// NewCoreContextSourceRegistryWithConfig builds the always-on collector used by
// prompt builders before any game-system-specific sources are appended.
func NewCoreContextSourceRegistryWithConfig(cfg CoreContextSourceConfig) *ContextSourceRegistry {
	reg := NewContextSourceRegistry()
	reg.RegisterAll(CoreContextSourcesWithConfig(cfg)...)
	return reg
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

func latestSessionRecapContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	sessionsRaw, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/sessions", input.CampaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read sessions for recap: %w", err)
	}
	var payload struct {
		Sessions []struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			EndedAt string `json:"ended_at"`
		} `json:"sessions"`
	}
	if err := json.Unmarshal([]byte(sessionsRaw), &payload); err != nil {
		return BriefContribution{}, fmt.Errorf("decode sessions for recap: %w", err)
	}
	var latest struct {
		ID      string
		EndedAt string
	}
	for _, session := range payload.Sessions {
		if !strings.EqualFold(strings.TrimSpace(session.Status), "ENDED") || strings.TrimSpace(session.EndedAt) == "" {
			continue
		}
		if latest.ID == "" || strings.TrimSpace(session.EndedAt) > latest.EndedAt {
			latest.ID = strings.TrimSpace(session.ID)
			latest.EndedAt = strings.TrimSpace(session.EndedAt)
		}
	}
	if latest.ID == "" {
		return BriefContribution{}, nil
	}
	recap, err := readOptionalResource(ctx, sess, fmt.Sprintf("campaign://%s/sessions/%s/recap", input.CampaignID, latest.ID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read latest session recap: %w", err)
	}
	recap = strings.TrimSpace(recap)
	if recap == "" {
		return BriefContribution{}, nil
	}
	return SectionContribution(BriefSection{
		ID:       "latest_session_recap",
		Priority: 300,
		Label:    "Session recap",
		Content:  recap,
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

func storyIndexContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	story, err := readOptionalResource(ctx, sess, fmt.Sprintf("campaign://%s/artifacts/story.md", input.CampaignID))
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read story artifact for index: %w", err)
	}
	index := strings.TrimSpace(BuildStoryContextIndex(input.CampaignID, story))
	if index == "" {
		return BriefContribution{}, nil
	}
	return SectionContribution(BriefSection{
		ID:       "story_index",
		Priority: 280,
		Label:    "Story index",
		Content:  index,
	}), nil
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
	sections := []BriefSection{{
		ID:       "interaction_state",
		Priority: 200,
		Label:    "Current interaction state",
		Content:  interaction,
	}}
	if guide := strings.TrimSpace(BuildPhaseGuide(snapshot.TurnMode(), input)); guide != "" {
		sections = append(sections, BriefSection{
			ID:       "phase_guide",
			Priority: 150,
			Label:    "Current phase guide",
			Content:  guide,
		})
	}
	if accessMap := strings.TrimSpace(BuildContextAccessMap(snapshot.TurnMode(), input)); accessMap != "" {
		sections = append(sections, BriefSection{
			ID:       "context_access_map",
			Priority: 210,
			Label:    "Context access map",
			Content:  accessMap,
		})
	}
	return BriefContribution{
		Sections:         sections,
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
