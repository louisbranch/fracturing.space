package participanttransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
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

func TestUpdateParticipant_DomainRejectsAIInvariant(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	eventStore := gametest.NewFakeEventStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			Name:           "Player One",
			Role:           participant.RolePlayer,
			Controller:     participant.ControllerHuman,
			CampaignAccess: participant.CampaignAccessMember,
		},
	}
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.update"): {
			Decision: command.Reject(command.Rejection{
				Code:    "PARTICIPANT_AI_ROLE_REQUIRED",
				Message: "ai-controlled participants must use gm role",
			}),
		},
	}}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}, Applier: projection.Applier{Campaign: campaignStore, Participant: participantStore}})
	ctx := requestctx.WithParticipantID("owner-1")
	_, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Controller:    statev1.Controller_CONTROLLER_AI,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want 1", domain.calls)
	}
}

func TestUpdateParticipant_Success(t *testing.T) {
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
	resp, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Name:          wrapperspb.String("Player Uno"),
		Controller:    statev1.Controller_CONTROLLER_AI,
	})
	if err != nil {
		t.Fatalf("UpdateParticipant returned error: %v", err)
	}
	if resp.Participant.Name != "Player Uno" {
		t.Errorf("Participant Name = %q, want %q", resp.Participant.Name, "Player Uno")
	}
	if resp.Participant.Controller != statev1.Controller_CONTROLLER_AI {
		t.Errorf("Participant Controller = %v, want %v", resp.Participant.Controller, statev1.Controller_CONTROLLER_AI)
	}

	stored, err := participantStore.GetParticipant(context.Background(), "c1", "p1")
	if err != nil {
		t.Fatalf("Participant not persisted: %v", err)
	}
	if stored.Name != "Player Uno" {
		t.Errorf("Stored participant Name = %q, want %q", stored.Name, "Player Uno")
	}
	if got := len(eventStore.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["c1"][0].Type != event.Type("participant.updated") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, event.Type("participant.updated"))
	}
}

func TestUpdateParticipant_UsesDomainEngine(t *testing.T) {
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

	svc := newParticipantServiceForTest(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
			Applier:     projection.Applier{Campaign: campaignStore, Participant: participantStore},
		},
		runtimekit.FixedClock(now),
		nil,
		nil,
	)

	ctx := requestctx.WithParticipantID("owner-1")
	resp, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Name:          wrapperspb.String("Player Uno"),
		Controller:    statev1.Controller_CONTROLLER_AI,
	})
	if err != nil {
		t.Fatalf("UpdateParticipant returned error: %v", err)
	}
	if resp.Participant == nil {
		t.Fatal("UpdateParticipant response has nil participant")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("participant.update") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "participant.update")
	}
	if got := len(eventStore.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["c1"][0].Type != event.Type("participant.updated") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, event.Type("participant.updated"))
	}
	stored, err := participantStore.GetParticipant(context.Background(), "c1", "p1")
	if err != nil {
		t.Fatalf("Participant not persisted: %v", err)
	}
	if stored.Name != "Player Uno" {
		t.Fatalf("Stored participant Name = %q, want %q", stored.Name, "Player Uno")
	}
}
