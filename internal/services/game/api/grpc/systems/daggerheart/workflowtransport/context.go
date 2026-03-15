package workflowtransport

import (
	"context"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc/metadata"
)

// WithCampaignSessionMetadata attaches campaign/session metadata to a context
// for internal workflow chaining between Daggerheart handlers.
func WithCampaignSessionMetadata(ctx context.Context, campaignID, sessionID string) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	md = metadata.Join(md, metadata.Pairs(grpcmeta.CampaignIDHeader, campaignID, grpcmeta.SessionIDHeader, sessionID))
	return metadata.NewIncomingContext(ctx, md)
}

// NormalizeTargets trims, deduplicates, and preserves order for target ids.
func NormalizeTargets(targets []string) []string {
	if len(targets) == 0 {
		return nil
	}

	result := make([]string, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		trimmed := strings.TrimSpace(target)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
