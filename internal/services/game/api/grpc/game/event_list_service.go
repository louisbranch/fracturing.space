package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListEventsPageSize = pageMedium
	maxListEventsPageSize     = pageLarge
)

type normalizedListEventsRequest struct {
	campaignID      string
	pageSize        int
	orderBy         string
	descending      bool
	filterStr       string
	filter          storage.EventQueryFilter
	afterSeq        uint64
	paginationScope string
	cursorSeq       uint64
	cursorDir       string
	cursorReverse   bool
}

// ListEvents returns a paginated, filtered, and sorted list of events for a campaign.
func (s *EventService) ListEvents(ctx context.Context, in *campaignv1.ListEventsRequest) (*campaignv1.ListEventsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list events request is required")
	}
	return newEventApplication(s).ListEvents(ctx, in)
}
