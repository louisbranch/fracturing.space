package game

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestCreateParticipant_NilRequest(t *testing.T) {
	svc := NewParticipantService(Stores{})
	_, err := svc.CreateParticipant(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateParticipant_MissingCampaignId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	_, err := svc.CreateParticipant(context.Background(), &statev1.CreateParticipantRequest{
		DisplayName: "Player 1",
		Role:        statev1.ParticipantRole_PLAYER,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateParticipant_CampaignNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	_, err := svc.CreateParticipant(context.Background(), &statev1.CreateParticipantRequest{
		CampaignId:  "nonexistent",
		DisplayName: "Player 1",
		Role:        statev1.ParticipantRole_PLAYER,
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestCreateParticipant_CampaignArchivedDisallowed(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusArchived,
	}

	eventStore := newFakeEventStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	_, err := svc.CreateParticipant(context.Background(), &statev1.CreateParticipantRequest{
		CampaignId:  "c1",
		DisplayName: "Player 1",
		Role:        statev1.ParticipantRole_PLAYER,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestCreateParticipant_EmptyDisplayName(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	participantStore.participants["c1"] = map[string]participant.Participant{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
	}
	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusDraft,
	}

	eventStore := newFakeEventStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	ctx := contextWithParticipantID("owner-1")
	_, err := svc.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId: "c1",
		Role:       statev1.ParticipantRole_PLAYER,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateParticipant_InvalidRole(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	participantStore.participants["c1"] = map[string]participant.Participant{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
	}
	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusDraft,
	}

	eventStore := newFakeEventStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	ctx := contextWithParticipantID("owner-1")
	_, err := svc.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId:  "c1",
		DisplayName: "Player 1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateParticipant_Success_GM(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	participantStore.participants["c1"] = map[string]participant.Participant{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusDraft,
	}

	eventStore := newFakeEventStore()
	svc := &ParticipantService{
		stores:      Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("participant-123"),
	}

	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId:  "c1",
		DisplayName: "Game Master",
		Role:        statev1.ParticipantRole_GM,
		Controller:  statev1.Controller_CONTROLLER_AI,
	})
	if err != nil {
		t.Fatalf("CreateParticipant returned error: %v", err)
	}
	if resp.Participant == nil {
		t.Fatal("CreateParticipant response has nil participant")
	}
	if resp.Participant.Id != "participant-123" {
		t.Errorf("Participant ID = %q, want %q", resp.Participant.Id, "participant-123")
	}
	if resp.Participant.DisplayName != "Game Master" {
		t.Errorf("Participant DisplayName = %q, want %q", resp.Participant.DisplayName, "Game Master")
	}
	if resp.Participant.Role != statev1.ParticipantRole_GM {
		t.Errorf("Participant Role = %v, want %v", resp.Participant.Role, statev1.ParticipantRole_GM)
	}
	if resp.Participant.Controller != statev1.Controller_CONTROLLER_AI {
		t.Errorf("Participant Controller = %v, want %v", resp.Participant.Controller, statev1.Controller_CONTROLLER_AI)
	}
	if resp.Participant.CampaignAccess != statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER {
		t.Errorf("Participant CampaignAccess = %v, want %v", resp.Participant.CampaignAccess, statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeParticipantJoined {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeParticipantJoined)
	}

	// Verify persisted
	stored, err := participantStore.GetParticipant(context.Background(), "c1", "participant-123")
	if err != nil {
		t.Fatalf("Participant not persisted: %v", err)
	}
	if stored.DisplayName != "Game Master" {
		t.Errorf("Stored participant DisplayName = %q, want %q", stored.DisplayName, "Game Master")
	}
}

func TestCreateParticipant_Success_Player(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	participantStore.participants["c1"] = map[string]participant.Participant{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
	}
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}

	eventStore := newFakeEventStore()
	svc := &ParticipantService{
		stores:      Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("participant-456"),
	}

	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.CreateParticipant(ctx, &statev1.CreateParticipantRequest{
		CampaignId:  "c1",
		DisplayName: "Player One",
		Role:        statev1.ParticipantRole_PLAYER,
		Controller:  statev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("CreateParticipant returned error: %v", err)
	}
	if resp.Participant.Role != statev1.ParticipantRole_PLAYER {
		t.Errorf("Participant Role = %v, want %v", resp.Participant.Role, statev1.ParticipantRole_PLAYER)
	}
	if resp.Participant.CampaignAccess != statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER {
		t.Errorf("Participant CampaignAccess = %v, want %v", resp.Participant.CampaignAccess, statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeParticipantJoined {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeParticipantJoined)
	}
}

func TestUpdateParticipant_NoFields(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	participantStore.participants["c1"] = map[string]participant.Participant{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", DisplayName: "Player One", Role: participant.ParticipantRolePlayer, Controller: participant.ControllerHuman},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	ctx := contextWithParticipantID("owner-1")
	_, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateParticipant_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	participantStore.participants["c1"] = map[string]participant.Participant{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", DisplayName: "Player One", Role: participant.ParticipantRolePlayer, Controller: participant.ControllerHuman},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	ctx := contextWithParticipantID("owner-1")
	resp, err := svc.UpdateParticipant(ctx, &statev1.UpdateParticipantRequest{
		CampaignId:    "c1",
		ParticipantId: "p1",
		DisplayName:   wrapperspb.String("Player Uno"),
		Controller:    statev1.Controller_CONTROLLER_AI,
	})
	if err != nil {
		t.Fatalf("UpdateParticipant returned error: %v", err)
	}
	if resp.Participant.DisplayName != "Player Uno" {
		t.Errorf("Participant DisplayName = %q, want %q", resp.Participant.DisplayName, "Player Uno")
	}
	if resp.Participant.Controller != statev1.Controller_CONTROLLER_AI {
		t.Errorf("Participant Controller = %v, want %v", resp.Participant.Controller, statev1.Controller_CONTROLLER_AI)
	}

	stored, err := participantStore.GetParticipant(context.Background(), "c1", "p1")
	if err != nil {
		t.Fatalf("Participant not persisted: %v", err)
	}
	if stored.DisplayName != "Player Uno" {
		t.Errorf("Stored participant DisplayName = %q, want %q", stored.DisplayName, "Player Uno")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeParticipantUpdated {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeParticipantUpdated)
	}
}

func TestUpdateParticipant_CampaignAccess(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	participantStore.participants["c1"] = map[string]participant.Participant{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", DisplayName: "Player One", CampaignAccess: participant.CampaignAccessMember},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	ctx := contextWithParticipantID("owner-1")
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

func TestDeleteParticipant_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive, ParticipantCount: 1}
	participantStore.participants["c1"] = map[string]participant.Participant{
		"owner-1": {ID: "owner-1", CampaignID: "c1", CampaignAccess: participant.CampaignAccessOwner},
		"p1":      {ID: "p1", CampaignID: "c1", DisplayName: "Player One", Role: participant.ParticipantRolePlayer},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore, Event: eventStore})
	ctx := contextWithParticipantID("owner-1")
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
	if updatedCampaign.ParticipantCount != 0 {
		t.Errorf("ParticipantCount = %d, want 0", updatedCampaign.ParticipantCount)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeParticipantLeft {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeParticipantLeft)
	}
}

func TestListParticipants_NilRequest(t *testing.T) {
	svc := NewParticipantService(Stores{})
	_, err := svc.ListParticipants(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListParticipants_MissingCampaignId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.ListParticipants(context.Background(), &statev1.ListParticipantsRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListParticipants_CampaignNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.ListParticipants(context.Background(), &statev1.ListParticipantsRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestListParticipants_CampaignArchivedAllowed(t *testing.T) {
	// ListParticipants uses CampaignOpRead which is allowed for all campaign statuses,
	// including archived campaigns. This allows viewing historical campaign participants.
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusArchived,
	}
	participantStore.participants["c1"] = map[string]participant.Participant{
		"p1": {ID: "p1", CampaignID: "c1", DisplayName: "GM", Role: participant.ParticipantRoleGM, CreatedAt: now},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	resp, err := svc.ListParticipants(context.Background(), &statev1.ListParticipantsRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListParticipants returned error: %v", err)
	}
	if len(resp.Participants) != 1 {
		t.Errorf("ListParticipants returned %d participants, want 1", len(resp.Participants))
	}
}

func TestListParticipants_EmptyList(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	resp, err := svc.ListParticipants(context.Background(), &statev1.ListParticipantsRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListParticipants returned error: %v", err)
	}
	if len(resp.Participants) != 0 {
		t.Errorf("ListParticipants returned %d participants, want 0", len(resp.Participants))
	}
}

func TestListParticipants_WithParticipants(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}
	participantStore.participants["c1"] = map[string]participant.Participant{
		"p1": {ID: "p1", CampaignID: "c1", DisplayName: "GM", Role: participant.ParticipantRoleGM, CreatedAt: now},
		"p2": {ID: "p2", CampaignID: "c1", DisplayName: "Player 1", Role: participant.ParticipantRolePlayer, CreatedAt: now},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	resp, err := svc.ListParticipants(context.Background(), &statev1.ListParticipantsRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListParticipants returned error: %v", err)
	}
	if len(resp.Participants) != 2 {
		t.Errorf("ListParticipants returned %d participants, want 2", len(resp.Participants))
	}
}

func TestGetParticipant_NilRequest(t *testing.T) {
	svc := NewParticipantService(Stores{})
	_, err := svc.GetParticipant(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetParticipant_MissingCampaignId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetParticipant(context.Background(), &statev1.GetParticipantRequest{ParticipantId: "p1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetParticipant_MissingParticipantId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetParticipant(context.Background(), &statev1.GetParticipantRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetParticipant_CampaignNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetParticipant(context.Background(), &statev1.GetParticipantRequest{CampaignId: "c1", ParticipantId: "p1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetParticipant_ParticipantNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetParticipant(context.Background(), &statev1.GetParticipantRequest{CampaignId: "c1", ParticipantId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetParticipant_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}
	participantStore.participants["c1"] = map[string]participant.Participant{
		"p1": {
			ID:          "p1",
			CampaignID:  "c1",
			DisplayName: "Game Master",
			Role:        participant.ParticipantRoleGM,
			Controller:  participant.ControllerAI,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	svc := NewParticipantService(Stores{Campaign: campaignStore, Participant: participantStore})
	resp, err := svc.GetParticipant(context.Background(), &statev1.GetParticipantRequest{CampaignId: "c1", ParticipantId: "p1"})
	if err != nil {
		t.Fatalf("GetParticipant returned error: %v", err)
	}
	if resp.Participant == nil {
		t.Fatal("GetParticipant response has nil participant")
	}
	if resp.Participant.Id != "p1" {
		t.Errorf("Participant ID = %q, want %q", resp.Participant.Id, "p1")
	}
	if resp.Participant.DisplayName != "Game Master" {
		t.Errorf("Participant DisplayName = %q, want %q", resp.Participant.DisplayName, "Game Master")
	}
	if resp.Participant.Role != statev1.ParticipantRole_GM {
		t.Errorf("Participant Role = %v, want %v", resp.Participant.Role, statev1.ParticipantRole_GM)
	}
	if resp.Participant.Controller != statev1.Controller_CONTROLLER_AI {
		t.Errorf("Participant Controller = %v, want %v", resp.Participant.Controller, statev1.Controller_CONTROLLER_AI)
	}
}
