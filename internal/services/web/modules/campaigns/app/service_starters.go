package app

import (
	"context"
	"strings"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// StarterPreview returns protected starter preview state.
func (s starterService) StarterPreview(ctx context.Context, starterKey string) (CampaignStarterPreview, error) {
	return s.starterPreview(ctx, starterKey)
}

// LaunchStarter launches a new campaign from one protected starter template.
func (s starterService) LaunchStarter(ctx context.Context, starterKey string, input LaunchStarterInput) (StarterLaunchResult, error) {
	return s.launchStarter(ctx, starterKey, input)
}

// starterPreview keeps starter-key validation close to the protected read flow.
func (s starterService) starterPreview(ctx context.Context, starterKey string) (CampaignStarterPreview, error) {
	starterKey = strings.TrimSpace(starterKey)
	if starterKey == "" {
		return CampaignStarterPreview{}, apperrors.E(apperrors.KindInvalidInput, "starter key is required")
	}
	return s.gateway.StarterPreview(ctx, starterKey)
}

// launchStarter shares starter-key validation across transport entry points.
func (s starterService) launchStarter(ctx context.Context, starterKey string, input LaunchStarterInput) (StarterLaunchResult, error) {
	starterKey = strings.TrimSpace(starterKey)
	if starterKey == "" {
		return StarterLaunchResult{}, apperrors.E(apperrors.KindInvalidInput, "starter key is required")
	}
	return s.gateway.LaunchStarter(ctx, starterKey, input)
}
