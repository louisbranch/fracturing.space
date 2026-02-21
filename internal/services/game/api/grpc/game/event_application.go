package game

import (
	"context"
	"errors"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
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
	input := event.Event{
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
	}
	if a.stores.Domain != nil {
		if cmdType, ok := domainCommandTypeForEvent(input.Type); ok {
			actorType := command.ActorTypeSystem
			switch input.ActorType {
			case event.ActorTypeParticipant:
				actorType = command.ActorTypeParticipant
			case event.ActorTypeGM:
				actorType = command.ActorTypeGM
			}
			result, err := a.stores.Domain.Execute(ctx, command.Command{
				CampaignID:   input.CampaignID,
				Type:         cmdType,
				ActorType:    actorType,
				ActorID:      input.ActorID,
				SessionID:    input.SessionID,
				RequestID:    input.RequestID,
				InvocationID: input.InvocationID,
				EntityType:   input.EntityType,
				EntityID:     input.EntityID,
				PayloadJSON:  input.PayloadJSON,
			})
			if err != nil {
				return event.Event{}, status.Errorf(codes.Internal, "execute domain command: %v", err)
			}
			if len(result.Decision.Rejections) > 0 {
				return event.Event{}, status.Error(codes.FailedPrecondition, result.Decision.Rejections[0].Message)
			}
			if len(result.Decision.Events) == 0 {
				return event.Event{}, status.Error(codes.Internal, "append event did not emit an event")
			}
			return result.Decision.Events[0], nil
		}
		return event.Event{}, status.Error(codes.FailedPrecondition, "event type is not supported for append")
	}

	stored, err := a.stores.Event.AppendEvent(ctx, input)
	if err != nil {
		if isEventValidationError(err) {
			return event.Event{}, status.Error(codes.InvalidArgument, err.Error())
		}
		return event.Event{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	return stored, nil
}

func isEventValidationError(err error) bool {
	return errors.Is(err, event.ErrCampaignIDRequired) ||
		errors.Is(err, event.ErrTypeRequired) ||
		errors.Is(err, event.ErrTypeUnknown) ||
		errors.Is(err, event.ErrActorTypeInvalid) ||
		errors.Is(err, event.ErrActorIDRequired) ||
		errors.Is(err, event.ErrSystemMetadataRequired) ||
		errors.Is(err, event.ErrSystemMetadataForbidden) ||
		errors.Is(err, event.ErrPayloadInvalid) ||
		errors.Is(err, event.ErrStorageFieldsSet)
}

func domainCommandTypeForEvent(eventType event.Type) (command.Type, bool) {
	switch eventType {
	case event.Type("story.note_added"):
		return command.Type("story.note.add"), true
	case event.Type("action.roll_resolved"):
		return command.Type("action.roll.resolve"), true
	case event.Type("action.outcome_applied"):
		return command.Type("action.outcome.apply"), true
	case event.Type("action.outcome_rejected"):
		return command.Type("action.outcome.reject"), true
	default:
		return "", false
	}
}
