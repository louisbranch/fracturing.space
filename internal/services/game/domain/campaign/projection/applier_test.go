package projection

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type fakeInviteStore struct {
	invites       map[string]invite.Invite
	updatedStatus map[string]invite.Status
	updatedAt     map[string]time.Time
}

func newFakeInviteStore() *fakeInviteStore {
	return &fakeInviteStore{
		invites:       make(map[string]invite.Invite),
		updatedStatus: make(map[string]invite.Status),
		updatedAt:     make(map[string]time.Time),
	}
}

func (s *fakeInviteStore) PutInvite(_ context.Context, inv invite.Invite) error {
	s.invites[inv.ID] = inv
	return nil
}

func (s *fakeInviteStore) GetInvite(_ context.Context, inviteID string) (invite.Invite, error) {
	inv, ok := s.invites[inviteID]
	if !ok {
		return invite.Invite{}, storage.ErrNotFound
	}
	return inv, nil
}

func (s *fakeInviteStore) ListInvites(context.Context, string, string, invite.Status, int, string) (storage.InvitePage, error) {
	return storage.InvitePage{}, nil
}

func (s *fakeInviteStore) ListPendingInvites(context.Context, string, int, string) (storage.InvitePage, error) {
	return storage.InvitePage{}, nil
}

func (s *fakeInviteStore) ListPendingInvitesForRecipient(context.Context, string, int, string) (storage.InvitePage, error) {
	return storage.InvitePage{}, nil
}

func (s *fakeInviteStore) UpdateInviteStatus(_ context.Context, inviteID string, status invite.Status, updatedAt time.Time) error {
	s.updatedStatus[inviteID] = status
	s.updatedAt[inviteID] = updatedAt
	return nil
}

type fakeSessionStore struct {
	last session.Session
}

func (s *fakeSessionStore) PutSession(_ context.Context, sess session.Session) error {
	s.last = sess
	return nil
}

func (s *fakeSessionStore) EndSession(context.Context, string, string, time.Time) (session.Session, bool, error) {
	return session.Session{}, true, nil
}

func (s *fakeSessionStore) GetSession(context.Context, string, string) (session.Session, error) {
	return session.Session{}, storage.ErrNotFound
}

func (s *fakeSessionStore) GetActiveSession(context.Context, string) (session.Session, error) {
	return session.Session{}, storage.ErrNotFound
}

func (s *fakeSessionStore) ListSessions(context.Context, string, int, string) (storage.SessionPage, error) {
	return storage.SessionPage{}, nil
}

type fakeClaimIndexStore struct {
	claims    map[string]storage.ParticipantClaim
	deleted   []string
	lastPut   storage.ParticipantClaim
	lastPutOK bool
}

func newFakeClaimIndexStore() *fakeClaimIndexStore {
	return &fakeClaimIndexStore{claims: make(map[string]storage.ParticipantClaim)}
}

func (s *fakeClaimIndexStore) PutParticipantClaim(_ context.Context, campaignID, userID, participantID string, claimedAt time.Time) error {
	claim := storage.ParticipantClaim{CampaignID: campaignID, UserID: userID, ParticipantID: participantID, ClaimedAt: claimedAt}
	s.claims[campaignID+":"+userID] = claim
	s.lastPut = claim
	s.lastPutOK = true
	return nil
}

func (s *fakeClaimIndexStore) GetParticipantClaim(_ context.Context, campaignID, userID string) (storage.ParticipantClaim, error) {
	claim, ok := s.claims[campaignID+":"+userID]
	if !ok {
		return storage.ParticipantClaim{}, storage.ErrNotFound
	}
	return claim, nil
}

func (s *fakeClaimIndexStore) DeleteParticipantClaim(_ context.Context, campaignID, userID string) error {
	delete(s.claims, campaignID+":"+userID)
	s.deleted = append(s.deleted, userID)
	return nil
}

type fakeAdapter struct {
	called bool
}

func (a *fakeAdapter) ID() commonv1.GameSystem {
	return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
}

func (a *fakeAdapter) Version() string {
	return "v1"
}

func (a *fakeAdapter) ApplyEvent(context.Context, event.Event) error {
	a.called = true
	return nil
}

func (a *fakeAdapter) Snapshot(context.Context, string) (any, error) {
	return nil, errors.New("not implemented")
}

func TestApplyCampaignUpdated_StatusAndName(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = campaign.Campaign{ID: "camp-1", Status: campaign.CampaignStatusDraft, Name: "Old"}
	applier := Applier{Campaign: store}

	payload := event.CampaignUpdatedPayload{
		Fields: map[string]any{
			"status": "ACTIVE",
			"name":   "  New Name  ",
		},
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	evt := event.Event{CampaignID: "camp-1", Type: event.TypeCampaignUpdated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	updated, err := store.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if updated.Status != campaign.CampaignStatusActive {
		t.Fatalf("Status = %v, want %v", updated.Status, campaign.CampaignStatusActive)
	}
	if updated.Name != "New Name" {
		t.Fatalf("Name = %q, want %q", updated.Name, "New Name")
	}
	if !updated.UpdatedAt.Equal(stamp) {
		t.Fatalf("UpdatedAt = %v, want %v", updated.UpdatedAt, stamp)
	}
}

func TestApplyParticipantUnbound_RejectsMismatch(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = participant.Participant{ID: "part-1", CampaignID: "camp-1", UserID: "user-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = campaign.Campaign{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	payload := event.ParticipantUnboundPayload{UserID: "user-2"}
	data, _ := json.Marshal(payload)
	evt := event.Event{CampaignID: "camp-1", EntityID: "part-1", Type: event.TypeParticipantUnbound, PayloadJSON: data}

	if err := applier.Apply(ctx, evt); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestApplySeatReassigned_UpdatesClaims(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = participant.Participant{ID: "part-1", CampaignID: "camp-1", UserID: "user-old"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = campaign.Campaign{ID: "camp-1"}
	claimStore := newFakeClaimIndexStore()
	applier := Applier{Participant: participantStore, Campaign: campaignStore, ClaimIndex: claimStore}

	payload := event.SeatReassignedPayload{UserID: "user-new", PriorUserID: "user-old"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 12, 30, 0, 0, time.UTC)
	evt := event.Event{CampaignID: "camp-1", EntityID: "part-1", Type: event.TypeSeatReassigned, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	updated, err := participantStore.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if updated.UserID != "user-new" {
		t.Fatalf("UserID = %q, want %q", updated.UserID, "user-new")
	}
	if !claimStore.lastPutOK || claimStore.lastPut.UserID != "user-new" {
		t.Fatal("expected claim to be recorded for new user")
	}
}

func TestApplyInviteCreated_UsesEntityID(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = campaign.Campaign{ID: "camp-1"}
	inviteStore := newFakeInviteStore()
	applier := Applier{Campaign: campaignStore, Invite: inviteStore}

	payload := event.InviteCreatedPayload{
		InviteID:      "",
		ParticipantID: "part-1",
		Status:        "pending",
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)
	evt := event.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: event.TypeInviteCreated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	inv, err := inviteStore.GetInvite(ctx, "inv-1")
	if err != nil {
		t.Fatalf("get invite: %v", err)
	}
	if inv.ParticipantID != "part-1" {
		t.Fatalf("ParticipantID = %q, want %q", inv.ParticipantID, "part-1")
	}
	updatedCampaign, err := campaignStore.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if !updatedCampaign.UpdatedAt.Equal(stamp) {
		t.Fatalf("UpdatedAt = %v, want %v", updatedCampaign.UpdatedAt, stamp)
	}
}

func TestApplySessionStarted_UsesEntityID(t *testing.T) {
	ctx := context.Background()
	sessionStore := &fakeSessionStore{}
	applier := Applier{Session: sessionStore}

	payload := event.SessionStartedPayload{SessionID: "", SessionName: "Session 1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 14, 0, 0, 0, time.UTC)
	evt := event.Event{CampaignID: "camp-1", EntityID: "sess-1", Type: event.TypeSessionStarted, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if sessionStore.last.ID != "sess-1" {
		t.Fatalf("Session ID = %q, want %q", sessionStore.last.ID, "sess-1")
	}
	if sessionStore.last.Status != session.SessionStatusActive {
		t.Fatalf("Status = %v, want %v", sessionStore.last.Status, session.SessionStatusActive)
	}
}

func TestApplySystemEvent_UsesAdapter(t *testing.T) {
	ctx := context.Background()
	adapter := &fakeAdapter{}
	registry := systems.NewAdapterRegistry()
	registry.Register(adapter)
	applier := Applier{Adapters: registry}

	evt := event.Event{
		Type:          event.Type("system.custom"),
		SystemID:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		SystemVersion: "",
		PayloadJSON:   []byte("{}"),
	}

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !adapter.called {
		t.Fatal("expected adapter to be called")
	}
}
