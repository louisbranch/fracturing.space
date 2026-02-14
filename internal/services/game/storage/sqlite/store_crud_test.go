package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestCampaignPutGet(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 2, 9, 0, 0, 0, time.UTC)
	completed := now.Add(2 * time.Hour)
	archived := now.Add(24 * time.Hour)

	expected := campaign.Campaign{
		ID:               "camp-crud",
		Name:             "Shimmering Fields",
		System:           commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:           campaign.CampaignStatusCompleted,
		GmMode:           campaign.GmModeHybrid,
		ParticipantCount: 1,
		CharacterCount:   2,
		ThemePrompt:      "Drowned ruins",
		CreatedAt:        now,
		UpdatedAt:        now.Add(30 * time.Minute),
		CompletedAt:      &completed,
		ArchivedAt:       &archived,
	}

	if err := store.Put(context.Background(), expected); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	seedParticipant(t, store, expected.ID, "part-1", "user-1", now)
	if err := store.PutCharacter(context.Background(), character.Character{
		CampaignID: expected.ID,
		ID:         "char-1",
		Name:       "Aria",
		Kind:       character.CharacterKindPC,
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("put character: %v", err)
	}
	if err := store.PutCharacter(context.Background(), character.Character{
		CampaignID: expected.ID,
		ID:         "char-2",
		Name:       "Brim",
		Kind:       character.CharacterKindNPC,
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("put character: %v", err)
	}

	got, err := store.Get(context.Background(), expected.ID)
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}

	if got.ID != expected.ID || got.Name != expected.Name {
		t.Fatalf("expected campaign identity to match")
	}
	if got.System != expected.System || got.Status != expected.Status || got.GmMode != expected.GmMode {
		t.Fatalf("expected campaign metadata to match")
	}
	if got.ParticipantCount != expected.ParticipantCount || got.CharacterCount != expected.CharacterCount {
		t.Fatalf("expected campaign counts to match")
	}
	if got.ThemePrompt != expected.ThemePrompt {
		t.Fatalf("expected campaign theme prompt to match")
	}
	if !got.CreatedAt.Equal(expected.CreatedAt) || !got.UpdatedAt.Equal(expected.UpdatedAt) {
		t.Fatalf("expected campaign timestamps to match")
	}
	if got.CompletedAt == nil || !got.CompletedAt.Equal(*expected.CompletedAt) {
		t.Fatalf("expected campaign completed timestamp to match")
	}
	if got.ArchivedAt == nil || !got.ArchivedAt.Equal(*expected.ArchivedAt) {
		t.Fatalf("expected campaign archived timestamp to match")
	}
}

func TestParticipantClaimLifecycle(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 2, 10, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-claims", now)
	seedParticipant(t, store, "camp-claims", "part-1", "user-1", now)

	claimedAt := now.Add(5 * time.Minute)
	if err := store.PutParticipantClaim(context.Background(), "camp-claims", "user-1", "part-1", claimedAt); err != nil {
		t.Fatalf("put participant claim: %v", err)
	}

	claim, err := store.GetParticipantClaim(context.Background(), "camp-claims", "user-1")
	if err != nil {
		t.Fatalf("get participant claim: %v", err)
	}
	if claim.ParticipantID != "part-1" {
		t.Fatalf("expected participant id to match")
	}
	if !claim.ClaimedAt.Equal(claimedAt) {
		t.Fatalf("expected claim timestamp to match")
	}

	if err := store.DeleteParticipantClaim(context.Background(), "camp-claims", "user-1"); err != nil {
		t.Fatalf("delete participant claim: %v", err)
	}

	_, err = store.GetParticipantClaim(context.Background(), "camp-claims", "user-1")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete")
	}
}

func TestInviteListingAndUpdate(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 2, 11, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-invites", now)
	seedParticipant(t, store, "camp-invites", "seat-1", "user-1", now)
	seedParticipant(t, store, "camp-invites", "seat-2", "user-2", now)

	invitePending := invite.Invite{
		ID:                     "inv-1",
		CampaignID:             "camp-invites",
		ParticipantID:          "seat-1",
		RecipientUserID:        "user-1",
		Status:                 invite.StatusPending,
		CreatedByParticipantID: "seat-1",
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	inviteClaimed := invite.Invite{
		ID:                     "inv-2",
		CampaignID:             "camp-invites",
		ParticipantID:          "seat-2",
		RecipientUserID:        "user-2",
		Status:                 invite.StatusClaimed,
		CreatedByParticipantID: "seat-2",
		CreatedAt:              now,
		UpdatedAt:              now,
	}

	if err := store.PutInvite(context.Background(), invitePending); err != nil {
		t.Fatalf("put invite pending: %v", err)
	}
	if err := store.PutInvite(context.Background(), inviteClaimed); err != nil {
		t.Fatalf("put invite claimed: %v", err)
	}

	pendingPage, err := store.ListPendingInvites(context.Background(), "camp-invites", 10, "")
	if err != nil {
		t.Fatalf("list pending invites: %v", err)
	}
	if len(pendingPage.Invites) != 1 || pendingPage.Invites[0].ID != invitePending.ID {
		t.Fatalf("expected pending invite to be listed")
	}

	recipientPage, err := store.ListPendingInvitesForRecipient(context.Background(), "user-1", 10, "")
	if err != nil {
		t.Fatalf("list pending invites for recipient: %v", err)
	}
	if len(recipientPage.Invites) != 1 || recipientPage.Invites[0].ID != invitePending.ID {
		t.Fatalf("expected pending invite for recipient to be listed")
	}

	claimedPage, err := store.ListInvites(context.Background(), "camp-invites", "", invite.StatusClaimed, 10, "")
	if err != nil {
		t.Fatalf("list claimed invites: %v", err)
	}
	if len(claimedPage.Invites) != 1 || claimedPage.Invites[0].ID != inviteClaimed.ID {
		t.Fatalf("expected claimed invite to be listed")
	}

	updatedAt := now.Add(10 * time.Minute)
	if err := store.UpdateInviteStatus(context.Background(), invitePending.ID, invite.StatusClaimed, updatedAt); err != nil {
		t.Fatalf("update invite status: %v", err)
	}

	got, err := store.GetInvite(context.Background(), invitePending.ID)
	if err != nil {
		t.Fatalf("get invite: %v", err)
	}
	if got.Status != invite.StatusClaimed {
		t.Fatalf("expected invite status to update")
	}
	if !got.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("expected invite updated timestamp to match")
	}
}

func TestSessionLifecycle(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-sessions", now)

	sess := session.Session{
		ID:         "sess-1",
		CampaignID: "camp-sessions",
		Name:       "First Session",
		Status:     session.SessionStatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
	}
	if err := store.PutSession(context.Background(), sess); err != nil {
		t.Fatalf("put session: %v", err)
	}

	active, err := store.GetActiveSession(context.Background(), "camp-sessions")
	if err != nil {
		t.Fatalf("get active session: %v", err)
	}
	if active.ID != sess.ID || active.Status != session.SessionStatusActive {
		t.Fatalf("expected active session to match")
	}

	other := session.Session{
		ID:         "sess-2",
		CampaignID: "camp-sessions",
		Name:       "Second Session",
		Status:     session.SessionStatusActive,
		StartedAt:  now.Add(30 * time.Minute),
		UpdatedAt:  now.Add(30 * time.Minute),
	}
	err = store.PutSession(context.Background(), other)
	if err == nil || !errors.Is(err, storage.ErrActiveSessionExists) {
		t.Fatalf("expected active session conflict")
	}

	endedAt := now.Add(2 * time.Hour)
	ended, transitioned, err := store.EndSession(context.Background(), "camp-sessions", sess.ID, endedAt)
	if err != nil {
		t.Fatalf("end session: %v", err)
	}
	if !transitioned {
		t.Fatalf("expected session to transition to ended")
	}
	if ended.Status != session.SessionStatusEnded {
		t.Fatalf("expected session status to be ended")
	}
	if ended.EndedAt == nil || !ended.EndedAt.Equal(endedAt.UTC()) {
		t.Fatalf("expected ended timestamp to match")
	}

	_, err = store.GetActiveSession(context.Background(), "camp-sessions")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected no active session after ending")
	}
}

func TestSessionGateAndSpotlight(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 2, 13, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-gates", now)

	gate := storage.SessionGate{
		CampaignID:         "camp-gates",
		SessionID:          "sess-1",
		GateID:             "gate-1",
		GateType:           "prompt",
		Status:             "open",
		Reason:             "Need consent",
		CreatedAt:          now,
		CreatedByActorType: "system",
		CreatedByActorID:   "",
		MetadataJSON:       []byte("{\"flag\":true}"),
		ResolutionJSON:     []byte("{}"),
	}

	if err := store.PutSessionGate(context.Background(), gate); err != nil {
		t.Fatalf("put session gate: %v", err)
	}

	gotGate, err := store.GetSessionGate(context.Background(), gate.CampaignID, gate.SessionID, gate.GateID)
	if err != nil {
		t.Fatalf("get session gate: %v", err)
	}
	if gotGate.GateType != gate.GateType || gotGate.Status != gate.Status {
		t.Fatalf("expected session gate to match")
	}
	if string(gotGate.MetadataJSON) != string(gate.MetadataJSON) {
		t.Fatalf("expected session gate metadata to match")
	}

	openGate, err := store.GetOpenSessionGate(context.Background(), gate.CampaignID, gate.SessionID)
	if err != nil {
		t.Fatalf("get open session gate: %v", err)
	}
	if openGate.GateID != gate.GateID {
		t.Fatalf("expected open session gate to match")
	}

	spotlight := storage.SessionSpotlight{
		CampaignID:         "camp-gates",
		SessionID:          "sess-1",
		SpotlightType:      "character",
		CharacterID:        "char-1",
		UpdatedAt:          now,
		UpdatedByActorType: "participant",
		UpdatedByActorID:   "part-1",
	}
	if err := store.PutSessionSpotlight(context.Background(), spotlight); err != nil {
		t.Fatalf("put session spotlight: %v", err)
	}

	gotSpotlight, err := store.GetSessionSpotlight(context.Background(), spotlight.CampaignID, spotlight.SessionID)
	if err != nil {
		t.Fatalf("get session spotlight: %v", err)
	}
	if gotSpotlight.SpotlightType != spotlight.SpotlightType || gotSpotlight.CharacterID != spotlight.CharacterID {
		t.Fatalf("expected session spotlight to match")
	}

	if err := store.ClearSessionSpotlight(context.Background(), spotlight.CampaignID, spotlight.SessionID); err != nil {
		t.Fatalf("clear session spotlight: %v", err)
	}
	_, err = store.GetSessionSpotlight(context.Background(), spotlight.CampaignID, spotlight.SessionID)
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected no session spotlight after clear")
	}
}

func seedCampaign(t *testing.T, store *Store, id string, now time.Time) campaign.Campaign {
	t.Helper()

	c := campaign.Campaign{
		ID:        id,
		Name:      "Campaign",
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.CampaignStatusActive,
		GmMode:    campaign.GmModeHuman,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Put(context.Background(), c); err != nil {
		t.Fatalf("seed campaign: %v", err)
	}
	return c
}

func seedParticipant(t *testing.T, store *Store, campaignID, participantID, userID string, now time.Time) participant.Participant {
	t.Helper()

	p := participant.Participant{
		CampaignID:     campaignID,
		ID:             participantID,
		UserID:         userID,
		DisplayName:    participantID,
		Role:           participant.ParticipantRolePlayer,
		Controller:     participant.ControllerHuman,
		CampaignAccess: participant.CampaignAccessMember,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := store.PutParticipant(context.Background(), p); err != nil {
		t.Fatalf("seed participant: %v", err)
	}
	return p
}
