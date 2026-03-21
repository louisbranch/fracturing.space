package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type defaultPromptBuilder struct{}

type sessionBrief struct {
	skills       string
	story        string
	memory       string
	current      string
	campaign     string
	participants string
	characters   string
	sessions     string
	scenes       string
	interaction  string
	bootstrap    bool
}

func newDefaultPromptBuilder() PromptBuilder {
	return defaultPromptBuilder{}
}

func (defaultPromptBuilder) Build(ctx context.Context, sess Session, input Input) (string, error) {
	brief, err := buildBrief(ctx, sess, input)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("Skills:\n")
	b.WriteString(brief.skills)
	b.WriteString("\n\nInteraction contract:\n")
	b.WriteString("You are the AI GM for this campaign turn. You manage narration and authoritative game-state changes together.\n")
	b.WriteString("Keep in-character narration and out-of-character coordination separate.\n")
	b.WriteString("Use interaction_scene_gm_output_commit for authoritative in-character narration.\n")
	b.WriteString("Use interaction_ooc_* tools for out-of-character rules guidance, coordination, pauses, and resumptions.\n")
	b.WriteString("Use system_reference_search and system_reference_read before improvising Daggerheart rules or mechanics.\n")
	b.WriteString("Use tools for authoritative state changes; do not rely on free-form narration to mutate game state.\n")
	b.WriteString("\nAuthority:\n")
	b.WriteString("Campaign, session, and participant authority are fixed for this turn.\n")
	if pid := strings.TrimSpace(input.ParticipantID); pid != "" {
		b.WriteString("Fixed participant authority:\n")
		b.WriteString(pid)
		b.WriteString("\n")
	}
	if brief.bootstrap {
		b.WriteString("\nBootstrap mode: there is no active scene yet.\n")
		b.WriteString("You are responsible for creating or choosing the opening scene from campaign, participant, and character context, setting it active, and committing authoritative GM output.\n")
		b.WriteString("If there are no suitable scenes yet, create one that fits the campaign theme and the player characters.\n")
		b.WriteString("After the scene is active and narrated, start the first player phase when the acting characters are clear.\n\n")
	} else {
		b.WriteString("\nActive scene mode: continue the session from the current interaction state and use tools for authoritative changes.\n\n")
	}
	b.WriteString("Current context:\n")
	b.WriteString(brief.current)
	b.WriteString("\n\nCampaign:\n")
	b.WriteString(brief.campaign)
	b.WriteString("\n\nParticipants:\n")
	b.WriteString(brief.participants)
	b.WriteString("\n\nCharacters:\n")
	b.WriteString(brief.characters)
	b.WriteString("\n\nSessions:\n")
	b.WriteString(brief.sessions)
	b.WriteString("\n\nScenes:\n")
	b.WriteString(brief.scenes)
	b.WriteString("\n\nCurrent interaction state:\n")
	b.WriteString(brief.interaction)
	if text := strings.TrimSpace(brief.story); text != "" {
		b.WriteString("\n\nstory.md:\n")
		b.WriteString(text)
	}
	if text := strings.TrimSpace(brief.memory); text != "" {
		b.WriteString("\n\nmemory.md:\n")
		b.WriteString(text)
	}
	if text := strings.TrimSpace(input.Input); text != "" {
		b.WriteString("\n\nTurn input:\n")
		b.WriteString(text)
	}
	b.WriteString("\n\nReturn narrated GM output once you have enough information.")
	return b.String(), nil
}

func buildBrief(ctx context.Context, sess Session, input Input) (sessionBrief, error) {
	campaignID := strings.TrimSpace(input.CampaignID)
	sessionID := strings.TrimSpace(input.SessionID)
	skills, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/artifacts/skills.md", campaignID))
	if err != nil {
		return sessionBrief{}, fmt.Errorf("read skills artifact: %w", err)
	}
	current, err := sess.ReadResource(ctx, "context://current")
	if err != nil {
		return sessionBrief{}, fmt.Errorf("read mcp context: %w", err)
	}
	campaign, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s", campaignID))
	if err != nil {
		return sessionBrief{}, fmt.Errorf("read campaign: %w", err)
	}
	participants, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/participants", campaignID))
	if err != nil {
		return sessionBrief{}, fmt.Errorf("read participants: %w", err)
	}
	characters, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/characters", campaignID))
	if err != nil {
		return sessionBrief{}, fmt.Errorf("read characters: %w", err)
	}
	sessions, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/sessions", campaignID))
	if err != nil {
		return sessionBrief{}, fmt.Errorf("read sessions: %w", err)
	}
	scenes, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/sessions/%s/scenes", campaignID, sessionID))
	if err != nil {
		return sessionBrief{}, fmt.Errorf("read scenes: %w", err)
	}
	interaction, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/interaction", campaignID))
	if err != nil {
		return sessionBrief{}, fmt.Errorf("read interaction state: %w", err)
	}
	memory, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/artifacts/memory.md", campaignID))
	if err != nil {
		return sessionBrief{}, fmt.Errorf("read memory artifact: %w", err)
	}
	story, err := readOptionalResource(ctx, sess, fmt.Sprintf("campaign://%s/artifacts/story.md", campaignID))
	if err != nil {
		return sessionBrief{}, fmt.Errorf("read story artifact: %w", err)
	}

	var state struct {
		ActiveScene struct {
			SceneID string `json:"scene_id"`
		} `json:"active_scene"`
	}
	if err := json.Unmarshal([]byte(interaction), &state); err != nil {
		return sessionBrief{}, fmt.Errorf("decode interaction state: %w", err)
	}

	return sessionBrief{
		skills:       skills,
		story:        story,
		memory:       memory,
		current:      current,
		campaign:     campaign,
		participants: participants,
		characters:   characters,
		sessions:     sessions,
		scenes:       scenes,
		interaction:  interaction,
		bootstrap:    strings.TrimSpace(state.ActiveScene.SceneID) == "",
	}, nil
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
