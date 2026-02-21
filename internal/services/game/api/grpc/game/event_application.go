package game

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var compatibilityAppendEnabled atomic.Bool

type eventApplication struct {
	stores Stores
}

func newEventApplication(service *EventService) eventApplication {
	return eventApplication{stores: service.stores}
}

// SetCompatibilityAppendEnabled toggles direct event-store append in
// EventService.AppendEvent when the domain engine is not configured.
func SetCompatibilityAppendEnabled(enabled bool) {
	compatibilityAppendEnabled.Store(enabled)
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
			result, err := a.stores.Domain.Execute(ctx, command.Command{
				CampaignID:   input.CampaignID,
				Type:         cmdType,
				ActorType:    commandActorTypeForEventActor(input.ActorType),
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
			for _, emitted := range result.Decision.Events {
				if emitted.Type == input.Type {
					return emitted, nil
				}
			}
			return event.Event{}, status.Errorf(
				codes.FailedPrecondition,
				"append event did not emit requested event type %s",
				input.Type,
			)
		}
		return event.Event{}, status.Error(codes.FailedPrecondition, "event type is not supported for append")
	}

	if !compatibilityAppendEnabled.Load() {
		return event.Event{}, status.Error(codes.FailedPrecondition, "append event compatibility mode is disabled")
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

var compatibilityEventCommandMap = map[event.Type]command.Type{
	event.Type("story.note_added"):        command.Type("story.note.add"),
	event.Type("action.roll_resolved"):    command.Type("action.roll.resolve"),
	event.Type("action.outcome_applied"):  command.Type("action.outcome.apply"),
	event.Type("action.outcome_rejected"): command.Type("action.outcome.reject"),
}

func domainCommandTypeForEvent(eventType event.Type) (command.Type, bool) {
	cmdType, ok := compatibilityEventCommandMap[eventType]
	if !ok {
		return "", false
	}
	return cmdType, true
}

func commandActorTypeForEventActor(actorType event.ActorType) command.ActorType {
	switch actorType {
	case event.ActorTypeParticipant:
		return command.ActorTypeParticipant
	case event.ActorTypeGM:
		return command.ActorTypeGM
	default:
		return command.ActorTypeSystem
	}
}
