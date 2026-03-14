package app

import (
	"context"
	"fmt"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	workerdomain "github.com/louisbranch/fracturing.space/internal/services/worker/domain"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type authIntegrationOutboxClient interface {
	LeaseIntegrationOutboxEvents(ctx context.Context, in *authv1.LeaseIntegrationOutboxEventsRequest, opts ...grpc.CallOption) (*authv1.LeaseIntegrationOutboxEventsResponse, error)
	AckIntegrationOutboxEvent(ctx context.Context, in *authv1.AckIntegrationOutboxEventRequest, opts ...grpc.CallOption) (*authv1.AckIntegrationOutboxEventResponse, error)
}

type gameIntegrationOutboxClient interface {
	LeaseIntegrationOutboxEvents(ctx context.Context, in *gamev1.LeaseIntegrationOutboxEventsRequest, opts ...grpc.CallOption) (*gamev1.LeaseIntegrationOutboxEventsResponse, error)
	AckIntegrationOutboxEvent(ctx context.Context, in *gamev1.AckIntegrationOutboxEventRequest, opts ...grpc.CallOption) (*gamev1.AckIntegrationOutboxEventResponse, error)
}

type authOutboxClientAdapter struct {
	client authIntegrationOutboxClient
}

func newAuthOutboxClientAdapter(client authIntegrationOutboxClient) OutboxClient {
	if client == nil {
		return nil
	}
	return authOutboxClientAdapter{client: client}
}

func (a authOutboxClientAdapter) Lease(ctx context.Context, req LeaseRequest) ([]workerdomain.OutboxEvent, error) {
	resp, err := a.client.LeaseIntegrationOutboxEvents(serviceContext(ctx), &authv1.LeaseIntegrationOutboxEventsRequest{
		Consumer:   req.Consumer,
		Limit:      int32(req.Limit),
		LeaseTtlMs: int64(req.LeaseTTL / time.Millisecond),
		Now:        timestamppb.New(req.Now),
	})
	if err != nil {
		return nil, err
	}
	return authEventsToOutboxEvents(resp.GetEvents()), nil
}

func (a authOutboxClientAdapter) Ack(ctx context.Context, req AckRequest) error {
	outcome, err := authAckOutcome(req.Outcome)
	if err != nil {
		return err
	}
	ackReq := &authv1.AckIntegrationOutboxEventRequest{
		EventId:   req.EventID,
		Consumer:  req.Consumer,
		Outcome:   outcome,
		LastError: req.LastError,
	}
	if req.Outcome == workerdomain.AckOutcomeRetry {
		ackReq.NextAttemptAt = timestamppb.New(req.NextAttemptAt)
	}
	_, err = a.client.AckIntegrationOutboxEvent(serviceContext(ctx), ackReq)
	return err
}

type gameOutboxClientAdapter struct {
	client gameIntegrationOutboxClient
}

func newGameOutboxClientAdapter(client gameIntegrationOutboxClient) OutboxClient {
	if client == nil {
		return nil
	}
	return gameOutboxClientAdapter{client: client}
}

func (a gameOutboxClientAdapter) Lease(ctx context.Context, req LeaseRequest) ([]workerdomain.OutboxEvent, error) {
	resp, err := a.client.LeaseIntegrationOutboxEvents(serviceContext(ctx), &gamev1.LeaseIntegrationOutboxEventsRequest{
		Consumer:   req.Consumer,
		Limit:      int32(req.Limit),
		LeaseTtlMs: int64(req.LeaseTTL / time.Millisecond),
		Now:        timestamppb.New(req.Now),
	})
	if err != nil {
		return nil, err
	}
	return gameEventsToOutboxEvents(resp.GetEvents()), nil
}

func (a gameOutboxClientAdapter) Ack(ctx context.Context, req AckRequest) error {
	outcome, err := gameAckOutcome(req.Outcome)
	if err != nil {
		return err
	}
	ackReq := &gamev1.AckIntegrationOutboxEventRequest{
		EventId:   req.EventID,
		Consumer:  req.Consumer,
		Outcome:   outcome,
		LastError: req.LastError,
	}
	if req.Outcome == workerdomain.AckOutcomeRetry {
		ackReq.NextAttemptAt = timestamppb.New(req.NextAttemptAt)
	}
	_, err = a.client.AckIntegrationOutboxEvent(serviceContext(ctx), ackReq)
	return err
}

func serviceContext(ctx context.Context) context.Context {
	return grpcauthctx.WithServiceID(ctx, serviceaddr.ServiceWorker)
}

func authEventsToOutboxEvents(events []*authv1.IntegrationOutboxEvent) []workerdomain.OutboxEvent {
	out := make([]workerdomain.OutboxEvent, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		out = append(out, event)
	}
	return out
}

func gameEventsToOutboxEvents(events []*gamev1.IntegrationOutboxEvent) []workerdomain.OutboxEvent {
	out := make([]workerdomain.OutboxEvent, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		out = append(out, event)
	}
	return out
}

func authAckOutcome(outcome workerdomain.AckOutcome) (authv1.IntegrationOutboxAckOutcome, error) {
	switch outcome {
	case workerdomain.AckOutcomeSucceeded:
		return authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED, nil
	case workerdomain.AckOutcomeRetry:
		return authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY, nil
	case workerdomain.AckOutcomeDead:
		return authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD, nil
	default:
		return authv1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_UNSPECIFIED, fmt.Errorf("unknown ack outcome %q", outcome.String())
	}
}

func gameAckOutcome(outcome workerdomain.AckOutcome) (gamev1.IntegrationOutboxAckOutcome, error) {
	switch outcome {
	case workerdomain.AckOutcomeSucceeded:
		return gamev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_SUCCEEDED, nil
	case workerdomain.AckOutcomeRetry:
		return gamev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_RETRY, nil
	case workerdomain.AckOutcomeDead:
		return gamev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_DEAD, nil
	default:
		return gamev1.IntegrationOutboxAckOutcome_INTEGRATION_OUTBOX_ACK_OUTCOME_UNSPECIFIED, fmt.Errorf("unknown ack outcome %q", outcome.String())
	}
}
