package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	workerdomain "github.com/louisbranch/fracturing.space/internal/services/worker/domain"
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
	Handle(ctx context.Context, event workerdomain.OutboxEvent) error
}

// EventHandlerFunc adapts a function to EventHandler.
type EventHandlerFunc func(ctx context.Context, event workerdomain.OutboxEvent) error

// Handle runs the adapted function.
func (fn EventHandlerFunc) Handle(ctx context.Context, event workerdomain.OutboxEvent) error {
	return fn(ctx, event)
}

// LeaseRequest describes one outbox lease pass.
type LeaseRequest struct {
	Consumer string
	Limit    int
	LeaseTTL time.Duration
	Now      time.Time
}

// AckRequest acknowledges one leased event outcome.
type AckRequest struct {
	EventID       string
	Consumer      string
	Outcome       workerdomain.AckOutcome
	NextAttemptAt time.Time
	LastError     string
}

// OutboxClient leases and acknowledges one authoritative service outbox.
type OutboxClient interface {
	Lease(ctx context.Context, req LeaseRequest) ([]workerdomain.OutboxEvent, error)
	Ack(ctx context.Context, req AckRequest) error
}

// Attempt captures one worker processing outcome for observability persistence.
type Attempt struct {
	EventID      string
	EventType    string
	Outcome      workerdomain.AckOutcome
	AttemptCount int32
	Error        string
	CreatedAt    time.Time
}

// AttemptRecorder persists worker attempt outcomes.
type AttemptRecorder interface {
	RecordAttempt(ctx context.Context, attempt Attempt) error
}

// Server leases outbox work from one source and dispatches to handlers.
type Server struct {
	name     string
	client   OutboxClient
	recorder AttemptRecorder
	handlers map[string]EventHandler
	config   Config
	clock    func() time.Time
}

// New builds a worker processing server for one outbox source.
func New(name string, client OutboxClient, recorder AttemptRecorder, handlers map[string]EventHandler, cfg Config, clock func() time.Time) *Server {
	if handlers == nil {
		handlers = map[string]EventHandler{}
	}
	if clock == nil {
		clock = time.Now
	}
	return &Server{
		name:     name,
		client:   client,
		recorder: recorder,
		handlers: handlers,
		config:   cfg.normalized(),
		clock:    clock,
	}
}

// Run starts the worker lease/poll loop until context cancellation.
func (s *Server) Run(ctx context.Context) error {
	if s == nil {
		return errors.New("worker server is nil")
	}
	if s.client == nil {
		return fmt.Errorf("%s outbox client is not configured", s.name)
	}
	if ctx == nil {
		ctx = context.Background()
	}

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
	events, err := s.client.Lease(ctx, LeaseRequest{
		Consumer: s.config.Consumer,
		Limit:    s.config.MaxAttempts * 4,
		LeaseTTL: s.config.LeaseTTL,
		Now:      now,
	})
	if err != nil {
		log.Printf("worker: lease %s integration outbox events: %v", s.name, err)
		return
	}
	for _, outboxEvent := range events {
		s.processEvent(ctx, outboxEvent)
	}
}

func (s *Server) processEvent(ctx context.Context, outboxEvent workerdomain.OutboxEvent) {
	if outboxEvent == nil {
		return
	}
	now := s.clock().UTC()
	handler, ok := s.handlers[outboxEvent.GetEventType()]
	if !ok {
		s.ackWithObservation(
			ctx,
			outboxEvent,
			workerdomain.AckOutcomeDead,
			now,
			time.Time{},
			fmt.Sprintf("no handler registered for event type %q", outboxEvent.GetEventType()),
		)
		return
	}

	err := handler.Handle(ctx, outboxEvent)
	if err == nil {
		s.ackWithObservation(ctx, outboxEvent, workerdomain.AckOutcomeSucceeded, now, time.Time{}, "")
		return
	}

	if workerdomain.IsPermanent(err) || int(outboxEvent.GetAttemptCount())+1 >= s.config.MaxAttempts {
		s.ackWithObservation(ctx, outboxEvent, workerdomain.AckOutcomeDead, now, time.Time{}, err.Error())
		return
	}

	nextAttempt := now.Add(s.retryDelay(outboxEvent.GetAttemptCount()))
	s.ackWithObservation(ctx, outboxEvent, workerdomain.AckOutcomeRetry, now, nextAttempt, err.Error())
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
			delay *= 2
		}
	}
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

func (s *Server) ack(ctx context.Context, outboxEvent workerdomain.OutboxEvent, outcome workerdomain.AckOutcome, now time.Time, nextAttemptAt time.Time, lastError string) error {
	if outboxEvent == nil {
		return nil
	}
	if err := s.client.Ack(ctx, AckRequest{
		EventID:       outboxEvent.GetId(),
		Consumer:      s.config.Consumer,
		Outcome:       outcome,
		NextAttemptAt: nextAttemptAt,
		LastError:     lastError,
	}); err != nil {
		return err
	}

	if s.recorder != nil {
		_ = s.recorder.RecordAttempt(ctx, Attempt{
			EventID:      outboxEvent.GetId(),
			EventType:    outboxEvent.GetEventType(),
			Outcome:      outcome,
			AttemptCount: outboxEvent.GetAttemptCount() + 1,
			Error:        lastError,
			CreatedAt:    now,
		})
	}
	return nil
}

func (s *Server) ackWithObservation(ctx context.Context, outboxEvent workerdomain.OutboxEvent, outcome workerdomain.AckOutcome, now time.Time, nextAttemptAt time.Time, lastError string) {
	if err := s.ack(ctx, outboxEvent, outcome, now, nextAttemptAt, lastError); err != nil {
		log.Printf("worker: ack %s integration outbox event id=%s outcome=%s: %v", s.name, outboxEvent.GetId(), outcome.String(), err)
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

type parallelLoop struct {
	loops []workerLoop
}

func (l parallelLoop) Run(ctx context.Context) error {
	if len(l.loops) == 0 {
		return nil
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	errCh := make(chan error, len(l.loops))
	activeLoops := 0
	for _, loop := range l.loops {
		if loop == nil {
			continue
		}
		activeLoops++
		go func(loop workerLoop) {
			errCh <- loop.Run(runCtx)
		}(loop)
	}

	for i := 0; i < activeLoops; i++ {
		err := <-errCh
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			cancel()
			return err
		}
	}
	return nil
}
