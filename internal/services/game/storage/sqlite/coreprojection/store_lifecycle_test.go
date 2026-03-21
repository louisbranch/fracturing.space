package coreprojection

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestOpenEmptyPath(t *testing.T) {
	_, err := openStore("", nil, "projections")
	if err == nil {
		t.Fatal("expected error for empty path")
	}

	_, err = openStore("   ", nil, "projections")
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
	if err := store.Put(context.Background(), storage.CampaignRecord{
		ID:        "camp-open",
		Name:      "Test",
		Locale:    "en-US",
		System:    bridge.SystemIDDaggerheart,
		Status:    campaign.StatusActive,
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

	p1 := storage.ParticipantRecord{
		CampaignID:     "camp-dup-user",
		ID:             "part-1",
		UserID:         "user-shared",
		Name:           "Player 1",
		Role:           participant.RolePlayer,
		Controller:     participant.ControllerHuman,
		CampaignAccess: participant.CampaignAccessMember,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := store.PutParticipant(context.Background(), p1); err != nil {
		t.Fatalf("put first participant: %v", err)
	}

	p2 := storage.ParticipantRecord{
		CampaignID:     "camp-dup-user",
		ID:             "part-2",
		UserID:         "user-shared",
		Name:           "Player 2",
		Role:           participant.RolePlayer,
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

	sess := storage.SessionRecord{
		ID:         "sess-1",
		CampaignID: "camp-end-twice",
		Name:       "Session One",
		Status:     session.StatusActive,
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

func TestPutSessionNonActive(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-non-active", now)

	// Storing a session with non-active status skips the active check
	endedAt := now.Add(time.Hour)
	sess := storage.SessionRecord{
		ID:         "sess-ended",
		CampaignID: "camp-non-active",
		Name:       "Ended Session",
		Status:     session.StatusEnded,
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
	if got.Status != session.StatusEnded {
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
		Metadata:            map[string]any{},
		Resolution:          map[string]any{"approved": true},
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
	if got.Resolution["approved"] != true {
		t.Fatalf("expected resolution to match, got %#v", got.Resolution)
	}
}

func TestCampaignPutUpdate(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 10, 0, 0, 0, time.UTC)

	c := storage.CampaignRecord{
		ID:        "camp-update",
		Name:      "Original",
		Locale:    "en-US",
		System:    bridge.SystemIDDaggerheart,
		Status:    campaign.StatusDraft,
		GmMode:    campaign.GmModeAI,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := store.Put(context.Background(), c); err != nil {
		t.Fatalf("put campaign: %v", err)
	}

	// Update with new fields
	c.Name = "Updated"
	c.Status = campaign.StatusActive
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
	if got.Status != campaign.StatusActive {
		t.Fatalf("expected active status")
	}
}

func TestEnumRoundTripCoverage(t *testing.T) {
	// enumToStorage covers all enum types
	if enumToStorage(campaign.StatusDraft) != "DRAFT" {
		t.Fatal("expected DRAFT")
	}
	if enumToStorage(campaign.StatusCompleted) != "COMPLETED" {
		t.Fatal("expected COMPLETED")
	}
	if enumToStorage(campaign.StatusArchived) != "ARCHIVED" {
		t.Fatal("expected ARCHIVED")
	}
	if enumToStorage(campaign.GmModeHuman) != "HUMAN" {
		t.Fatal("expected HUMAN")
	}
	if enumToStorage(campaign.GmModeAI) != "AI" {
		t.Fatal("expected AI gm mode")
	}
	if enumToStorage(participant.CampaignAccessManager) != "MANAGER" {
		t.Fatal("expected MANAGER")
	}
	if enumToStorage(participant.CampaignAccessOwner) != "OWNER" {
		t.Fatal("expected OWNER")
	}
	if enumToStorage(participant.RoleGM) != "GM" {
		t.Fatal("expected GM")
	}
	if enumToStorage(participant.ControllerAI) != "AI" {
		t.Fatal("expected AI controller")
	}

	// enumFromStorage round-trips through domain Normalize functions
	if enumFromStorage("DRAFT", campaign.NormalizeStatus) != campaign.StatusDraft {
		t.Fatal("expected draft status")
	}
	if enumFromStorage("COMPLETED", campaign.NormalizeStatus) != campaign.StatusCompleted {
		t.Fatal("expected completed status")
	}
	if enumFromStorage("ARCHIVED", campaign.NormalizeStatus) != campaign.StatusArchived {
		t.Fatal("expected archived status")
	}
	if enumFromStorage("UNKNOWN", campaign.NormalizeStatus) != campaign.StatusUnspecified {
		t.Fatal("expected unspecified status for unknown string")
	}
	if enumFromStorage("HUMAN", campaign.NormalizeGmMode) != campaign.GmModeHuman {
		t.Fatal("expected human mode")
	}
	if enumFromStorage("HYBRID", campaign.NormalizeGmMode) != campaign.GmModeHybrid {
		t.Fatal("expected hybrid mode")
	}
	if enumFromStorage("AI", campaign.NormalizeGmMode) != campaign.GmModeAI {
		t.Fatal("expected AI mode")
	}
	if enumFromStorage("UNKNOWN", campaign.NormalizeGmMode) != campaign.GmModeUnspecified {
		t.Fatal("expected unspecified mode for unknown string")
	}
	if enumFromStorage("MANAGER", participant.NormalizeCampaignAccess) != participant.CampaignAccessManager {
		t.Fatal("expected manager access")
	}
	if enumFromStorage("OWNER", participant.NormalizeCampaignAccess) != participant.CampaignAccessOwner {
		t.Fatal("expected owner access")
	}
	if enumFromStorage("GM", participant.NormalizeRole) != participant.RoleGM {
		t.Fatal("expected GM role")
	}
	if enumFromStorage("AI", participant.NormalizeController) != participant.ControllerAI {
		t.Fatal("expected AI controller")
	}
	if enumFromStorage("HUMAN", participant.NormalizeController) != participant.ControllerHuman {
		t.Fatal("expected human controller")
	}

	// Zero-value enums produce UNSPECIFIED in storage
	if enumToStorage(campaign.GmModeUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected UNSPECIFIED gm mode")
	}
	if enumToStorage(campaign.StatusUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected UNSPECIFIED campaign status")
	}
	if enumToStorage(participant.ControllerUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected UNSPECIFIED controller")
	}
	if enumToStorage(character.KindUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected UNSPECIFIED kind")
	}
	if enumToStorage(session.StatusUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected UNSPECIFIED session status")
	}
	if enumToStorage(participant.RoleUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected UNSPECIFIED role")
	}
	if enumToStorage(participant.CampaignAccessUnspecified) != "UNSPECIFIED" {
		t.Fatal("expected UNSPECIFIED access")
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
