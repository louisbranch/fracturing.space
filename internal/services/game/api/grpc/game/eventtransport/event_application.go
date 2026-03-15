package eventtransport

import (
	"context"
	"strings"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	appendEventScopeHeader      = "x-fracturing-space-append-event-scope"
	appendEventScopeMaintenance = "maintenance"
	appendEventScopeAdmin       = "admin"
)

// Deps holds all dependencies needed by the event transport layer.
type Deps struct {
	Auth        authz.PolicyDeps
	Event       storage.EventStore
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Character   storage.CharacterStore
	Session     storage.SessionStore
	Write       domainwriteexec.WritePath
}

type eventApplication struct {
	auth   authz.PolicyDeps
	stores eventApplicationStores
	write  domainwriteexec.WritePath
	clock  func() time.Time
}

type eventApplicationStores struct {
	Event       storage.EventStore
	Campaign    storage.CampaignStore
	Participant storage.ParticipantStore
	Character   storage.CharacterStore
	Session     storage.SessionStore
}

func newEventApplication(service *Service) eventApplication {
	if service == nil {
		return eventApplication{}
	}
	return service.app
}

func newEventApplicationWithDependencies(deps Deps, clock func() time.Time) eventApplication {
	app := eventApplication{
		auth: deps.Auth,
		stores: eventApplicationStores{
			Event:       deps.Event,
			Campaign:    deps.Campaign,
			Participant: deps.Participant,
			Character:   deps.Character,
			Session:     deps.Session,
		},
		write: deps.Write,
		clock: clock,
	}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}

func (a eventApplication) AppendEvent(ctx context.Context, in *campaignv1.AppendEventRequest) (event.Event, error) {
	if !appendEventScopeAllowed(ctx) {
		return event.Event{}, status.Error(codes.PermissionDenied, "append event is restricted to maintenance/admin scope")
	}
	input := event.Event{
		CampaignID:   ids.CampaignID(in.GetCampaignId()),
		Timestamp:    a.clock().UTC(),
		Type:         event.Type(strings.TrimSpace(in.GetType())),
		SessionID:    ids.SessionID(strings.TrimSpace(in.GetSessionId())),
		SceneID:      ids.SceneID(strings.TrimSpace(in.GetSceneId())),
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    event.ActorType(strings.TrimSpace(in.GetActorType())),
		ActorID:      strings.TrimSpace(in.GetActorId()),
		EntityType:   strings.TrimSpace(in.GetEntityType()),
		EntityID:     strings.TrimSpace(in.GetEntityId()),
		PayloadJSON:  in.GetPayloadJson(),
	}
	if a.write.Executor == nil {
		return event.Event{}, status.Error(codes.FailedPrecondition, "append event requires domain engine")
	}
	cmdType, ok := domainCommandTypeForEvent(input.Type)
	if !ok {
		return event.Event{}, status.Error(codes.FailedPrecondition, "event type is not supported for append")
	}
	cmd := commandbuild.Core(commandbuild.CoreInput{
		CampaignID:   string(input.CampaignID),
		Type:         cmdType,
		ActorType:    handler.CommandActorTypeForEventActor(input.ActorType),
		ActorID:      input.ActorID,
		SessionID:    input.SessionID.String(),
		SceneID:      input.SceneID.String(),
		RequestID:    input.RequestID,
		InvocationID: input.InvocationID,
		EntityType:   input.EntityType,
		EntityID:     input.EntityID,
		PayloadJSON:  input.PayloadJSON,
	})
	result, err := handler.ExecuteWithoutInlineApply(ctx, a.write, cmd, domainwrite.Options{
		RequireEvents:   true,
		MissingEventMsg: "append event did not emit an event",
		ExecuteErr: func(err error) error {
			return grpcerror.Internal("execute domain command", err)
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
	case handler.EventTypeStoryNoteAdded:
		return handler.CommandTypeStoryNoteAdd, true
	case handler.EventTypeActionRollResolved:
		return handler.CommandTypeActionRollResolve, true
	case handler.EventTypeActionOutcomeApplied:
		return handler.CommandTypeActionOutcomeApply, true
	case handler.EventTypeActionOutcomeRejected:
		return handler.CommandTypeActionOutcomeReject, true
	default:
		return "", false
	}
}
