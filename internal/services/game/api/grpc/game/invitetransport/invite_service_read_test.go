package invitetransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/testclients"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestListPendingInvites_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner, UserID: "user-1"},
		"seat-1":  {ID: "seat-1", CampaignID: "campaign-1", Name: "Seat 1", Role: participant.RolePlayer},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:                     "invite-1",
		CampaignID:             "campaign-1",
		ParticipantID:          "seat-1",
		Status:                 invite.StatusPending,
		CreatedByParticipantID: "owner-1",
	}
	inviteStore.Invites["invite-2"] = storage.InviteRecord{
		ID:            "invite-2",
		CampaignID:    "campaign-1",
		ParticipantID: "seat-1",
		Status:        invite.StatusClaimed,
	}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		nil,
		nil,
		&testclients.FakeAuthClient{User: &authv1.User{Id: "user-1", Username: "owner"}},
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.ListPendingInvites(ctx, &statev1.ListPendingInvitesRequest{CampaignId: "campaign-1"})
	if err != nil {
		t.Fatalf("ListPendingInvites returned error: %v", err)
	}
	if len(resp.Invites) != 1 {
		t.Fatalf("pending invite count = %d, want 1", len(resp.Invites))
	}
	entry := resp.Invites[0]
	if entry.Invite.Id != "invite-1" {
		t.Fatalf("invite id = %s, want invite-1", entry.Invite.Id)
	}
	if entry.Participant == nil || entry.Participant.Id != "seat-1" {
		t.Fatalf("participant id = %v, want seat-1", entry.Participant)
	}
	if entry.CreatedByUser == nil || entry.CreatedByUser.Id != "user-1" {
		t.Fatalf("created_by_user id = %v, want user-1", entry.CreatedByUser)
	}
}

func TestGetInvite_NilRequest(t *testing.T) {
	svc := NewService(Deps{}, nil)
	_, err := svc.GetInvite(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetInvite_MissingInviteId(t *testing.T) {
	svc := NewService(Deps{
		Invite:   gametest.NewFakeInviteStore(),
		Campaign: gametest.NewFakeCampaignStore(),
	}, nil)
	_, err := svc.GetInvite(context.Background(), &statev1.GetInviteRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetInvite_InviteNotFound(t *testing.T) {
	svc := NewService(Deps{
		Invite:   gametest.NewFakeInviteStore(),
		Campaign: gametest.NewFakeCampaignStore(),
	}, nil)
	_, err := svc.GetInvite(context.Background(), &statev1.GetInviteRequest{InviteId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetInvite_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:         "invite-1",
		CampaignID: "campaign-1",
		Status:     invite.StatusPending,
	}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		nil,
		nil,
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.GetInvite(ctx, &statev1.GetInviteRequest{InviteId: "invite-1"})
	if err != nil {
		t.Fatalf("GetInvite returned error: %v", err)
	}
	if resp.Invite == nil {
		t.Fatal("GetInvite response has nil invite")
	}
	if resp.Invite.Id != "invite-1" {
		t.Fatalf("invite id = %s, want invite-1", resp.Invite.Id)
	}
}

func TestGetInvite_MissingParticipantIdentity(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	inviteStore := gametest.NewFakeInviteStore()
	participantStore := gametest.NewFakeParticipantStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:         "invite-1",
		CampaignID: "campaign-1",
		Status:     invite.StatusPending,
	}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		nil,
		nil,
		nil,
		nil,
	)

	_, err := svc.GetInvite(context.Background(), &statev1.GetInviteRequest{InviteId: "invite-1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListInvites_NilRequest(t *testing.T) {
	svc := NewService(Deps{}, nil)
	_, err := svc.ListInvites(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListInvites_MissingCampaignId(t *testing.T) {
	svc := NewService(Deps{
		Invite:   gametest.NewFakeInviteStore(),
		Campaign: gametest.NewFakeCampaignStore(),
	}, nil)
	_, err := svc.ListInvites(context.Background(), &statev1.ListInvitesRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListInvites_CampaignNotFound(t *testing.T) {
	svc := NewService(Deps{
		Invite:   gametest.NewFakeInviteStore(),
		Campaign: gametest.NewFakeCampaignStore(),
	}, nil)
	_, err := svc.ListInvites(context.Background(), &statev1.ListInvitesRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestListInvites_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:         "invite-1",
		CampaignID: "campaign-1",
		Status:     invite.StatusPending,
	}
	inviteStore.Invites["invite-2"] = storage.InviteRecord{
		ID:         "invite-2",
		CampaignID: "campaign-1",
		Status:     invite.StatusClaimed,
	}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		nil,
		nil,
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.ListInvites(ctx, &statev1.ListInvitesRequest{CampaignId: "campaign-1"})
	if err != nil {
		t.Fatalf("ListInvites returned error: %v", err)
	}
	if len(resp.Invites) != 2 {
		t.Fatalf("invite count = %d, want 2", len(resp.Invites))
	}
}

func TestListInvites_WithStatusFilter(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:         "invite-1",
		CampaignID: "campaign-1",
		Status:     invite.StatusPending,
	}
	inviteStore.Invites["invite-2"] = storage.InviteRecord{
		ID:         "invite-2",
		CampaignID: "campaign-1",
		Status:     invite.StatusClaimed,
	}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		nil,
		nil,
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.ListInvites(ctx, &statev1.ListInvitesRequest{
		CampaignId: "campaign-1",
		Status:     statev1.InviteStatus_PENDING,
	})
	if err != nil {
		t.Fatalf("ListInvites returned error: %v", err)
	}
	if len(resp.Invites) != 1 {
		t.Fatalf("invite count = %d, want 1", len(resp.Invites))
	}
	if resp.Invites[0].Id != "invite-1" {
		t.Fatalf("invite id = %s, want invite-1", resp.Invites[0].Id)
	}
}

func TestListInvites_EmptyResult(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		nil,
		nil,
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.ListInvites(ctx, &statev1.ListInvitesRequest{CampaignId: "campaign-1"})
	if err != nil {
		t.Fatalf("ListInvites returned error: %v", err)
	}
	if len(resp.Invites) != 0 {
		t.Fatalf("invite count = %d, want 0", len(resp.Invites))
	}
}

func TestListPendingInvitesForUser_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	inviteStore := gametest.NewFakeInviteStore()

	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"seat-1": {ID: "seat-1", CampaignID: "campaign-1", Name: "Seat 1", Role: participant.RolePlayer},
	}
	inviteStore.Invites["invite-1"] = storage.InviteRecord{
		ID:              "invite-1",
		CampaignID:      "campaign-1",
		ParticipantID:   "seat-1",
		RecipientUserID: "user-1",
		Status:          invite.StatusPending,
		CreatedAt:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	inviteStore.Invites["invite-2"] = storage.InviteRecord{
		ID:              "invite-2",
		CampaignID:      "campaign-1",
		ParticipantID:   "seat-1",
		RecipientUserID: "user-2",
		Status:          invite.StatusPending,
	}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Campaign:    campaignStore,
			Participant: participantStore,
			Invite:      inviteStore,
		},
		nil,
		nil,
		nil,
		nil,
	)

	ctx := gametest.ContextWithUserID("user-1")
	resp, err := svc.ListPendingInvitesForUser(ctx, &statev1.ListPendingInvitesForUserRequest{})
	if err != nil {
		t.Fatalf("ListPendingInvitesForUser returned error: %v", err)
	}
	if len(resp.Invites) != 1 {
		t.Fatalf("pending invite count = %d, want 1", len(resp.Invites))
	}
	entry := resp.Invites[0]
	if entry.Invite.Id != "invite-1" {
		t.Fatalf("invite id = %s, want invite-1", entry.Invite.Id)
	}
	if entry.Campaign == nil || entry.Campaign.Id != "campaign-1" {
		t.Fatalf("campaign id = %v, want campaign-1", entry.Campaign)
	}
	if entry.Participant == nil || entry.Participant.Id != "seat-1" {
		t.Fatalf("participant id = %v, want seat-1", entry.Participant)
	}
}
