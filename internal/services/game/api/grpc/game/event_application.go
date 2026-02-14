package game

import (
	"context"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type eventApplication struct {
	stores Stores
}

func newEventApplication(service *EventService) eventApplication {
	return eventApplication{stores: service.stores}
}

func (a eventApplication) AppendEvent(ctx context.Context, in *campaignv1.AppendEventRequest) (event.Event, error) {
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
		return event.Event{}, status.Error(codes.InvalidArgument, err.Error())
	}

	stored, err := a.stores.Event.AppendEvent(ctx, input)
	if err != nil {
		return event.Event{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	return stored, nil
}
