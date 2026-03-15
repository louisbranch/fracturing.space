package game

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"

	"context"
	"strconv"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/filter"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type normalizedTimelineRequest struct {
	campaignID    string
	pageSize      int
	orderBy       string
	descending    bool
	filterStr     string
	filter        storage.EventQueryFilter
	cursorSeq     uint64
	cursorDir     string
	cursorReverse bool
}

// ListEvents loads paginated event history behind the event application seam.
func (a eventApplication) ListEvents(ctx context.Context, in *campaignv1.ListEventsRequest) (*campaignv1.ListEventsResponse, error) {
	normalized, err := normalizeListEventsRequest(in)
	if err != nil {
		return nil, err
	}

	if err := authz.RequireReadPolicy(ctx, a.auth, storage.CampaignRecord{ID: normalized.campaignID}); err != nil {
		return nil, err
	}

	result, err := a.stores.Event.ListEventsPage(ctx, storage.ListEventsPageRequest{
		CampaignID:    normalized.campaignID,
		AfterSeq:      normalized.afterSeq,
		PageSize:      normalized.pageSize,
		CursorSeq:     normalized.cursorSeq,
		CursorDir:     normalized.cursorDir,
		CursorReverse: normalized.cursorReverse,
		Descending:    normalized.descending,
		Filter:        normalized.filter,
	})
	if err != nil {
		return nil, grpcerror.Internal("list events", err)
	}

	response := &campaignv1.ListEventsResponse{
		Events:    make([]*campaignv1.Event, 0, len(result.Events)),
		TotalSize: int32(result.TotalCount),
	}
	for _, evt := range result.Events {
		response.Events = append(response.Events, eventToProto(evt))
	}

	if len(result.Events) > 0 {
		if result.HasNextPage {
			lastSeq := result.Events[len(result.Events)-1].Seq
			nextCursor := pagination.NewNextPageCursor(
				[]pagination.CursorValue{pagination.UintValue("seq", lastSeq)},
				normalized.descending,
				normalized.paginationScope,
				normalized.orderBy,
			)
			if token, err := pagination.Encode(nextCursor); err == nil {
				response.NextPageToken = token
			}
		}
		if result.HasPrevPage {
			firstSeq := result.Events[0].Seq
			prevCursor := pagination.NewPrevPageCursor(
				[]pagination.CursorValue{pagination.UintValue("seq", firstSeq)},
				normalized.descending,
				normalized.paginationScope,
				normalized.orderBy,
			)
			if token, err := pagination.Encode(prevCursor); err == nil {
				response.PreviousPageToken = token
			}
		}
	}

	return response, nil
}

// SubscribeCampaignUpdates streams campaign updates behind the event
// application seam so the service handler only validates transport inputs.
func (a eventApplication) SubscribeCampaignUpdates(
	in *campaignv1.SubscribeCampaignUpdatesRequest,
	stream grpc.ServerStreamingServer[campaignv1.CampaignUpdate],
) error {
	normalized, err := normalizeSubscribeCampaignUpdatesRequest(in)
	if err != nil {
		return err
	}
	if a.stores.Event == nil {
		return status.Error(codes.Internal, "event store is not configured")
	}
	if stream == nil {
		return status.Error(codes.Internal, "stream is required")
	}

	ctx := stream.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := authz.RequireReadPolicy(ctx, a.auth, storage.CampaignRecord{ID: normalized.campaignID}); err != nil {
		return err
	}

	lastSeq := normalized.afterSeq
	sendAvailable := func() error {
		events, err := a.stores.Event.ListEvents(ctx, normalized.campaignID, lastSeq, maxListEventsPageSize)
		if err != nil {
			return grpcerror.Internal("list events", err)
		}

		for _, evt := range events {
			if normalized.includeEventCommitted {
				if err := stream.Send(campaignUpdateEventCommitted(evt)); err != nil {
					if ctx.Err() != nil {
						return nil
					}
					return status.Errorf(codes.Unavailable, "send event_committed update: %v", err)
				}
			}
			if normalized.includeProjection {
				scopes := projectionScopesForEventType(string(evt.Type))
				if hasProjectionScopeIntersection(scopes, normalized.projectionScopes) {
					if err := stream.Send(campaignUpdateProjectionApplied(evt, scopes)); err != nil {
						if ctx.Err() != nil {
							return nil
						}
						return status.Errorf(codes.Unavailable, "send projection_applied update: %v", err)
					}
				}
			}
			if evt.Seq > lastSeq {
				lastSeq = evt.Seq
			}
		}
		return nil
	}

	if err := sendAvailable(); err != nil {
		return err
	}

	ticker := time.NewTicker(normalized.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := sendAvailable(); err != nil {
				return err
			}
		}
	}
}

// ListTimelineEntries loads the timeline view behind the event application seam.
func (a eventApplication) ListTimelineEntries(ctx context.Context, in *campaignv1.ListTimelineEntriesRequest) (*campaignv1.ListTimelineEntriesResponse, error) {
	normalized, err := normalizeListTimelineEntriesRequest(in)
	if err != nil {
		return nil, err
	}
	if err := authz.RequireReadPolicy(ctx, a.auth, storage.CampaignRecord{ID: normalized.campaignID}); err != nil {
		return nil, err
	}

	result, err := a.stores.Event.ListEventsPage(ctx, storage.ListEventsPageRequest{
		CampaignID:    normalized.campaignID,
		PageSize:      normalized.pageSize,
		CursorSeq:     normalized.cursorSeq,
		CursorDir:     normalized.cursorDir,
		CursorReverse: normalized.cursorReverse,
		Descending:    normalized.descending,
		Filter:        normalized.filter,
	})
	if err != nil {
		return nil, grpcerror.Internal("list timeline entries", err)
	}

	resolver := newTimelineProjectionResolver(timelineProjectionStores{
		Campaign:    a.stores.Campaign,
		Participant: a.stores.Participant,
		Character:   a.stores.Character,
		Session:     a.stores.Session,
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
				normalized.descending,
				normalized.filterStr,
				normalized.orderBy,
			)
			if token, err := pagination.Encode(nextCursor); err == nil {
				response.NextPageToken = token
			}
		}
		if result.HasPrevPage {
			firstSeq := result.Events[0].Seq
			prevCursor := pagination.NewPrevPageCursor(
				[]pagination.CursorValue{pagination.UintValue("seq", firstSeq)},
				normalized.descending,
				normalized.filterStr,
				normalized.orderBy,
			)
			if token, err := pagination.Encode(prevCursor); err == nil {
				response.PreviousPageToken = token
			}
		}
	}

	return response, nil
}

func normalizeListEventsRequest(in *campaignv1.ListEventsRequest) (normalizedListEventsRequest, error) {
	if in == nil {
		return normalizedListEventsRequest{}, status.Error(codes.InvalidArgument, "request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return normalizedListEventsRequest{}, status.Error(codes.InvalidArgument, "campaign_id is required")
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
		return normalizedListEventsRequest{}, status.Errorf(codes.InvalidArgument, "invalid order_by: %s (must be 'seq' or 'seq desc')", strings.TrimSpace(in.GetOrderBy()))
	}
	descending := orderBy == "seq desc"

	filterStr := strings.TrimSpace(in.GetFilter())
	afterSeq := in.GetAfterSeq()
	paginationScope := filterStr + "|after_seq=" + strconv.FormatUint(afterSeq, 10)

	if filterStr != "" {
		if _, err := filter.ParseEventFilter(filterStr); err != nil {
			return normalizedListEventsRequest{}, status.Errorf(codes.InvalidArgument, "invalid filter: %v", err)
		}
	}

	var cursorSeq uint64
	var cursorDir string
	var cursorReverse bool
	pageToken := strings.TrimSpace(in.GetPageToken())
	if pageToken != "" {
		c, err := pagination.Decode(pageToken)
		if err != nil {
			return normalizedListEventsRequest{}, status.Errorf(codes.InvalidArgument, "invalid page_token: %v", err)
		}
		if err := pagination.ValidateFilterHash(c, paginationScope); err != nil {
			return normalizedListEventsRequest{}, status.Errorf(codes.InvalidArgument, "page_token invalid: %v", err)
		}
		if err := pagination.ValidateOrderHash(c, orderBy); err != nil {
			return normalizedListEventsRequest{}, status.Errorf(codes.InvalidArgument, "page_token invalid: %v", err)
		}
		seqValue, err := pagination.ValueUint(c, "seq")
		if err != nil {
			return normalizedListEventsRequest{}, status.Errorf(codes.InvalidArgument, "page_token invalid: %v", err)
		}
		cursorSeq = seqValue
		cursorDir = string(c.Dir)
		cursorReverse = c.Reverse
	}

	return normalizedListEventsRequest{
		campaignID:      campaignID,
		pageSize:        pageSize,
		orderBy:         orderBy,
		descending:      descending,
		filterStr:       filterStr,
		filter:          storage.EventQueryFilter{Expression: filterStr},
		afterSeq:        afterSeq,
		paginationScope: paginationScope,
		cursorSeq:       cursorSeq,
		cursorDir:       cursorDir,
		cursorReverse:   cursorReverse,
	}, nil
}

func normalizeListTimelineEntriesRequest(in *campaignv1.ListTimelineEntriesRequest) (normalizedTimelineRequest, error) {
	if in == nil {
		return normalizedTimelineRequest{}, status.Error(codes.InvalidArgument, "request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return normalizedTimelineRequest{}, status.Error(codes.InvalidArgument, "campaign_id is required")
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
		return normalizedTimelineRequest{}, status.Errorf(codes.InvalidArgument, "invalid order_by: %s (must be 'seq' or 'seq desc')", strings.TrimSpace(in.GetOrderBy()))
	}
	descending := orderBy == "seq desc"

	filterStr := strings.TrimSpace(in.GetFilter())
	if filterStr != "" {
		if _, err := filter.ParseEventFilter(filterStr); err != nil {
			return normalizedTimelineRequest{}, status.Errorf(codes.InvalidArgument, "invalid filter: %v", err)
		}
	}

	var cursorSeq uint64
	var cursorDir string
	var cursorReverse bool
	pageToken := strings.TrimSpace(in.GetPageToken())
	if pageToken != "" {
		c, err := pagination.Decode(pageToken)
		if err != nil {
			return normalizedTimelineRequest{}, status.Errorf(codes.InvalidArgument, "invalid page_token: %v", err)
		}
		if err := pagination.ValidateFilterHash(c, filterStr); err != nil {
			return normalizedTimelineRequest{}, status.Errorf(codes.InvalidArgument, "page_token invalid: %v", err)
		}
		if err := pagination.ValidateOrderHash(c, orderBy); err != nil {
			return normalizedTimelineRequest{}, status.Errorf(codes.InvalidArgument, "page_token invalid: %v", err)
		}
		seqValue, err := pagination.ValueUint(c, "seq")
		if err != nil {
			return normalizedTimelineRequest{}, status.Errorf(codes.InvalidArgument, "page_token invalid: %v", err)
		}
		cursorSeq = seqValue
		cursorDir = string(c.Dir)
		cursorReverse = c.Reverse
	}

	return normalizedTimelineRequest{
		campaignID:    campaignID,
		pageSize:      pageSize,
		orderBy:       orderBy,
		descending:    descending,
		filterStr:     filterStr,
		filter:        storage.EventQueryFilter{Expression: filterStr},
		cursorSeq:     cursorSeq,
		cursorDir:     cursorDir,
		cursorReverse: cursorReverse,
	}, nil
}
