package invitetransport

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
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
