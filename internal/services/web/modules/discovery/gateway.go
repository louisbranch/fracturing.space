package discovery

import (
	"context"
	"fmt"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
)

// ListingClient exposes listing operations needed by the discovery module.
type ListingClient interface {
	ListCampaignListings(ctx context.Context, in *listingv1.ListCampaignListingsRequest, opts ...grpc.CallOption) (*listingv1.ListCampaignListingsResponse, error)
}

// StarterListing is a presentation-ready listing card for the discovery page.
type StarterListing struct {
	CampaignID  string
	Title       string
	Description string
	Tags        []string
	Difficulty  string
	Duration    string
	GmMode      string
	System      string
	Level       int32
	Players     string
}

// Gateway abstracts listing data access for the discovery module.
type Gateway interface {
	ListStarterListings(ctx context.Context) ([]StarterListing, error)
}

// GRPCGateway implements Gateway backed by the listing gRPC service.
type GRPCGateway struct {
	client ListingClient
}

// NewGRPCGateway returns a Gateway backed by the given listing client.
// Returns an unavailable gateway when client is nil (fail-closed).
func NewGRPCGateway(client ListingClient) Gateway {
	if client == nil {
		return unavailableGateway{}
	}
	return GRPCGateway{client: client}
}

// IsGatewayHealthy reports whether a discovery gateway is configured and usable.
func IsGatewayHealthy(gw Gateway) bool {
	if gw == nil {
		return false
	}
	_, unavailable := gw.(unavailableGateway)
	return !unavailable
}

// unavailableGateway is a fail-closed Gateway returned when the listing client is nil.
type unavailableGateway struct{}

// ListStarterListings always returns an unavailable error.
func (unavailableGateway) ListStarterListings(context.Context) ([]StarterListing, error) {
	return nil, apperrors.E(apperrors.KindUnavailable, "listing service client is not configured")
}

// ListStarterListings fetches campaign listings and filters to starter intent.
func (g GRPCGateway) ListStarterListings(ctx context.Context) ([]StarterListing, error) {
	resp, err := g.client.ListCampaignListings(ctx, &listingv1.ListCampaignListingsRequest{
		PageSize: 50,
	})
	if err != nil {
		return nil, apperrors.MapGRPCTransportError(err, apperrors.GRPCStatusMapping{
			FallbackKind:    apperrors.KindUnavailable,
			FallbackMessage: "listing service is unavailable",
		})
	}
	if resp == nil {
		return nil, nil
	}

	var results []StarterListing
	for _, listing := range resp.GetListings() {
		if listing.GetIntent() != listingv1.CampaignListingIntent_CAMPAIGN_LISTING_INTENT_STARTER {
			continue
		}
		results = append(results, mapProtoToStarterListing(listing))
	}
	return results, nil
}

// mapProtoToStarterListing converts a proto CampaignListing to a presentation-ready StarterListing.
func mapProtoToStarterListing(l *listingv1.CampaignListing) StarterListing {
	return StarterListing{
		CampaignID:  strings.TrimSpace(l.GetCampaignId()),
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
func difficultyLabel(tier listingv1.CampaignDifficultyTier) string {
	switch tier {
	case listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER:
		return "Beginner"
	case listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_INTERMEDIATE:
		return "Intermediate"
	case listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_ADVANCED:
		return "Advanced"
	default:
		return ""
	}
}

// gmModeLabel maps a proto GM mode to a display string.
func gmModeLabel(mode listingv1.CampaignListingGmMode) string {
	switch mode {
	case listingv1.CampaignListingGmMode_CAMPAIGN_LISTING_GM_MODE_HUMAN:
		return "Human"
	case listingv1.CampaignListingGmMode_CAMPAIGN_LISTING_GM_MODE_AI:
		return "AI"
	case listingv1.CampaignListingGmMode_CAMPAIGN_LISTING_GM_MODE_HYBRID:
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
