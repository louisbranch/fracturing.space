package outcometransport

import (
	"context"
	"time"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var testTimestamp = time.Date(2026, time.February, 14, 0, 0, 0, 0, time.UTC)

type testingT interface {
	Helper()
	Fatalf(format string, args ...any)
}

func assertStatusCode(t testingT, err error, want codes.Code) {
	t.Helper()
	got := status.Code(err)
	if got != want {
		t.Fatalf("status code = %v, want %v (err=%v)", got, want, err)
	}
}

func testSessionContext(campaignID, sessionID string) context.Context {
	md := metadata.Pairs(grpcmeta.CampaignIDHeader, campaignID, grpcmeta.SessionIDHeader, sessionID)
	return metadata.NewIncomingContext(context.Background(), md)
}
