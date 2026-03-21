package invitetransport

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestRevokeInvite_Success(t *testing.T) {
	inviteStore := gametest.NewFakeInviteStore()
	inviteStore.Invites["invite-1"] = storage.InviteRecord{ID: "invite-1", CampaignID: "campaign-1", Status: invite.StatusPending}
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "campaign-1",
			Type:        event.Type("invite.revoked"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "owner-1",
			EntityType:  "invite",
			EntityID:    "invite-1",
			PayloadJSON: []byte(`{"invite_id":"invite-1"}`),
		}),
	}}

	svc := newServiceWithDependencies(
		Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Invite: inviteStore, Participant: participantStore, Campaign: campaignStore, Event: eventStore, Write: domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime}},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("x"),
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.RevokeInvite(ctx, &statev1.RevokeInviteRequest{InviteId: "invite-1"})
	if err != nil {
		t.Fatalf("RevokeInvite returned error: %v", err)
	}
	if resp.Invite == nil {
		t.Fatal("RevokeInvite response has nil invite")
	}
	if resp.Invite.Status != statev1.InviteStatus_REVOKED {
		t.Fatalf("invite status = %v, want REVOKED", resp.Invite.Status)
	}
	if len(eventStore.Events["campaign-1"]) != 1 {
		t.Fatalf("event count = %d, want 1", len(eventStore.Events["campaign-1"]))
	}
	if eventStore.Events["campaign-1"][0].Type != event.Type("invite.revoked") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["campaign-1"][0].Type, event.Type("invite.revoked"))
	}
}

func TestRevokeInvite_RequiresDomainEngine(t *testing.T) {
	inviteStore := gametest.NewFakeInviteStore()
	inviteStore.Invites["invite-1"] = storage.InviteRecord{ID: "invite-1", CampaignID: "campaign-1", Status: invite.StatusPending}
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}

	svc := newServiceWithDependencies(
		Deps{Auth: authz.PolicyDeps{Participant: participantStore}, Invite: inviteStore, Participant: participantStore, Campaign: campaignStore, Event: gametest.NewFakeEventStore()},
		gametest.FixedClock(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
		gametest.FixedIDGenerator("x"),
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	_, err := svc.RevokeInvite(ctx, &statev1.RevokeInviteRequest{InviteId: "invite-1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestRevokeInvite_UsesDomainEngine(t *testing.T) {
	inviteStore := gametest.NewFakeInviteStore()
	inviteStore.Invites["invite-1"] = storage.InviteRecord{ID: "invite-1", CampaignID: "campaign-1", Status: invite.StatusPending}
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["campaign-1"] = gametest.DraftCampaignRecord("campaign-1")
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["campaign-1"] = map[string]storage.ParticipantRecord{
		"owner-1": {ID: "owner-1", CampaignID: "campaign-1", CampaignAccess: participant.CampaignAccessOwner},
	}
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "campaign-1",
			Type:        event.Type("invite.revoked"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "owner-1",
			EntityType:  "invite",
			EntityID:    "invite-1",
			PayloadJSON: []byte(`{"invite_id":"invite-1"}`),
		}),
	}}

	svc := newServiceWithDependencies(
		Deps{
			Auth:        authz.PolicyDeps{Participant: participantStore},
			Invite:      inviteStore,
			Participant: participantStore,
			Campaign:    campaignStore,
			Event:       eventStore,
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("x"),
		nil,
		nil,
	)

	ctx := gametest.ContextWithParticipantID("owner-1")
	resp, err := svc.RevokeInvite(ctx, &statev1.RevokeInviteRequest{InviteId: "invite-1"})
	if err != nil {
		t.Fatalf("RevokeInvite returned error: %v", err)
	}
	if resp.Invite == nil {
		t.Fatal("RevokeInvite response has nil invite")
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("invite.revoke") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "invite.revoke")
	}
	if len(eventStore.Events["campaign-1"]) != 1 {
		t.Fatalf("event count = %d, want 1", len(eventStore.Events["campaign-1"]))
	}
	if eventStore.Events["campaign-1"][0].Type != event.Type("invite.revoked") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["campaign-1"][0].Type, event.Type("invite.revoked"))
	}
}
