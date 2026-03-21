package projection

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
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

func (s *fakeSessionStore) CountSessions(context.Context, string) (int, error) {
	return 0, nil
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
		CampaignID:     ids.CampaignID(strings.TrimSpace(string(evt.CampaignID))),
		Seq:            evt.Seq,
		Hash:           evt.Hash,
		PrevHash:       evt.PrevHash,
		ChainHash:      evt.ChainHash,
		Signature:      evt.Signature,
		SignatureKeyID: evt.SignatureKeyID,
		Type:           event.Type(strings.TrimSpace(string(evt.Type))),
		Timestamp:      evt.Timestamp,
		SessionID:      ids.SessionID(evt.SessionID),
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

type fakeSessionInteractionStore struct {
	interactions map[string]storage.SessionInteraction
}

func newFakeSessionInteractionStore() *fakeSessionInteractionStore {
	return &fakeSessionInteractionStore{interactions: make(map[string]storage.SessionInteraction)}
}

func (s *fakeSessionInteractionStore) PutSessionInteraction(_ context.Context, interaction storage.SessionInteraction) error {
	s.interactions[interaction.CampaignID+":"+interaction.SessionID] = interaction
	return nil
}

func (s *fakeSessionInteractionStore) GetSessionInteraction(_ context.Context, campaignID, sessionID string) (storage.SessionInteraction, error) {
	interaction, ok := s.interactions[campaignID+":"+sessionID]
	if !ok {
		return storage.SessionInteraction{}, storage.ErrNotFound
	}
	return interaction, nil
}

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

func (s *fakeCharacterStore) ListCharactersByOwnerParticipant(context.Context, string, string) ([]storage.CharacterRecord, error) {
	return nil, nil
}

func (s *fakeCharacterStore) ListCharactersByControllerParticipant(context.Context, string, string) ([]storage.CharacterRecord, error) {
	return nil, nil
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

type fakeSceneStore struct {
	scenes map[string]storage.SceneRecord
}

func newFakeSceneStore() *fakeSceneStore {
	return &fakeSceneStore{scenes: make(map[string]storage.SceneRecord)}
}

func (s *fakeSceneStore) PutScene(_ context.Context, rec storage.SceneRecord) error {
	s.scenes[rec.CampaignID+":"+rec.SceneID] = rec
	return nil
}

func (s *fakeSceneStore) EndScene(_ context.Context, campaignID, sceneID string, endedAt time.Time) error {
	key := campaignID + ":" + sceneID
	rec, ok := s.scenes[key]
	if !ok {
		return storage.ErrNotFound
	}
	rec.Active = false
	rec.EndedAt = &endedAt
	rec.UpdatedAt = endedAt
	s.scenes[key] = rec
	return nil
}

func (s *fakeSceneStore) GetScene(_ context.Context, campaignID, sceneID string) (storage.SceneRecord, error) {
	rec, ok := s.scenes[campaignID+":"+sceneID]
	if !ok {
		return storage.SceneRecord{}, storage.ErrNotFound
	}
	return rec, nil
}

func (s *fakeSceneStore) ListScenes(_ context.Context, _, _ string, _ int, _ string) (storage.ScenePage, error) {
	return storage.ScenePage{}, nil
}

func (s *fakeSceneStore) ListActiveScenes(_ context.Context, _ string) ([]storage.SceneRecord, error) {
	return nil, nil
}

func (s *fakeSceneStore) ListVisibleActiveScenesForCharacters(context.Context, string, string, []string) ([]storage.SceneRecord, error) {
	return nil, nil
}

type fakeSceneCharacterStore struct {
	characters map[string][]storage.SceneCharacterRecord
}

func newFakeSceneCharacterStore() *fakeSceneCharacterStore {
	return &fakeSceneCharacterStore{characters: make(map[string][]storage.SceneCharacterRecord)}
}

func (s *fakeSceneCharacterStore) PutSceneCharacter(_ context.Context, rec storage.SceneCharacterRecord) error {
	key := rec.CampaignID + ":" + rec.SceneID
	s.characters[key] = append(s.characters[key], rec)
	return nil
}

func (s *fakeSceneCharacterStore) DeleteSceneCharacter(_ context.Context, campaignID, sceneID, characterID string) error {
	key := campaignID + ":" + sceneID
	chars := s.characters[key]
	for i, c := range chars {
		if c.CharacterID == characterID {
			s.characters[key] = append(chars[:i], chars[i+1:]...)
			return nil
		}
	}
	return nil
}

func (s *fakeSceneCharacterStore) ListSceneCharacters(_ context.Context, campaignID, sceneID string) ([]storage.SceneCharacterRecord, error) {
	return s.characters[campaignID+":"+sceneID], nil
}

type fakeSceneGateStore struct {
	gates map[string]storage.SceneGate
}

func newFakeSceneGateStore() *fakeSceneGateStore {
	return &fakeSceneGateStore{gates: make(map[string]storage.SceneGate)}
}

func (s *fakeSceneGateStore) PutSceneGate(_ context.Context, gate storage.SceneGate) error {
	s.gates[gate.CampaignID+":"+gate.SceneID+":"+gate.GateID] = gate
	return nil
}

func (s *fakeSceneGateStore) GetSceneGate(_ context.Context, campaignID, sceneID, gateID string) (storage.SceneGate, error) {
	gate, ok := s.gates[campaignID+":"+sceneID+":"+gateID]
	if !ok {
		return storage.SceneGate{}, storage.ErrNotFound
	}
	return gate, nil
}

func (s *fakeSceneGateStore) GetOpenSceneGate(_ context.Context, campaignID, sceneID string) (storage.SceneGate, error) {
	for _, gate := range s.gates {
		if gate.CampaignID == campaignID && gate.SceneID == sceneID && gate.Status == "open" {
			return gate, nil
		}
	}
	return storage.SceneGate{}, storage.ErrNotFound
}

type fakeSceneSpotlightStore struct {
	spotlights map[string]storage.SceneSpotlight
	cleared    []string
}

func newFakeSceneSpotlightStore() *fakeSceneSpotlightStore {
	return &fakeSceneSpotlightStore{spotlights: make(map[string]storage.SceneSpotlight)}
}

func (s *fakeSceneSpotlightStore) PutSceneSpotlight(_ context.Context, spotlight storage.SceneSpotlight) error {
	s.spotlights[spotlight.CampaignID+":"+spotlight.SceneID] = spotlight
	return nil
}

func (s *fakeSceneSpotlightStore) GetSceneSpotlight(_ context.Context, campaignID, sceneID string) (storage.SceneSpotlight, error) {
	spotlight, ok := s.spotlights[campaignID+":"+sceneID]
	if !ok {
		return storage.SceneSpotlight{}, storage.ErrNotFound
	}
	return spotlight, nil
}

func (s *fakeSceneSpotlightStore) ClearSceneSpotlight(_ context.Context, campaignID, sceneID string) error {
	s.cleared = append(s.cleared, campaignID+":"+sceneID)
	return nil
}

type fakeSceneInteractionStore struct {
	interactions map[string]storage.SceneInteraction
}

func newFakeSceneInteractionStore() *fakeSceneInteractionStore {
	return &fakeSceneInteractionStore{interactions: make(map[string]storage.SceneInteraction)}
}

func (s *fakeSceneInteractionStore) PutSceneInteraction(_ context.Context, interaction storage.SceneInteraction) error {
	s.interactions[interaction.CampaignID+":"+interaction.SceneID] = interaction
	return nil
}

func (s *fakeSceneInteractionStore) GetSceneInteraction(_ context.Context, campaignID, sceneID string) (storage.SceneInteraction, error) {
	interaction, ok := s.interactions[campaignID+":"+sceneID]
	if !ok {
		return storage.SceneInteraction{}, storage.ErrNotFound
	}
	return interaction, nil
}

type fakeSceneGMInteractionStore struct {
	interactions map[string][]storage.SceneGMInteraction
}

func newFakeSceneGMInteractionStore() *fakeSceneGMInteractionStore {
	return &fakeSceneGMInteractionStore{interactions: make(map[string][]storage.SceneGMInteraction)}
}

func (s *fakeSceneGMInteractionStore) PutSceneGMInteraction(_ context.Context, interaction storage.SceneGMInteraction) error {
	key := interaction.CampaignID + ":" + interaction.SceneID
	s.interactions[key] = append(s.interactions[key], interaction)
	return nil
}

func (s *fakeSceneGMInteractionStore) ListSceneGMInteractions(_ context.Context, campaignID, sceneID string) ([]storage.SceneGMInteraction, error) {
	return append([]storage.SceneGMInteraction(nil), s.interactions[campaignID+":"+sceneID]...), nil
}
