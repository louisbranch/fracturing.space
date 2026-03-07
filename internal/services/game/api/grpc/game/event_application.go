package game

import (
	"context"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	appendEventScopeHeader      = "x-fracturing-space-append-event-scope"
	appendEventScopeMaintenance = "maintenance"
	appendEventScopeAdmin       = "admin"
)

type eventApplication struct {
	stores Stores
}

func newEventApplication(service *EventService) eventApplication {
	return eventApplication{
		stores: service.stores,
	}
}

func (a eventApplication) AppendEvent(ctx context.Context, in *campaignv1.AppendEventRequest) (event.Event, error) {
	if !appendEventScopeAllowed(ctx) {
		return event.Event{}, status.Error(codes.PermissionDenied, "append event is restricted to maintenance/admin scope")
	}
	input := event.Event{
		CampaignID:   in.GetCampaignId(),
		Timestamp:    time.Now().UTC(),
		Type:         event.Type(strings.TrimSpace(in.GetType())),
		SessionID:    strings.TrimSpace(in.GetSessionId()),
		SceneID:      strings.TrimSpace(in.GetSceneId()),
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    event.ActorType(strings.TrimSpace(in.GetActorType())),
		ActorID:      strings.TrimSpace(in.GetActorId()),
		EntityType:   strings.TrimSpace(in.GetEntityType()),
		EntityID:     strings.TrimSpace(in.GetEntityId()),
		PayloadJSON:  in.GetPayloadJson(),
	}
	if a.stores.Domain == nil {
		return event.Event{}, status.Error(codes.FailedPrecondition, "append event requires domain engine")
	}
	cmdType, ok := domainCommandTypeForEvent(input.Type)
	if !ok {
		return event.Event{}, status.Error(codes.FailedPrecondition, "event type is not supported for append")
	}
	cmd := commandbuild.Core(commandbuild.CoreInput{
		CampaignID:   input.CampaignID,
		Type:         cmdType,
		ActorType:    commandActorTypeForEventActor(input.ActorType),
		ActorID:      input.ActorID,
		SessionID:    input.SessionID,
		SceneID:      input.SceneID,
		RequestID:    input.RequestID,
		InvocationID: input.InvocationID,
		EntityType:   input.EntityType,
		EntityID:     input.EntityID,
		PayloadJSON:  input.PayloadJSON,
	})
	result, err := executeDomainCommandWithoutInlineApply(ctx, a.stores, cmd, domainwrite.Options{
		RequireEvents:   true,
		MissingEventMsg: "append event did not emit an event",
		ExecuteErr: func(err error) error {
			return status.Errorf(codes.Internal, "execute domain command: %v", err)
		},
	})
	if err != nil {
		return event.Event{}, err
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

func appendEventScopeAllowed(ctx context.Context) bool {
	if ctx == nil {
		return false
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return false
	}
	scope := strings.ToLower(strings.TrimSpace(grpcmeta.FirstMetadataValue(md, appendEventScopeHeader)))
	switch scope {
	case appendEventScopeMaintenance, appendEventScopeAdmin:
		return true
	default:
		return false
	}
}

func domainCommandTypeForEvent(eventType event.Type) (command.Type, bool) {
	switch eventType {
	case eventTypeStoryNoteAdded:
		return commandTypeStoryNoteAdd, true
	case eventTypeActionRollResolved:
		return commandTypeActionRollResolve, true
	case eventTypeActionOutcomeApplied:
		return commandTypeActionOutcomeApply, true
	case eventTypeActionOutcomeRejected:
		return commandTypeActionOutcomeReject, true
	default:
		return "", false
	}
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
