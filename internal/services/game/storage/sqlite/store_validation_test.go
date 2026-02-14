package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestNilStoreErrors(t *testing.T) {
	ctx := context.Background()
	var s *Store

	if err := s.Put(ctx, campaign.Campaign{ID: "x"}); err == nil {
		t.Fatal("expected error from nil store Put")
	}
	if _, err := s.Get(ctx, "x"); err == nil {
		t.Fatal("expected error from nil store Get")
	}
	if _, err := s.List(ctx, 10, ""); err == nil {
		t.Fatal("expected error from nil store List")
	}
	if err := s.PutParticipant(ctx, participant.Participant{CampaignID: "c", ID: "p"}); err == nil {
		t.Fatal("expected error from nil store PutParticipant")
	}
	if err := s.DeleteParticipant(ctx, "c", "p"); err == nil {
		t.Fatal("expected error from nil store DeleteParticipant")
	}
	if _, err := s.GetParticipant(ctx, "c", "p"); err == nil {
		t.Fatal("expected error from nil store GetParticipant")
	}
	if _, err := s.ListParticipantsByCampaign(ctx, "c"); err == nil {
		t.Fatal("expected error from nil store ListParticipantsByCampaign")
	}
	if _, err := s.ListParticipants(ctx, "c", 10, ""); err == nil {
		t.Fatal("expected error from nil store ListParticipants")
	}
	if err := s.PutParticipantClaim(ctx, "c", "u", "p", time.Now()); err == nil {
		t.Fatal("expected error from nil store PutParticipantClaim")
	}
	if _, err := s.GetParticipantClaim(ctx, "c", "u"); err == nil {
		t.Fatal("expected error from nil store GetParticipantClaim")
	}
	if err := s.DeleteParticipantClaim(ctx, "c", "u"); err == nil {
		t.Fatal("expected error from nil store DeleteParticipantClaim")
	}
	if err := s.PutInvite(ctx, invite.Invite{ID: "i", CampaignID: "c", ParticipantID: "p"}); err == nil {
		t.Fatal("expected error from nil store PutInvite")
	}
	if _, err := s.GetInvite(ctx, "i"); err == nil {
		t.Fatal("expected error from nil store GetInvite")
	}
	if _, err := s.ListInvites(ctx, "c", "", invite.StatusPending, 10, ""); err == nil {
		t.Fatal("expected error from nil store ListInvites")
	}
	if _, err := s.ListPendingInvites(ctx, "c", 10, ""); err == nil {
		t.Fatal("expected error from nil store ListPendingInvites")
	}
	if _, err := s.ListPendingInvitesForRecipient(ctx, "u", 10, ""); err == nil {
		t.Fatal("expected error from nil store ListPendingInvitesForRecipient")
	}
	if err := s.UpdateInviteStatus(ctx, "i", invite.StatusClaimed, time.Now()); err == nil {
		t.Fatal("expected error from nil store UpdateInviteStatus")
	}
	if err := s.PutCharacter(ctx, character.Character{CampaignID: "c", ID: "ch"}); err == nil {
		t.Fatal("expected error from nil store PutCharacter")
	}
	if _, err := s.GetCharacter(ctx, "c", "ch"); err == nil {
		t.Fatal("expected error from nil store GetCharacter")
	}
	if err := s.DeleteCharacter(ctx, "c", "ch"); err == nil {
		t.Fatal("expected error from nil store DeleteCharacter")
	}
	if _, err := s.ListCharacters(ctx, "c", 10, ""); err == nil {
		t.Fatal("expected error from nil store ListCharacters")
	}
	if err := s.PutSession(ctx, session.Session{CampaignID: "c", ID: "s"}); err == nil {
		t.Fatal("expected error from nil store PutSession")
	}
	if _, _, err := s.EndSession(ctx, "c", "s", time.Now()); err == nil {
		t.Fatal("expected error from nil store EndSession")
	}
	if _, err := s.GetSession(ctx, "c", "s"); err == nil {
		t.Fatal("expected error from nil store GetSession")
	}
	if _, err := s.GetActiveSession(ctx, "c"); err == nil {
		t.Fatal("expected error from nil store GetActiveSession")
	}
	if _, err := s.ListSessions(ctx, "c", 10, ""); err == nil {
		t.Fatal("expected error from nil store ListSessions")
	}
	if err := s.PutSessionGate(ctx, storage.SessionGate{CampaignID: "c", SessionID: "s", GateID: "g", GateType: "t", Status: "open"}); err == nil {
		t.Fatal("expected error from nil store PutSessionGate")
	}
	if _, err := s.GetSessionGate(ctx, "c", "s", "g"); err == nil {
		t.Fatal("expected error from nil store GetSessionGate")
	}
	if _, err := s.GetOpenSessionGate(ctx, "c", "s"); err == nil {
		t.Fatal("expected error from nil store GetOpenSessionGate")
	}
	if err := s.PutSessionSpotlight(ctx, storage.SessionSpotlight{CampaignID: "c", SessionID: "s", SpotlightType: "t"}); err == nil {
		t.Fatal("expected error from nil store PutSessionSpotlight")
	}
	if _, err := s.GetSessionSpotlight(ctx, "c", "s"); err == nil {
		t.Fatal("expected error from nil store GetSessionSpotlight")
	}
	if err := s.ClearSessionSpotlight(ctx, "c", "s"); err == nil {
		t.Fatal("expected error from nil store ClearSessionSpotlight")
	}
	if err := s.PutSnapshot(ctx, storage.Snapshot{CampaignID: "c", SessionID: "s"}); err == nil {
		t.Fatal("expected error from nil store PutSnapshot")
	}
	if _, err := s.GetSnapshot(ctx, "c", "s"); err == nil {
		t.Fatal("expected error from nil store GetSnapshot")
	}
	if _, err := s.GetLatestSnapshot(ctx, "c"); err == nil {
		t.Fatal("expected error from nil store GetLatestSnapshot")
	}
	if _, err := s.ListSnapshots(ctx, "c", 10); err == nil {
		t.Fatal("expected error from nil store ListSnapshots")
	}
	if err := s.AppendTelemetryEvent(ctx, storage.TelemetryEvent{EventName: "e", Severity: "info"}); err == nil {
		t.Fatal("expected error from nil store AppendTelemetryEvent")
	}
	if _, err := s.GetGameStatistics(ctx, nil); err == nil {
		t.Fatal("expected error from nil store GetGameStatistics")
	}
	if _, err := s.GetCampaignForkMetadata(ctx, "c"); err == nil {
		t.Fatal("expected error from nil store GetCampaignForkMetadata")
	}
	if err := s.SetCampaignForkMetadata(ctx, "c", storage.ForkMetadata{}); err == nil {
		t.Fatal("expected error from nil store SetCampaignForkMetadata")
	}
}

func TestCancelledContextErrors(t *testing.T) {
	store := openTestStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := store.Put(ctx, campaign.Campaign{ID: "x"}); err == nil {
		t.Fatal("expected context error from Put")
	}
	if _, err := store.Get(ctx, "x"); err == nil {
		t.Fatal("expected context error from Get")
	}
	if _, err := store.List(ctx, 10, ""); err == nil {
		t.Fatal("expected context error from List")
	}
	if err := store.PutParticipant(ctx, participant.Participant{CampaignID: "c", ID: "p"}); err == nil {
		t.Fatal("expected context error from PutParticipant")
	}
	if err := store.DeleteParticipant(ctx, "c", "p"); err == nil {
		t.Fatal("expected context error from DeleteParticipant")
	}
	if _, err := store.GetParticipant(ctx, "c", "p"); err == nil {
		t.Fatal("expected context error from GetParticipant")
	}
	if _, err := store.ListParticipantsByCampaign(ctx, "c"); err == nil {
		t.Fatal("expected context error from ListParticipantsByCampaign")
	}
	if _, err := store.ListParticipants(ctx, "c", 10, ""); err == nil {
		t.Fatal("expected context error from ListParticipants")
	}
	if err := store.PutParticipantClaim(ctx, "c", "u", "p", time.Now()); err == nil {
		t.Fatal("expected context error from PutParticipantClaim")
	}
	if _, err := store.GetParticipantClaim(ctx, "c", "u"); err == nil {
		t.Fatal("expected context error from GetParticipantClaim")
	}
	if err := store.DeleteParticipantClaim(ctx, "c", "u"); err == nil {
		t.Fatal("expected context error from DeleteParticipantClaim")
	}
	if err := store.PutInvite(ctx, invite.Invite{ID: "i", CampaignID: "c", ParticipantID: "p"}); err == nil {
		t.Fatal("expected context error from PutInvite")
	}
	if _, err := store.GetInvite(ctx, "i"); err == nil {
		t.Fatal("expected context error from GetInvite")
	}
	if _, err := store.ListInvites(ctx, "c", "", invite.StatusPending, 10, ""); err == nil {
		t.Fatal("expected context error from ListInvites")
	}
	if _, err := store.ListPendingInvites(ctx, "c", 10, ""); err == nil {
		t.Fatal("expected context error from ListPendingInvites")
	}
	if _, err := store.ListPendingInvitesForRecipient(ctx, "u", 10, ""); err == nil {
		t.Fatal("expected context error from ListPendingInvitesForRecipient")
	}
	if err := store.UpdateInviteStatus(ctx, "i", invite.StatusClaimed, time.Now()); err == nil {
		t.Fatal("expected context error from UpdateInviteStatus")
	}
	if err := store.PutCharacter(ctx, character.Character{CampaignID: "c", ID: "ch"}); err == nil {
		t.Fatal("expected context error from PutCharacter")
	}
	if _, err := store.GetCharacter(ctx, "c", "ch"); err == nil {
		t.Fatal("expected context error from GetCharacter")
	}
	if err := store.DeleteCharacter(ctx, "c", "ch"); err == nil {
		t.Fatal("expected context error from DeleteCharacter")
	}
	if _, err := store.ListCharacters(ctx, "c", 10, ""); err == nil {
		t.Fatal("expected context error from ListCharacters")
	}
	if err := store.PutSession(ctx, session.Session{CampaignID: "c", ID: "s"}); err == nil {
		t.Fatal("expected context error from PutSession")
	}
	if _, _, err := store.EndSession(ctx, "c", "s", time.Now()); err == nil {
		t.Fatal("expected context error from EndSession")
	}
	if _, err := store.GetSession(ctx, "c", "s"); err == nil {
		t.Fatal("expected context error from GetSession")
	}
	if _, err := store.GetActiveSession(ctx, "c"); err == nil {
		t.Fatal("expected context error from GetActiveSession")
	}
	if _, err := store.ListSessions(ctx, "c", 10, ""); err == nil {
		t.Fatal("expected context error from ListSessions")
	}
	if err := store.PutSessionGate(ctx, storage.SessionGate{CampaignID: "c", SessionID: "s", GateID: "g", GateType: "t", Status: "open"}); err == nil {
		t.Fatal("expected context error from PutSessionGate")
	}
	if _, err := store.GetSessionGate(ctx, "c", "s", "g"); err == nil {
		t.Fatal("expected context error from GetSessionGate")
	}
	if _, err := store.GetOpenSessionGate(ctx, "c", "s"); err == nil {
		t.Fatal("expected context error from GetOpenSessionGate")
	}
	if err := store.PutSessionSpotlight(ctx, storage.SessionSpotlight{CampaignID: "c", SessionID: "s", SpotlightType: "t"}); err == nil {
		t.Fatal("expected context error from PutSessionSpotlight")
	}
	if _, err := store.GetSessionSpotlight(ctx, "c", "s"); err == nil {
		t.Fatal("expected context error from GetSessionSpotlight")
	}
	if err := store.ClearSessionSpotlight(ctx, "c", "s"); err == nil {
		t.Fatal("expected context error from ClearSessionSpotlight")
	}
	if err := store.PutSnapshot(ctx, storage.Snapshot{CampaignID: "c", SessionID: "s"}); err == nil {
		t.Fatal("expected context error from PutSnapshot")
	}
	if _, err := store.GetSnapshot(ctx, "c", "s"); err == nil {
		t.Fatal("expected context error from GetSnapshot")
	}
	if _, err := store.GetLatestSnapshot(ctx, "c"); err == nil {
		t.Fatal("expected context error from GetLatestSnapshot")
	}
	if _, err := store.ListSnapshots(ctx, "c", 10); err == nil {
		t.Fatal("expected context error from ListSnapshots")
	}
	if err := store.AppendTelemetryEvent(ctx, storage.TelemetryEvent{EventName: "e", Severity: "info"}); err == nil {
		t.Fatal("expected context error from AppendTelemetryEvent")
	}
	if _, err := store.GetGameStatistics(ctx, nil); err == nil {
		t.Fatal("expected context error from GetGameStatistics")
	}
	if _, err := store.GetCampaignForkMetadata(ctx, "c"); err == nil {
		t.Fatal("expected context error from GetCampaignForkMetadata")
	}
	if err := store.SetCampaignForkMetadata(ctx, "c", storage.ForkMetadata{}); err == nil {
		t.Fatal("expected context error from SetCampaignForkMetadata")
	}
}

func TestEmptyIDValidation(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	// Campaign
	if err := store.Put(ctx, campaign.Campaign{}); err == nil {
		t.Fatal("expected error for empty campaign ID in Put")
	}
	if _, err := store.Get(ctx, ""); err == nil {
		t.Fatal("expected error for empty campaign ID in Get")
	}
	if _, err := store.Get(ctx, "  "); err == nil {
		t.Fatal("expected error for whitespace campaign ID in Get")
	}
	if _, err := store.List(ctx, 0, ""); err == nil {
		t.Fatal("expected error for zero page size in List")
	}

	// Participant
	if err := store.PutParticipant(ctx, participant.Participant{ID: "p"}); err == nil {
		t.Fatal("expected error for empty campaign ID in PutParticipant")
	}
	if err := store.PutParticipant(ctx, participant.Participant{CampaignID: "c"}); err == nil {
		t.Fatal("expected error for empty participant ID in PutParticipant")
	}
	if err := store.DeleteParticipant(ctx, "", "p"); err == nil {
		t.Fatal("expected error for empty campaign ID in DeleteParticipant")
	}
	if err := store.DeleteParticipant(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty participant ID in DeleteParticipant")
	}
	if _, err := store.GetParticipant(ctx, "", "p"); err == nil {
		t.Fatal("expected error for empty campaign ID in GetParticipant")
	}
	if _, err := store.GetParticipant(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty participant ID in GetParticipant")
	}
	if _, err := store.ListParticipantsByCampaign(ctx, ""); err == nil {
		t.Fatal("expected error for empty campaign ID in ListParticipantsByCampaign")
	}
	if _, err := store.ListParticipants(ctx, "", 10, ""); err == nil {
		t.Fatal("expected error for empty campaign ID in ListParticipants")
	}
	if _, err := store.ListParticipants(ctx, "c", 0, ""); err == nil {
		t.Fatal("expected error for zero page size in ListParticipants")
	}

	// Participant Claim
	if err := store.PutParticipantClaim(ctx, "", "u", "p", time.Now()); err == nil {
		t.Fatal("expected error for empty campaign ID in PutParticipantClaim")
	}
	if err := store.PutParticipantClaim(ctx, "c", "", "p", time.Now()); err == nil {
		t.Fatal("expected error for empty user ID in PutParticipantClaim")
	}
	if err := store.PutParticipantClaim(ctx, "c", "u", "", time.Now()); err == nil {
		t.Fatal("expected error for empty participant ID in PutParticipantClaim")
	}
	if _, err := store.GetParticipantClaim(ctx, "", "u"); err == nil {
		t.Fatal("expected error for empty campaign ID in GetParticipantClaim")
	}
	if _, err := store.GetParticipantClaim(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty user ID in GetParticipantClaim")
	}
	if err := store.DeleteParticipantClaim(ctx, "", "u"); err == nil {
		t.Fatal("expected error for empty campaign ID in DeleteParticipantClaim")
	}
	if err := store.DeleteParticipantClaim(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty user ID in DeleteParticipantClaim")
	}

	// Invite
	if err := store.PutInvite(ctx, invite.Invite{CampaignID: "c", ParticipantID: "p"}); err == nil {
		t.Fatal("expected error for empty invite ID in PutInvite")
	}
	if err := store.PutInvite(ctx, invite.Invite{ID: "i", ParticipantID: "p"}); err == nil {
		t.Fatal("expected error for empty campaign ID in PutInvite")
	}
	if err := store.PutInvite(ctx, invite.Invite{ID: "i", CampaignID: "c"}); err == nil {
		t.Fatal("expected error for empty participant ID in PutInvite")
	}
	if _, err := store.GetInvite(ctx, ""); err == nil {
		t.Fatal("expected error for empty invite ID in GetInvite")
	}
	if _, err := store.ListInvites(ctx, "", "", invite.StatusPending, 10, ""); err == nil {
		t.Fatal("expected error for empty campaign ID in ListInvites")
	}
	if _, err := store.ListInvites(ctx, "c", "", invite.StatusPending, 0, ""); err == nil {
		t.Fatal("expected error for zero page size in ListInvites")
	}
	if _, err := store.ListPendingInvites(ctx, "", 10, ""); err == nil {
		t.Fatal("expected error for empty campaign ID in ListPendingInvites")
	}
	if _, err := store.ListPendingInvites(ctx, "c", 0, ""); err == nil {
		t.Fatal("expected error for zero page size in ListPendingInvites")
	}
	if _, err := store.ListPendingInvitesForRecipient(ctx, "", 10, ""); err == nil {
		t.Fatal("expected error for empty user ID in ListPendingInvitesForRecipient")
	}
	if _, err := store.ListPendingInvitesForRecipient(ctx, "u", 0, ""); err == nil {
		t.Fatal("expected error for zero page size in ListPendingInvitesForRecipient")
	}
	if err := store.UpdateInviteStatus(ctx, "", invite.StatusClaimed, time.Now()); err == nil {
		t.Fatal("expected error for empty invite ID in UpdateInviteStatus")
	}

	// Character
	if err := store.PutCharacter(ctx, character.Character{ID: "ch"}); err == nil {
		t.Fatal("expected error for empty campaign ID in PutCharacter")
	}
	if err := store.PutCharacter(ctx, character.Character{CampaignID: "c"}); err == nil {
		t.Fatal("expected error for empty character ID in PutCharacter")
	}
	if _, err := store.GetCharacter(ctx, "", "ch"); err == nil {
		t.Fatal("expected error for empty campaign ID in GetCharacter")
	}
	if _, err := store.GetCharacter(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty character ID in GetCharacter")
	}
	if err := store.DeleteCharacter(ctx, "", "ch"); err == nil {
		t.Fatal("expected error for empty campaign ID in DeleteCharacter")
	}
	if err := store.DeleteCharacter(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty character ID in DeleteCharacter")
	}
	if _, err := store.ListCharacters(ctx, "", 10, ""); err == nil {
		t.Fatal("expected error for empty campaign ID in ListCharacters")
	}
	if _, err := store.ListCharacters(ctx, "c", 0, ""); err == nil {
		t.Fatal("expected error for zero page size in ListCharacters")
	}

	// Session
	if err := store.PutSession(ctx, session.Session{ID: "s"}); err == nil {
		t.Fatal("expected error for empty campaign ID in PutSession")
	}
	if err := store.PutSession(ctx, session.Session{CampaignID: "c"}); err == nil {
		t.Fatal("expected error for empty session ID in PutSession")
	}
	if _, _, err := store.EndSession(ctx, "", "s", time.Now()); err == nil {
		t.Fatal("expected error for empty campaign ID in EndSession")
	}
	if _, _, err := store.EndSession(ctx, "c", "", time.Now()); err == nil {
		t.Fatal("expected error for empty session ID in EndSession")
	}
	if _, err := store.GetSession(ctx, "", "s"); err == nil {
		t.Fatal("expected error for empty campaign ID in GetSession")
	}
	if _, err := store.GetSession(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty session ID in GetSession")
	}
	if _, err := store.GetActiveSession(ctx, ""); err == nil {
		t.Fatal("expected error for empty campaign ID in GetActiveSession")
	}
	if _, err := store.ListSessions(ctx, "", 10, ""); err == nil {
		t.Fatal("expected error for empty campaign ID in ListSessions")
	}
	if _, err := store.ListSessions(ctx, "c", 0, ""); err == nil {
		t.Fatal("expected error for zero page size in ListSessions")
	}

	// Session Gate
	if err := store.PutSessionGate(ctx, storage.SessionGate{SessionID: "s", GateID: "g", GateType: "t", Status: "open"}); err == nil {
		t.Fatal("expected error for empty campaign ID in PutSessionGate")
	}
	if err := store.PutSessionGate(ctx, storage.SessionGate{CampaignID: "c", GateID: "g", GateType: "t", Status: "open"}); err == nil {
		t.Fatal("expected error for empty session ID in PutSessionGate")
	}
	if err := store.PutSessionGate(ctx, storage.SessionGate{CampaignID: "c", SessionID: "s", GateType: "t", Status: "open"}); err == nil {
		t.Fatal("expected error for empty gate ID in PutSessionGate")
	}
	if err := store.PutSessionGate(ctx, storage.SessionGate{CampaignID: "c", SessionID: "s", GateID: "g", Status: "open"}); err == nil {
		t.Fatal("expected error for empty gate type in PutSessionGate")
	}
	if err := store.PutSessionGate(ctx, storage.SessionGate{CampaignID: "c", SessionID: "s", GateID: "g", GateType: "t"}); err == nil {
		t.Fatal("expected error for empty gate status in PutSessionGate")
	}
	if _, err := store.GetSessionGate(ctx, "", "s", "g"); err == nil {
		t.Fatal("expected error for empty campaign ID in GetSessionGate")
	}
	if _, err := store.GetSessionGate(ctx, "c", "", "g"); err == nil {
		t.Fatal("expected error for empty session ID in GetSessionGate")
	}
	if _, err := store.GetSessionGate(ctx, "c", "s", ""); err == nil {
		t.Fatal("expected error for empty gate ID in GetSessionGate")
	}
	if _, err := store.GetOpenSessionGate(ctx, "", "s"); err == nil {
		t.Fatal("expected error for empty campaign ID in GetOpenSessionGate")
	}
	if _, err := store.GetOpenSessionGate(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty session ID in GetOpenSessionGate")
	}

	// Session Spotlight
	if err := store.PutSessionSpotlight(ctx, storage.SessionSpotlight{SessionID: "s", SpotlightType: "t"}); err == nil {
		t.Fatal("expected error for empty campaign ID in PutSessionSpotlight")
	}
	if err := store.PutSessionSpotlight(ctx, storage.SessionSpotlight{CampaignID: "c", SpotlightType: "t"}); err == nil {
		t.Fatal("expected error for empty session ID in PutSessionSpotlight")
	}
	if err := store.PutSessionSpotlight(ctx, storage.SessionSpotlight{CampaignID: "c", SessionID: "s"}); err == nil {
		t.Fatal("expected error for empty spotlight type in PutSessionSpotlight")
	}
	if _, err := store.GetSessionSpotlight(ctx, "", "s"); err == nil {
		t.Fatal("expected error for empty campaign ID in GetSessionSpotlight")
	}
	if _, err := store.GetSessionSpotlight(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty session ID in GetSessionSpotlight")
	}
	if err := store.ClearSessionSpotlight(ctx, "", "s"); err == nil {
		t.Fatal("expected error for empty campaign ID in ClearSessionSpotlight")
	}
	if err := store.ClearSessionSpotlight(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty session ID in ClearSessionSpotlight")
	}

	// Snapshot
	if err := store.PutSnapshot(ctx, storage.Snapshot{SessionID: "s"}); err == nil {
		t.Fatal("expected error for empty campaign ID in PutSnapshot")
	}
	if err := store.PutSnapshot(ctx, storage.Snapshot{CampaignID: "c"}); err == nil {
		t.Fatal("expected error for empty session ID in PutSnapshot")
	}
	if _, err := store.GetSnapshot(ctx, "", "s"); err == nil {
		t.Fatal("expected error for empty campaign ID in GetSnapshot")
	}
	if _, err := store.GetSnapshot(ctx, "c", ""); err == nil {
		t.Fatal("expected error for empty session ID in GetSnapshot")
	}
	if _, err := store.GetLatestSnapshot(ctx, ""); err == nil {
		t.Fatal("expected error for empty campaign ID in GetLatestSnapshot")
	}
	if _, err := store.ListSnapshots(ctx, "", 10); err == nil {
		t.Fatal("expected error for empty campaign ID in ListSnapshots")
	}
	if _, err := store.ListSnapshots(ctx, "c", 0); err == nil {
		t.Fatal("expected error for zero limit in ListSnapshots")
	}

	// Fork Metadata
	if _, err := store.GetCampaignForkMetadata(ctx, ""); err == nil {
		t.Fatal("expected error for empty campaign ID in GetCampaignForkMetadata")
	}
	if err := store.SetCampaignForkMetadata(ctx, "", storage.ForkMetadata{}); err == nil {
		t.Fatal("expected error for empty campaign ID in SetCampaignForkMetadata")
	}
}

func TestCloseNilStore(t *testing.T) {
	var s *Store
	if err := s.Close(); err != nil {
		t.Fatalf("expected nil close to succeed, got %v", err)
	}

	s = &Store{}
	if err := s.Close(); err != nil {
		t.Fatalf("expected close with nil sqlDB to succeed, got %v", err)
	}
}
