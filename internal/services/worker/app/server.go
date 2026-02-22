package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	workerdomain "github.com/louisbranch/fracturing.space/internal/services/worker/domain"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultConsumer      = "worker"
	defaultPollInterval  = 2 * time.Second
	defaultLeaseTTL      = 30 * time.Second
	defaultMaxAttempts   = 8
	defaultRetryBackoff  = 5 * time.Second
	defaultRetryMaxDelay = 5 * time.Minute
	defaultAckBackoff    = 10 * time.Millisecond
)

// Config controls worker poll and retry behavior.
type Config struct {
	Consumer      string
	PollInterval  time.Duration
	LeaseTTL      time.Duration
	MaxAttempts   int
	RetryBackoff  time.Duration
	RetryMaxDelay time.Duration
	AckBackoff    time.Duration
}

func (c Config) normalized() Config {
	result := c
	if result.Consumer == "" {
		result.Consumer = defaultConsumer
	}
	if result.PollInterval <= 0 {
		result.PollInterval = defaultPollInterval
	}
	if result.LeaseTTL <= 0 {
		result.LeaseTTL = defaultLeaseTTL
	}
	if result.MaxAttempts <= 0 {
		result.MaxAttempts = defaultMaxAttempts
	}
	if result.RetryBackoff <= 0 {
		result.RetryBackoff = defaultRetryBackoff
	}
	if result.RetryMaxDelay <= 0 {
		result.RetryMaxDelay = defaultRetryMaxDelay
	}
	if result.AckBackoff <= 0 {
		result.AckBackoff = defaultAckBackoff
	}
	return result
}

// EventHandler processes one leased integration outbox event.
type EventHandler interface {
	Handle(ctx context.Context, event *authv1.IntegrationOutboxEvent) error
}

// EventHandlerFunc adapts a function to EventHandler.
type EventHandlerFunc func(ctx context.Context, event *authv1.IntegrationOutboxEvent) error

// Handle runs the adapted function.
func (fn EventHandlerFunc) Handle(ctx context.Context, event *authv1.IntegrationOutboxEvent) error {
	return fn(ctx, event)
}

type authOutboxClient interface {
	LeaseIntegrationOutboxEvents(ctx context.Context, in *authv1.LeaseIntegrationOutboxEventsRequest, opts ...grpc.CallOption) (*authv1.LeaseIntegrationOutboxEventsResponse, error)
	AckIntegrationOutboxEvent(ctx context.Context, in *authv1.AckIntegrationOutboxEventRequest, opts ...grpc.CallOption) (*authv1.AckIntegrationOutboxEventResponse, error)
}

// Attempt captures one worker processing outcome for observability persistence.
type Attempt struct {
	EventID      string
	EventType    string
	Outcome      authv1.IntegrationOutboxAckOutcome
	AttemptCount int32
	Error        string
	CreatedAt    time.Time
}

// AttemptRecorder persists worker attempt outcomes.
type AttemptRecorder interface {
	RecordAttempt(ctx context.Context, attempt Attempt) error
}

// Server leases auth outbox work and dispatches to handlers.
type Server struct {
	authClient authOutboxClient
	recorder   AttemptRecorder
	handlers   map[string]EventHandler
	config     Config
	clock      func() time.Time
}

// New builds a worker processing server.
func New(authClient authOutboxClient, recorder AttemptRecorder, handlers map[string]EventHandler, cfg Config, clock func() time.Time) *Server {
	if handlers == nil {
		handlers = map[string]EventHandler{}
	}
	if clock == nil {
		clock = time.Now
	}
	return &Server{
		authClient: authClient,
		recorder:   recorder,
		handlers:   handlers,
		config:     cfg.normalized(),
		clock:      clock,
	}
}

// Run starts the worker lease/poll loop until context cancellation.
func (s *Server) Run(ctx context.Context) error {
	if s == nil {
		return errors.New("worker server is nil")
	}
	if s.authClient == nil {
		return errors.New("auth outbox client is not configured")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// Run once immediately so startup does not wait for first ticker interval.
	s.runOnce(ctx)

	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.runOnce(ctx)
		}
	}
}

func (s *Server) runOnce(ctx context.Context) {
	now := s.clock().UTC()
	resp, err := s.authClient.LeaseIntegrationOutboxEvents(ctx, &authv1.LeaseIntegrationOutboxEventsRequest{
		Consumer:   s.config.Consumer,
		Limit:      int32(s.config.MaxAttempts * 4),
		LeaseTtlMs: int64(s.config.LeaseTTL / time.Millisecond),
		Now:        timestamppb.New(now),
	})
	if err != nil {
		log.Printf("worker: lease integration outbox events: %v", err)
		return
	}
	if resp == nil {
		return
	}

	for _, event := range resp.GetEvents() {
		s.processEvent(ctx, event)
	}
}

func (s *Server) processEvent(ctx context.Context, event *authv1.IntegrationOutboxEvent) {
	if event == nil {
		return
	}
	now := s.clock().UTC()
	handler, ok := s.handlers[event.GetEventType()]
	if !ok {
		s.ackWithObservation(
			ctx,
			event,
			authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD,
			now,
			time.Time{},
			fmt.Sprintf("no handler registered for event type %q", event.GetEventType()),
		)
		return
	}

	err := handler.Handle(ctx, event)
	if err == nil {
		s.ackWithObservation(
			ctx,
			event,
			authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED,
			now,
			time.Time{},
			"",
		)
		return
	}

	if workerdomain.IsPermanent(err) || int(event.GetAttemptCount())+1 >= s.config.MaxAttempts {
		s.ackWithObservation(
			ctx,
			event,
			authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD,
			now,
			time.Time{},
			err.Error(),
		)
		return
	}

	nextAttempt := now.Add(s.retryDelay(event.GetAttemptCount()))
	s.ackWithObservation(
		ctx,
		event,
		authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY,
		now,
		nextAttempt,
		err.Error(),
	)
}

func (s *Server) retryDelay(attemptCount int32) time.Duration {
	delay := s.config.RetryBackoff
	if delay <= 0 {
		delay = defaultRetryBackoff
	}
	maxDelay := s.config.RetryMaxDelay
	if maxDelay <= 0 {
		maxDelay = defaultRetryMaxDelay
	}
	if attemptCount > 0 {
		for i := int32(0); i < attemptCount; i++ {
			if delay >= maxDelay {
				return maxDelay
			}
			delay = delay * 2
		}
	}
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

func (s *Server) ack(ctx context.Context, event *authv1.IntegrationOutboxEvent, outcome authv1.IntegrationOutboxAckOutcome, now time.Time, nextAttemptAt time.Time, lastError string) error {
	if event == nil {
		return nil
	}
	req := &authv1.AckIntegrationOutboxEventRequest{
		EventId:   event.GetId(),
		Consumer:  s.config.Consumer,
		Outcome:   outcome,
		LastError: lastError,
	}
	switch outcome {
	case authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY:
		req.NextAttemptAt = timestamppb.New(nextAttemptAt)
	default:
		req.ProcessedAt = timestamppb.New(now)
	}
	if _, err := s.authClient.AckIntegrationOutboxEvent(ctx, req); err != nil {
		return err
	}

	if s.recorder != nil {
		_ = s.recorder.RecordAttempt(ctx, Attempt{
			EventID:      event.GetId(),
			EventType:    event.GetEventType(),
			Outcome:      outcome,
			AttemptCount: event.GetAttemptCount() + 1,
			Error:        lastError,
			CreatedAt:    now,
		})
	}
	return nil
}

func (s *Server) ackWithObservation(ctx context.Context, event *authv1.IntegrationOutboxEvent, outcome authv1.IntegrationOutboxAckOutcome, now time.Time, nextAttemptAt time.Time, lastError string) {
	if err := s.ack(ctx, event, outcome, now, nextAttemptAt, lastError); err != nil {
		log.Printf("worker: ack integration outbox event id=%s outcome=%s: %v", event.GetId(), outcome.String(), err)
		s.sleepAckBackoff(ctx)
	}
}

func (s *Server) sleepAckBackoff(ctx context.Context) {
	delay := s.config.AckBackoff
	if delay <= 0 {
		return
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
