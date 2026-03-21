package invite

import (
	"context"
	"time"

	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Service) LeaseIntegrationOutboxEvents(ctx context.Context, in *invitev1.LeaseIntegrationOutboxEventsRequest) (*invitev1.LeaseIntegrationOutboxEventsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	consumer := in.GetConsumer()
	if consumer == "" {
		return nil, status.Error(codes.InvalidArgument, "consumer is required")
	}
	limit := int(in.GetLimit())
	if limit <= 0 {
		limit = 10
	}
	leaseTTL := time.Duration(in.GetLeaseTtlMs()) * time.Millisecond
	if leaseTTL <= 0 {
		leaseTTL = 30 * time.Second
	}
	now := in.GetNow().AsTime()
	if now.IsZero() {
		now = s.clock()
	}
	events, err := s.outbox.LeaseOutboxEvents(ctx, consumer, limit, leaseTTL, now)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "lease outbox events: %v", err)
	}
	protoEvents := make([]*invitev1.IntegrationOutboxEvent, 0, len(events))
	for _, evt := range events {
		protoEvents = append(protoEvents, &invitev1.IntegrationOutboxEvent{
			Id:           evt.ID,
			EventType:    evt.EventType,
			PayloadJson:  evt.PayloadJSON,
			DedupeKey:    evt.DedupeKey,
			Status:       evt.Status,
			AttemptCount: int32(evt.AttemptCount),
			LeaseOwner:   evt.LeaseOwner,
			CreatedAt:    timestamppb.New(evt.CreatedAt),
		})
	}
	return &invitev1.LeaseIntegrationOutboxEventsResponse{Events: protoEvents}, nil
}

func (s *Service) AckIntegrationOutboxEvent(ctx context.Context, in *invitev1.AckIntegrationOutboxEventRequest) (*invitev1.AckIntegrationOutboxEventResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	eventID := in.GetEventId()
	if eventID == "" {
		return nil, status.Error(codes.InvalidArgument, "event_id is required")
	}
	consumer := in.GetConsumer()
	if consumer == "" {
		return nil, status.Error(codes.InvalidArgument, "consumer is required")
	}
	var outcome string
	switch in.GetOutcome() {
	case invitev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED:
		outcome = "succeeded"
	case invitev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY:
		outcome = "retry"
	case invitev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD:
		outcome = "dead"
	default:
		return nil, status.Error(codes.InvalidArgument, "outcome is required")
	}
	now := s.clock()
	nextAttemptAt := in.GetNextAttemptAt().AsTime()
	if err := s.outbox.AckOutboxEvent(ctx, eventID, consumer, outcome, nextAttemptAt, in.GetLastError(), now); err != nil {
		return nil, status.Errorf(codes.Internal, "ack outbox event: %v", err)
	}
	return &invitev1.AckIntegrationOutboxEventResponse{}, nil
}
