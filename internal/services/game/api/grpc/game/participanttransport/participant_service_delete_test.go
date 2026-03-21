package participanttransport

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestDeleteParticipant_DeniesManagerRemovingOwner(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1":   {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"manager-1": gametest.ManagerParticipantRecord("c1", "manager-1"),
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore, Character: characterStore}, Campaign: campaignStore, Participant: participantStore, Character: characterStore})
	ctx := gametest.ContextWithParticipantID("manager-1")
	_, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "owner-1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestDeleteParticipant_DeniesRemovingFinalOwner(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1":  {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"member-1": gametest.MemberParticipantRecord("c1", "member-1"),
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore, Character: characterStore}, Campaign: campaignStore, Participant: participantStore, Character: characterStore})
	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "owner-1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestDeleteParticipant_DeniesMemberWithoutManageAccess(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecordWithParticipantCount("c1", 2)
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1": gametest.MemberParticipantRecord("c1", "member-1"),
		"p1":       {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, CampaignAccess: participant.CampaignAccessMember},
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	ctx := gametest.ContextWithParticipantID("member-1")
	_, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestDeleteParticipant_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecordWithParticipantCount("c1", 1)
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.leave"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.left"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","reason":"left"}`),
			}),
		},
	}}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore, Character: characterStore}, Campaign: campaignStore, Participant: participantStore, Character: characterStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}, Applier: projection.Applier{Campaign: campaignStore, Participant: participantStore, Character: characterStore}})
	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Reason:        "left",
	})
	if err != nil {
		t.Fatalf("DeleteParticipant returned error: %v", err)
	}
	if resp.Participant.Id != "p1" {
		t.Errorf("Participant ID = %q, want %q", resp.Participant.Id, "p1")
	}
	if _, err := participantStore.GetParticipant(context.Background(), "c1", "p1"); err == nil {
		t.Fatal("expected participant to be deleted")
	}
	updatedCampaign, err := campaignStore.Get(context.Background(), "c1")
	if err != nil {
		t.Fatalf("campaign not found: %v", err)
	}
	if updatedCampaign.ParticipantCount != 1 {
		t.Errorf("ParticipantCount = %d, want 1", updatedCampaign.ParticipantCount)
	}
	if got := len(eventStore.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["c1"][0].Type != event.Type("participant.left") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, event.Type("participant.left"))
	}
}

func TestDeleteParticipant_DeniesWhenParticipantOwnsCharacter(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 20, 19, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecordWithParticipantCount("c1", 2)
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, CampaignAccess: participant.CampaignAccessMember},
	}
	eventStore.Events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "p1",
			EntityType:  "character",
			EntityID:    "ch-1",
			PayloadJSON: []byte(`{"character_id":"ch-1","name":"Hero","kind":"pc","owner_participant_id":"p1"}`),
		},
	}
	eventStore.NextSeq["c1"] = 2
	characterStore.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch-1": {ID: "ch-1", CampaignID: "c1", OwnerParticipantID: "p1", Name: "Hero", Kind: character.KindPC},
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.leave"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.left"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","reason":"left"}`),
			}),
		},
	}}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore, Character: characterStore}, Campaign: campaignStore, Participant: participantStore, Character: characterStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}, Applier: projection.Applier{Campaign: campaignStore, Participant: participantStore, Character: characterStore}})
	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Reason:        "left",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
	if domain.calls != 0 {
		t.Fatalf("domain calls = %d, want 0", domain.calls)
	}
}

func TestDeleteParticipant_DeniesWhenParticipantOwnsCharacterFromActorFallback(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 20, 19, 5, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecordWithParticipantCount("c1", 2)
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, CampaignAccess: participant.CampaignAccessMember},
	}
	eventStore.Events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "p1",
			EntityType:  "character",
			EntityID:    "ch-1",
			PayloadJSON: []byte(`{"character_id":"ch-1","name":"Hero","kind":"pc"}`),
		},
	}
	eventStore.NextSeq["c1"] = 2
	characterStore.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch-1": {ID: "ch-1", CampaignID: "c1", OwnerParticipantID: "p1", Name: "Hero", Kind: character.KindPC},
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.leave"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.left"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","reason":"left"}`),
			}),
		},
	}}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore, Character: characterStore}, Campaign: campaignStore, Participant: participantStore, Character: characterStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}, Applier: projection.Applier{Campaign: campaignStore, Participant: participantStore, Character: characterStore}})
	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Reason:        "left",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
	if domain.calls != 0 {
		t.Fatalf("domain calls = %d, want 0", domain.calls)
	}
}

func TestDeleteParticipant_AllowsWhenOwnedCharacterAlreadyDeleted(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 20, 19, 10, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecordWithParticipantCount("c1", 2)
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, CampaignAccess: participant.CampaignAccessMember},
	}
	eventStore.Events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "p1",
			EntityType:  "character",
			EntityID:    "ch-1",
			PayloadJSON: []byte(`{"character_id":"ch-1","name":"Hero","kind":"pc","owner_participant_id":"p1"}`),
		},
		{
			Seq:         2,
			CampaignID:  "c1",
			Type:        event.Type("character.deleted"),
			Timestamp:   now.Add(time.Minute),
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "p1",
			EntityType:  "character",
			EntityID:    "ch-1",
			PayloadJSON: []byte(`{"character_id":"ch-1","reason":"retired"}`),
		},
	}
	eventStore.NextSeq["c1"] = 3

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.leave"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.left"),
				Timestamp:   now.Add(2 * time.Minute),
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","reason":"left"}`),
			}),
		},
	}}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore, Character: characterStore}, Campaign: campaignStore, Participant: participantStore, Character: characterStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}, Applier: projection.Applier{Campaign: campaignStore, Participant: participantStore, Character: characterStore}})
	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Reason:        "left",
	})
	if err != nil {
		t.Fatalf("DeleteParticipant returned error: %v", err)
	}
	if resp.GetParticipant().GetId() != "p1" {
		t.Fatalf("participant id = %q, want %q", resp.GetParticipant().GetId(), "p1")
	}
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want 1", domain.calls)
	}
}

func TestDeleteParticipant_AllowsWhenCharacterOwnershipTransferredAway(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2026, 2, 20, 19, 25, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecordWithParticipantCount("c1", 3)
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer, CampaignAccess: participant.CampaignAccessMember},
		"p2":      {ID: "p2", CampaignID: "c1", Name: "Player Two", Role: participant.RolePlayer, CampaignAccess: participant.CampaignAccessMember},
	}
	eventStore.Events["c1"] = []event.Event{
		{
			Seq:         1,
			CampaignID:  "c1",
			Type:        event.Type("character.created"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "p1",
			EntityType:  "character",
			EntityID:    "ch-1",
			PayloadJSON: []byte(`{"character_id":"ch-1","name":"Hero","kind":"pc","owner_participant_id":"p1"}`),
		},
		{
			Seq:         2,
			CampaignID:  "c1",
			Type:        event.Type("character.updated"),
			Timestamp:   now.Add(time.Minute),
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "owner-1",
			EntityType:  "character",
			EntityID:    "ch-1",
			PayloadJSON: []byte(`{"character_id":"ch-1","fields":{"owner_participant_id":"p2"}}`),
		},
	}
	eventStore.NextSeq["c1"] = 3
	characterStore.Characters["c1"] = map[string]storage.CharacterRecord{
		"ch-1": {ID: "ch-1", CampaignID: "c1", OwnerParticipantID: "p2", Name: "Hero", Kind: character.KindPC},
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.leave"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.left"),
				Timestamp:   now.Add(2 * time.Minute),
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","reason":"left"}`),
			}),
		},
	}}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore, Character: characterStore}, Campaign: campaignStore, Participant: participantStore, Character: characterStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}, Applier: projection.Applier{Campaign: campaignStore, Participant: participantStore, Character: characterStore}})
	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Reason:        "left",
	})
	if err != nil {
		t.Fatalf("DeleteParticipant returned error: %v", err)
	}
	if resp.GetParticipant().GetId() != "p1" {
		t.Fatalf("participant id = %q, want %q", resp.GetParticipant().GetId(), "p1")
	}
	if domain.calls != 1 {
		t.Fatalf("domain calls = %d, want 1", domain.calls)
	}
}

func TestDeleteParticipant_RequiresDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecordWithParticipantCount("c1", 1)
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer},
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore, Character: characterStore}, Campaign: campaignStore, Participant: participantStore, Character: characterStore})
	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestDeleteParticipant_UsesDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecordWithParticipantCount("c1", 1)
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", Role: participant.RolePlayer},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.leave"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.left"),
				Timestamp:   now,
				ActorType:   event.ActorTypeParticipant,
				ActorID:     "owner-1",
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","reason":"left"}`),
			}),
		},
	}}

	svc := newParticipantServiceForTest(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore, Character: characterStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Character:   characterStore,
			Write:       domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
			Applier:     projection.Applier{Campaign: campaignStore, Participant: participantStore, Character: characterStore},
		},
		gametest.FixedClock(now),
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.DeleteParticipant(ctx, &statev1.DeleteParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Reason:        "left",
	})
	if err != nil {
		t.Fatalf("DeleteParticipant returned error: %v", err)
	}
	if resp.Participant.Id != "p1" {
		t.Fatalf("Participant ID = %q, want %q", resp.Participant.Id, "p1")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("participant.leave") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "participant.leave")
	}
	if got := len(eventStore.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["c1"][0].Type != event.Type("participant.left") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, event.Type("participant.left"))
	}
	if _, err := participantStore.GetParticipant(context.Background(), "c1", "p1"); err == nil {
		t.Fatal("expected participant to be deleted")
	}
	updatedCampaign, err := campaignStore.Get(context.Background(), "c1")
	if err != nil {
		t.Fatalf("campaign not found: %v", err)
	}
	if updatedCampaign.ParticipantCount != 1 {
		t.Fatalf("ParticipantCount = %d, want 1", updatedCampaign.ParticipantCount)
	}
}
