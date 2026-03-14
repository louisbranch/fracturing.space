package game

import (
	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/filter"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ListTimelineEntries returns a paginated timeline view for a campaign.
func (s *EventService) ListTimelineEntries(ctx context.Context, in *campaignv1.ListTimelineEntriesRequest) (*campaignv1.ListTimelineEntriesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign_id is required")
	}

	pageSize := pagination.ClampPageSize(in.GetPageSize(), pagination.PageSizeConfig{
		Default: defaultListEventsPageSize,
		Max:     maxListEventsPageSize,
	})

	orderBy, err := pagination.NormalizeOrderBy(strings.TrimSpace(in.GetOrderBy()), pagination.OrderByConfig{
		Default: "seq",
		Allowed: []string{"seq", "seq desc"},
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_by: %s (must be 'seq' or 'seq desc')", strings.TrimSpace(in.GetOrderBy()))
	}
	descending := orderBy == "seq desc"

	filterStr := strings.TrimSpace(in.GetFilter())
	if filterStr != "" {
		if _, err := filter.ParseEventFilter(filterStr); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter: %v", err)
		}
	}

	var cursorSeq uint64
	var cursorDir string
	var cursorReverse bool
	pageToken := strings.TrimSpace(in.GetPageToken())
	if pageToken != "" {
		c, err := pagination.Decode(pageToken)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid page_token: %v", err)
		}
		if err := pagination.ValidateFilterHash(c, filterStr); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "page_token invalid: %v", err)
		}
		if err := pagination.ValidateOrderHash(c, orderBy); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "page_token invalid: %v", err)
		}
		seqValue, err := pagination.ValueUint(c, "seq")
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "page_token invalid: %v", err)
		}
		cursorSeq = seqValue
		cursorDir = string(c.Dir)
		cursorReverse = c.Reverse
	}

	if err := requireReadPolicy(ctx, s.stores, storage.CampaignRecord{ID: campaignID}); err != nil {
		return nil, err
	}

	req := storage.ListEventsPageRequest{
		CampaignID:    campaignID,
		PageSize:      pageSize,
		CursorSeq:     cursorSeq,
		CursorDir:     cursorDir,
		CursorReverse: cursorReverse,
		Descending:    descending,
		Filter: storage.EventQueryFilter{
			Expression: filterStr,
		},
	}

	result, err := s.stores.Event.ListEventsPage(ctx, req)
	if err != nil {
		return nil, grpcerror.Internal("list timeline entries", err)
	}

	resolver := newTimelineProjectionResolver(timelineProjectionStores{
		Campaign:    s.stores.Campaign,
		Participant: s.stores.Participant,
		Character:   s.stores.Character,
		Session:     s.stores.Session,
	})
	response := &campaignv1.ListTimelineEntriesResponse{
		Entries:   make([]*campaignv1.TimelineEntry, 0, len(result.Events)),
		TotalSize: int32(result.TotalCount),
	}

	for _, evt := range result.Events {
		entry, err := timelineEntryFromEvent(ctx, resolver, evt)
		if err != nil {
			return nil, grpcerror.Internal("resolve timeline entry", err)
		}
		response.Entries = append(response.Entries, entry)
	}

	if len(result.Events) > 0 {
		if result.HasNextPage {
			lastSeq := result.Events[len(result.Events)-1].Seq
			nextCursor := pagination.NewNextPageCursor(
				[]pagination.CursorValue{pagination.UintValue("seq", lastSeq)},
				descending,
				filterStr,
				orderBy,
			)
			token, err := pagination.Encode(nextCursor)
			if err == nil {
				response.NextPageToken = token
			}
		}
		if result.HasPrevPage {
			firstSeq := result.Events[0].Seq
			prevCursor := pagination.NewPrevPageCursor(
				[]pagination.CursorValue{pagination.UintValue("seq", firstSeq)},
				descending,
				filterStr,
				orderBy,
			)
			token, err := pagination.Encode(prevCursor)
			if err == nil {
				response.PreviousPageToken = token
			}
		}
	}

	return response, nil
}
