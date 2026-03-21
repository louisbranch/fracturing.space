package participanttransport

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestUpdateParticipant_DeniesHumanGMForAIGMCampaign(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
		GmMode: campaign.GmModeAI,
	}
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

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.UpdateParticipant(gametest.ContextWithParticipantID("owner-1"), &statev1.UpdateParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		Role:          statev1.ParticipantRole_GM,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateParticipant_CampaignAccess(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	eventStore := gametest.NewFakeEventStore()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1": gametest.OwnerParticipantRecord("c1", "owner-1"),
		"p1":      {ID: "p1", CampaignID: "c1", Name: "Player One", CampaignAccess: participant.CampaignAccessMember},
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
				PayloadJSON: []byte(`{"participant_id":"p1","fields":{"campaign_access":"manager"}}`),
			}),
		},
	}}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}, Applier: projection.Applier{Campaign: campaignStore, Participant: participantStore}})
	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:     "c1",
		ParticipantId:  "p1",
		CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
	})
	if err != nil {
		t.Fatalf("UpdateParticipant returned error: %v", err)
	}
	if resp.Participant.CampaignAccess != statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER {
		t.Errorf("Participant CampaignAccess = %v, want %v", resp.Participant.CampaignAccess, statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER)
	}

	stored, err := participantStore.GetParticipant(context.Background(), "c1", "p1")
	if err != nil {
		t.Fatalf("Participant not persisted: %v", err)
	}
	if stored.CampaignAccess != participant.CampaignAccessManager {
		t.Errorf("Stored participant CampaignAccess = %v, want %v", stored.CampaignAccess, participant.CampaignAccessManager)
	}
}

func TestUpdateParticipant_DeniesManagerAssigningOwnerAccess(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1":   {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"manager-1": gametest.ManagerParticipantRecord("c1", "manager-1"),
		"member-1":  {ID: "member-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessMember},
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	ctx := gametest.ContextWithParticipantID("manager-1")
	_, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:     "c1",
		ParticipantId:  "member-1",
		CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestUpdateParticipant_DeniesManagerMutatingOwnerWithoutAccessChange(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1":   {ID: "owner-1", CampaignID: "c1", Name: "Owner", CampaignAccess: participant.CampaignAccessOwner},
		"manager-1": gametest.ManagerParticipantRecord("c1", "manager-1"),
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	ctx := gametest.ContextWithParticipantID("manager-1")
	_, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "owner-1",
		Name:          wrapperspb.String("Updated Owner"),
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestUpdateParticipant_AllowsSelfOwnedProfileChanges(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			UserID:         "user-1",
			Name:           "Player One",
			Role:           participant.RolePlayer,
			Controller:     participant.ControllerHuman,
			CampaignAccess: participant.CampaignAccessMember,
			Pronouns:       "she/her",
		},
	}
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("participant.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("participant.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "participant",
				EntityID:    "p1",
				PayloadJSON: []byte(`{"participant_id":"p1","fields":{"name":"Player Prime","pronouns":"they/them"}}`),
			}),
		},
	}}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore, Write: domainwrite.WritePath{Executor: domain, Runtime: testRuntime}, Applier: projection.Applier{Campaign: campaignStore, Participant: participantStore}})
	resp, err := svc.UpdateParticipant(gametest.ContextWithUserID("user-1"), &statev1.UpdateParticipantRequest{
		CampaignId:     "c1",
		ParticipantId:  "p1",
		Name:           wrapperspb.String("Player Prime"),
		Pronouns:       sharedpronouns.ToProto("they/them"),
		Role:           statev1.ParticipantRole_PLAYER,
		CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
	})
	if err != nil {
		t.Fatalf("UpdateParticipant returned error: %v", err)
	}
	if resp.Participant.GetName() != "Player Prime" {
		t.Fatalf("participant name = %q, want %q", resp.Participant.GetName(), "Player Prime")
	}
	if got := sharedpronouns.FromProto(resp.Participant.GetPronouns()); got != "they/them" {
		t.Fatalf("participant pronouns = %q, want %q", got, "they/them")
	}
}

func TestUpdateParticipant_DeniesSelfOwnedGovernanceChange(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			UserID:         "user-1",
			Name:           "Player One",
			Role:           participant.RolePlayer,
			Controller:     participant.ControllerHuman,
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.UpdateParticipant(gametest.ContextWithUserID("user-1"), &statev1.UpdateParticipantRequest{
		CampaignId:     "c1",
		ParticipantId:  "p1",
		Name:           wrapperspb.String("Player Prime"),
		Role:           statev1.ParticipantRole_GM,
		CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestUpdateParticipant_DeniesDemotingFinalOwner(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"owner-1":  {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"member-1": gametest.MemberParticipantRecord("c1", "member-1"),
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:     "c1",
		ParticipantId:  "owner-1",
		CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}
