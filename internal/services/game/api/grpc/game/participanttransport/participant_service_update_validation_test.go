package participanttransport

import (
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestUpdateParticipant_NoFields(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	eventStore := gametest.NewFakeEventStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, Controller: participant.ControllerHuman},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","fields":{"name":"Player Uno","controller":"ai"}}`),
			}),
		},
	}}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}, Applier: projection.Applier{Campaign: campaignStore, Participant: participantStore}})
	ctx := requestctx.WithParticipantID("owner-1")
	_, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateParticipant_RequiresDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, Controller: participant.ControllerHuman},
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	ctx := requestctx.WithParticipantID("owner-1")
	_, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Name:          wrapperspb.String("Player Uno"),
	})
	assertStatusCode(t, err, codes.Internal)
}
