package outcometransport

import (
	"context"
	"time"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"google.golang.org/grpc/metadata"
)

var testTimestamp = time.Date(2026, time.February, 14, 0, 0, 0, 0, time.UTC)

type testingT interface {
	Helper()
	Fatalf(format string, args ...any)
}

func testSessionContext(campaignID, sessionID string) context.Context {
	md := metadata.Pairs(grpcmeta.CampaignIDHeader, campaignID, grpcmeta.SessionIDHeader, sessionID)
	return metadata.NewIncomingContext(context.Background(), md)
}
