package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListTimelineEntries returns a paginated timeline view for a campaign.
func (s *EventService) ListTimelineEntries(ctx context.Context, in *campaignv1.ListTimelineEntriesRequest) (*campaignv1.ListTimelineEntriesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list timeline entries request is required")
	}
	return newEventApplication(s).ListTimelineEntries(ctx, in)
}
