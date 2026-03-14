package discovery

import (
	"context"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	discoveryapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
)

// DiscoveryClient exposes discovery operations needed by the discovery module.
type DiscoveryClient interface {
	ListDiscoveryEntries(ctx context.Context, in *discoveryv1.ListDiscoveryEntriesRequest, opts ...grpc.CallOption) (*discoveryv1.ListDiscoveryEntriesResponse, error)
}

// StarterEntry is the transport-facing alias for discovery page entries.
type StarterEntry = discoveryapp.StarterEntry

// Gateway abstracts discovery-entry data access for the discovery module.
type Gateway = discoveryapp.Gateway

// GRPCGateway implements Gateway backed by the discovery gRPC service.
type GRPCGateway struct {
	client DiscoveryClient
}

// NewGRPCGateway returns a Gateway backed by the given discovery client.
// Returns an unavailable gateway when client is nil (fail-closed).
func NewGRPCGateway(client DiscoveryClient) Gateway {
	if client == nil {
		return discoveryapp.NewUnavailableGateway()
	}
	return GRPCGateway{client: client}
}

// IsGatewayHealthy reports whether a discovery gateway is configured and usable.
func IsGatewayHealthy(gw Gateway) bool {
	return discoveryapp.IsGatewayHealthy(gw)
}

// ListStarterEntries fetches discovery entries and filters to starter intent.
func (g GRPCGateway) ListStarterEntries(ctx context.Context) ([]StarterEntry, error) {
	resp, err := g.client.ListDiscoveryEntries(ctx, &discoveryv1.ListDiscoveryEntriesRequest{
		PageSize: 50,
		Kind:     discoveryv1.DiscoveryEntryKind_DISCOVERY_ENTRY_KIND_CAMPAIGN_STARTER,
	})
	if err != nil {
		return nil, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnavailable,
			FallbackMessage: "discovery service is unavailable",
		})
	}
	if resp == nil {
		return nil, nil
	}

	var results []StarterEntry
	for _, entry := range resp.GetEntries() {
		if entry.GetIntent() != discoveryv1.DiscoveryIntent_DISCOVERY_INTENT_STARTER {
			continue
		}
		results = append(results, mapProtoToStarterEntry(entry))
	}
	return results, nil
}

// mapProtoToStarterEntry converts a proto DiscoveryEntry to a presentation-ready StarterEntry.
func mapProtoToStarterEntry(l *discoveryv1.DiscoveryEntry) StarterEntry {
	campaignID := strings.TrimSpace(l.GetSourceId())
	if campaignID == "" {
		campaignID = strings.TrimSpace(l.GetEntryId())
	}
	return StarterEntry{
		CampaignID:  campaignID,
		Title:       strings.TrimSpace(l.GetTitle()),
		Description: strings.TrimSpace(l.GetDescription()),
		Tags:        l.GetTags(),
		Difficulty:  difficultyLabel(l.GetDifficultyTier()),
		Duration:    strings.TrimSpace(l.GetExpectedDurationLabel()),
		GmMode:      gmModeLabel(l.GetGmMode()),
		System:      gameSystemLabel(l.GetSystem()),
		Level:       l.GetLevel(),
		Players:     playersLabel(l.GetRecommendedParticipantsMin(), l.GetRecommendedParticipantsMax()),
	}
}

// difficultyLabel maps a proto difficulty tier to a display string.
func difficultyLabel(tier discoveryv1.DiscoveryDifficultyTier) string {
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

// gmModeLabel maps a proto GM mode to a display string.
func gmModeLabel(mode discoveryv1.DiscoveryGmMode) string {
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

// gameSystemLabel maps a proto game system to a display string.
func gameSystemLabel(sys commonv1.GameSystem) string {
	switch sys {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return "Daggerheart"
	default:
		return ""
	}
}

// playersLabel formats a participant range like "2-4".
func playersLabel(min, max int32) string {
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
