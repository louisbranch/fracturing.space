package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestOpenEmptyPath(t *testing.T) {
	_, err := openStore("", nil, "projections", nil)
	if err == nil {
		t.Fatal("expected error for empty path")
	}

	_, err = openStore("   ", nil, "projections", nil)
	if err == nil {
		t.Fatal("expected error for whitespace path")
	}
}

func TestOpenAlias(t *testing.T) {
	path := filepath.Join(t.TempDir(), "alias.sqlite")
	store, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer store.Close()

	// Verify it behaves like a projections store
	if err := store.Put(context.Background(), campaign.Campaign{
		ID:        "camp-open",
		Name:      "Test",
		Locale:    platformi18n.DefaultLocale(),
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.CampaignStatusActive,
		GmMode:    campaign.GmModeHuman,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("put campaign via Open: %v", err)
	}
}

func TestPutParticipantDuplicateUser(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-dup-user", now)

	p1 := participant.Participant{
		CampaignID:     "camp-dup-user",
		ID:             "part-1",
		UserID:         "user-shared",
		DisplayName:    "Player 1",
		Role:           participant.ParticipantRolePlayer,
		Controller:     participant.ControllerHuman,
		CampaignAccess: participant.CampaignAccessMember,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := store.PutParticipant(context.Background(), p1); err != nil {
		t.Fatalf("put first participant: %v", err)
	}

	p2 := participant.Participant{
		CampaignID:     "camp-dup-user",
		ID:             "part-2",
		UserID:         "user-shared",
		DisplayName:    "Player 2",
		Role:           participant.ParticipantRolePlayer,
		Controller:     participant.ControllerHuman,
		CampaignAccess: participant.CampaignAccessMember,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	err := store.PutParticipant(context.Background(), p2)
	if err == nil {
		t.Fatal("expected error for duplicate user in same campaign")
	}
	if !apperrors.IsCode(err, apperrors.CodeParticipantUserAlreadyClaimed) {
		t.Fatalf("expected CodeParticipantUserAlreadyClaimed, got %v", err)
	}
}

func TestPutParticipantClaimConflict(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-claim-dup", now)
	seedParticipant(t, store, "camp-claim-dup", "part-1", "user-1", now)
	seedParticipant(t, store, "camp-claim-dup", "part-2", "user-2", now)

	// First claim succeeds
	if err := store.PutParticipantClaim(context.Background(), "camp-claim-dup", "user-1", "part-1", now); err != nil {
		t.Fatalf("put first claim: %v", err)
	}

	// Idempotent re-claim for same participant succeeds
	if err := store.PutParticipantClaim(context.Background(), "camp-claim-dup", "user-1", "part-1", now); err != nil {
		t.Fatalf("idempotent re-claim: %v", err)
	}

	// Claim for different participant by same user fails
	err := store.PutParticipantClaim(context.Background(), "camp-claim-dup", "user-1", "part-2", now)
	if err == nil {
		t.Fatal("expected error for conflicting claim")
	}
	if !apperrors.IsCode(err, apperrors.CodeParticipantUserAlreadyClaimed) {
		t.Fatalf("expected CodeParticipantUserAlreadyClaimed, got %v", err)
	}
}

func TestPutParticipantClaimZeroTime(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-claim-zero", now)
	seedParticipant(t, store, "camp-claim-zero", "part-1", "user-1", now)

	// Zero time should auto-fill
	if err := store.PutParticipantClaim(context.Background(), "camp-claim-zero", "user-1", "part-1", time.Time{}); err != nil {
		t.Fatalf("put claim with zero time: %v", err)
	}

	claim, err := store.GetParticipantClaim(context.Background(), "camp-claim-zero", "user-1")
	if err != nil {
		t.Fatalf("get claim: %v", err)
	}
	if claim.ClaimedAt.IsZero() {
		t.Fatal("expected non-zero claimed_at when zero time provided")
	}
}

func TestEndSessionAlreadyEnded(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-end-twice", now)

	sess := session.Session{
		ID:         "sess-1",
		CampaignID: "camp-end-twice",
		Name:       "Session One",
		Status:     session.SessionStatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
	}
	if err := store.PutSession(context.Background(), sess); err != nil {
		t.Fatalf("put session: %v", err)
	}

	endedAt := now.Add(time.Hour)
	_, transitioned, err := store.EndSession(context.Background(), "camp-end-twice", "sess-1", endedAt)
	if err != nil {
		t.Fatalf("first end: %v", err)
	}
	if !transitioned {
		t.Fatal("expected first end to transition")
	}

	// Second end should not transition
	_, transitioned, err = store.EndSession(context.Background(), "camp-end-twice", "sess-1", endedAt.Add(time.Hour))
	if err != nil {
		t.Fatalf("second end: %v", err)
	}
	if transitioned {
		t.Fatal("expected second end to not transition")
	}
}

func TestEndSessionNotFound(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-end-nf", now)

	_, _, err := store.EndSession(context.Background(), "camp-end-nf", "no-sess", now)
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSessionGateNotFound(t *testing.T) {
	store := openTestStore(t)

	_, err := store.GetSessionGate(context.Background(), "no-camp", "no-sess", "no-gate")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for GetSessionGate, got %v", err)
	}

	_, err = store.GetOpenSessionGate(context.Background(), "no-camp", "no-sess")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for GetOpenSessionGate, got %v", err)
	}
}

func TestSessionSpotlightNotFound(t *testing.T) {
	store := openTestStore(t)

	_, err := store.GetSessionSpotlight(context.Background(), "no-camp", "no-sess")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for GetSessionSpotlight, got %v", err)
	}
}

func TestGetInviteNotFound(t *testing.T) {
	store := openTestStore(t)

	_, err := store.GetInvite(context.Background(), "no-invite")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for GetInvite, got %v", err)
	}
}

func TestUpdateInviteStatusZeroTime(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-inv-zero", now)
	seedParticipant(t, store, "camp-inv-zero", "part-1", "user-1", now)

	inv := invite.Invite{
		ID:                     "inv-zero",
		CampaignID:             "camp-inv-zero",
		ParticipantID:          "part-1",
		Status:                 invite.StatusPending,
		CreatedByParticipantID: "part-1",
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if err := store.PutInvite(context.Background(), inv); err != nil {
		t.Fatalf("put invite: %v", err)
	}

	// Zero time should auto-fill
	if err := store.UpdateInviteStatus(context.Background(), "inv-zero", invite.StatusClaimed, time.Time{}); err != nil {
		t.Fatalf("update invite status with zero time: %v", err)
	}

	got, err := store.GetInvite(context.Background(), "inv-zero")
	if err != nil {
		t.Fatalf("get invite: %v", err)
	}
	if got.Status != invite.StatusClaimed {
		t.Fatalf("expected status claimed, got %v", got.Status)
	}
}

func TestListInvitesWithRecipientFilter(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-inv-recip", now)
	seedParticipant(t, store, "camp-inv-recip", "part-1", "user-1", now)
	seedParticipant(t, store, "camp-inv-recip", "part-2", "user-2", now)

	inv1 := invite.Invite{
		ID: "inv-r1", CampaignID: "camp-inv-recip", ParticipantID: "part-1",
		RecipientUserID: "user-1", Status: invite.StatusPending,
		CreatedByParticipantID: "part-1", CreatedAt: now, UpdatedAt: now,
	}
	inv2 := invite.Invite{
		ID: "inv-r2", CampaignID: "camp-inv-recip", ParticipantID: "part-2",
		RecipientUserID: "user-2", Status: invite.StatusPending,
		CreatedByParticipantID: "part-2", CreatedAt: now, UpdatedAt: now,
	}
	if err := store.PutInvite(context.Background(), inv1); err != nil {
		t.Fatalf("put invite 1: %v", err)
	}
	if err := store.PutInvite(context.Background(), inv2); err != nil {
		t.Fatalf("put invite 2: %v", err)
	}

	// Filter by recipient
	page, err := store.ListInvites(context.Background(), "camp-inv-recip", "user-1", invite.StatusUnspecified, 10, "")
	if err != nil {
		t.Fatalf("list invites with recipient filter: %v", err)
	}
	if len(page.Invites) != 1 || page.Invites[0].ID != "inv-r1" {
		t.Fatalf("expected 1 invite for user-1, got %d", len(page.Invites))
	}

	// Filter by recipient + status
	page2, err := store.ListInvites(context.Background(), "camp-inv-recip", "user-1", invite.StatusPending, 10, "")
	if err != nil {
		t.Fatalf("list invites with recipient+status filter: %v", err)
	}
	if len(page2.Invites) != 1 {
		t.Fatalf("expected 1 invite with pending status for user-1, got %d", len(page2.Invites))
	}

	// No filter returns all
	all, err := store.ListInvites(context.Background(), "camp-inv-recip", "", invite.StatusUnspecified, 10, "")
	if err != nil {
		t.Fatalf("list invites no filter: %v", err)
	}
	if len(all.Invites) != 2 {
		t.Fatalf("expected 2 invites, got %d", len(all.Invites))
	}
}

func TestPutSessionNonActive(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-non-active", now)

	// Storing a session with non-active status skips the active check
	endedAt := now.Add(time.Hour)
	sess := session.Session{
		ID:         "sess-ended",
		CampaignID: "camp-non-active",
		Name:       "Ended Session",
		Status:     session.SessionStatusEnded,
		StartedAt:  now,
		UpdatedAt:  now,
		EndedAt:    &endedAt,
	}
	if err := store.PutSession(context.Background(), sess); err != nil {
		t.Fatalf("put non-active session: %v", err)
	}

	got, err := store.GetSession(context.Background(), "camp-non-active", "sess-ended")
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if got.Status != session.SessionStatusEnded {
		t.Fatalf("expected ended status, got %v", got.Status)
	}
	if got.EndedAt == nil {
		t.Fatal("expected ended_at to be set")
	}
}

func TestSessionGateResolvedFields(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	resolvedAt := now.Add(time.Hour)

	gate := storage.SessionGate{
		CampaignID:          "camp-gate-resolve",
		SessionID:           "sess-1",
		GateID:              "gate-resolved",
		GateType:            "consent",
		Status:              "resolved",
		Reason:              "Done",
		CreatedAt:           now,
		CreatedByActorType:  "system",
		CreatedByActorID:    "sys-1",
		ResolvedAt:          &resolvedAt,
		ResolvedByActorType: "participant",
		ResolvedByActorID:   "part-1",
		MetadataJSON:        []byte(`{}`),
		ResolutionJSON:      []byte(`{"approved":true}`),
	}
	if err := store.PutSessionGate(context.Background(), gate); err != nil {
		t.Fatalf("put resolved gate: %v", err)
	}

	got, err := store.GetSessionGate(context.Background(), gate.CampaignID, gate.SessionID, gate.GateID)
	if err != nil {
		t.Fatalf("get resolved gate: %v", err)
	}
	if got.Status != "resolved" {
		t.Fatalf("expected resolved status, got %q", got.Status)
	}
	if got.ResolvedAt == nil || !got.ResolvedAt.Equal(resolvedAt.UTC()) {
		t.Fatal("expected resolved_at to match")
	}
	if got.ResolvedByActorType != "participant" || got.ResolvedByActorID != "part-1" {
		t.Fatal("expected resolved actor fields to match")
	}
	if string(got.ResolutionJSON) != `{"approved":true}` {
		t.Fatalf("expected resolution json to match, got %s", string(got.ResolutionJSON))
	}
}

func TestCampaignPutUpdate(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)

	c := campaign.Campaign{
		ID:        "camp-update",
		Name:      "Original",
		Locale:    platformi18n.DefaultLocale(),
		System:    commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Status:    campaign.CampaignStatusDraft,
		GmMode:    campaign.GmModeAI,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Put(context.Background(), c); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	// Update with new fields
	c.Name = "Updated"
	c.Status = campaign.CampaignStatusActive
	c.GmMode = campaign.GmModeHybrid
	c.UpdatedAt = now.Add(time.Hour)
	if err := store.Put(context.Background(), c); err != nil {
		t.Fatalf("update campaign: %v", err)
	}

	got, err := store.Get(context.Background(), "camp-update")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if got.Name != "Updated" {
		t.Fatalf("expected updated name, got %q", got.Name)
	}
	if got.Status != campaign.CampaignStatusActive {
		t.Fatalf("expected active status")
	}
}

func TestConversionHelpersCoverage(t *testing.T) {
	// Draft status
	if campaignStatusToString(campaign.CampaignStatusDraft) != "DRAFT" {
		t.Fatal("expected DRAFT")
	}
	if campaignStatusToString(campaign.CampaignStatusCompleted) != "COMPLETED" {
		t.Fatal("expected COMPLETED")
	}
	if stringToCampaignStatus("DRAFT") != campaign.CampaignStatusDraft {
		t.Fatal("expected draft status")
	}
	if stringToCampaignStatus("COMPLETED") != campaign.CampaignStatusCompleted {
		t.Fatal("expected completed status")
	}
	if stringToCampaignStatus("ARCHIVED") != campaign.CampaignStatusArchived {
		t.Fatal("expected archived status")
	}

	// GM Mode: HUMAN
	if gmModeToString(campaign.GmModeHuman) != "HUMAN" {
		t.Fatal("expected HUMAN")
	}
	if stringToGmMode("HUMAN") != campaign.GmModeHuman {
		t.Fatal("expected human mode")
	}
	if stringToGmMode("HYBRID") != campaign.GmModeHybrid {
		t.Fatal("expected hybrid mode")
	}

	// Participant: Manager access
	if participantAccessToString(participant.CampaignAccessManager) != "MANAGER" {
		t.Fatal("expected MANAGER")
	}
	if stringToParticipantAccess("MANAGER") != participant.CampaignAccessManager {
		t.Fatal("expected manager access")
	}
	if stringToParticipantAccess("OWNER") != participant.CampaignAccessOwner {
		t.Fatal("expected owner access")
	}

	// Participant: Unspecified controller
	if participantControllerToString(participant.ControllerUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected UNSPECIFIED controller")
	}

	// Character: Unspecified kind
	if characterKindToString(character.CharacterKindUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected UNSPECIFIED kind")
	}

	// Session: Unspecified status
	if sessionStatusToString(session.SessionStatusUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected UNSPECIFIED session status")
	}

	// Participant role: Unspecified
	if participantRoleToString(participant.ParticipantRoleUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected UNSPECIFIED role")
	}

	// Participant access: Unspecified
	if participantAccessToString(participant.CampaignAccessUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected UNSPECIFIED access")
	}

	// stringToParticipantRole: GM case
	if stringToParticipantRole("GM") != participant.ParticipantRoleGM {
		t.Fatal("expected GM role")
	}

	// stringToParticipantController: AI case
	if stringToParticipantController("AI") != participant.ControllerAI {
		t.Fatal("expected AI controller")
	}

	// participantControllerToString: AI case
	if participantControllerToString(participant.ControllerAI) != "AI" {
		t.Fatal("expected AI string")
	}

	// participantRoleToString: GM case
	if participantRoleToString(participant.ParticipantRoleGM) != "GM" {
		t.Fatal("expected GM string")
	}

	// campaignStatusToString: Archived
	if campaignStatusToString(campaign.CampaignStatusArchived) != "ARCHIVED" {
		t.Fatal("expected ARCHIVED")
	}

	// gmModeToString: AI and Unspecified
	if gmModeToString(campaign.GmModeAI) != "AI" {
		t.Fatal("expected AI")
	}
	if gmModeToString(campaign.GmModeUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected UNSPECIFIED gm mode")
	}
	if stringToGmMode("AI") != campaign.GmModeAI {
		t.Fatal("expected AI mode")
	}
	if stringToGmMode("UNKNOWN") != campaign.GmModeUnspecified {
		t.Fatal("expected unspecified mode for unknown string")
	}

	// campaignStatusToString: Unspecified
	if campaignStatusToString(campaign.CampaignStatusUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected UNSPECIFIED campaign status")
	}
	if stringToCampaignStatus("UNKNOWN") != campaign.CampaignStatusUnspecified {
		t.Fatal("expected unspecified status for unknown string")
	}
}

func TestIsConstraintErrorFalseForNonSqlite(t *testing.T) {
	if isConstraintError(errors.New("random error")) {
		t.Fatal("expected false for non-sqlite error")
	}
	if isParticipantUserConflict(errors.New("random error")) {
		t.Fatal("expected false for non-constraint error")
	}
	if isParticipantClaimConflict(errors.New("random error")) {
		t.Fatal("expected false for non-constraint error")
	}
}
