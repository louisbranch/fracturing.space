package campaign

import (
	"context"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/core/filter"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"github.com/louisbranch/fracturing.space/internal/storage/cursor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultListEventsPageSize = 50
	maxListEventsPageSize     = 200
)

// EventService implements the campaign.v1.EventService gRPC API.
type EventService struct {
	campaignv1.UnimplementedEventServiceServer
	stores Stores
}

// NewEventService creates an EventService with the provided stores.
func NewEventService(stores Stores) *EventService {
	return &EventService{
		stores: stores,
	}
}

// AppendEvent appends a new event to the campaign journal.
func (s *EventService) AppendEvent(ctx context.Context, in *campaignv1.AppendEventRequest) (*campaignv1.AppendEventResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "append event request is required")
	}

	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	input, err := event.NormalizeForAppend(event.Event{
		CampaignID:   in.GetCampaignId(),
		Timestamp:    time.Now().UTC(),
		Type:         event.Type(strings.TrimSpace(in.GetType())),
		SessionID:    strings.TrimSpace(in.GetSessionId()),
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    event.ActorType(strings.TrimSpace(in.GetActorType())),
		ActorID:      strings.TrimSpace(in.GetActorId()),
		EntityType:   strings.TrimSpace(in.GetEntityType()),
		EntityID:     strings.TrimSpace(in.GetEntityId()),
		PayloadJSON:  in.GetPayloadJson(),
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	stored, err := s.stores.Event.AppendEvent(ctx, input)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	return &campaignv1.AppendEventResponse{Event: eventToProto(stored)}, nil
}

// ListEvents returns a paginated, filtered, and sorted list of events for a campaign.
func (s *EventService) ListEvents(ctx context.Context, in *campaignv1.ListEventsRequest) (*campaignv1.ListEventsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign_id is required")
	}

	// Validate page size
	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = defaultListEventsPageSize
	}
	if pageSize > maxListEventsPageSize {
		pageSize = maxListEventsPageSize
	}

	// Parse ordering
	descending := false
	orderBy := strings.TrimSpace(in.GetOrderBy())
	switch orderBy {
	case "", "seq":
		descending = false
	case "seq desc":
		descending = true
	default:
		return nil, status.Errorf(codes.InvalidArgument, "invalid order_by: %s (must be 'seq' or 'seq desc')", orderBy)
	}

	// Parse filter
	filterStr := strings.TrimSpace(in.GetFilter())
	var filterClause string
	var filterParams []any
	if filterStr != "" {
		cond, err := filter.ParseEventFilter(filterStr)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid filter: %v", err)
		}
		filterClause = cond.Clause
		filterParams = cond.Params
	}

	// Parse cursor
	var cursorSeq uint64
	var cursorDir string
	var cursorReverse bool
	pageToken := strings.TrimSpace(in.GetPageToken())
	if pageToken != "" {
		c, err := cursor.Decode(pageToken)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid page_token: %v", err)
		}
		// Validate filter hasn't changed
		if err := cursor.ValidateFilterHash(c, filterStr); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "page_token invalid: %v", err)
		}
		// Validate order_by hasn't changed
		if err := cursor.ValidateOrderHash(c, orderBy); err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "page_token invalid: %v", err)
		}
		cursorSeq = c.Seq
		cursorDir = string(c.Dir)
		cursorReverse = c.Reverse
	}

	// Build request
	req := storage.ListEventsPageRequest{
		CampaignID:    campaignID,
		PageSize:      pageSize,
		CursorSeq:     cursorSeq,
		CursorDir:     cursorDir,
		CursorReverse: cursorReverse,
		Descending:    descending,
		FilterClause:  filterClause,
		FilterParams:  filterParams,
	}

	result, err := s.stores.Event.ListEventsPage(ctx, req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list events: %v", err)
	}

	// Build response
	response := &campaignv1.ListEventsResponse{
		Events:    make([]*campaignv1.Event, 0, len(result.Events)),
		TotalSize: int32(result.TotalCount),
	}

	for _, evt := range result.Events {
		response.Events = append(response.Events, eventToProto(evt))
	}

	// Generate next/previous page tokens
	if len(result.Events) > 0 {
		if result.HasNextPage {
			lastSeq := result.Events[len(result.Events)-1].Seq
			nextCursor := cursor.NewNextPageCursor(lastSeq, descending, filterStr, orderBy)
			token, err := cursor.Encode(nextCursor)
			if err == nil {
				response.NextPageToken = token
			}
		}
		if result.HasPrevPage {
			firstSeq := result.Events[0].Seq
			prevCursor := cursor.NewPrevPageCursor(firstSeq, descending, filterStr, orderBy)
			token, err := cursor.Encode(prevCursor)
			if err == nil {
				response.PreviousPageToken = token
			}
		}
	}

	return response, nil
}

// eventToProto converts a domain event to a proto Event message.
func eventToProto(evt event.Event) *campaignv1.Event {
	return &campaignv1.Event{
		CampaignId:   evt.CampaignID,
		Seq:          evt.Seq,
		Hash:         evt.Hash,
		Ts:           timestamppb.New(evt.Timestamp),
		Type:         string(evt.Type),
		SessionId:    evt.SessionID,
		RequestId:    evt.RequestID,
		InvocationId: evt.InvocationID,
		ActorType:    string(evt.ActorType),
		ActorId:      evt.ActorID,
		EntityType:   evt.EntityType,
		EntityId:     evt.EntityID,
		PayloadJson:  evt.PayloadJSON,
	}
}
