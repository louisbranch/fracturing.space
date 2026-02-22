package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	"github.com/louisbranch/fracturing.space/internal/services/auth/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultIntegrationOutboxLeaseLimit = 50
	defaultIntegrationOutboxLeaseTTL   = 30 * time.Second
)

func (s *AuthService) LeaseIntegrationOutboxEvents(ctx context.Context, in *authv1.LeaseIntegrationOutboxEventsRequest) (*authv1.LeaseIntegrationOutboxEventsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "lease integration outbox events request is required")
	}

	outboxStore, ok := s.store.(storage.IntegrationOutboxStore)
	if s == nil || s.store == nil || !ok {
		return nil, status.Error(codes.Internal, "integration outbox store is not configured")
	}

	consumer := strings.TrimSpace(in.GetConsumer())
	if consumer == "" {
		return nil, status.Error(codes.InvalidArgument, "consumer is required")
	}

	limit := int(in.GetLimit())
	if limit <= 0 {
		limit = defaultIntegrationOutboxLeaseLimit
	}
	leaseTTL := time.Duration(in.GetLeaseTtlMs()) * time.Millisecond
	if leaseTTL <= 0 {
		leaseTTL = defaultIntegrationOutboxLeaseTTL
	}
	now := nowUTC(s.clock)
	if ts := in.GetNow(); ts != nil && ts.IsValid() {
		now = ts.AsTime().UTC()
	}

	leased, err := outboxStore.LeaseIntegrationOutboxEvents(ctx, consumer, limit, now, leaseTTL)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "lease integration outbox events: %v", err)
	}

	resp := &authv1.LeaseIntegrationOutboxEventsResponse{
		Events: make([]*authv1.IntegrationOutboxEvent, 0, len(leased)),
	}
	for _, event := range leased {
		resp.Events = append(resp.Events, integrationOutboxEventToProto(event))
	}
	return resp, nil
}

func (s *AuthService) AckIntegrationOutboxEvent(ctx context.Context, in *authv1.AckIntegrationOutboxEventRequest) (*authv1.AckIntegrationOutboxEventResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "ack integration outbox event request is required")
	}

	outboxStore, ok := s.store.(storage.IntegrationOutboxStore)
	if s == nil || s.store == nil || !ok {
		return nil, status.Error(codes.Internal, "integration outbox store is not configured")
	}

	eventID := strings.TrimSpace(in.GetEventId())
	if eventID == "" {
		return nil, status.Error(codes.InvalidArgument, "event id is required")
	}
	consumer := strings.TrimSpace(in.GetConsumer())
	if consumer == "" {
		return nil, status.Error(codes.InvalidArgument, "consumer is required")
	}

	now := nowUTC(s.clock)
	var err error
	switch in.GetOutcome() {
	case authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED:
		processedAt := now
		if ts := in.GetProcessedAt(); ts != nil && ts.IsValid() {
			processedAt = ts.AsTime().UTC()
		}
		err = outboxStore.MarkIntegrationOutboxSucceeded(ctx, eventID, consumer, processedAt)
	case authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY:
		nextAttemptAt := in.GetNextAttemptAt()
		if nextAttemptAt == nil || !nextAttemptAt.IsValid() {
			return nil, status.Error(codes.InvalidArgument, "next attempt at is required for retry outcome")
		}
		err = outboxStore.MarkIntegrationOutboxRetry(ctx, eventID, consumer, nextAttemptAt.AsTime().UTC(), in.GetLastError())
	case authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD:
		processedAt := now
		if ts := in.GetProcessedAt(); ts != nil && ts.IsValid() {
			processedAt = ts.AsTime().UTC()
		}
		err = outboxStore.MarkIntegrationOutboxDead(ctx, eventID, consumer, in.GetLastError(), processedAt)
	default:
		return nil, status.Error(codes.InvalidArgument, "ack outcome is required")
	}

	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, storage.ErrNotFound.Error())
		}
		return nil, status.Errorf(codes.Internal, "ack integration outbox event: %v", err)
	}
	return &authv1.AckIntegrationOutboxEventResponse{}, nil
}

func integrationOutboxEventToProto(event storage.IntegrationOutboxEvent) *authv1.IntegrationOutboxEvent {
	result := &authv1.IntegrationOutboxEvent{
		Id:            event.ID,
		EventType:     event.EventType,
		PayloadJson:   event.PayloadJSON,
		DedupeKey:     event.DedupeKey,
		Status:        event.Status,
		AttemptCount:  int32(event.AttemptCount),
		NextAttemptAt: timestamppb.New(event.NextAttemptAt),
		LeaseOwner:    event.LeaseOwner,
		LastError:     event.LastError,
		CreatedAt:     timestamppb.New(event.CreatedAt),
		UpdatedAt:     timestamppb.New(event.UpdatedAt),
	}
	if event.LeaseExpiresAt != nil {
		result.LeaseExpiresAt = timestamppb.New(*event.LeaseExpiresAt)
	}
	if event.ProcessedAt != nil {
		result.ProcessedAt = timestamppb.New(*event.ProcessedAt)
	}
	return result
}

func nowUTC(clock func() time.Time) time.Time {
	if clock == nil {
		return time.Now().UTC()
	}
	return clock().UTC()
}
