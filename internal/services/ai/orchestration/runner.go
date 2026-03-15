package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/mcpbridge"
)

const defaultMaxSteps = 8

// ErrStepLimit indicates the model exceeded the allowed tool loop depth.
var ErrStepLimit = errors.New("campaign orchestration exceeded tool loop limit")
var ErrNarrationNotCommitted = errors.New("campaign orchestration did not commit gm output")

type runner struct {
	dialer Dialer
	max    int
}

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

// NewRunner builds a campaign-turn runner over one MCP dialer.
func NewRunner(dialer Dialer, max int) CampaignTurnRunner {
	if max <= 0 {
		max = defaultMaxSteps
	}
	return &runner{dialer: dialer, max: max}
}

// Run executes one MCP-backed provider turn.
func (r *runner) Run(ctx context.Context, input Input) (Result, error) {
	if r == nil || r.dialer == nil {
		return Result{}, fmt.Errorf("campaign turn runner is not configured")
	}
	if input.Provider == nil {
		return Result{}, fmt.Errorf("campaign turn provider is required")
	}
	if strings.TrimSpace(input.CampaignID) == "" || strings.TrimSpace(input.SessionID) == "" || strings.TrimSpace(input.ParticipantID) == "" {
		return Result{}, fmt.Errorf("campaign, session, and participant are required")
	}
	if strings.TrimSpace(input.Model) == "" {
		return Result{}, fmt.Errorf("model is required")
	}
	if strings.TrimSpace(input.CredentialSecret) == "" {
		return Result{}, fmt.Errorf("credential secret is required")
	}

	ctx = mcpbridge.WithSessionContext(ctx, mcpbridge.SessionContext{
		CampaignID:    input.CampaignID,
		SessionID:     input.SessionID,
		ParticipantID: input.ParticipantID,
	})

	sess, err := r.dialer.Dial(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("dial mcp: %w", err)
	}
	defer sess.Close()

	tools, err := sess.ListTools(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("list mcp tools: %w", err)
	}
	allowedTools := filterTools(tools)
	allowedToolNames := make(map[string]struct{}, len(allowedTools))
	for _, tool := range allowedTools {
		allowedToolNames[tool.Name] = struct{}{}
	}
	prompt, err := buildPrompt(ctx, sess, input)
	if err != nil {
		return Result{}, err
	}

	var convo string
	committed := false
	var results []ProviderToolResult
	commitReminderUsed := false
	var followUpPrompt string

	for i := 0; i < r.max; i++ {
		step, err := input.Provider.Run(ctx, ProviderInput{
			Model:            input.Model,
			ReasoningEffort:  input.ReasoningEffort,
			Instructions:     input.Instructions,
			Prompt:           prompt,
			FollowUpPrompt:   followUpPrompt,
			CredentialSecret: input.CredentialSecret,
			Tools:            allowedTools,
			ConversationID:   convo,
			Results:          results,
		})
		if err != nil {
			return Result{}, fmt.Errorf("invoke provider: %w", err)
		}
		convo = strings.TrimSpace(step.ConversationID)
		followUpPrompt = ""
		if len(step.ToolCalls) == 0 {
			text := strings.TrimSpace(step.OutputText)
			if text == "" {
				return Result{}, fmt.Errorf("provider returned no tool calls or output")
			}
			if !committed {
				if !commitReminderUsed && i+1 < r.max {
					commitReminderUsed = true
					results = nil
					followUpPrompt = buildCommitReminder(text)
					continue
				}
				return Result{}, ErrNarrationNotCommitted
			}
			return Result{OutputText: text}, nil
		}

		results = make([]ProviderToolResult, 0, len(step.ToolCalls))
		for _, call := range step.ToolCalls {
			if _, ok := allowedToolNames[strings.TrimSpace(call.Name)]; !ok {
				results = append(results, ProviderToolResult{
					CallID:  call.CallID,
					Output:  fmt.Sprintf("tool %q is not allowed for campaign orchestration", strings.TrimSpace(call.Name)),
					IsError: true,
				})
				continue
			}
			args, err := decodeArgs(call.Arguments)
			if err != nil {
				results = append(results, ProviderToolResult{
					CallID:  call.CallID,
					Output:  fmt.Sprintf("invalid tool arguments: %v", err),
					IsError: true,
				})
				continue
			}
			res, err := sess.CallTool(ctx, call.Name, args)
			if err != nil {
				results = append(results, ProviderToolResult{
					CallID:  call.CallID,
					Output:  fmt.Sprintf("tool call failed: %v", err),
					IsError: true,
				})
				continue
			}
			if call.Name == "interaction_scene_gm_output_commit" && !res.IsError {
				committed = true
			}
			results = append(results, ProviderToolResult{
				CallID:  call.CallID,
				Output:  res.Output,
				IsError: res.IsError,
			})
		}
	}

	return Result{}, ErrStepLimit
}

func buildCommitReminder(text string) string {
	text = strings.TrimSpace(text)
	var b strings.Builder
	b.WriteString("You returned narration without calling interaction_scene_gm_output_commit.\n")
	b.WriteString("Convert that draft into an authoritative tool call before returning final text.\n")
	b.WriteString("If there is no active scene, set one active first.\n")
	if text != "" {
		b.WriteString("Use this draft narration as the commit text unless you need a small correction:\n")
		b.WriteString(text)
	}
	return b.String()
}

func filterTools(tools []Tool) []Tool {
	filtered := make([]Tool, 0, len(tools))
	for _, tool := range tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		if !mcpbridge.ProductionToolAllowed(name) {
			continue
		}
		filtered = append(filtered, tool)
	}
	return filtered
}

func buildPrompt(ctx context.Context, sess Session, input Input) (string, error) {
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
	b.WriteString("Use MCP tools for authoritative state changes; do not rely on free-form narration to mutate game state.\n")
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
		b.WriteString("\nActive scene mode: continue the session from the current interaction state and use MCP tools for authoritative changes.\n\n")
	}
	b.WriteString("Current MCP context:\n")
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

func decodeArgs(raw string) (any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return nil, err
	}
	return value, nil
}
