package readiness

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
)

func evaluateSessionBoundaryBlockers(state aggregate.State, options ReportOptions) []Blocker {
	if !options.IncludeSessionBoundary {
		return nil
	}

	blockers := make([]Blocker, 0, 2)
	if !campaignStatusAllowsSessionStart(state.Campaign.Status) {
		blockers = append(blockers, newBlocker(
			RejectionCodeSessionReadinessCampaignStatusDisallowsStart,
			fmt.Sprintf("campaign readiness requires campaign status draft or active before session start (status=%s)", normalizeCampaignStatus(state.Campaign.Status)),
			map[string]string{
				"status": string(normalizeCampaignStatus(state.Campaign.Status)),
			},
		))
	}
	if options.HasActiveSession {
		blockers = append(blockers, newBlocker(
			RejectionCodeSessionReadinessActiveSessionExists,
			"campaign readiness requires no active session",
			nil,
		))
	}
	return blockers
}

func campaignStatusAllowsSessionStart(status campaign.Status) bool {
	switch normalizeCampaignStatus(status) {
	case campaign.StatusDraft, campaign.StatusActive:
		return true
	default:
		return false
	}
}

func normalizeCampaignStatus(status campaign.Status) campaign.Status {
	trimmed := strings.TrimSpace(string(status))
	normalized, ok := campaign.NormalizeStatus(trimmed)
	if ok {
		return normalized
	}
	return campaign.Status(trimmed)
}

func isAIGMMode(mode campaign.GmMode) bool {
	switch mode {
	case campaign.GmModeAI, campaign.GmModeHybrid:
		return true
	default:
		return false
	}
}
