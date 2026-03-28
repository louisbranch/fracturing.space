package game

import (
	"context"
	"strings"
	"time"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultIntegrationOutboxLeaseLimit = 50
	defaultIntegrationOutboxLeaseTTL   = 30 * time.Second
)

// IntegrationService exposes worker-facing game integration outbox leasing.
type IntegrationService struct {
	gamev1.UnimplementedIntegrationServiceServer
	store storage.IntegrationOutboxWorkerStore
	clock func() time.Time
}

// NewIntegrationService creates the internal game integration outbox service.
func NewIntegrationService(store storage.IntegrationOutboxWorkerStore) *IntegrationService {
	return &IntegrationService{
		store: store,
		clock: time.Now,
	}
}

func (s *IntegrationService) LeaseIntegrationOutboxEvents(ctx context.Context, in *gamev1.LeaseIntegrationOutboxEventsRequest) (*gamev1.LeaseIntegrationOutboxEventsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Lease integration outbox events request is required.")
	}

	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "Integration outbox store is not configured.")
	}

	consumer := strings.TrimSpace(in.GetConsumer())
	if consumer == "" {
		return nil, status.Error(codes.InvalidArgument, "Consumer is required.")
	}

	limit := int(in.GetLimit())
	if limit <= 0 {
		limit = defaultIntegrationOutboxLeaseLimit
	}
	leaseTTL := time.Duration(in.GetLeaseTtlMs()) * time.Millisecond
	if leaseTTL <= 0 {
		leaseTTL = defaultIntegrationOutboxLeaseTTL
	}
	now := integrationNowUTC(s.clock)
	if ts := in.GetNow(); ts != nil && ts.IsValid() {
		now = ts.AsTime().UTC()
	}

	leased, err := s.store.LeaseIntegrationOutboxEvents(ctx, consumer, limit, now, leaseTTL)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Lease integration outbox events: %v", err)
	}

	resp := &gamev1.LeaseIntegrationOutboxEventsResponse{
		Events: make([]*gamev1.IntegrationOutboxEvent, 0, len(leased)),
	}
	for _, outboxEvent := range leased {
		resp.Events = append(resp.Events, integrationOutboxEventToProto(outboxEvent))
	}
	return resp, nil
}

func (s *IntegrationService) AckIntegrationOutboxEvent(ctx context.Context, in *gamev1.AckIntegrationOutboxEventRequest) (*gamev1.AckIntegrationOutboxEventResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "Ack integration outbox event request is required.")
	}

	if s == nil || s.store == nil {
		return nil, status.Error(codes.Internal, "Integration outbox store is not configured.")
	}

	eventID := strings.TrimSpace(in.GetEventId())
	if eventID == "" {
		return nil, status.Error(codes.InvalidArgument, "Event ID is required.")
	}
	consumer := strings.TrimSpace(in.GetConsumer())
	if consumer == "" {
		return nil, status.Error(codes.InvalidArgument, "Consumer is required.")
	}

	now := integrationNowUTC(s.clock)
	var err error
	switch in.GetOutcome() {
	case gamev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED:
		err = s.store.MarkIntegrationOutboxSucceeded(ctx, eventID, consumer, now)
	case gamev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY:
		nextAttemptAt := in.GetNextAttemptAt()
		if nextAttemptAt == nil || !nextAttemptAt.IsValid() {
			return nil, status.Error(codes.InvalidArgument, "Next attempt at is required for retry outcome.")
		}
		err = s.store.MarkIntegrationOutboxRetry(ctx, eventID, consumer, nextAttemptAt.AsTime().UTC(), in.GetLastError())
	case gamev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD:
		err = s.store.MarkIntegrationOutboxDead(ctx, eventID, consumer, in.GetLastError(), now)
	default:
		return nil, status.Error(codes.InvalidArgument, "Ack outcome is required.")
	}

	if err != nil {
		return nil, grpcerror.LookupErrorContext(ctx, err, "ack integration outbox event", storage.ErrNotFound.Error())
	}
	return &gamev1.AckIntegrationOutboxEventResponse{}, nil
}

func integrationOutboxEventToProto(outboxEvent storage.IntegrationOutboxEvent) *gamev1.IntegrationOutboxEvent {
	result := &gamev1.IntegrationOutboxEvent{
		Id:            outboxEvent.ID,
		EventType:     outboxEvent.EventType,
		PayloadJson:   outboxEvent.PayloadJSON,
		DedupeKey:     outboxEvent.DedupeKey,
		Status:        outboxEvent.Status,
		AttemptCount:  int32(outboxEvent.AttemptCount),
		NextAttemptAt: timestamppb.New(outboxEvent.NextAttemptAt),
		LeaseOwner:    outboxEvent.LeaseOwner,
		LastError:     outboxEvent.LastError,
		CreatedAt:     timestamppb.New(outboxEvent.CreatedAt),
		UpdatedAt:     timestamppb.New(outboxEvent.UpdatedAt),
	}
	if outboxEvent.LeaseExpiresAt != nil {
		result.LeaseExpiresAt = timestamppb.New(*outboxEvent.LeaseExpiresAt)
	}
	if outboxEvent.ProcessedAt != nil {
		result.ProcessedAt = timestamppb.New(*outboxEvent.ProcessedAt)
	}
	return result
}

func integrationNowUTC(clock func() time.Time) time.Time {
	if clock == nil {
		return time.Now().UTC()
	}
	return clock().UTC()
}
