package game

import (
	"context"
	"strconv"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	"github.com/louisbranch/fracturing.space/internal/services/game/core/filter"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultListEventsPageSize         = 50
	maxListEventsPageSize             = 200
	defaultCampaignUpdatePollInterval = time.Second
	minCampaignUpdatePollInterval     = 50 * time.Millisecond
)

var defaultCampaignProjectionScopes = []string{
	"campaign_summary",
	"campaign_participants",
	"campaign_sessions",
	"campaign_characters",
	"campaign_invites",
}

// EventService implements the game.v1.EventService gRPC API.
type EventService struct {
	campaignv1.UnimplementedEventServiceServer
	stores Stores
}

type normalizedListEventsRequest struct {
	campaignID      string
	pageSize        int
	orderBy         string
	descending      bool
	filterStr       string
	afterSeq        uint64
	paginationScope string
	filterClause    string
	filterParams    []any
	cursorSeq       uint64
	cursorDir       string
	cursorReverse   bool
}

type normalizedSubscribeCampaignUpdatesRequest struct {
	campaignID            string
	afterSeq              uint64
	includeEventCommitted bool
	includeProjection     bool
	projectionScopes      map[string]struct{}
	pollInterval          time.Duration
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

	stored, err := newEventApplication(s).AppendEvent(ctx, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.AppendEventResponse{Event: eventToProto(stored)}, nil
}

// ListEvents returns a paginated, filtered, and sorted list of events for a campaign.
func (s *EventService) ListEvents(ctx context.Context, in *campaignv1.ListEventsRequest) (*campaignv1.ListEventsResponse, error) {
	normalized, err := normalizeListEventsRequest(in)
	if err != nil {
		return nil, err
	}

	if err := requireReadPolicy(ctx, s.stores, storage.CampaignRecord{ID: normalized.campaignID}); err != nil {
		return nil, err
	}

	// Build request
	storeReq := storage.ListEventsPageRequest{
		CampaignID:    normalized.campaignID,
		AfterSeq:      normalized.afterSeq,
		PageSize:      normalized.pageSize,
		CursorSeq:     normalized.cursorSeq,
		CursorDir:     normalized.cursorDir,
		CursorReverse: normalized.cursorReverse,
		Descending:    normalized.descending,
		FilterClause:  normalized.filterClause,
		FilterParams:  normalized.filterParams,
	}

	result, err := s.stores.Event.ListEventsPage(ctx, storeReq)
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
			nextCursor := pagination.NewNextPageCursor(
				[]pagination.CursorValue{pagination.UintValue("seq", lastSeq)},
				normalized.descending,
				normalized.paginationScope,
				normalized.orderBy,
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
				normalized.descending,
				normalized.paginationScope,
				normalized.orderBy,
			)
			token, err := pagination.Encode(prevCursor)
			if err == nil {
				response.PreviousPageToken = token
			}
		}
	}

	return response, nil
}

// SubscribeCampaignUpdates streams campaign update envelopes for realtime clients.
func (s *EventService) SubscribeCampaignUpdates(in *campaignv1.SubscribeCampaignUpdatesRequest, stream grpc.ServerStreamingServer[campaignv1.CampaignUpdate]) error {
	normalized, err := normalizeSubscribeCampaignUpdatesRequest(in)
	if err != nil {
		return err
	}
	if s == nil || s.stores.Event == nil {
		return status.Error(codes.Internal, "event store is not configured")
	}
	if stream == nil {
		return status.Error(codes.Internal, "stream is required")
	}

	ctx := stream.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := requireReadPolicy(ctx, s.stores, storage.CampaignRecord{ID: normalized.campaignID}); err != nil {
		return err
	}
	lastSeq := normalized.afterSeq

	sendAvailable := func() error {
		events, err := s.stores.Event.ListEvents(ctx, normalized.campaignID, lastSeq, maxListEventsPageSize)
		if err != nil {
			return status.Errorf(codes.Internal, "list events: %v", err)
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

	var filterClause string
	var filterParams []any
	if filterStr != "" {
		cond, err := filter.ParseEventFilter(filterStr)
		if err != nil {
			return normalizedListEventsRequest{}, status.Errorf(codes.InvalidArgument, "invalid filter: %v", err)
		}
		filterClause = cond.Clause
		filterParams = cond.Params
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
		afterSeq:        afterSeq,
		paginationScope: paginationScope,
		filterClause:    filterClause,
		filterParams:    filterParams,
		cursorSeq:       cursorSeq,
		cursorDir:       cursorDir,
		cursorReverse:   cursorReverse,
	}, nil
}

func normalizeSubscribeCampaignUpdatesRequest(in *campaignv1.SubscribeCampaignUpdatesRequest) (normalizedSubscribeCampaignUpdatesRequest, error) {
	if in == nil {
		return normalizedSubscribeCampaignUpdatesRequest{}, status.Error(codes.InvalidArgument, "request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return normalizedSubscribeCampaignUpdatesRequest{}, status.Error(codes.InvalidArgument, "campaign_id is required")
	}

	includeEventCommitted := true
	includeProjection := true
	if len(in.GetKinds()) > 0 {
		includeEventCommitted = false
		includeProjection = false
		for _, kind := range in.GetKinds() {
			switch kind {
			case campaignv1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_EVENT_COMMITTED:
				includeEventCommitted = true
			case campaignv1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_PROJECTION_APPLIED:
				includeProjection = true
			case campaignv1.CampaignUpdateKind_CAMPAIGN_UPDATE_KIND_UNSPECIFIED:
				// ignore
			default:
				return normalizedSubscribeCampaignUpdatesRequest{}, status.Errorf(codes.InvalidArgument, "invalid update kind: %d", kind)
			}
		}
		if !includeEventCommitted && !includeProjection {
			return normalizedSubscribeCampaignUpdatesRequest{}, status.Error(codes.InvalidArgument, "at least one update kind is required")
		}
	}

	pollInterval := defaultCampaignUpdatePollInterval
	if in.GetPollIntervalMs() > 0 {
		pollInterval = time.Duration(in.GetPollIntervalMs()) * time.Millisecond
	}
	if pollInterval < minCampaignUpdatePollInterval {
		pollInterval = minCampaignUpdatePollInterval
	}

	var projectionScopes map[string]struct{}
	if len(in.GetProjectionScopes()) > 0 {
		projectionScopes = make(map[string]struct{}, len(in.GetProjectionScopes()))
		for _, scope := range in.GetProjectionScopes() {
			scope = strings.TrimSpace(scope)
			if scope == "" {
				continue
			}
			projectionScopes[scope] = struct{}{}
		}
	}

	return normalizedSubscribeCampaignUpdatesRequest{
		campaignID:            campaignID,
		afterSeq:              in.GetAfterSeq(),
		includeEventCommitted: includeEventCommitted,
		includeProjection:     includeProjection,
		projectionScopes:      projectionScopes,
		pollInterval:          pollInterval,
	}, nil
}

// eventToProto converts a domain event to a proto Event message.
func eventToProto(evt event.Event) *campaignv1.Event {
	return &campaignv1.Event{
		CampaignId:    evt.CampaignID,
		Seq:           evt.Seq,
		Hash:          evt.Hash,
		Ts:            timestamppb.New(evt.Timestamp),
		Type:          string(evt.Type),
		SystemId:      evt.SystemID,
		SystemVersion: evt.SystemVersion,
		SessionId:     evt.SessionID,
		RequestId:     evt.RequestID,
		InvocationId:  evt.InvocationID,
		ActorType:     string(evt.ActorType),
		ActorId:       evt.ActorID,
		EntityType:    evt.EntityType,
		EntityId:      evt.EntityID,
		PayloadJson:   evt.PayloadJSON,
	}
}

func campaignUpdateEventCommitted(evt event.Event) *campaignv1.CampaignUpdate {
	return &campaignv1.CampaignUpdate{
		CampaignId: evt.CampaignID,
		Seq:        evt.Seq,
		EventType:  string(evt.Type),
		EventTime:  timestamppb.New(evt.Timestamp),
		EntityType: evt.EntityType,
		EntityId:   evt.EntityID,
		Update: &campaignv1.CampaignUpdate_EventCommitted{
			EventCommitted: &campaignv1.EventCommitted{},
		},
	}
}

func campaignUpdateProjectionApplied(evt event.Event, scopes []string) *campaignv1.CampaignUpdate {
	return &campaignv1.CampaignUpdate{
		CampaignId: evt.CampaignID,
		Seq:        evt.Seq,
		EventType:  string(evt.Type),
		EventTime:  timestamppb.New(evt.Timestamp),
		EntityType: evt.EntityType,
		EntityId:   evt.EntityID,
		Update: &campaignv1.CampaignUpdate_ProjectionApplied{
			ProjectionApplied: &campaignv1.ProjectionApplied{
				SourceSeq: evt.Seq,
				Scopes:    append([]string(nil), scopes...),
			},
		},
	}
}

func projectionScopesForEventType(eventType string) []string {
	eventType = strings.TrimSpace(eventType)
	switch {
	case strings.HasPrefix(eventType, "campaign."):
		return []string{"campaign_summary"}
	case eventType == "seat.reassigned":
		return []string{"campaign_participants", "campaign_summary"}
	case strings.HasPrefix(eventType, "participant."):
		return []string{"campaign_participants", "campaign_summary"}
	case strings.HasPrefix(eventType, "session."):
		return []string{"campaign_sessions"}
	case strings.HasPrefix(eventType, "character."):
		return []string{"campaign_characters", "campaign_summary"}
	case strings.HasPrefix(eventType, "invite."):
		return []string{"campaign_invites"}
	default:
		return append([]string(nil), defaultCampaignProjectionScopes...)
	}
}

func hasProjectionScopeIntersection(scopes []string, filter map[string]struct{}) bool {
	if len(scopes) == 0 {
		return false
	}
	if len(filter) == 0 {
		return true
	}
	for _, scope := range scopes {
		if _, ok := filter[scope]; ok {
			return true
		}
	}
	return false
}
