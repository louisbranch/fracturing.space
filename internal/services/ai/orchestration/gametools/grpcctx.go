package gametools

import (
	"context"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/id"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc/metadata"
)

const callTimeout = 30 * time.Second

// SessionContext carries the fixed campaign/session/participant authority for
// one orchestration run.
type SessionContext struct {
	CampaignID    string
	SessionID     string
	ParticipantID string
}

// outgoingContext attaches gRPC metadata for campaign authority and a fresh
// request ID to ctx. It also applies a call timeout when none is already set.
func outgoingContext(ctx context.Context, sc SessionContext) (context.Context, context.CancelFunc) {
	requestID, _ := id.NewID()

	ctx = metadata.AppendToOutgoingContext(ctx, grpcmeta.RequestIDHeader, requestID)
	if sc.CampaignID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, grpcmeta.CampaignIDHeader, strings.TrimSpace(sc.CampaignID))
	}
	if sc.SessionID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, grpcmeta.SessionIDHeader, strings.TrimSpace(sc.SessionID))
	}
	if sc.ParticipantID != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, grpcmeta.ParticipantIDHeader, strings.TrimSpace(sc.ParticipantID))
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		return context.WithTimeout(ctx, callTimeout)
	}
	return context.WithCancel(ctx)
}
