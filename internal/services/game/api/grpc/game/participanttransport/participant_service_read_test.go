package participanttransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestListParticipants_NilRequest(t *testing.T) {
	svc := NewService(Deps{})
	_, err := svc.ListParticipants(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListParticipants_MissingCampaignId(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.ListParticipants(context.Background(), &statev1.ListParticipantsRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListParticipants_CampaignNotFound(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.ListParticipants(context.Background(), &statev1.ListParticipantsRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestListParticipants_DeniesMissingIdentity(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			UserID:         "user-1",
			Name:           "GM",
			Role:           participant.RoleGM,
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.ListParticipants(context.Background(), &statev1.ListParticipantsRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListParticipants_CampaignArchivedAllowed(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.Campaigns["c1"] = gametest.ArchivedCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			Name:           "GM",
			Role:           participant.RoleGM,
			CampaignAccess: participant.CampaignAccessMember,
			CreatedAt:      now,
		},
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	resp, err := svc.ListParticipants(requestctx.WithParticipantID("p1"), &statev1.ListParticipantsRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListParticipants returned error: %v", err)
	}
	if len(resp.Participants) != 1 {
		t.Errorf("ListParticipants returned %d participants, want 1", len(resp.Participants))
	}
}

func TestListParticipants_DeniesNonMember(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			Name:           "GM",
			Role:           participant.RoleGM,
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.ListParticipants(requestctx.WithParticipantID("outsider-1"), &statev1.ListParticipantsRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListParticipants_WithParticipants(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			Name:           "GM",
			Role:           participant.RoleGM,
			CampaignAccess: participant.CampaignAccessMember,
			CreatedAt:      now,
		},
		"p2": {
			ID:             "p2",
			CampaignID:     "c1",
			Name:           "Player 1",
			Role:           participant.RolePlayer,
			CampaignAccess: participant.CampaignAccessMember,
			CreatedAt:      now,
		},
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	resp, err := svc.ListParticipants(requestctx.WithParticipantID("p1"), &statev1.ListParticipantsRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListParticipants returned error: %v", err)
	}
	if len(resp.Participants) != 2 {
		t.Errorf("ListParticipants returned %d participants, want 2", len(resp.Participants))
	}
}

func TestGetParticipant_NilRequest(t *testing.T) {
	svc := NewService(Deps{})
	_, err := svc.GetParticipant(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetParticipant_MissingCampaignId(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetParticipant(context.Background(), &statev1.GetParticipantRequest{ParticipantId: "p1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetParticipant_MissingParticipantId(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetParticipant(context.Background(), &statev1.GetParticipantRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetParticipant_CampaignNotFound(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetParticipant(context.Background(), &statev1.GetParticipantRequest{CampaignId: "c1", ParticipantId: "p1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetParticipant_DeniesMissingIdentity(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			Name:           "GM",
			Role:           participant.RoleGM,
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetParticipant(context.Background(), &statev1.GetParticipantRequest{CampaignId: "c1", ParticipantId: "p1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetParticipant_ParticipantNotFound(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			Name:           "GM",
			Role:           participant.RoleGM,
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	_, err := svc.GetParticipant(requestctx.WithParticipantID("p1"), &statev1.GetParticipantRequest{CampaignId: "c1", ParticipantId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetParticipant_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"p1": {
			ID:             "p1",
			CampaignID:     "c1",
			Name:           "Game Master",
			Role:           participant.RoleGM,
			Controller:     participant.ControllerAI,
			CampaignAccess: participant.CampaignAccessMember,
			CreatedAt:      now,
			UpdatedAt:      now,
		},
	}

	svc := NewService(Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Campaign: campaignStore, Participant: participantStore})
	resp, err := svc.GetParticipant(requestctx.WithParticipantID("p1"), &statev1.GetParticipantRequest{CampaignId: "c1", ParticipantId: "p1"})
	if err != nil {
		t.Fatalf("GetParticipant returned error: %v", err)
	}
	if resp.Participant == nil {
		t.Fatal("GetParticipant response has nil participant")
	}
	if resp.Participant.Id != "p1" {
		t.Errorf("Participant ID = %q, want %q", resp.Participant.Id, "p1")
	}
	if resp.Participant.Name != "Game Master" {
		t.Errorf("Participant Name = %q, want %q", resp.Participant.Name, "Game Master")
	}
	if resp.Participant.Role != statev1.ParticipantRole_GM {
		t.Errorf("Participant Role = %v, want %v", resp.Participant.Role, statev1.ParticipantRole_GM)
	}
	if resp.Participant.Controller != statev1.Controller_CONTROLLER_AI {
		t.Errorf("Participant Controller = %v, want %v", resp.Participant.Controller, statev1.Controller_CONTROLLER_AI)
	}
}
