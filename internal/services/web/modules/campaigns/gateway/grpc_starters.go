package gateway

import (
	"context"
	"fmt"
	"strings"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// NewStarterGateway builds the protected starter preview/launch adapter from explicit dependencies.
func NewStarterGateway(deps StarterDeps) campaignapp.CampaignStarterGateway {
	if deps.Discovery == nil || deps.Agent == nil || deps.Campaign == nil || deps.Fork == nil {
		return nil
	}
	return starterGateway{deps: deps}
}

// StarterPreview returns protected starter preview state plus the caller's selectable AI agents.
func (g starterGateway) StarterPreview(ctx context.Context, starterKey string) (campaignapp.CampaignStarterPreview, error) {
	entry, err := g.loadStarterEntry(ctx, starterKey)
	if err != nil {
		return campaignapp.CampaignStarterPreview{}, err
	}

	options, err := g.listStarterAIAgents(ctx)
	if err != nil {
		return campaignapp.CampaignStarterPreview{}, err
	}

	preview := campaignapp.CampaignStarterPreview{
		EntryID:            strings.TrimSpace(entry.GetEntryId()),
		TemplateCampaignID: strings.TrimSpace(entry.GetSourceId()),
		Title:              strings.TrimSpace(entry.GetTitle()),
		Description:        strings.TrimSpace(entry.GetDescription()),
		CampaignTheme:      starterCampaignTheme(entry),
		Hook:               strings.TrimSpace(entry.GetPreviewHook()),
		PlaystyleLabel:     strings.TrimSpace(entry.GetPreviewPlaystyleLabel()),
		CharacterName:      strings.TrimSpace(entry.GetPreviewCharacterName()),
		CharacterSummary:   strings.TrimSpace(entry.GetPreviewCharacterSummary()),
		System:             starterGameSystemLabel(entry.GetSystem()),
		Difficulty:         starterDifficultyLabel(entry.GetDifficultyTier()),
		Duration:           strings.TrimSpace(entry.GetExpectedDurationLabel()),
		GmMode:             starterGMModeLabel(entry.GetGmMode()),
		Players:            starterPlayersLabel(entry.GetRecommendedParticipantsMin(), entry.GetRecommendedParticipantsMax()),
		Tags:               append([]string(nil), entry.GetTags()...),
		AIAgentOptions:     options,
	}
	for _, option := range options {
		if option.Enabled {
			preview.HasAvailableAIAgents = true
			break
		}
	}
	return preview, nil
}

// LaunchStarter forks the canonical starter template and binds the selected AI agent.
func (g starterGateway) LaunchStarter(ctx context.Context, starterKey string, input campaignapp.LaunchStarterInput) (campaignapp.StarterLaunchResult, error) {
	entry, err := g.loadStarterEntry(ctx, starterKey)
	if err != nil {
		return campaignapp.StarterLaunchResult{}, err
	}
	templateCampaignID := strings.TrimSpace(entry.GetSourceId())
	if templateCampaignID == "" {
		return campaignapp.StarterLaunchResult{}, apperrors.EK(apperrors.KindUnavailable, "error.web.message.starter_template_is_unavailable", "starter template is unavailable")
	}

	aiAgentID := strings.TrimSpace(input.AIAgentID)
	if aiAgentID == "" {
		return campaignapp.StarterLaunchResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.ai_agent_is_required", "AI agent is required")
	}
	options, err := g.listStarterAIAgents(ctx)
	if err != nil {
		return campaignapp.StarterLaunchResult{}, err
	}
	if !containsEnabledAIAgent(options, aiAgentID) {
		return campaignapp.StarterLaunchResult{}, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.ai_agent_is_required", "AI agent is required")
	}

	forkResp, err := g.deps.Fork.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
		SourceCampaignId: templateCampaignID,
		NewCampaignName:  strings.TrimSpace(entry.GetTitle()),
		CopyParticipants: true,
	})
	if err != nil {
		return campaignapp.StarterLaunchResult{}, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_launch_starter",
			FallbackMessage: "failed to launch starter",
		})
	}
	campaignID := strings.TrimSpace(forkResp.GetCampaign().GetId())
	if campaignID == "" {
		return campaignapp.StarterLaunchResult{}, apperrors.E(apperrors.KindUnknown, "forked campaign id is required")
	}

	_, err = g.deps.Campaign.SetCampaignAIBinding(ctx, &statev1.SetCampaignAIBindingRequest{
		CampaignId: campaignID,
		AiAgentId:  aiAgentID,
	})
	if err != nil {
		return campaignapp.StarterLaunchResult{}, mapCampaignAIBindingMutationError(err)
	}
	return campaignapp.StarterLaunchResult{CampaignID: campaignID}, nil
}

// loadStarterEntry keeps discovery lookup and transport error mapping together.
func (g starterGateway) loadStarterEntry(ctx context.Context, starterKey string) (*discoveryv1.DiscoveryEntry, error) {
	starterKey = strings.TrimSpace(starterKey)
	if starterKey == "" {
		return nil, apperrors.E(apperrors.KindInvalidInput, "starter key is required")
	}
	resp, err := g.deps.Discovery.GetDiscoveryEntry(ctx, &discoveryv1.GetDiscoveryEntryRequest{EntryId: starterKey})
	if err != nil {
		return nil, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_load_starter",
			FallbackMessage: "failed to load starter",
		})
	}
	if resp == nil || resp.GetEntry() == nil {
		return nil, apperrors.E(apperrors.KindNotFound, "starter not found")
	}
	return resp.GetEntry(), nil
}

// listStarterAIAgents filters the caller-owned AI agents down to launchable choices.
func (g starterGateway) listStarterAIAgents(ctx context.Context) ([]campaignapp.CampaignAIAgentOption, error) {
	resp, err := g.deps.Agent.ListAgents(ctx, &aiv1.ListAgentsRequest{PageSize: campaignAIAgentsPageSize})
	if err != nil {
		return nil, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnknown,
			FallbackKey:     "error.web.message.failed_to_list_ai_agents",
			FallbackMessage: "failed to list AI agents",
		})
	}
	options := make([]campaignapp.CampaignAIAgentOption, 0, len(resp.GetAgents()))
	for _, agent := range resp.GetAgents() {
		if agent == nil {
			continue
		}
		agentID := strings.TrimSpace(agent.GetId())
		if agentID == "" {
			continue
		}
		options = append(options, campaignapp.CampaignAIAgentOption{
			ID:    agentID,
			Label: campaignAIAgentDisplayName(agent),
			Enabled: agent.GetStatus() == aiv1.AgentStatus_AGENT_STATUS_ACTIVE &&
				agent.GetAuthState() == aiv1.AgentAuthState_AGENT_AUTH_STATE_READY,
		})
	}
	return options, nil
}

// containsEnabledAIAgent enforces that launch only binds a ready caller-owned agent.
func containsEnabledAIAgent(options []campaignapp.CampaignAIAgentOption, aiAgentID string) bool {
	aiAgentID = strings.TrimSpace(aiAgentID)
	if aiAgentID == "" {
		return false
	}
	for _, option := range options {
		if strings.TrimSpace(option.ID) == aiAgentID && option.Enabled {
			return true
		}
	}
	return false
}

// starterDifficultyLabel centralizes human-readable discovery difficulty text for the preview page.
func starterDifficultyLabel(tier discoveryv1.DiscoveryDifficultyTier) string {
	switch tier {
	case discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER:
		return "Beginner"
	case discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_INTERMEDIATE:
		return "Intermediate"
	case discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_ADVANCED:
		return "Advanced"
	default:
		return ""
	}
}

// starterGMModeLabel keeps discovery GM-mode labels aligned with campaign UI copy.
func starterGMModeLabel(mode discoveryv1.DiscoveryGmMode) string {
	switch mode {
	case discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_HUMAN:
		return "Human"
	case discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_AI:
		return "AI"
	case discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_HYBRID:
		return "Hybrid"
	default:
		return ""
	}
}

// starterGameSystemLabel limits the preview to user-facing system labels we actually support.
func starterGameSystemLabel(system commonv1.GameSystem) string {
	switch system {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return "Daggerheart"
	default:
		return ""
	}
}

// starterPlayersLabel renders the recommended player count range for the starter card.
func starterPlayersLabel(min, max int32) string {
	if min <= 0 && max <= 0 {
		return ""
	}
	if min > 0 && max > 0 && min != max {
		return fmt.Sprintf("%d-%d", min, max)
	}
	if min > 0 && max > 0 {
		return fmt.Sprintf("%d", min)
	}
	if min > 0 {
		return fmt.Sprintf("%d+", min)
	}
	return fmt.Sprintf("up to %d", max)
}

// starterCampaignTheme resolves the spoiler-safe starter preview theme with a
// description fallback for entries that have not adopted the richer field yet.
func starterCampaignTheme(entry *discoveryv1.DiscoveryEntry) string {
	if entry == nil {
		return ""
	}
	theme := strings.TrimSpace(entry.GetCampaignTheme())
	if theme != "" {
		return theme
	}
	return strings.TrimSpace(entry.GetDescription())
}
