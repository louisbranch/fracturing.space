package invitetransport

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	gamegrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/checkpoint"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/inviteclaimworkflow"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var testRuntime *domainwrite.Runtime

func TestMain(m *testing.M) {
	testRuntime = gametest.SetupRuntime()
	os.Exit(m.Run())
}

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with code %v", want)
	}
	statusErr, ok := status.FromError(err)
	if !ok {
		err = grpcerror.HandleDomainError(err)
		statusErr, ok = status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %T", err)
		}
	}
	if statusErr.Code() != want {
		t.Fatalf("status code = %v, want %v (message: %s)", statusErr.Code(), want, statusErr.Message())
	}
}

type fakeDomainEngine struct {
	store         storage.EventStore
	result        engine.Result
	resultsByType map[command.Type]engine.Result
	calls         int
	lastCommand   command.Command
	commands      []command.Command
}

func (f *fakeDomainEngine) Execute(ctx context.Context, cmd command.Command) (engine.Result, error) {
	f.calls++
	f.lastCommand = cmd
	f.commands = append(f.commands, cmd)

	result := f.result
	if len(f.resultsByType) > 0 {
		if selected, ok := f.resultsByType[cmd.Type]; ok {
			result = selected
		}
	}
	if len(result.Decision.Events) == 0 && cmd.Type == inviteclaimworkflow.CommandTypeClaimBind {
		result = synthesizeInviteClaimWorkflowResult(timedNow(cmd), cmd)
	}
	if f.store == nil {
		return result, nil
	}
	if len(result.Decision.Events) == 0 {
		return result, nil
	}
	stored := make([]event.Event, 0, len(result.Decision.Events))
	for _, evt := range result.Decision.Events {
		storedEvent, err := f.store.AppendEvent(ctx, evt)
		if err != nil {
			return engine.Result{}, err
		}
		stored = append(stored, storedEvent)
	}
	result.Decision.Events = stored
	return result, nil
}

func withInviteClaimWrite(t *testing.T, deps Deps) Deps {
	t.Helper()
	if deps.Write.Executor != nil {
		return deps
	}
	deps.Write = newInviteClaimWritePath(t, deps.Event)
	return deps
}

func newInviteClaimWritePath(t *testing.T, store storage.EventStore) domainwrite.WritePath {
	t.Helper()
	if store == nil {
		t.Fatal("event store is required")
	}

	registries, err := engine.BuildRegistries()
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}
	decider, err := engine.NewCoreDecider(registries.Systems, registries.Commands.ListDefinitions())
	if err != nil {
		t.Fatalf("build core decider: %v", err)
	}

	checkpoints := checkpoint.NewNoop()
	folder := &aggregate.Folder{
		Events:         registries.Events,
		SystemRegistry: registries.Systems,
	}
	stateLoader := engine.ReplayStateLoader{
		Events:       gamegrpc.NewEventStoreAdapter(store),
		Checkpoints:  checkpoints,
		Snapshots:    checkpoints,
		Folder:       folder,
		StateFactory: func() any { return aggregate.NewState() },
	}
	gateStateLoader := engine.ReplayGateStateLoader{StateLoader: stateLoader}
	domain, err := engine.NewHandler(engine.Handler{
		Commands:             registries.Commands,
		Events:               registries.Events,
		Journal:              gamegrpc.NewJournalAdapter(store),
		Checkpoints:          checkpoints,
		Snapshots:            checkpoints,
		Gate:                 engine.DecisionGate{Registry: registries.Commands},
		GateStateLoader:      gateStateLoader,
		SceneGateStateLoader: gateStateLoader,
		StateLoader:          stateLoader,
		Decider:              decider,
		Folder:               folder,
	})
	if err != nil {
		t.Fatalf("build domain handler: %v", err)
	}
	return domainwrite.WritePath{Executor: domain, Runtime: testRuntime}
}

func synthesizeInviteClaimWorkflowResult(now time.Time, cmd command.Command) engine.Result {
	var payload inviteclaimworkflow.ClaimBindPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return engine.Result{}
	}

	bindPayloadJSON, _ := json.Marshal(participant.BindPayload{
		ParticipantID: payload.ParticipantID,
		UserID:        payload.UserID,
	})
	claimPayloadJSON, _ := json.Marshal(invite.ClaimPayload{
		InviteID:      payload.InviteID,
		ParticipantID: payload.ParticipantID,
		UserID:        payload.UserID,
		JWTID:         payload.JWTID,
	})

	return engine.Result{
		Decision: command.Accept(
			event.Event{
				CampaignID:  cmd.CampaignID,
				Type:        participant.EventTypeBound,
				Timestamp:   now,
				ActorType:   event.ActorType(cmd.ActorType),
				ActorID:     cmd.ActorID,
				EntityType:  "participant",
				EntityID:    string(payload.ParticipantID),
				PayloadJSON: bindPayloadJSON,
			},
			event.Event{
				CampaignID:  cmd.CampaignID,
				Type:        invite.EventTypeClaimed,
				Timestamp:   now,
				ActorType:   event.ActorType(cmd.ActorType),
				ActorID:     cmd.ActorID,
				EntityType:  "invite",
				EntityID:    string(payload.InviteID),
				PayloadJSON: claimPayloadJSON,
			},
		),
	}
}

func timedNow(cmd command.Command) time.Time {
	return time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
}

type eventAppender interface {
	AppendEvent(context.Context, event.Event) (event.Event, error)
}

func seedParticipantJoinedEvent(t *testing.T, store eventAppender, record storage.ParticipantRecord, stamp time.Time) {
	t.Helper()

	role := record.Role
	if role == "" {
		role = participant.RolePlayer
	}
	controller := record.Controller
	if controller == "" {
		controller = participant.ControllerHuman
	}
	access := record.CampaignAccess
	if access == "" {
		access = participant.CampaignAccessMember
	}
	name := record.Name
	if name == "" {
		name = record.ID
	}
	payloadJSON, err := json.Marshal(participant.JoinPayload{
		ParticipantID:  ids.ParticipantID(record.ID),
		UserID:         ids.UserID(record.UserID),
		Name:           name,
		Role:           string(role),
		Controller:     string(controller),
		CampaignAccess: string(access),
		AvatarSetID:    record.AvatarSetID,
		AvatarAssetID:  record.AvatarAssetID,
		Pronouns:       record.Pronouns,
	})
	if err != nil {
		t.Fatalf("marshal participant join payload: %v", err)
	}
	if _, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  ids.CampaignID(record.CampaignID),
		Type:        participant.EventTypeJoined,
		Timestamp:   stamp,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    record.ID,
		PayloadJSON: payloadJSON,
	}); err != nil {
		t.Fatalf("append participant join event: %v", err)
	}
}

func seedInviteCreatedEvent(t *testing.T, store eventAppender, record storage.InviteRecord, stamp time.Time) {
	t.Helper()

	status := record.Status
	if status == invite.StatusUnspecified {
		status = invite.StatusPending
	}
	payloadJSON, err := json.Marshal(invite.CreatePayload{
		InviteID:               ids.InviteID(record.ID),
		ParticipantID:          ids.ParticipantID(record.ParticipantID),
		RecipientUserID:        ids.UserID(record.RecipientUserID),
		CreatedByParticipantID: ids.ParticipantID(record.CreatedByParticipantID),
		Status:                 string(status),
	})
	if err != nil {
		t.Fatalf("marshal invite create payload: %v", err)
	}
	if _, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  ids.CampaignID(record.CampaignID),
		Type:        invite.EventTypeCreated,
		Timestamp:   stamp,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "invite",
		EntityID:    record.ID,
		PayloadJSON: payloadJSON,
	}); err != nil {
		t.Fatalf("append invite create event: %v", err)
	}
}

func seedParticipantLeftEvent(t *testing.T, store eventAppender, campaignID string, participantID string, stamp time.Time) {
	t.Helper()

	payloadJSON, err := json.Marshal(participant.LeavePayload{
		ParticipantID: ids.ParticipantID(participantID),
	})
	if err != nil {
		t.Fatalf("marshal participant leave payload: %v", err)
	}
	if _, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  ids.CampaignID(campaignID),
		Type:        participant.EventTypeLeft,
		Timestamp:   stamp,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    participantID,
		PayloadJSON: payloadJSON,
	}); err != nil {
		t.Fatalf("append participant left event: %v", err)
	}
}

func seedParticipantUnboundEvent(t *testing.T, store eventAppender, campaignID string, participantID string, userID string, stamp time.Time) {
	t.Helper()

	payloadJSON, err := json.Marshal(participant.UnbindPayload{
		ParticipantID: ids.ParticipantID(participantID),
		UserID:        ids.UserID(userID),
	})
	if err != nil {
		t.Fatalf("marshal participant unbind payload: %v", err)
	}
	if _, err := store.AppendEvent(context.Background(), event.Event{
		CampaignID:  ids.CampaignID(campaignID),
		Type:        participant.EventTypeUnbound,
		Timestamp:   stamp,
		ActorType:   event.ActorTypeSystem,
		EntityType:  "participant",
		EntityID:    participantID,
		PayloadJSON: payloadJSON,
	}); err != nil {
		t.Fatalf("append participant unbind event: %v", err)
	}
}
