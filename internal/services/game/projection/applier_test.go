package projection

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	daggerheartsys "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection/testevent"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

type fakeInviteStore struct {
	invites       map[string]storage.InviteRecord
	updatedStatus map[string]invite.Status
	updatedAt     map[string]time.Time
}

func newFakeInviteStore() *fakeInviteStore {
	return &fakeInviteStore{
		invites:       make(map[string]storage.InviteRecord),
		updatedStatus: make(map[string]invite.Status),
		updatedAt:     make(map[string]time.Time),
	}
}

func (s *fakeInviteStore) PutInvite(_ context.Context, inv storage.InviteRecord) error {
	s.invites[inv.ID] = inv
	return nil
}

func (s *fakeInviteStore) GetInvite(_ context.Context, inviteID string) (storage.InviteRecord, error) {
	inv, ok := s.invites[inviteID]
	if !ok {
		return storage.InviteRecord{}, storage.ErrNotFound
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
	last storage.SessionRecord
}

func (s *fakeSessionStore) PutSession(_ context.Context, sess storage.SessionRecord) error {
	s.last = sess
	return nil
}

func (s *fakeSessionStore) EndSession(context.Context, string, string, time.Time) (storage.SessionRecord, bool, error) {
	return storage.SessionRecord{}, true, nil
}

func (s *fakeSessionStore) GetSession(context.Context, string, string) (storage.SessionRecord, error) {
	return storage.SessionRecord{}, storage.ErrNotFound
}

func (s *fakeSessionStore) GetActiveSession(context.Context, string) (storage.SessionRecord, error) {
	return storage.SessionRecord{}, storage.ErrNotFound
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

func (a *fakeAdapter) ID() string {
	return "daggerheart"
}

func (a *fakeAdapter) Version() string {
	return "v1"
}

func (a *fakeAdapter) Apply(context.Context, event.Event) error {
	a.called = true
	return nil
}

func (a *fakeAdapter) Snapshot(context.Context, string) (any, error) {
	return nil, errors.New("not implemented")
}

func (a *fakeAdapter) HandledTypes() []event.Type {
	return nil
}

// testEventRegistry builds a fully-wired event registry with aliases for test
// appliers that need to resolve legacy event types.
func testEventRegistry(t *testing.T) *event.Registry {
	t.Helper()
	registries, err := engine.BuildRegistries()
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}
	return registries.Events
}

func eventToEvent(evt testevent.Event) event.Event {
	return event.Event{
		CampaignID:     strings.TrimSpace(evt.CampaignID),
		Seq:            evt.Seq,
		Hash:           evt.Hash,
		PrevHash:       evt.PrevHash,
		ChainHash:      evt.ChainHash,
		Signature:      evt.Signature,
		SignatureKeyID: evt.SignatureKeyID,
		Type:           event.Type(strings.TrimSpace(string(evt.Type))),
		Timestamp:      evt.Timestamp,
		SessionID:      evt.SessionID,
		RequestID:      evt.RequestID,
		InvocationID:   evt.InvocationID,
		ActorType:      event.ActorType(evt.ActorType),
		ActorID:        evt.ActorID,
		EntityType:     evt.EntityType,
		EntityID:       evt.EntityID,
		SystemID:       evt.SystemID,
		SystemVersion:  evt.SystemVersion,
		PayloadJSON:    evt.PayloadJSON,
	}
}

func TestApplyCampaignUpdated_StatusAndName(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", Status: campaign.StatusDraft, Name: "Old"}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{
		Fields: map[string]any{
			"status": "ACTIVE",
			"name":   "  New Name  ",
		},
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	updated, err := store.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get campaign: %v", err)
	}
	if updated.Status != campaign.StatusActive {
		t.Fatalf("Status = %v, want %v", updated.Status, campaign.StatusActive)
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
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "user-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	payload := testevent.ParticipantUnboundPayload{UserID: "user-2"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUnbound, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestApplySeatReassigned_UpdatesClaims(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "user-old"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	claimStore := newFakeClaimIndexStore()
	applier := Applier{Events: testEventRegistry(t), Participant: participantStore, Campaign: campaignStore, ClaimIndex: claimStore}

	payload := testevent.SeatReassignedPayload{UserID: "user-new", PriorUserID: "user-old"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 12, 30, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeSeatReassigned, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
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
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	inviteStore := newFakeInviteStore()
	applier := Applier{Campaign: campaignStore, Invite: inviteStore}

	payload := testevent.InviteCreatedPayload{
		InviteID:      "",
		ParticipantID: "part-1",
		Status:        "pending",
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 13, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteCreated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
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

	payload := testevent.SessionStartedPayload{SessionID: "", SessionName: "Session 1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 14, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "sess-1", Type: testevent.TypeSessionStarted, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if sessionStore.last.ID != "sess-1" {
		t.Fatalf("Session ID = %q, want %q", sessionStore.last.ID, "sess-1")
	}
	if sessionStore.last.Status != session.StatusActive {
		t.Fatalf("Status = %v, want %v", sessionStore.last.Status, session.StatusActive)
	}
}

type fakeSessionGateStore struct {
	gates map[string]storage.SessionGate
}

func newFakeSessionGateStore() *fakeSessionGateStore {
	return &fakeSessionGateStore{gates: make(map[string]storage.SessionGate)}
}

func (s *fakeSessionGateStore) PutSessionGate(_ context.Context, gate storage.SessionGate) error {
	key := gate.CampaignID + ":" + gate.SessionID + ":" + gate.GateID
	s.gates[key] = gate
	return nil
}

func (s *fakeSessionGateStore) GetSessionGate(_ context.Context, campaignID, sessionID, gateID string) (storage.SessionGate, error) {
	key := campaignID + ":" + sessionID + ":" + gateID
	gate, ok := s.gates[key]
	if !ok {
		return storage.SessionGate{}, storage.ErrNotFound
	}
	return gate, nil
}

func (s *fakeSessionGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

type fakeSessionSpotlightStore struct {
	spotlights map[string]storage.SessionSpotlight
	cleared    []string
}

func newFakeSessionSpotlightStore() *fakeSessionSpotlightStore {
	return &fakeSessionSpotlightStore{
		spotlights: make(map[string]storage.SessionSpotlight),
	}
}

func (s *fakeSessionSpotlightStore) PutSessionSpotlight(_ context.Context, spotlight storage.SessionSpotlight) error {
	key := spotlight.CampaignID + ":" + spotlight.SessionID
	s.spotlights[key] = spotlight
	return nil
}

func (s *fakeSessionSpotlightStore) GetSessionSpotlight(_ context.Context, campaignID, sessionID string) (storage.SessionSpotlight, error) {
	key := campaignID + ":" + sessionID
	spotlight, ok := s.spotlights[key]
	if !ok {
		return storage.SessionSpotlight{}, storage.ErrNotFound
	}
	return spotlight, nil
}

func (s *fakeSessionSpotlightStore) ClearSessionSpotlight(_ context.Context, campaignID, sessionID string) error {
	s.cleared = append(s.cleared, campaignID+":"+sessionID)
	return nil
}

func TestApplyInviteRevoked(t *testing.T) {
	ctx := context.Background()
	inviteStore := newFakeInviteStore()
	inviteStore.invites["inv-1"] = storage.InviteRecord{ID: "inv-1", CampaignID: "camp-1", Status: invite.StatusPending}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Invite: inviteStore, Campaign: campaignStore}

	payload := testevent.InviteRevokedPayload{InviteID: "inv-1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteRevoked, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if inviteStore.updatedStatus["inv-1"] != invite.StatusRevoked {
		t.Fatalf("expected invite status revoked, got %v", inviteStore.updatedStatus["inv-1"])
	}
	updated, _ := campaignStore.Get(ctx, "camp-1")
	if !updated.UpdatedAt.Equal(stamp) {
		t.Fatalf("campaign UpdatedAt = %v, want %v", updated.UpdatedAt, stamp)
	}
}

func TestApplyInviteRevoked_MissingStores(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteRevokedPayload{InviteID: "inv-1"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteRevoked, PayloadJSON: data}

	// Missing invite store
	if err := (Applier{Campaign: newProjectionCampaignStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing invite store")
	}
	// Missing campaign store
	if err := (Applier{Invite: newFakeInviteStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyInviteRevoked_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteRevokedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeInviteRevoked, PayloadJSON: data}
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

func TestApplyInviteUpdated(t *testing.T) {
	ctx := context.Background()
	inviteStore := newFakeInviteStore()
	applier := Applier{Invite: inviteStore}

	payload := testevent.InviteUpdatedPayload{InviteID: "inv-1", Status: "REVOKED"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 11, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteUpdated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if inviteStore.updatedStatus["inv-1"] != invite.StatusRevoked {
		t.Fatalf("expected invite status revoked, got %v", inviteStore.updatedStatus["inv-1"])
	}
}

func TestApplyInviteUpdated_EntityIDFallback(t *testing.T) {
	ctx := context.Background()
	inviteStore := newFakeInviteStore()
	applier := Applier{Invite: inviteStore}

	payload := testevent.InviteUpdatedPayload{InviteID: "", Status: "CLAIMED"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-2", Type: testevent.TypeInviteUpdated, PayloadJSON: data, Timestamp: time.Now()}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, ok := inviteStore.updatedStatus["inv-2"]; !ok {
		t.Fatal("expected invite update to use EntityID as fallback")
	}
}

func TestApplyInviteUpdated_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteUpdatedPayload{InviteID: "inv-1", Status: "PENDING"})
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeInviteUpdated, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing invite store")
	}
}

func TestApplyInviteUpdated_MissingInviteID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteUpdatedPayload{InviteID: "", Status: "PENDING"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeInviteUpdated, PayloadJSON: data}
	applier := Applier{Invite: newFakeInviteStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing invite ID")
	}
}

func TestApplySessionGateAbandoned(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSessionGateStore()
	gateStore.gates["camp-1:sess-1:gate-1"] = storage.SessionGate{
		CampaignID: "camp-1", SessionID: "sess-1", GateID: "gate-1", Status: session.GateStatusOpen,
	}
	applier := Applier{SessionGate: gateStore}

	payload := testevent.SessionGateAbandonedPayload{GateID: "gate-1", Reason: "timeout"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 12, 0, 0, 0, time.UTC)
	evt := testevent.Event{
		CampaignID: "camp-1", SessionID: "sess-1", EntityID: "gate-1",
		Type: testevent.TypeSessionGateAbandoned, PayloadJSON: data, Timestamp: stamp,
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gate, err := gateStore.GetSessionGate(ctx, "camp-1", "sess-1", "gate-1")
	if err != nil {
		t.Fatalf("get gate: %v", err)
	}
	if gate.Status != session.GateStatusAbandoned {
		t.Fatalf("gate status = %q, want %q", gate.Status, session.GateStatusAbandoned)
	}
	if gate.ResolvedAt == nil || !gate.ResolvedAt.Equal(stamp) {
		t.Fatalf("gate resolved at mismatch")
	}
}

func TestApplySessionGateAbandoned_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateAbandonedPayload{GateID: "gate-1"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionGateAbandoned, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session gate store")
	}
}

func TestApplySessionGateAbandoned_MissingSessionID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateAbandonedPayload{GateID: "gate-1"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "", Type: testevent.TypeSessionGateAbandoned, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

func TestApplySessionSpotlightSet(t *testing.T) {
	ctx := context.Background()
	spotlightStore := newFakeSessionSpotlightStore()
	applier := Applier{SessionSpotlight: spotlightStore}

	payload := testevent.SessionSpotlightSetPayload{SpotlightType: "character", CharacterID: "char-1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 13, 0, 0, 0, time.UTC)
	evt := testevent.Event{
		CampaignID: "camp-1", SessionID: "sess-1",
		Type: testevent.TypeSessionSpotlightSet, PayloadJSON: data, Timestamp: stamp,
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	spotlight, err := spotlightStore.GetSessionSpotlight(ctx, "camp-1", "sess-1")
	if err != nil {
		t.Fatalf("get spotlight: %v", err)
	}
	if spotlight.CharacterID != "char-1" {
		t.Fatalf("spotlight character = %q, want %q", spotlight.CharacterID, "char-1")
	}
}

func TestApplySessionSpotlightSet_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionSpotlightSetPayload{SpotlightType: "character", CharacterID: "c1"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionSpotlightSet, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing spotlight store")
	}
}

func TestApplySessionSpotlightSet_MissingSessionID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionSpotlightSetPayload{SpotlightType: "character", CharacterID: "c1"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "", Type: testevent.TypeSessionSpotlightSet, PayloadJSON: data}
	applier := Applier{SessionSpotlight: newFakeSessionSpotlightStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

func TestApplySessionSpotlightCleared(t *testing.T) {
	ctx := context.Background()
	spotlightStore := newFakeSessionSpotlightStore()
	applier := Applier{SessionSpotlight: spotlightStore}

	data, _ := json.Marshal(testevent.SessionSpotlightClearedPayload{})
	evt := testevent.Event{
		CampaignID: "camp-1", SessionID: "sess-1",
		Type: testevent.TypeSessionSpotlightCleared, PayloadJSON: data,
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(spotlightStore.cleared) != 1 || spotlightStore.cleared[0] != "camp-1:sess-1" {
		t.Fatalf("expected spotlight to be cleared for camp-1:sess-1")
	}
}

func TestApplySessionSpotlightCleared_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionSpotlightClearedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionSpotlightCleared, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing spotlight store")
	}
}

func TestApplySessionSpotlightCleared_MissingSessionID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionSpotlightClearedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "", Type: testevent.TypeSessionSpotlightCleared, PayloadJSON: data}
	applier := Applier{SessionSpotlight: newFakeSessionSpotlightStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

func TestParseGameSystem(t *testing.T) {
	// Exact proto enum name
	sys, err := parseGameSystem("GAME_SYSTEM_DAGGERHEART")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sys != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("expected DAGGERHEART, got %v", sys)
	}

	// Uppercase shorthand
	sys, err = parseGameSystem("DAGGERHEART")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sys != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("expected DAGGERHEART, got %v", sys)
	}

	// Empty
	_, err = parseGameSystem("")
	if err == nil {
		t.Fatal("expected error for empty game system")
	}

	// Unknown
	_, err = parseGameSystem("NONEXISTENT")
	if err == nil {
		t.Fatal("expected error for unknown game system")
	}
}

func TestApplySystemEvent_MissingAdapters(t *testing.T) {
	ctx := context.Background()
	evt := testevent.Event{Type: testevent.Type("system.custom"), SystemID: "daggerheart", PayloadJSON: []byte("{}")}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing adapters")
	}
}

func TestApplySystemEvent_UnknownAdapter(t *testing.T) {
	ctx := context.Background()
	registry := bridge.NewAdapterRegistry()
	applier := Applier{Adapters: registry}
	evt := testevent.Event{Type: testevent.Type("system.custom"), SystemID: "daggerheart", PayloadJSON: []byte("{}")}
	// No adapter registered for daggerheart in this registry, should error
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing adapter")
	}
}

func TestApplySessionEnded(t *testing.T) {
	ctx := context.Background()
	sessionStore := &fakeSessionStore{}
	applier := Applier{Session: sessionStore}

	payload := testevent.SessionEndedPayload{SessionID: "sess-1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 14, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "sess-1", Type: testevent.TypeSessionEnded, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
}

func TestApplySessionEnded_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionEndedPayload{SessionID: "sess-1"})
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeSessionEnded, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session store")
	}
}

func TestApplySessionEnded_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionEndedPayload{SessionID: "sess-1"})
	evt := testevent.Event{CampaignID: "", Type: testevent.TypeSessionEnded, PayloadJSON: data}
	applier := Applier{Session: &fakeSessionStore{}}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

func TestEnsureTimestamp(t *testing.T) {
	// Non-zero timestamp should be converted to UTC
	ts := time.Date(2026, 1, 1, 12, 0, 0, 0, time.FixedZone("EST", -5*3600))
	result, err := ensureTimestamp(ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Equal(ts) {
		t.Fatalf("expected equal time, got %v", result)
	}
	if result.Location() != time.UTC {
		t.Fatalf("expected UTC, got %v", result.Location())
	}

	// Zero timestamp should return an error for replay determinism
	_, err = ensureTimestamp(time.Time{})
	if err == nil {
		t.Fatal("expected error for zero timestamp")
	}
}

func TestApplySystemEvent_UsesAdapter(t *testing.T) {
	ctx := context.Background()
	adapter := &fakeAdapter{}
	registry := bridge.NewAdapterRegistry()
	if err := registry.Register(adapter); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	applier := Applier{Adapters: registry}

	evt := testevent.Event{
		Type:          testevent.Type("system.custom"),
		SystemID:      "daggerheart",
		SystemVersion: "",
		PayloadJSON:   []byte("{}"),
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !adapter.called {
		t.Fatal("expected adapter to be called")
	}
}

func TestApplySystemEvent_UsesDaggerheartAdapterForSysPrefixedEventType(t *testing.T) {
	ctx := context.Background()
	daggerheartStore := newProjectionDaggerheartStore()
	registry := bridge.NewAdapterRegistry()
	if err := registry.Register(daggerheartsys.NewAdapter(daggerheartStore)); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	applier := Applier{
		Adapters: registry,
	}

	payload, err := json.Marshal(daggerheartsys.GMFearChangedPayload{
		Before: 1,
		After:  4,
		Reason: "test",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	evt := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("sys." + daggerheartsys.SystemID + ".gm_fear_changed"),
		SystemID:      daggerheartsys.SystemID,
		SystemVersion: daggerheartsys.SystemVersion,
		EntityType:    "campaign",
		EntityID:      "camp-1",
		PayloadJSON:   payload,
	}

	if err := applier.Apply(ctx, evt); err != nil {
		t.Fatalf("apply: %v", err)
	}

	snapshot, err := daggerheartStore.GetDaggerheartSnapshot(ctx, "camp-1")
	if err != nil {
		t.Fatalf("load snapshot: %v", err)
	}
	if snapshot.GMFear != 4 {
		t.Fatalf("snapshot gm fear = %d, want %d", snapshot.GMFear, 4)
	}
}

// --- Parse helper tests ---

func TestParseCampaignStatus(t *testing.T) {
	tests := []struct {
		input string
		want  campaign.Status
		err   bool
	}{
		{"draft", campaign.StatusDraft, false},
		{"ACTIVE", campaign.StatusActive, false},
		{"completed", campaign.StatusCompleted, false},
		{"ARCHIVED", campaign.StatusArchived, false},
		{"CAMPAIGN_STATUS_DRAFT", campaign.StatusDraft, false},
		{"CAMPAIGN_STATUS_ACTIVE", campaign.StatusActive, false},
		{"CAMPAIGN_STATUS_COMPLETED", campaign.StatusCompleted, false},
		{"CAMPAIGN_STATUS_ARCHIVED", campaign.StatusArchived, false},
		{"", campaign.StatusUnspecified, true},
		{"   ", campaign.StatusUnspecified, true},
		{"unknown", campaign.StatusUnspecified, true},
	}
	for _, tt := range tests {
		got, err := parseCampaignStatus(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseCampaignStatus(%q) error = %v, wantErr %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseCampaignStatus(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseGmMode(t *testing.T) {
	tests := []struct {
		input string
		want  campaign.GmMode
		err   bool
	}{
		{"human", campaign.GmModeHuman, false},
		{"AI", campaign.GmModeAI, false},
		{"hybrid", campaign.GmModeHybrid, false},
		{"GM_MODE_HUMAN", campaign.GmModeHuman, false},
		{"GM_MODE_AI", campaign.GmModeAI, false},
		{"GM_MODE_HYBRID", campaign.GmModeHybrid, false},
		{"", campaign.GmModeUnspecified, true},
		{"unknown", campaign.GmModeUnspecified, true},
	}
	for _, tt := range tests {
		got, err := parseGmMode(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseGmMode(%q) error = %v, wantErr %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseGmMode(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseParticipantRole(t *testing.T) {
	tests := []struct {
		input string
		want  participant.Role
		err   bool
	}{
		{"gm", participant.RoleGM, false},
		{"PLAYER", participant.RolePlayer, false},
		{"GM", participant.RoleGM, false},
		{"player", participant.RolePlayer, false},
		{"", participant.RoleUnspecified, true},
		{"observer", participant.RoleUnspecified, true},
	}
	for _, tt := range tests {
		got, err := parseParticipantRole(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseParticipantRole(%q) error = %v, wantErr %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseParticipantRole(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseParticipantController(t *testing.T) {
	tests := []struct {
		input string
		want  participant.Controller
		err   bool
	}{
		{"human", participant.ControllerHuman, false},
		{"AI", participant.ControllerAI, false},
		{"CONTROLLER_HUMAN", participant.ControllerHuman, false},
		{"CONTROLLER_AI", participant.ControllerAI, false},
		{"", participant.ControllerUnspecified, true},
		{"bot", participant.ControllerUnspecified, true},
	}
	for _, tt := range tests {
		got, err := parseParticipantController(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseParticipantController(%q) error = %v, wantErr %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseParticipantController(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseCampaignAccess(t *testing.T) {
	tests := []struct {
		input string
		want  participant.CampaignAccess
		err   bool
	}{
		{"member", participant.CampaignAccessMember, false},
		{"MANAGER", participant.CampaignAccessManager, false},
		{"owner", participant.CampaignAccessOwner, false},
		{"CAMPAIGN_ACCESS_MEMBER", participant.CampaignAccessMember, false},
		{"CAMPAIGN_ACCESS_MANAGER", participant.CampaignAccessManager, false},
		{"CAMPAIGN_ACCESS_OWNER", participant.CampaignAccessOwner, false},
		{"", participant.CampaignAccessUnspecified, true},
		{"admin", participant.CampaignAccessUnspecified, true},
	}
	for _, tt := range tests {
		got, err := parseCampaignAccess(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseCampaignAccess(%q) error = %v, wantErr %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseCampaignAccess(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseInviteStatus(t *testing.T) {
	tests := []struct {
		input string
		want  invite.Status
		err   bool
	}{
		{"pending", invite.StatusPending, false},
		{"CLAIMED", invite.StatusClaimed, false},
		{"revoked", invite.StatusRevoked, false},
		{"PENDING", invite.StatusPending, false},
		{"", invite.StatusUnspecified, true},
		{"unknown", invite.StatusUnspecified, true},
	}
	for _, tt := range tests {
		got, err := parseInviteStatus(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseInviteStatus(%q) error = %v, wantErr %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseInviteStatus(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseCharacterKind(t *testing.T) {
	tests := []struct {
		input string
		want  character.Kind
		err   bool
	}{
		{"pc", character.KindPC, false},
		{"NPC", character.KindNPC, false},
		{"CHARACTER_KIND_PC", character.KindPC, false},
		{"CHARACTER_KIND_NPC", character.KindNPC, false},
		{"PC", character.KindPC, false},
		{"npc", character.KindNPC, false},
		{"", character.KindUnspecified, true},
		{"enemy", character.KindUnspecified, true},
	}
	for _, tt := range tests {
		got, err := parseCharacterKind(tt.input)
		if (err != nil) != tt.err {
			t.Errorf("parseCharacterKind(%q) error = %v, wantErr %v", tt.input, err, tt.err)
		}
		if got != tt.want {
			t.Errorf("parseCharacterKind(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// --- Fake character store ---

type fakeCharacterStore struct {
	characters map[string]storage.CharacterRecord
}

func newFakeCharacterStore() *fakeCharacterStore {
	return &fakeCharacterStore{characters: make(map[string]storage.CharacterRecord)}
}

func (s *fakeCharacterStore) PutCharacter(_ context.Context, c storage.CharacterRecord) error {
	s.characters[c.CampaignID+":"+c.ID] = c
	return nil
}

func (s *fakeCharacterStore) GetCharacter(_ context.Context, campaignID, characterID string) (storage.CharacterRecord, error) {
	c, ok := s.characters[campaignID+":"+characterID]
	if !ok {
		return storage.CharacterRecord{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *fakeCharacterStore) DeleteCharacter(_ context.Context, campaignID, characterID string) error {
	key := campaignID + ":" + characterID
	if _, ok := s.characters[key]; !ok {
		return storage.ErrNotFound
	}
	delete(s.characters, key)
	return nil
}

func (s *fakeCharacterStore) CountCharacters(_ context.Context, campaignID string) (int, error) {
	count := 0
	for key := range s.characters {
		if strings.HasPrefix(key, campaignID+":") {
			count++
		}
	}
	return count, nil
}

func (s *fakeCharacterStore) ListCharacters(context.Context, string, int, string) (storage.CharacterPage, error) {
	return storage.CharacterPage{}, nil
}

// --- Fake campaign fork store ---

type fakeCampaignForkStore struct {
	metadata map[string]storage.ForkMetadata
}

func newFakeCampaignForkStore() *fakeCampaignForkStore {
	return &fakeCampaignForkStore{metadata: make(map[string]storage.ForkMetadata)}
}

func (s *fakeCampaignForkStore) GetCampaignForkMetadata(_ context.Context, campaignID string) (storage.ForkMetadata, error) {
	m, ok := s.metadata[campaignID]
	if !ok {
		return storage.ForkMetadata{}, storage.ErrNotFound
	}
	return m, nil
}

func (s *fakeCampaignForkStore) SetCampaignForkMetadata(_ context.Context, campaignID string, metadata storage.ForkMetadata) error {
	s.metadata[campaignID] = metadata
	return nil
}

// --- applyCampaignForked tests ---

func TestApplyCampaignForked(t *testing.T) {
	ctx := context.Background()
	forkStore := newFakeCampaignForkStore()
	applier := Applier{CampaignFork: forkStore}

	payload := testevent.CampaignForkedPayload{
		ParentCampaignID: "parent-1",
		ForkEventSeq:     42,
		OriginCampaignID: "origin-1",
	}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignForked, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	m, err := forkStore.GetCampaignForkMetadata(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get metadata: %v", err)
	}
	if m.ParentCampaignID != "parent-1" {
		t.Fatalf("ParentCampaignID = %q, want %q", m.ParentCampaignID, "parent-1")
	}
	if m.ForkEventSeq != 42 {
		t.Fatalf("ForkEventSeq = %d, want 42", m.ForkEventSeq)
	}
	if m.OriginCampaignID != "origin-1" {
		t.Fatalf("OriginCampaignID = %q, want %q", m.OriginCampaignID, "origin-1")
	}
}

func TestApplyCampaignForked_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignForkedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignForked, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing fork store")
	}
}

func TestApplyCampaignForked_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignForkedPayload{})
	evt := testevent.Event{CampaignID: "", Type: testevent.TypeCampaignForked, PayloadJSON: data}
	applier := Applier{CampaignFork: newFakeCampaignForkStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

// --- applyParticipantUpdated tests ---

func TestApplyParticipantUpdated(t *testing.T) {
	ctx := context.Background()
	pStore := newProjectionParticipantStore()
	pStore.participants["camp-1:part-1"] = storage.ParticipantRecord{
		ID: "part-1", CampaignID: "camp-1", UserID: "user-1",
		Name: "Old Name", Role: participant.RolePlayer,
		Controller: participant.ControllerHuman, CampaignAccess: participant.CampaignAccessMember,
	}
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: pStore, Campaign: cStore}

	payload := testevent.ParticipantUpdatedPayload{Fields: map[string]any{
		"name":            "New Name",
		"role":            "GM",
		"controller":      "AI",
		"campaign_access": "OWNER",
		"user_id":         "user-2",
	}}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 15, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUpdated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	updated, err := pStore.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if updated.Name != "New Name" {
		t.Fatalf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Role != participant.RoleGM {
		t.Fatalf("Role = %v, want GM", updated.Role)
	}
	if updated.Controller != participant.ControllerAI {
		t.Fatalf("Controller = %v, want AI", updated.Controller)
	}
	if updated.CampaignAccess != participant.CampaignAccessOwner {
		t.Fatalf("CampaignAccess = %v, want OWNER", updated.CampaignAccess)
	}
	if updated.UserID != "user-2" {
		t.Fatalf("UserID = %q, want %q", updated.UserID, "user-2")
	}
}

func TestApplyParticipantUpdated_EmptyFields(t *testing.T) {
	ctx := context.Background()
	pStore := newProjectionParticipantStore()
	cStore := newProjectionCampaignStore()
	applier := Applier{Participant: pStore, Campaign: cStore}

	payload := testevent.ParticipantUpdatedPayload{Fields: map[string]any{}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply with empty fields should succeed: %v", err)
	}
}

func TestApplyParticipantUpdated_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantUpdatedPayload{Fields: map[string]any{"name": "x"}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUpdated, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing participant store")
	}
}

// --- applyParticipantLeft tests ---

func TestApplyParticipantLeft(t *testing.T) {
	ctx := context.Background()
	pStore := newProjectionParticipantStore()
	pStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", ParticipantCount: 3}
	applier := Applier{Participant: pStore, Campaign: cStore}

	data, _ := json.Marshal(testevent.ParticipantLeftPayload{})
	stamp := time.Date(2026, 2, 11, 15, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantLeft, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := pStore.GetParticipant(ctx, "camp-1", "part-1"); err == nil {
		t.Fatal("expected participant to be deleted")
	}
	c, _ := cStore.Get(ctx, "camp-1")
	// Count is derived from actual store records (0 remaining), not arithmetic.
	if c.ParticipantCount != 0 {
		t.Fatalf("ParticipantCount = %d, want 0", c.ParticipantCount)
	}
}

func TestApplyParticipantLeft_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantLeftPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantLeft, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplyParticipantLeft_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantLeftPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeParticipantLeft, PayloadJSON: data}
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

// --- applyParticipantBound tests ---

func TestApplyParticipantBound(t *testing.T) {
	ctx := context.Background()
	pStore := newProjectionParticipantStore()
	pStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	claimStore := newFakeClaimIndexStore()
	applier := Applier{Participant: pStore, Campaign: cStore, ClaimIndex: claimStore}

	payload := testevent.ParticipantBoundPayload{UserID: "user-1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 16, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantBound, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	updated, err := pStore.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if updated.UserID != "user-1" {
		t.Fatalf("UserID = %q, want %q", updated.UserID, "user-1")
	}
	if !claimStore.lastPutOK || claimStore.lastPut.UserID != "user-1" {
		t.Fatal("expected claim to be recorded")
	}
}

func TestApplyParticipantBound_MissingUserID(t *testing.T) {
	ctx := context.Background()
	pStore := newProjectionParticipantStore()
	pStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	cStore := newProjectionCampaignStore()
	applier := Applier{Participant: pStore, Campaign: cStore}

	payload := testevent.ParticipantBoundPayload{UserID: ""}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantBound, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing user ID")
	}
}

func TestApplyParticipantBound_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantBoundPayload{UserID: "u1"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantBound, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

// --- applyParticipantUnbound tests ---

func TestApplyParticipantUnbound_Success(t *testing.T) {
	ctx := context.Background()
	pStore := newProjectionParticipantStore()
	pStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "user-1"}
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	claimStore := newFakeClaimIndexStore()
	applier := Applier{Participant: pStore, Campaign: cStore, ClaimIndex: claimStore}

	payload := testevent.ParticipantUnboundPayload{UserID: "user-1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 16, 30, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUnbound, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	updated, err := pStore.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if updated.UserID != "" {
		t.Fatalf("UserID = %q, want empty", updated.UserID)
	}
	if len(claimStore.deleted) != 1 || claimStore.deleted[0] != "user-1" {
		t.Fatal("expected claim to be deleted for user-1")
	}
}

func TestApplyParticipantUnbound_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantUnboundPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUnbound, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplyParticipantUnbound_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantUnboundPayload{})
	evt := testevent.Event{CampaignID: "", EntityID: "part-1", Type: testevent.TypeParticipantUnbound, PayloadJSON: data}
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

// --- applyInviteClaimed tests ---

func TestApplyInviteClaimed(t *testing.T) {
	ctx := context.Background()
	inviteStore := newFakeInviteStore()
	inviteStore.invites["inv-1"] = storage.InviteRecord{ID: "inv-1", CampaignID: "camp-1", Status: invite.StatusPending}
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Invite: inviteStore, Campaign: cStore}

	payload := testevent.InviteClaimedPayload{InviteID: "inv-1"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 17, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteClaimed, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if inviteStore.updatedStatus["inv-1"] != invite.StatusClaimed {
		t.Fatalf("status = %v, want claimed", inviteStore.updatedStatus["inv-1"])
	}
	c, _ := cStore.Get(ctx, "camp-1")
	if !c.UpdatedAt.Equal(stamp) {
		t.Fatalf("campaign UpdatedAt = %v, want %v", c.UpdatedAt, stamp)
	}
}

func TestApplyInviteClaimed_MismatchID(t *testing.T) {
	ctx := context.Background()
	inviteStore := newFakeInviteStore()
	cStore := newProjectionCampaignStore()
	applier := Applier{Invite: inviteStore, Campaign: cStore}

	payload := testevent.InviteClaimedPayload{InviteID: "inv-other"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteClaimed, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invite ID mismatch")
	}
}

func TestApplyInviteClaimed_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteClaimedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteClaimed, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplyInviteClaimed_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteClaimedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeInviteClaimed, PayloadJSON: data}
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

// --- applyCharacterCreated tests ---

func TestApplyCharacterCreated(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", CharacterCount: 0}
	applier := Applier{Character: charStore, Campaign: cStore}

	payload := testevent.CharacterCreatedPayload{Name: "Aragorn", Kind: "PC", Notes: "A ranger"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 17, 30, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterCreated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	ch, err := charStore.GetCharacter(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get character: %v", err)
	}
	if ch.Name != "Aragorn" {
		t.Fatalf("Name = %q, want %q", ch.Name, "Aragorn")
	}
	if ch.Kind != character.KindPC {
		t.Fatalf("Kind = %v, want PC", ch.Kind)
	}
	c, _ := cStore.Get(ctx, "camp-1")
	// Count is derived from actual store records (1 character created).
	if c.CharacterCount != 1 {
		t.Fatalf("CharacterCount = %d, want 1", c.CharacterCount)
	}
}

func TestApplyCharacterCreated_IdempotentCount(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", CharacterCount: 0}
	applier := Applier{Character: charStore, Campaign: cStore}

	payload := testevent.CharacterCreatedPayload{Name: "Aragorn", Kind: "PC"}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 17, 30, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterCreated, PayloadJSON: data, Timestamp: stamp}

	// Apply the same event twice (idempotent replay).
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	c, _ := cStore.Get(ctx, "camp-1")
	if c.CharacterCount != 1 {
		t.Fatalf("CharacterCount = %d, want 1 (idempotent)", c.CharacterCount)
	}
}

func TestApplyCharacterCreated_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CharacterCreatedPayload{Name: "A", Kind: "PC"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterCreated, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplyCharacterCreated_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CharacterCreatedPayload{Name: "A", Kind: "PC"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeCharacterCreated, PayloadJSON: data}
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

// --- applyCharacterUpdated tests ---

func TestApplyCharacterUpdated(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{
		ID: "char-1", CampaignID: "camp-1", Name: "Old", Kind: character.KindPC,
	}
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Character: charStore, Campaign: cStore}

	payload := testevent.CharacterUpdatedPayload{Fields: map[string]any{
		"name":           "New Name",
		"kind":           "NPC",
		"notes":          "Some notes",
		"participant_id": "part-1",
	}}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 18, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	updated, err := charStore.GetCharacter(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get character: %v", err)
	}
	if updated.Name != "New Name" {
		t.Fatalf("Name = %q, want %q", updated.Name, "New Name")
	}
	if updated.Kind != character.KindNPC {
		t.Fatalf("Kind = %v, want NPC", updated.Kind)
	}
	if updated.Notes != "Some notes" {
		t.Fatalf("Notes = %q, want %q", updated.Notes, "Some notes")
	}
	if updated.ParticipantID != "part-1" {
		t.Fatalf("ParticipantID = %q, want %q", updated.ParticipantID, "part-1")
	}
}

func TestApplyCharacterUpdated_EmptyFields(t *testing.T) {
	ctx := context.Background()
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	payload := testevent.CharacterUpdatedPayload{Fields: map[string]any{}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply with empty fields should succeed: %v", err)
	}
}

func TestApplyCharacterUpdated_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CharacterUpdatedPayload{Fields: map[string]any{"name": "x"}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

// --- applyCharacterDeleted tests ---

func TestApplyCharacterDeleted(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1"}
	cStore := newProjectionCampaignStore()
	cStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", CharacterCount: 3}
	applier := Applier{Character: charStore, Campaign: cStore}

	data, _ := json.Marshal(testevent.CharacterDeletedPayload{})
	stamp := time.Date(2026, 2, 11, 18, 30, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterDeleted, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := charStore.GetCharacter(ctx, "camp-1", "char-1"); err == nil {
		t.Fatal("expected character to be deleted")
	}
	c, _ := cStore.Get(ctx, "camp-1")
	// Count is derived from actual store records (0 remaining), not arithmetic.
	if c.CharacterCount != 0 {
		t.Fatalf("CharacterCount = %d, want 0", c.CharacterCount)
	}
}

func TestApplyCharacterDeleted_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CharacterDeletedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterDeleted, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplyCharacterDeleted_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CharacterDeletedPayload{})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeCharacterDeleted, PayloadJSON: data}
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

// --- applyProfileUpdated tests ---

func TestApplyProfileUpdated(t *testing.T) {
	ctx := context.Background()
	dhStore := newProjectionDaggerheartStore()
	adapters := bridge.NewAdapterRegistry()
	_ = adapters.Register(daggerheartsys.NewAdapter(dhStore))
	applier := Applier{Adapters: adapters}

	payload := testevent.ProfileUpdatedPayload{
		SystemProfile: map[string]any{
			"daggerheart": map[string]any{
				"level":            1,
				"hp_max":           6,
				"stress_max":       6,
				"evasion":          10,
				"major_threshold":  4,
				"severe_threshold": 8,
				"proficiency":      1,
				"armor_score":      0,
				"armor_max":        0,
				"agility":          1,
				"strength":         0,
				"finesse":          2,
				"instinct":         1,
				"presence":         0,
				"knowledge":        -1,
				"experiences": []map[string]any{
					{"name": "Ranger", "modifier": 2},
				},
			},
		},
	}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeProfileUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	profile, err := dhStore.GetDaggerheartCharacterProfile(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.Level != 1 || profile.HpMax != 6 {
		t.Fatalf("profile level=%d hpMax=%d, want 1/6", profile.Level, profile.HpMax)
	}
	if profile.Agility != 1 || profile.Knowledge != -1 {
		t.Fatalf("traits agility=%d knowledge=%d, want 1/-1", profile.Agility, profile.Knowledge)
	}
}

func TestApplyProfileUpdated_NilSystemProfile(t *testing.T) {
	ctx := context.Background()
	adapters := bridge.NewAdapterRegistry()
	applier := Applier{Adapters: adapters}

	payload := testevent.ProfileUpdatedPayload{SystemProfile: nil}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeProfileUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply with nil system profile should succeed: %v", err)
	}
}

func TestApplyProfileUpdated_UnknownSystemSkipped(t *testing.T) {
	ctx := context.Background()
	adapters := bridge.NewAdapterRegistry()
	applier := Applier{Adapters: adapters}

	payload := testevent.ProfileUpdatedPayload{SystemProfile: map[string]any{"other": "data"}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeProfileUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply with unrecognized system key should succeed: %v", err)
	}
}

func TestApplyProfileUpdated_RoutedThroughAdapter(t *testing.T) {
	ctx := context.Background()
	dhStore := newProjectionDaggerheartStore()
	adapters := bridge.NewAdapterRegistry()
	if err := adapters.Register(daggerheartsys.NewAdapter(dhStore)); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	applier := Applier{Adapters: adapters}

	payload := testevent.ProfileUpdatedPayload{
		SystemProfile: map[string]any{
			"daggerheart": map[string]any{
				"level":            1,
				"hp_max":           6,
				"stress_max":       6,
				"evasion":          10,
				"major_threshold":  4,
				"severe_threshold": 8,
				"proficiency":      1,
				"armor_score":      0,
				"armor_max":        0,
				"agility":          1,
				"strength":         0,
				"finesse":          2,
				"instinct":         1,
				"presence":         0,
				"knowledge":        -1,
				"experiences": []map[string]any{
					{"name": "Ranger", "modifier": 2},
				},
			},
		},
	}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeProfileUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply via adapter: %v", err)
	}
	profile, err := dhStore.GetDaggerheartCharacterProfile(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if profile.Level != 1 || profile.HpMax != 6 {
		t.Fatalf("profile level=%d hpMax=%d, want 1/6", profile.Level, profile.HpMax)
	}
}

func TestApplyProfileUpdated_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ProfileUpdatedPayload{SystemProfile: map[string]any{"daggerheart": map[string]any{}}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeProfileUpdated, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplyProfileUpdated_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ProfileUpdatedPayload{SystemProfile: map[string]any{"daggerheart": map[string]any{}}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeProfileUpdated, PayloadJSON: data}
	adapters := bridge.NewAdapterRegistry()
	_ = adapters.Register(daggerheartsys.NewAdapter(newProjectionDaggerheartStore()))
	applier := Applier{Adapters: adapters}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

// --- applySessionGateOpened tests ---

func TestApplySessionGateOpened(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSessionGateStore()
	applier := Applier{SessionGate: gateStore}

	payload := testevent.SessionGateOpenedPayload{
		GateID:   "gate-1",
		GateType: "choice",
		Reason:   "Player decision needed",
		Metadata: map[string]any{"key": "value"},
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 19, 0, 0, 0, time.UTC)
	evt := testevent.Event{
		CampaignID: "camp-1", SessionID: "sess-1",
		Type: testevent.TypeSessionGateOpened, PayloadJSON: data, Timestamp: stamp,
		ActorType: testevent.ActorTypeGM, ActorID: "gm-1",
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gate, err := gateStore.GetSessionGate(ctx, "camp-1", "sess-1", "gate-1")
	if err != nil {
		t.Fatalf("get gate: %v", err)
	}
	if gate.Status != session.GateStatusOpen {
		t.Fatalf("Status = %q, want open", gate.Status)
	}
	if gate.GateType != "choice" {
		t.Fatalf("GateType = %q, want %q", gate.GateType, "choice")
	}
	if gate.CreatedByActorType != "gm" {
		t.Fatalf("CreatedByActorType = %q, want gm", gate.CreatedByActorType)
	}
}

func TestApplySessionGateOpened_FallbackEntityID(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSessionGateStore()
	applier := Applier{SessionGate: gateStore}

	payload := testevent.SessionGateOpenedPayload{GateID: "", GateType: "choice", Reason: "test"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{
		CampaignID: "camp-1", SessionID: "sess-1", EntityID: "gate-fallback",
		Type: testevent.TypeSessionGateOpened, PayloadJSON: data,
		Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if _, err := gateStore.GetSessionGate(ctx, "camp-1", "sess-1", "gate-fallback"); err != nil {
		t.Fatalf("expected gate with entity ID fallback, got err: %v", err)
	}
}

func TestApplySessionGateOpened_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateOpenedPayload{GateID: "g", GateType: "choice"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionGateOpened, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplySessionGateOpened_MissingSessionID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateOpenedPayload{GateID: "g", GateType: "choice"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "", Type: testevent.TypeSessionGateOpened, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

func TestApplySessionGateOpened_MissingGateID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateOpenedPayload{GateID: "", GateType: "choice"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", EntityID: "", Type: testevent.TypeSessionGateOpened, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing gate ID")
	}
}

// --- applySessionGateResolved tests ---

func TestApplySessionGateResolved(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSessionGateStore()
	gateStore.gates["camp-1:sess-1:gate-1"] = storage.SessionGate{
		CampaignID: "camp-1", SessionID: "sess-1", GateID: "gate-1",
		Status: session.GateStatusOpen,
	}
	applier := Applier{SessionGate: gateStore}

	payload := testevent.SessionGateResolvedPayload{
		GateID:     "gate-1",
		Decision:   "approved",
		Resolution: map[string]any{"detail": "yes"},
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 19, 30, 0, 0, time.UTC)
	evt := testevent.Event{
		CampaignID: "camp-1", SessionID: "sess-1",
		Type: testevent.TypeSessionGateResolved, PayloadJSON: data, Timestamp: stamp,
		ActorType: testevent.ActorTypeGM, ActorID: "gm-1",
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gate, err := gateStore.GetSessionGate(ctx, "camp-1", "sess-1", "gate-1")
	if err != nil {
		t.Fatalf("get gate: %v", err)
	}
	if gate.Status != session.GateStatusResolved {
		t.Fatalf("Status = %q, want resolved", gate.Status)
	}
	if gate.ResolvedAt == nil || !gate.ResolvedAt.Equal(stamp) {
		t.Fatal("ResolvedAt mismatch")
	}
	if gate.ResolvedByActorType != "gm" {
		t.Fatalf("ResolvedByActorType = %q, want gm", gate.ResolvedByActorType)
	}
}

func TestApplySessionGateResolved_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateResolvedPayload{GateID: "g"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionGateResolved, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing store")
	}
}

func TestApplySessionGateResolved_MissingGateID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateResolvedPayload{GateID: ""})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", EntityID: "", Type: testevent.TypeSessionGateResolved, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing gate ID")
	}
}

// --- applyCampaignCreated tests ---

func TestApplyCampaignCreated(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	applier := Applier{Campaign: store}

	payload := testevent.CampaignCreatedPayload{
		Name:        "Test Campaign",
		GameSystem:  "GAME_SYSTEM_DAGGERHEART",
		GmMode:      "human",
		ThemePrompt: "dark forest",
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	evt := testevent.Event{EntityID: "camp-1", Type: testevent.TypeCampaignCreated, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	c, err := store.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if c.Name != "Test Campaign" {
		t.Fatalf("Name = %q, want %q", c.Name, "Test Campaign")
	}
	if c.Status != campaign.StatusDraft {
		t.Fatalf("Status = %v, want Draft", c.Status)
	}
	if c.ThemePrompt != "dark forest" {
		t.Fatalf("ThemePrompt = %q", c.ThemePrompt)
	}
}

func TestApplyCampaignCreated_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignCreatedPayload{Name: "X", GameSystem: "GAME_SYSTEM_DAGGERHEART", GmMode: "human"})
	evt := testevent.Event{EntityID: "camp-1", Type: testevent.TypeCampaignCreated, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyCampaignCreated_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignCreatedPayload{Name: "X", GameSystem: "GAME_SYSTEM_DAGGERHEART", GmMode: "human"})
	evt := testevent.Event{EntityID: "", Type: testevent.TypeCampaignCreated, PayloadJSON: data}
	applier := Applier{Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

func TestApplyCampaignCreated_InvalidJSON(t *testing.T) {
	ctx := context.Background()
	evt := testevent.Event{EntityID: "camp-1", Type: testevent.TypeCampaignCreated, PayloadJSON: []byte("{")}
	applier := Applier{Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyCampaignCreated_InvalidGameSystem(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignCreatedPayload{Name: "X", GameSystem: "", GmMode: "human"})
	evt := testevent.Event{EntityID: "camp-1", Type: testevent.TypeCampaignCreated, PayloadJSON: data}
	applier := Applier{Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid game system")
	}
}

func TestApplyCampaignCreated_InvalidGmMode(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignCreatedPayload{Name: "X", GameSystem: "DAGGERHEART", GmMode: ""})
	evt := testevent.Event{EntityID: "camp-1", Type: testevent.TypeCampaignCreated, PayloadJSON: data}
	applier := Applier{Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid gm mode")
	}
}

// --- applyCampaignUpdated additional tests ---

func TestApplyCampaignUpdated_ThemePrompt(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", Status: campaign.StatusDraft, ThemePrompt: "old"}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{"theme_prompt": "  new theme  "}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data, Timestamp: time.Now()}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	c, _ := store.Get(ctx, "camp-1")
	if c.ThemePrompt != "new theme" {
		t.Fatalf("ThemePrompt = %q, want %q", c.ThemePrompt, "new theme")
	}
}

func TestApplyCampaignUpdated_EmptyFields(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
}

func TestApplyCampaignUpdated_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignUpdatedPayload{Fields: map[string]any{"name": "X"}})
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyCampaignUpdated_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.CampaignUpdatedPayload{Fields: map[string]any{"name": "X"}})
	evt := testevent.Event{CampaignID: "", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}
	applier := Applier{Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

func TestApplyCampaignUpdated_InvalidNameType(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{"name": 42}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid name type")
	}
}

func TestApplyCampaignUpdated_EmptyName(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{"name": "  "}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestApplyCampaignUpdated_InvalidStatusType(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", Status: campaign.StatusDraft}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{"status": 42}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid status type")
	}
}

func TestApplyCampaignUpdated_InvalidThemePromptType(t *testing.T) {
	ctx := context.Background()
	store := newProjectionCampaignStore()
	store.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Campaign: store}

	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{"theme_prompt": 42}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}

	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid theme_prompt type")
	}
}

// --- applyParticipantJoined tests ---

func TestApplyParticipantJoined(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", ParticipantCount: 0}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	payload := testevent.ParticipantJoinedPayload{
		UserID:         "user-1",
		Name:           "Alice",
		Role:           "player",
		Controller:     "human",
		CampaignAccess: "member",
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantJoined, PayloadJSON: data, Timestamp: stamp}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	p, err := participantStore.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("get participant: %v", err)
	}
	if p.Name != "Alice" {
		t.Fatalf("Name = %q, want Alice", p.Name)
	}
	c, _ := campaignStore.Get(ctx, "camp-1")
	if c.ParticipantCount != 1 {
		t.Fatalf("ParticipantCount = %d, want 1", c.ParticipantCount)
	}
}

func TestApplyParticipantJoined_IdempotentCount(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", ParticipantCount: 0}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	payload := testevent.ParticipantJoinedPayload{
		UserID: "user-1", Name: "Alice", Role: "player",
		Controller: "human", CampaignAccess: "member",
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 11, 10, 0, 0, 0, time.UTC)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantJoined, PayloadJSON: data, Timestamp: stamp}

	// Apply the same event twice (idempotent replay).
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	c, _ := campaignStore.Get(ctx, "camp-1")
	if c.ParticipantCount != 1 {
		t.Fatalf("ParticipantCount = %d, want 1 (idempotent)", c.ParticipantCount)
	}
}

func TestApplyParticipantJoined_MissingStores(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantJoinedPayload{UserID: "u", Name: "A", Role: "player", Controller: "human", CampaignAccess: "member"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantJoined, PayloadJSON: data}

	if err := (Applier{Campaign: newProjectionCampaignStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing participant store")
	}
	if err := (Applier{Participant: newProjectionParticipantStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyParticipantJoined_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantJoinedPayload{UserID: "u", Name: "A", Role: "player", Controller: "human", CampaignAccess: "member"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeParticipantJoined, PayloadJSON: data}
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

func TestApplyParticipantJoined_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.ParticipantJoinedPayload{UserID: "u", Name: "A", Role: "player", Controller: "human", CampaignAccess: "member"})
	evt := testevent.Event{CampaignID: "", EntityID: "part-1", Type: testevent.TypeParticipantJoined, PayloadJSON: data}
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

// --- applySeatReassigned additional tests ---

func TestApplySeatReassigned_MissingStores(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SeatReassignedPayload{UserID: "user-new"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeSeatReassigned, PayloadJSON: data}

	if err := (Applier{Campaign: newProjectionCampaignStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing participant store")
	}
	if err := (Applier{Participant: newProjectionParticipantStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplySeatReassigned_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SeatReassignedPayload{UserID: "user-new"})
	evt := testevent.Event{CampaignID: "", EntityID: "part-1", Type: testevent.TypeSeatReassigned, PayloadJSON: data}
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

func TestApplySeatReassigned_MissingEntityID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SeatReassignedPayload{UserID: "user-new"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeSeatReassigned, PayloadJSON: data}
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity ID")
	}
}

func TestApplySeatReassigned_MissingUserID(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "user-old"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.SeatReassignedPayload{UserID: "", PriorUserID: "user-old"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeSeatReassigned, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing user ID")
	}
}

func TestApplySeatReassigned_PriorUserMismatch(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "user-old"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.SeatReassignedPayload{UserID: "user-new", PriorUserID: "user-wrong"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeSeatReassigned, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for prior user mismatch")
	}
}

func TestApplySeatReassigned_NoClaims(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "user-old"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Events: testEventRegistry(t), Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.SeatReassignedPayload{UserID: "user-new"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeSeatReassigned, PayloadJSON: data, Timestamp: time.Now()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	p, _ := participantStore.GetParticipant(ctx, "camp-1", "part-1")
	if p.UserID != "user-new" {
		t.Fatalf("UserID = %q, want user-new", p.UserID)
	}
}

// --- applyInviteCreated additional tests ---

func TestApplyInviteCreated_MissingStores(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteCreatedPayload{InviteID: "inv-1", ParticipantID: "p1", Status: "pending"})
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeInviteCreated, PayloadJSON: data}

	if err := (Applier{Campaign: newProjectionCampaignStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing invite store")
	}
	if err := (Applier{Invite: newFakeInviteStore()}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyInviteCreated_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteCreatedPayload{InviteID: "inv-1", ParticipantID: "p1", Status: "pending"})
	evt := testevent.Event{CampaignID: "", Type: testevent.TypeInviteCreated, PayloadJSON: data}
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

func TestApplyInviteCreated_MissingInviteID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteCreatedPayload{InviteID: "", ParticipantID: "p1", Status: "pending"})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeInviteCreated, PayloadJSON: data}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Invite: newFakeInviteStore(), Campaign: campaignStore}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing invite ID")
	}
}

func TestApplyInviteCreated_MissingParticipantID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.InviteCreatedPayload{InviteID: "inv-1", ParticipantID: "", Status: "pending"})
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeInviteCreated, PayloadJSON: data}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Invite: newFakeInviteStore(), Campaign: campaignStore}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing participant ID")
	}
}

// --- applySessionStarted additional tests ---

func TestApplySessionStarted_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionStartedPayload{SessionID: "sess-1"})
	evt := testevent.Event{CampaignID: "", Type: testevent.TypeSessionStarted, PayloadJSON: data}
	applier := Applier{Session: &fakeSessionStore{}}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

func TestApplySessionStarted_MissingStore(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionStartedPayload{SessionID: "sess-1"})
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeSessionStarted, PayloadJSON: data}
	if err := (Applier{}).Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session store")
	}
}

func TestApplySessionStarted_MissingSessionID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionStartedPayload{SessionID: ""})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeSessionStarted, PayloadJSON: data}
	applier := Applier{Session: &fakeSessionStore{}}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

// --- applySessionEnded additional tests ---

func TestApplySessionEnded_EntityIDFallback(t *testing.T) {
	ctx := context.Background()
	sessionStore := &fakeSessionStore{}
	applier := Applier{Session: sessionStore}

	payload := testevent.SessionEndedPayload{SessionID: ""}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "sess-fallback", Type: testevent.TypeSessionEnded, PayloadJSON: data, Timestamp: time.Now()}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
}

func TestApplySessionEnded_MissingSessionID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionEndedPayload{SessionID: ""})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "", Type: testevent.TypeSessionEnded, PayloadJSON: data}
	applier := Applier{Session: &fakeSessionStore{}}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

// --- applySessionGateAbandoned additional tests ---

func TestApplySessionGateAbandoned_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateAbandonedPayload{GateID: "gate-1"})
	evt := testevent.Event{CampaignID: "", SessionID: "sess-1", Type: testevent.TypeSessionGateAbandoned, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

func TestApplySessionGateAbandoned_MissingGateID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateAbandonedPayload{GateID: ""})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", EntityID: "", Type: testevent.TypeSessionGateAbandoned, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing gate ID")
	}
}

// --- applySessionGateResolved additional tests ---

func TestApplySessionGateResolved_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateResolvedPayload{GateID: "g"})
	evt := testevent.Event{CampaignID: "", SessionID: "sess-1", Type: testevent.TypeSessionGateResolved, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

func TestApplySessionGateResolved_MissingSessionID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateResolvedPayload{GateID: "g"})
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "", Type: testevent.TypeSessionGateResolved, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing session ID")
	}
}

// --- applySessionGateOpened additional tests ---

func TestApplySessionGateOpened_MissingCampaignID(t *testing.T) {
	ctx := context.Background()
	data, _ := json.Marshal(testevent.SessionGateOpenedPayload{GateID: "g", GateType: "choice"})
	evt := testevent.Event{CampaignID: "", SessionID: "sess-1", Type: testevent.TypeSessionGateOpened, PayloadJSON: data}
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign ID")
	}
}

// --- applyParticipantUpdated type assertion errors ---

func TestApplyParticipantUpdated_InvalidUserIDType(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.ParticipantUpdatedPayload{Fields: map[string]any{"user_id": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid user_id type")
	}
}

func TestApplyParticipantUpdated_InvalidNameType(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.ParticipantUpdatedPayload{Fields: map[string]any{"name": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid display_name type")
	}
}

func TestApplyParticipantUpdated_EmptyName(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.ParticipantUpdatedPayload{Fields: map[string]any{"name": "  "}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for empty display name")
	}
}

func TestApplyParticipantUpdated_InvalidRoleType(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.ParticipantUpdatedPayload{Fields: map[string]any{"role": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid role type")
	}
}

func TestApplyParticipantUpdated_InvalidControllerType(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.ParticipantUpdatedPayload{Fields: map[string]any{"controller": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid controller type")
	}
}

func TestApplyParticipantUpdated_InvalidAccessType(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.ParticipantUpdatedPayload{Fields: map[string]any{"campaign_access": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid campaign_access type")
	}
}

// --- applyCharacterUpdated type assertion errors ---

func TestApplyCharacterUpdated_InvalidNameType(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Character: charStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.CharacterUpdatedPayload{Fields: map[string]any{"name": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid name type")
	}
}

func TestApplyCharacterUpdated_EmptyName(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Character: charStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.CharacterUpdatedPayload{Fields: map[string]any{"name": "  "}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for empty character name")
	}
}

func TestApplyCharacterUpdated_InvalidKindType(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Character: charStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.CharacterUpdatedPayload{Fields: map[string]any{"kind": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid kind type")
	}
}

func TestApplyCharacterUpdated_InvalidNotesType(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Character: charStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.CharacterUpdatedPayload{Fields: map[string]any{"notes": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid notes type")
	}
}

func TestApplyCharacterUpdated_InvalidParticipantIDType(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Character: charStore, Campaign: campaignStore}

	data, _ := json.Marshal(testevent.CharacterUpdatedPayload{Fields: map[string]any{"participant_id": 42}})
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid participant_id type")
	}
}

// --- marshalOptionalMap tests ---

// --- applyParticipantLeft missing branches ---

func TestApplyParticipantLeft_MissingCampaignStore(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantLeft, PayloadJSON: []byte("{}")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyParticipantLeft_MissingCampaignID(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "  ", EntityID: "part-1", Type: testevent.TypeParticipantLeft, PayloadJSON: []byte("{}")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign id")
	}
}

func TestApplyParticipantLeft_ZeroCount(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", ParticipantCount: 0}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantLeft, PayloadJSON: []byte("{}"), Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	c, _ := campaignStore.Get(ctx, "camp-1")
	if c.ParticipantCount != 0 {
		t.Fatalf("ParticipantCount = %d, want 0", c.ParticipantCount)
	}
}

// --- applyParticipantBound missing branches ---

func TestApplyParticipantBound_MissingCampaignStore(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantBound, PayloadJSON: []byte(`{"user_id":"u1"}`)}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyParticipantBound_MissingCampaignID(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "  ", EntityID: "part-1", Type: testevent.TypeParticipantBound, PayloadJSON: []byte(`{"user_id":"u1"}`)}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign id")
	}
}

func TestApplyParticipantBound_MissingEntityID(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "  ", Type: testevent.TypeParticipantBound, PayloadJSON: []byte(`{"user_id":"u1"}`)}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity id")
	}
}

func TestApplyParticipantBound_InvalidJSON(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantBound, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applyParticipantUnbound missing branches ---

func TestApplyParticipantUnbound_MissingEntityID(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "  ", Type: testevent.TypeParticipantUnbound, PayloadJSON: []byte(`{"user_id":"u1"}`)}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity id")
	}
}

func TestApplyParticipantUnbound_InvalidJSON(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUnbound, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyParticipantUnbound_NilClaimIndex(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "u1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}
	payload := testevent.ParticipantUnboundPayload{}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUnbound, PayloadJSON: data, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	p, _ := participantStore.GetParticipant(ctx, "camp-1", "part-1")
	if p.UserID != "" {
		t.Fatalf("UserID = %q, want empty", p.UserID)
	}
}

// --- applyInviteClaimed missing branches ---

func TestApplyInviteClaimed_MissingCampaignStore(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteClaimed, PayloadJSON: []byte("{}")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyInviteClaimed_InvalidJSON(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteClaimed, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applyInviteRevoked missing branches ---

func TestApplyInviteRevoked_MismatchID(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	payload := testevent.InviteRevokedPayload{InviteID: "inv-2"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteRevoked, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invite id mismatch")
	}
}

func TestApplyInviteRevoked_InvalidJSON(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteRevoked, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applyCharacterCreated missing branches ---

func TestApplyCharacterCreated_MissingCampaignStore(t *testing.T) {
	applier := Applier{Character: newFakeCharacterStore()}
	payload := testevent.CharacterCreatedPayload{Name: "Hero", Kind: "PC"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterCreated, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyCharacterCreated_MissingCampaignID(t *testing.T) {
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	payload := testevent.CharacterCreatedPayload{Name: "Hero", Kind: "PC"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "  ", EntityID: "char-1", Type: testevent.TypeCharacterCreated, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign id")
	}
}

func TestApplyCharacterCreated_InvalidJSON(t *testing.T) {
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterCreated, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyCharacterCreated_InvalidKind(t *testing.T) {
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	payload := testevent.CharacterCreatedPayload{Name: "Hero", Kind: "ALIEN"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterCreated, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

// --- applyCharacterDeleted missing branches ---

func TestApplyCharacterDeleted_MissingCampaignStore(t *testing.T) {
	applier := Applier{Character: newFakeCharacterStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterDeleted, PayloadJSON: []byte("{}")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign store")
	}
}

func TestApplyCharacterDeleted_MissingCampaignID(t *testing.T) {
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "  ", EntityID: "char-1", Type: testevent.TypeCharacterDeleted, PayloadJSON: []byte("{}")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign id")
	}
}

func TestApplyCharacterDeleted_ZeroCount(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1", CharacterCount: 0}
	applier := Applier{Character: charStore, Campaign: campaignStore}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterDeleted, PayloadJSON: []byte("{}"), Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	c, _ := campaignStore.Get(ctx, "camp-1")
	if c.CharacterCount != 0 {
		t.Fatalf("CharacterCount = %d, want 0", c.CharacterCount)
	}
}

// --- applyProfileUpdated missing branches ---

func TestApplyProfileUpdated_MissingCampaignID(t *testing.T) {
	adapters := bridge.NewAdapterRegistry()
	_ = adapters.Register(daggerheartsys.NewAdapter(newProjectionDaggerheartStore()))
	applier := Applier{Adapters: adapters}
	evt := testevent.Event{CampaignID: "  ", EntityID: "char-1", Type: testevent.TypeProfileUpdated, PayloadJSON: []byte("{}")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign id")
	}
}

func TestApplyProfileUpdated_InvalidJSON(t *testing.T) {
	adapters := bridge.NewAdapterRegistry()
	_ = adapters.Register(daggerheartsys.NewAdapter(newProjectionDaggerheartStore()))
	applier := Applier{Adapters: adapters}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeProfileUpdated, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applySessionSpotlightSet missing branches ---

func TestApplySessionSpotlightSet_InvalidJSON(t *testing.T) {
	applier := Applier{SessionSpotlight: newFakeSessionSpotlightStore()}
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionSpotlightSet, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplySessionSpotlightSet_InvalidSpotlightType(t *testing.T) {
	applier := Applier{SessionSpotlight: newFakeSessionSpotlightStore()}
	payload := map[string]any{"spotlight_type": "INVALID_TYPE"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionSpotlightSet, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid spotlight type")
	}
}

func TestApplySessionSpotlightSet_InvalidTarget(t *testing.T) {
	applier := Applier{SessionSpotlight: newFakeSessionSpotlightStore()}
	payload := map[string]any{"spotlight_type": "CHARACTER", "character_id": ""}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionSpotlightSet, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid spotlight target")
	}
}

func TestApply_UnhandledCoreEventReturnsError(t *testing.T) {
	applier := Applier{}
	evt := event.Event{
		CampaignID:  "camp-1",
		Type:        event.Type("campaign.made_up"),
		PayloadJSON: []byte("{}"),
	}
	if err := applier.Apply(context.Background(), evt); err == nil {
		t.Fatal("expected error for unhandled core event")
	}
}

// --- applySystemEvent missing branches ---

func TestApplySystemEvent_MissingSystemID(t *testing.T) {
	registry := bridge.NewAdapterRegistry()
	if err := registry.Register(&fakeAdapter{}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	applier := Applier{Adapters: registry}
	// Call applySystemEvent directly to hit the empty SystemID guard
	evt := event.Event{CampaignID: "camp-1", Type: "system.custom", SystemID: "  ", PayloadJSON: []byte("{}")}
	if err := applier.applySystemEvent(context.Background(), evt); err == nil {
		t.Fatal("expected error for missing system_id")
	}
}

func TestApplySystemEvent_InvalidGameSystem(t *testing.T) {
	registry := bridge.NewAdapterRegistry()
	if err := registry.Register(&fakeAdapter{}); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	applier := Applier{Adapters: registry}
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.Type("system.custom"), SystemID: "INVALID_SYSTEM", PayloadJSON: []byte("{}")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid game system")
	}
}

func TestApplySystemEvent_UnhandledSystemEventReturnsError(t *testing.T) {
	registry := bridge.NewAdapterRegistry()
	if err := registry.Register(daggerheartsys.NewAdapter(newProjectionDaggerheartStore())); err != nil {
		t.Fatalf("register adapter: %v", err)
	}
	applier := Applier{Adapters: registry}
	evt := event.Event{
		CampaignID:    "camp-1",
		Type:          event.Type("sys.daggerheart.unhandled_system_event"),
		SystemID:      daggerheartsys.SystemID,
		SystemVersion: daggerheartsys.SystemVersion,
		EntityType:    "campaign",
		EntityID:      "camp-1",
		PayloadJSON:   []byte("{}"),
	}
	if err := applier.applySystemEvent(context.Background(), evt); err == nil {
		t.Fatal("expected error for unhandled system event")
	}
}

// --- applyCampaignForked missing branches ---

func TestApplyCampaignForked_InvalidJSON(t *testing.T) {
	applier := Applier{CampaignFork: newFakeCampaignForkStore()}
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignForked, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- marshalResolutionPayload missing branches ---

func TestMarshalResolutionPayload(t *testing.T) {
	// Empty decision and resolution returns nil
	result, err := marshalResolutionPayload("", nil)
	if err != nil || result != nil {
		t.Fatalf("expected nil, got %v, %v", result, err)
	}

	// Decision only
	result, err = marshalResolutionPayload("approve", nil)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for decision-only")
	}

	// Resolution only
	result, err = marshalResolutionPayload("", map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for resolution-only")
	}
}

// --- applySessionGateOpened missing branches ---

func TestApplySessionGateOpened_EmptyGateType(t *testing.T) {
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	payload := map[string]any{"gate_id": "gate-1", "gate_type": "  "}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionGateOpened, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for empty gate type")
	}
}

func TestApplySessionGateOpened_InvalidJSON(t *testing.T) {
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", Type: testevent.TypeSessionGateOpened, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applySessionGateResolved missing branches ---

func TestApplySessionGateResolved_EntityIDFallback(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSessionGateStore()
	gateStore.gates["camp-1:sess-1:gate-1"] = storage.SessionGate{
		CampaignID: "camp-1", SessionID: "sess-1", GateID: "gate-1", Status: session.GateStatusOpen,
	}
	applier := Applier{SessionGate: gateStore}
	payload := testevent.SessionGateResolvedPayload{Decision: "approve"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", EntityID: "gate-1", Type: testevent.TypeSessionGateResolved, PayloadJSON: data, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gate := gateStore.gates["camp-1:sess-1:gate-1"]
	if gate.Status != session.GateStatusResolved {
		t.Fatalf("gate status = %q, want %q", gate.Status, session.GateStatusResolved)
	}
}

func TestApplySessionGateResolved_InvalidJSON(t *testing.T) {
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", EntityID: "gate-1", Type: testevent.TypeSessionGateResolved, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applySessionGateAbandoned missing branches ---

func TestApplySessionGateAbandoned_EntityIDFallback(t *testing.T) {
	ctx := context.Background()
	gateStore := newFakeSessionGateStore()
	gateStore.gates["camp-1:sess-1:gate-1"] = storage.SessionGate{
		CampaignID: "camp-1", SessionID: "sess-1", GateID: "gate-1", Status: session.GateStatusOpen,
	}
	applier := Applier{SessionGate: gateStore}
	payload := testevent.SessionGateAbandonedPayload{Reason: "timeout"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", EntityID: "gate-1", Type: testevent.TypeSessionGateAbandoned, PayloadJSON: data, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	gate := gateStore.gates["camp-1:sess-1:gate-1"]
	if gate.Status != session.GateStatusAbandoned {
		t.Fatalf("gate status = %q, want %q", gate.Status, session.GateStatusAbandoned)
	}
}

func TestApplySessionGateAbandoned_InvalidJSON(t *testing.T) {
	applier := Applier{SessionGate: newFakeSessionGateStore()}
	evt := testevent.Event{CampaignID: "camp-1", SessionID: "sess-1", EntityID: "gate-1", Type: testevent.TypeSessionGateAbandoned, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applyInviteCreated missing branches ---

func TestApplyInviteCreated_InvalidStatus(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	payload := map[string]any{"invite_id": "inv-1", "participant_id": "part-1", "status": "INVALID"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeInviteCreated, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid invite status")
	}
}

func TestApplyInviteCreated_InvalidJSON(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeInviteCreated, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applyInviteUpdated missing branches ---

func TestApplyInviteUpdated_InvalidJSON(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "inv-1", Type: testevent.TypeInviteUpdated, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyInviteUpdated_InvalidStatus(t *testing.T) {
	applier := Applier{Invite: newFakeInviteStore()}
	payload := map[string]any{"invite_id": "inv-1", "status": "INVALID"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeInviteUpdated, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid invite status")
	}
}

// --- applySessionStarted missing branches ---

func TestApplySessionStarted_InvalidJSON(t *testing.T) {
	applier := Applier{Session: &fakeSessionStore{}}
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeSessionStarted, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applySessionEnded missing branches ---

func TestApplySessionEnded_InvalidJSON(t *testing.T) {
	applier := Applier{Session: &fakeSessionStore{}}
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeSessionEnded, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applyCampaignUpdated missing branches ---

func TestApplyCampaignUpdated_InvalidJSON(t *testing.T) {
	applier := Applier{Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyCampaignUpdated_InvalidStatus(t *testing.T) {
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Campaign: campaignStore}
	payload := testevent.CampaignUpdatedPayload{Fields: map[string]any{"status": "INVALID"}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", Type: testevent.TypeCampaignUpdated, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid status value")
	}
}

// --- applyParticipantJoined missing branches ---

func TestApplyParticipantJoined_InvalidJSON(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantJoined, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// --- applyCharacterUpdated missing branches ---

func TestApplyCharacterUpdated_InvalidJSON(t *testing.T) {
	applier := Applier{Character: newFakeCharacterStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyCharacterUpdated_InvalidKind(t *testing.T) {
	ctx := context.Background()
	charStore := newFakeCharacterStore()
	charStore.characters["camp-1:char-1"] = storage.CharacterRecord{ID: "char-1", CampaignID: "camp-1", Name: "Hero"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Character: charStore, Campaign: campaignStore}
	payload := testevent.CharacterUpdatedPayload{Fields: map[string]any{"kind": "ALIEN"}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeCharacterUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid kind")
	}
}

// --- applyParticipantUpdated missing branches ---

func TestApplyParticipantUpdated_MissingCampaignID(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "  ", EntityID: "part-1", Type: testevent.TypeParticipantUpdated, PayloadJSON: []byte(`{"fields":{"name":"X"}}`)}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing campaign id")
	}
}

func TestApplyParticipantUpdated_MissingEntityID(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "  ", Type: testevent.TypeParticipantUpdated, PayloadJSON: []byte(`{"fields":{"name":"X"}}`)}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for missing entity id")
	}
}

func TestApplyParticipantUpdated_InvalidJSON(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUpdated, PayloadJSON: []byte("{")}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestApplyParticipantUpdated_InvalidRole(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}
	payload := testevent.ParticipantUpdatedPayload{Fields: map[string]any{"role": "ALIEN"}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestApplyParticipantUpdated_InvalidController(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}
	payload := testevent.ParticipantUpdatedPayload{Fields: map[string]any{"controller": "ALIEN"}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid controller")
	}
}

func TestApplyParticipantUpdated_InvalidAccess(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	applier := Applier{Participant: participantStore, Campaign: campaignStore}
	payload := testevent.ParticipantUpdatedPayload{Fields: map[string]any{"campaign_access": "ALIEN"}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid access")
	}
}

// --- applyParticipantJoined parser error branches ---

func TestApplyParticipantJoined_InvalidRole(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	payload := testevent.ParticipantJoinedPayload{Name: "A", Role: "ALIEN", Controller: "HUMAN", CampaignAccess: "READ_ONLY"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantJoined, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestApplyParticipantJoined_InvalidController(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	payload := testevent.ParticipantJoinedPayload{Name: "A", Role: "PLAYER", Controller: "ALIEN", CampaignAccess: "READ_ONLY"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantJoined, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid controller")
	}
}

func TestApplyParticipantJoined_InvalidAccess(t *testing.T) {
	applier := Applier{Participant: newProjectionParticipantStore(), Campaign: newProjectionCampaignStore()}
	payload := testevent.ParticipantJoinedPayload{Name: "A", Role: "PLAYER", Controller: "HUMAN", CampaignAccess: "ALIEN"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantJoined, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid access")
	}
}

// --- applyParticipantBound with ClaimIndex ---

func TestApplyParticipantBound_WithClaimIndex(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	claimStore := newFakeClaimIndexStore()
	applier := Applier{Participant: participantStore, Campaign: campaignStore, ClaimIndex: claimStore}
	payload := testevent.ParticipantBoundPayload{UserID: "u1"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantBound, PayloadJSON: data, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if !claimStore.lastPutOK {
		t.Fatal("expected claim to be written")
	}
}

// --- applyParticipantUnbound with ClaimIndex ---

func TestApplyParticipantUnbound_WithClaimIndex(t *testing.T) {
	ctx := context.Background()
	participantStore := newProjectionParticipantStore()
	participantStore.participants["camp-1:part-1"] = storage.ParticipantRecord{ID: "part-1", CampaignID: "camp-1", UserID: "u1"}
	campaignStore := newProjectionCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	claimStore := newFakeClaimIndexStore()
	applier := Applier{Participant: participantStore, Campaign: campaignStore, ClaimIndex: claimStore}
	payload := testevent.ParticipantUnboundPayload{UserID: "u1"}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "part-1", Type: testevent.TypeParticipantUnbound, PayloadJSON: data, Timestamp: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(claimStore.deleted) != 1 || claimStore.deleted[0] != "u1" {
		t.Fatalf("expected claim deletion for u1, got %v", claimStore.deleted)
	}
}

// --- applyProfileUpdated validation profile branch ---

func TestApplyProfileUpdated_InvalidProfileData(t *testing.T) {
	adapters := bridge.NewAdapterRegistry()
	_ = adapters.Register(daggerheartsys.NewAdapter(newProjectionDaggerheartStore()))
	applier := Applier{Adapters: adapters}
	payload := map[string]any{"system_profile": map[string]any{"daggerheart": "not-an-object"}}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeProfileUpdated, PayloadJSON: data}
	if err := applier.Apply(context.Background(), eventToEvent(evt)); err == nil {
		t.Fatal("expected error for invalid profile data")
	}
}

func TestApplyProfileUpdated_DefaultLevel(t *testing.T) {
	ctx := context.Background()
	daggerheartStore := newProjectionDaggerheartStore()
	adapters := bridge.NewAdapterRegistry()
	_ = adapters.Register(daggerheartsys.NewAdapter(daggerheartStore))
	applier := Applier{Adapters: adapters}
	payload := map[string]any{
		"system_profile": map[string]any{
			"daggerheart": map[string]any{
				"hp_max":           float64(6),
				"stress_max":       float64(6),
				"evasion":          float64(10),
				"major_threshold":  float64(5),
				"severe_threshold": float64(10),
				"proficiency":      float64(0),
				"armor_score":      float64(0),
				"armor_max":        float64(0),
				"agility":          float64(1),
				"strength":         float64(0),
				"finesse":          float64(0),
				"instinct":         float64(0),
				"presence":         float64(0),
				"knowledge":        float64(0),
				"experiences":      []any{},
			},
		},
	}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{CampaignID: "camp-1", EntityID: "char-1", Type: testevent.TypeProfileUpdated, PayloadJSON: data}
	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
	profile, _ := daggerheartStore.GetDaggerheartCharacterProfile(ctx, "camp-1", "char-1")
	if profile.Level != 1 {
		t.Fatalf("Level = %d, want 1 (default)", profile.Level)
	}
}

func TestMarshalOptionalMap(t *testing.T) {
	// Empty map returns nil
	result, err := marshalOptionalMap(nil)
	if err != nil || result != nil {
		t.Fatalf("expected nil, got %v, %v", result, err)
	}

	result, err = marshalOptionalMap(map[string]any{})
	if err != nil || result != nil {
		t.Fatalf("expected nil for empty map, got %v, %v", result, err)
	}

	// Non-empty map returns JSON
	result, err = marshalOptionalMap(map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// fakeWatermarkStore records calls to SaveProjectionWatermark.
type fakeWatermarkStore struct {
	watermarks map[string]storage.ProjectionWatermark
}

func newFakeWatermarkStore() *fakeWatermarkStore {
	return &fakeWatermarkStore{watermarks: make(map[string]storage.ProjectionWatermark)}
}

func (s *fakeWatermarkStore) GetProjectionWatermark(_ context.Context, campaignID string) (storage.ProjectionWatermark, error) {
	wm, ok := s.watermarks[campaignID]
	if !ok {
		return storage.ProjectionWatermark{}, storage.ErrNotFound
	}
	return wm, nil
}

func (s *fakeWatermarkStore) SaveProjectionWatermark(_ context.Context, wm storage.ProjectionWatermark) error {
	s.watermarks[wm.CampaignID] = wm
	return nil
}

func (s *fakeWatermarkStore) ListProjectionWatermarks(_ context.Context) ([]storage.ProjectionWatermark, error) {
	var out []storage.ProjectionWatermark
	for _, wm := range s.watermarks {
		out = append(out, wm)
	}
	return out, nil
}

func TestApply_SavesWatermarkOnSuccess(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	watermarks := newFakeWatermarkStore()

	applier := Applier{
		Campaign:   campaignStore,
		Watermarks: watermarks,
	}

	payload := testevent.CampaignCreatedPayload{
		Name:       "Test",
		GameSystem: "DAGGERHEART",
		GmMode:     "HUMAN",
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	evt := testevent.Event{
		CampaignID:  "camp-1",
		EntityID:    "camp-1",
		Type:        testevent.TypeCampaignCreated,
		PayloadJSON: data,
		Timestamp:   stamp,
		Seq:         5,
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}

	wm, err := watermarks.GetProjectionWatermark(ctx, "camp-1")
	if err != nil {
		t.Fatalf("get watermark: %v", err)
	}
	if wm.AppliedSeq != 5 {
		t.Fatalf("applied_seq = %d, want 5", wm.AppliedSeq)
	}
}

func TestApply_SkipsWatermarkWhenNil(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()

	// No watermarks store configured — should not panic.
	applier := Applier{Campaign: campaignStore}

	payload := testevent.CampaignCreatedPayload{
		Name:       "Test",
		GameSystem: "DAGGERHEART",
		GmMode:     "HUMAN",
	}
	data, _ := json.Marshal(payload)
	stamp := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	evt := testevent.Event{
		CampaignID:  "camp-1",
		EntityID:    "camp-1",
		Type:        testevent.TypeCampaignCreated,
		PayloadJSON: data,
		Timestamp:   stamp,
		Seq:         5,
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}
}

func TestApply_SkipsWatermarkForZeroSeq(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	watermarks := newFakeWatermarkStore()

	applier := Applier{
		Campaign:   campaignStore,
		Watermarks: watermarks,
	}

	payload := testevent.CampaignCreatedPayload{
		Name:       "Test",
		GameSystem: "DAGGERHEART",
		GmMode:     "HUMAN",
	}
	data, _ := json.Marshal(payload)
	evt := testevent.Event{
		CampaignID:  "camp-1",
		EntityID:    "camp-1",
		Type:        testevent.TypeCampaignCreated,
		PayloadJSON: data,
		Timestamp:   time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC),
		Seq:         0, // no seq — should not save watermark
	}

	if err := applier.Apply(ctx, eventToEvent(evt)); err != nil {
		t.Fatalf("apply: %v", err)
	}

	_, err := watermarks.GetProjectionWatermark(ctx, "camp-1")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for zero-seq, got %v", err)
	}
}

func TestEnsureTimestamp_ZeroReturnsError(t *testing.T) {
	_, err := ensureTimestamp(time.Time{})
	if err == nil {
		t.Fatal("expected error for zero timestamp")
	}
	if !strings.Contains(err.Error(), "timestamp") {
		t.Fatalf("error should mention timestamp, got: %v", err)
	}
}

func TestEnsureTimestamp_NonZeroReturnsUTC(t *testing.T) {
	ts := time.Date(2025, 1, 1, 12, 0, 0, 0, time.FixedZone("EST", -5*60*60))
	got, err := ensureTimestamp(ts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Location() != time.UTC {
		t.Fatalf("expected UTC, got %v", got.Location())
	}
	if !got.Equal(ts) {
		t.Fatalf("time mismatch: got %v, want %v", got, ts)
	}
}
