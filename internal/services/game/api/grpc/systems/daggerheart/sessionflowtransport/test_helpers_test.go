package sessionflowtransport

import (
	"context"
	"testing"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
)

func assertContextIDs(t *testing.T, ctx context.Context, campaignID, sessionID string) {
	t.Helper()
	if got := grpcmeta.CampaignIDFromContext(ctx); got != campaignID {
		t.Fatalf("campaign_id metadata = %q, want %q", got, campaignID)
	}
	if got := grpcmeta.SessionIDFromContext(ctx); got != sessionID {
		t.Fatalf("session_id metadata = %q, want %q", got, sessionID)
	}
}
