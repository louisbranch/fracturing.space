package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	systems "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestReplayCampaign_AppliesEvents(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	participantStore := newProjectionParticipantStore()
	applier := Applier{Campaign: campaignStore, Participant: participantStore}
	eventStore := &projectionEventStore{
		events: []event.Event{
			newCampaignCreatedEvent("camp-1", 1),
			newParticipantJoinedEvent("camp-1", "part-1", 2),
		},
	}

	lastSeq, err := ReplayCampaign(ctx, eventStore, applier, "camp-1")
	if err != nil {
		t.Fatalf("ReplayCampaign returned error: %v", err)
	}
	if lastSeq != 2 {
		t.Fatalf("lastSeq = %d, want 2", lastSeq)
	}

	storedCampaign, err := campaignStore.Get(ctx, "camp-1")
	if err != nil {
		t.Fatalf("campaign not stored: %v", err)
	}
	if storedCampaign.System != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("campaign system = %v, want %v", storedCampaign.System, commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
	}
	if storedCampaign.ParticipantCount != 1 {
		t.Fatalf("campaign participant count = %d, want 1", storedCampaign.ParticipantCount)
	}

	storedParticipant, err := participantStore.GetParticipant(ctx, "camp-1", "part-1")
	if err != nil {
		t.Fatalf("participant not stored: %v", err)
	}
	if storedParticipant.Name != "Player One" {
		t.Fatalf("participant display name = %q, want %q", storedParticipant.Name, "Player One")
	}
}

func TestReplayCampaign_RequiresCampaignID(t *testing.T) {
	ctx := context.Background()
	_, err := ReplayCampaign(ctx, &projectionEventStore{}, Applier{}, "")
	if err == nil {
		t.Fatal("expected error for missing campaign id")
	}
}

func TestReplayCampaignWith_FilterSkipsEvents(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	applier := Applier{Campaign: campaignStore}
	eventStore := &projectionEventStore{
		events: []event.Event{
			newCampaignCreatedEvent("camp-1", 1),
		},
	}

	_, err := ReplayCampaignWith(ctx, eventStore, applier, "camp-1", ReplayOptions{
		Filter: func(event.Event) bool { return false },
	})
	if err != nil {
		t.Fatalf("ReplayCampaignWith returned error: %v", err)
	}
	if _, err := campaignStore.Get(ctx, "camp-1"); err == nil {
		t.Fatal("expected campaign to be skipped by filter")
	}
}

func TestReplaySnapshot_AppliesSnapshotEvents(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	daggerheartStore := newProjectionDaggerheartStore()
	applier := newProjectionApplier(campaignStore, daggerheartStore)
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	eventStore := &projectionEventStore{
		events: []event.Event{
			newCampaignCreatedEvent("camp-1", 1),
			newGMFearChangedEvent("camp-1", 2, 5),
			newCharacterStateChangedEvent("camp-1", "char-1", 3, 6, 2, 1),
		},
	}

	lastSeq, err := ReplaySnapshot(ctx, eventStore, applier, "camp-1", 0)
	if err != nil {
		t.Fatalf("ReplaySnapshot returned error: %v", err)
	}
	if lastSeq != 3 {
		t.Fatalf("lastSeq = %d, want 3", lastSeq)
	}

	snapshot, err := daggerheartStore.GetDaggerheartSnapshot(ctx, "camp-1")
	if err != nil {
		t.Fatalf("snapshot not stored: %v", err)
	}
	if snapshot.GMFear != 5 {
		t.Fatalf("snapshot GMFear = %d, want 5", snapshot.GMFear)
	}

	state, err := daggerheartStore.GetDaggerheartCharacterState(ctx, "camp-1", "char-1")
	if err != nil {
		t.Fatalf("state not stored: %v", err)
	}
	if state.Hp != 6 || state.Hope != 2 || state.Stress != 1 {
		t.Fatalf("state = %+v, want hp=6 hope=2 stress=1", state)
	}
}

func TestReplaySnapshot_RejectsInvalidState(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	daggerheartStore := newProjectionDaggerheartStore()
	applier := newProjectionApplier(campaignStore, daggerheartStore)
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	eventStore := &projectionEventStore{
		events: []event.Event{
			newCampaignCreatedEvent("camp-1", 1),
			newCharacterStateChangedEvent("camp-1", "char-1", 2, 6, 7, 1),
		},
	}

	_, err := ReplaySnapshot(ctx, eventStore, applier, "camp-1", 0)
	if err == nil {
		t.Fatal("expected error for invalid hope value")
	}
}

func TestReplaySnapshot_RejectsInvalidGMFear(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	daggerheartStore := newProjectionDaggerheartStore()
	applier := newProjectionApplier(campaignStore, daggerheartStore)
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	eventStore := &projectionEventStore{
		events: []event.Event{
			newCampaignCreatedEvent("camp-1", 1),
			newGMFearChangedEvent("camp-1", 2, 13),
		},
	}

	_, err := ReplaySnapshot(ctx, eventStore, applier, "camp-1", 0)
	if err == nil {
		t.Fatal("expected error for invalid gm fear value")
	}
}

func TestReplaySnapshot_AppliesDamageApplied(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	daggerheartStore := newProjectionDaggerheartStore()
	applier := newProjectionApplier(campaignStore, daggerheartStore)
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	daggerheartStore.states["camp-1:char-1"] = storage.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, Hope: 2, Stress: 1, Armor: 2}
	eventStore := &projectionEventStore{
		events: []event.Event{
			newCampaignCreatedEvent("camp-1", 1),
			newDamageAppliedEvent("camp-1", "char-1", 2, 4, 1),
		},
	}

	_, err := ReplaySnapshot(ctx, eventStore, applier, "camp-1", 0)
	if err != nil {
		t.Fatalf("ReplaySnapshot returned error: %v", err)
	}
	state := daggerheartStore.states["camp-1:char-1"]
	if state.Hp != 4 || state.Armor != 1 {
		t.Fatalf("state = %+v, want hp=4 armor=1", state)
	}
}

func TestReplaySnapshot_AppliesRestTaken(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	daggerheartStore := newProjectionDaggerheartStore()
	applier := newProjectionApplier(campaignStore, daggerheartStore)
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	daggerheartStore.snapshots["camp-1"] = storage.DaggerheartSnapshot{CampaignID: "camp-1", GMFear: 0}
	daggerheartStore.states["camp-1:char-1"] = storage.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, Hope: 1, Stress: 3, Armor: 0}
	eventStore := &projectionEventStore{
		events: []event.Event{
			newCampaignCreatedEvent("camp-1", 1),
			newRestTakenEvent("camp-1", "char-1", 2),
		},
	}

	_, err := ReplaySnapshot(ctx, eventStore, applier, "camp-1", 0)
	if err != nil {
		t.Fatalf("ReplaySnapshot returned error: %v", err)
	}
	if snap := daggerheartStore.snapshots["camp-1"]; snap.GMFear != 2 {
		t.Fatalf("gm_fear = %d, want 2", snap.GMFear)
	}
	state := daggerheartStore.states["camp-1:char-1"]
	if state.Hope != 2 || state.Stress != 0 {
		t.Fatalf("state = %+v, want hope=2 stress=0", state)
	}
}

func TestReplaySnapshot_AppliesDowntimeMove(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	daggerheartStore := newProjectionDaggerheartStore()
	applier := newProjectionApplier(campaignStore, daggerheartStore)
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	daggerheartStore.states["camp-1:char-1"] = storage.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, Hope: 1, Stress: 3, Armor: 0}
	eventStore := &projectionEventStore{
		events: []event.Event{
			newCampaignCreatedEvent("camp-1", 1),
			newDowntimeMoveAppliedEvent("camp-1", "char-1", 2),
		},
	}

	_, err := ReplaySnapshot(ctx, eventStore, applier, "camp-1", 0)
	if err != nil {
		t.Fatalf("ReplaySnapshot returned error: %v", err)
	}
	state := daggerheartStore.states["camp-1:char-1"]
	if state.Hope != 3 {
		t.Fatalf("hope = %d, want 3", state.Hope)
	}
}

func TestReplaySnapshot_AppliesLoadoutSwap(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	daggerheartStore := newProjectionDaggerheartStore()
	applier := newProjectionApplier(campaignStore, daggerheartStore)
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	daggerheartStore.states["camp-1:char-1"] = storage.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, Hope: 1, Stress: 3, Armor: 0}
	eventStore := &projectionEventStore{
		events: []event.Event{
			newCampaignCreatedEvent("camp-1", 1),
			newLoadoutSwappedEvent("camp-1", "char-1", 2),
		},
	}

	_, err := ReplaySnapshot(ctx, eventStore, applier, "camp-1", 0)
	if err != nil {
		t.Fatalf("ReplaySnapshot returned error: %v", err)
	}
	state := daggerheartStore.states["camp-1:char-1"]
	if state.Stress != 2 {
		t.Fatalf("stress = %d, want 2", state.Stress)
	}
}

type projectionEventStore struct {
	events []event.Event
}

func (s *projectionEventStore) AppendEvent(context.Context, event.Event) (event.Event, error) {
	return event.Event{}, nil
}

func (s *projectionEventStore) GetEventByHash(context.Context, string) (event.Event, error) {
	return event.Event{}, nil
}

func (s *projectionEventStore) GetEventBySeq(context.Context, string, uint64) (event.Event, error) {
	return event.Event{}, nil
}

func (s *projectionEventStore) ListEvents(_ context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	results := make([]event.Event, 0, limit)
	for _, evt := range s.events {
		if evt.CampaignID != campaignID {
			continue
		}
		if evt.Seq <= afterSeq {
			continue
		}
		results = append(results, evt)
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

func (s *projectionEventStore) ListEventsBySession(context.Context, string, string, uint64, int) ([]event.Event, error) {
	return nil, nil
}

func (s *projectionEventStore) GetLatestEventSeq(context.Context, string) (uint64, error) {
	return 0, nil
}

func (s *projectionEventStore) ListEventsPage(context.Context, storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	return storage.ListEventsPageResult{}, nil
}

type projectionCampaignStore struct {
	campaigns map[string]storage.CampaignRecord
}

func newProjectionCampaignStore() *projectionCampaignStore {
	return &projectionCampaignStore{campaigns: make(map[string]storage.CampaignRecord)}
}

func (s *projectionCampaignStore) Put(_ context.Context, c storage.CampaignRecord) error {
	s.campaigns[c.ID] = c
	return nil
}

func (s *projectionCampaignStore) Get(_ context.Context, id string) (storage.CampaignRecord, error) {
	c, ok := s.campaigns[id]
	if !ok {
		return storage.CampaignRecord{}, storage.ErrNotFound
	}
	return c, nil
}

func (s *projectionCampaignStore) List(context.Context, int, string) (storage.CampaignPage, error) {
	return storage.CampaignPage{}, nil
}

type projectionParticipantStore struct {
	participants map[string]storage.ParticipantRecord
}

func newProjectionParticipantStore() *projectionParticipantStore {
	return &projectionParticipantStore{participants: make(map[string]storage.ParticipantRecord)}
}

func (s *projectionParticipantStore) PutParticipant(_ context.Context, p storage.ParticipantRecord) error {
	s.participants[p.CampaignID+":"+p.ID] = p
	return nil
}

func (s *projectionParticipantStore) GetParticipant(_ context.Context, campaignID, participantID string) (storage.ParticipantRecord, error) {
	key := campaignID + ":" + participantID
	p, ok := s.participants[key]
	if !ok {
		return storage.ParticipantRecord{}, storage.ErrNotFound
	}
	return p, nil
}

func (s *projectionParticipantStore) DeleteParticipant(_ context.Context, campaignID, participantID string) error {
	key := campaignID + ":" + participantID
	if _, ok := s.participants[key]; !ok {
		return fmt.Errorf("not found")
	}
	delete(s.participants, key)
	return nil
}

func (s *projectionParticipantStore) ListParticipantsByCampaign(context.Context, string) ([]storage.ParticipantRecord, error) {
	return nil, nil
}

func (s *projectionParticipantStore) ListParticipants(context.Context, string, int, string) (storage.ParticipantPage, error) {
	return storage.ParticipantPage{}, nil
}

type projectionDaggerheartStore struct {
	profiles    map[string]storage.DaggerheartCharacterProfile
	states      map[string]storage.DaggerheartCharacterState
	snapshots   map[string]storage.DaggerheartSnapshot
	countdowns  map[string]storage.DaggerheartCountdown
	adversaries map[string]storage.DaggerheartAdversary
}

func newProjectionDaggerheartStore() *projectionDaggerheartStore {
	return &projectionDaggerheartStore{
		profiles:    make(map[string]storage.DaggerheartCharacterProfile),
		states:      make(map[string]storage.DaggerheartCharacterState),
		snapshots:   make(map[string]storage.DaggerheartSnapshot),
		countdowns:  make(map[string]storage.DaggerheartCountdown),
		adversaries: make(map[string]storage.DaggerheartAdversary),
	}
}

func newProjectionApplier(campaignStore *projectionCampaignStore, daggerheartStore *projectionDaggerheartStore) Applier {
	registry := systems.NewAdapterRegistry()
	if err := registry.Register(daggerheart.NewAdapter(daggerheartStore)); err != nil {
		panic(err)
	}
	return Applier{Campaign: campaignStore, Daggerheart: daggerheartStore, Adapters: registry}
}

func (s *projectionDaggerheartStore) PutDaggerheartCharacterProfile(_ context.Context, profile storage.DaggerheartCharacterProfile) error {
	key := profile.CampaignID + ":" + profile.CharacterID
	s.profiles[key] = profile
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) (storage.DaggerheartCharacterProfile, error) {
	key := campaignID + ":" + characterID
	profile, ok := s.profiles[key]
	if !ok {
		return storage.DaggerheartCharacterProfile{}, fmt.Errorf("not found")
	}
	return profile, nil
}

func (s *projectionDaggerheartStore) PutDaggerheartCharacterState(_ context.Context, state storage.DaggerheartCharacterState) error {
	key := state.CampaignID + ":" + state.CharacterID
	s.states[key] = state
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartCharacterState(_ context.Context, campaignID, characterID string) (storage.DaggerheartCharacterState, error) {
	key := campaignID + ":" + characterID
	state, ok := s.states[key]
	if !ok {
		return storage.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return state, nil
}

func (s *projectionDaggerheartStore) PutDaggerheartSnapshot(_ context.Context, snap storage.DaggerheartSnapshot) error {
	s.snapshots[snap.CampaignID] = snap
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartSnapshot(_ context.Context, campaignID string) (storage.DaggerheartSnapshot, error) {
	snap, ok := s.snapshots[campaignID]
	if !ok {
		return storage.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return snap, nil
}

func (s *projectionDaggerheartStore) PutDaggerheartCountdown(_ context.Context, countdown storage.DaggerheartCountdown) error {
	key := countdown.CampaignID + ":" + countdown.CountdownID
	s.countdowns[key] = countdown
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (storage.DaggerheartCountdown, error) {
	key := campaignID + ":" + countdownID
	countdown, ok := s.countdowns[key]
	if !ok {
		return storage.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return countdown, nil
}

func (s *projectionDaggerheartStore) ListDaggerheartCountdowns(_ context.Context, campaignID string) ([]storage.DaggerheartCountdown, error) {
	results := make([]storage.DaggerheartCountdown, 0)
	for _, countdown := range s.countdowns {
		if countdown.CampaignID == campaignID {
			results = append(results, countdown)
		}
	}
	return results, nil
}

func (s *projectionDaggerheartStore) DeleteDaggerheartCountdown(_ context.Context, campaignID, countdownID string) error {
	key := campaignID + ":" + countdownID
	if _, ok := s.countdowns[key]; !ok {
		return storage.ErrNotFound
	}
	delete(s.countdowns, key)
	return nil
}

func (s *projectionDaggerheartStore) PutDaggerheartAdversary(_ context.Context, adversary storage.DaggerheartAdversary) error {
	key := adversary.CampaignID + ":" + adversary.AdversaryID
	s.adversaries[key] = adversary
	return nil
}

func (s *projectionDaggerheartStore) GetDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) (storage.DaggerheartAdversary, error) {
	key := campaignID + ":" + adversaryID
	adversary, ok := s.adversaries[key]
	if !ok {
		return storage.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return adversary, nil
}

func (s *projectionDaggerheartStore) ListDaggerheartAdversaries(_ context.Context, campaignID, sessionID string) ([]storage.DaggerheartAdversary, error) {
	results := make([]storage.DaggerheartAdversary, 0)
	for _, adversary := range s.adversaries {
		if adversary.CampaignID != campaignID {
			continue
		}
		if strings.TrimSpace(sessionID) != "" && adversary.SessionID != sessionID {
			continue
		}
		results = append(results, adversary)
	}
	return results, nil
}

func (s *projectionDaggerheartStore) DeleteDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) error {
	key := campaignID + ":" + adversaryID
	if _, ok := s.adversaries[key]; !ok {
		return storage.ErrNotFound
	}
	delete(s.adversaries, key)
	return nil
}

func newCampaignCreatedEvent(campaignID string, seq uint64) event.Event {
	payload := campaign.CreatePayload{
		Name:       "Test Campaign",
		GameSystem: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:     "GM_MODE_HUMAN",
	}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:  campaignID,
		Seq:         seq,
		Timestamp:   time.Date(2025, 1, 10, 10, 0, 0, 0, time.UTC),
		Type:        event.Type("campaign.created"),
		EntityType:  "campaign",
		EntityID:    campaignID,
		PayloadJSON: data,
	}
}

func newParticipantJoinedEvent(campaignID, participantID string, seq uint64) event.Event {
	payload := participant.JoinPayload{
		ParticipantID:  participantID,
		Name:           "Player One",
		Role:           "PLAYER",
		Controller:     "CONTROLLER_HUMAN",
		CampaignAccess: "MEMBER",
	}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:  campaignID,
		Seq:         seq,
		Timestamp:   time.Date(2025, 1, 10, 10, 1, 0, 0, time.UTC),
		Type:        event.Type("participant.joined"),
		EntityType:  "participant",
		EntityID:    participantID,
		PayloadJSON: data,
	}
}

func newGMFearChangedEvent(campaignID string, seq uint64, gmFear int) event.Event {
	payload := daggerheart.GMFearChangedPayload{Before: 0, After: gmFear}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:    campaignID,
		Seq:           seq,
		Timestamp:     time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC),
		Type:          event.Type("sys.daggerheart.action.gm_fear_changed"),
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		EntityType:    "campaign",
		EntityID:      campaignID,
		PayloadJSON:   data,
	}
}

func newCharacterStateChangedEvent(campaignID, characterID string, seq uint64, hp, hope, stress int) event.Event {
	hpAfter := hp
	hopeAfter := hope
	stressAfter := stress
	payload := daggerheart.CharacterStatePatchPayload{
		CharacterID: characterID,
		HPAfter:     &hpAfter,
		HopeAfter:   &hopeAfter,
		StressAfter: &stressAfter,
	}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:    campaignID,
		Seq:           seq,
		Timestamp:     time.Date(2025, 1, 10, 12, 5, 0, 0, time.UTC),
		Type:          event.Type("sys.daggerheart.action.character_state_patched"),
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		EntityType:    "character",
		EntityID:      characterID,
		PayloadJSON:   data,
	}
}

func newDamageAppliedEvent(campaignID, characterID string, seq uint64, hpAfter, armorAfter int) event.Event {
	payload := daggerheart.DamageAppliedPayload{
		CharacterID: characterID,
		HpAfter:     &hpAfter,
		ArmorAfter:  &armorAfter,
		Severity:    "major",
		Marks:       2,
	}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:    campaignID,
		Seq:           seq,
		Timestamp:     time.Date(2025, 1, 10, 12, 6, 0, 0, time.UTC),
		Type:          event.Type("sys.daggerheart.action.damage_applied"),
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		EntityType:    "character",
		EntityID:      characterID,
		PayloadJSON:   data,
	}
}

func newRestTakenEvent(campaignID, characterID string, seq uint64) event.Event {
	hopeAfter := 2
	stressAfter := 0
	payload := daggerheart.RestTakePayload{
		RestType:         "short",
		Interrupted:      false,
		GMFearBefore:     0,
		GMFearAfter:      2,
		ShortRestsBefore: 0,
		ShortRestsAfter:  1,
		RefreshRest:      true,
		CharacterStates: []daggerheart.RestCharacterStatePatch{{
			CharacterID: characterID,
			HopeAfter:   &hopeAfter,
			StressAfter: &stressAfter,
		}},
	}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:    campaignID,
		Seq:           seq,
		Timestamp:     time.Date(2025, 1, 10, 12, 7, 0, 0, time.UTC),
		Type:          event.Type("sys.daggerheart.action.rest_taken"),
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		EntityType:    "session",
		EntityID:      campaignID,
		PayloadJSON:   data,
	}
}

func newDowntimeMoveAppliedEvent(campaignID, characterID string, seq uint64) event.Event {
	hopeAfter := 3
	payload := daggerheart.DowntimeMoveAppliedPayload{
		CharacterID: characterID,
		Move:        "prepare",
		HopeAfter:   &hopeAfter,
	}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:    campaignID,
		Seq:           seq,
		Timestamp:     time.Date(2025, 1, 10, 12, 8, 0, 0, time.UTC),
		Type:          event.Type("sys.daggerheart.action.downtime_move_applied"),
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		EntityType:    "character",
		EntityID:      characterID,
		PayloadJSON:   data,
	}
}

func newLoadoutSwappedEvent(campaignID, characterID string, seq uint64) event.Event {
	stressAfter := 2
	payload := daggerheart.LoadoutSwappedPayload{
		CharacterID: characterID,
		CardID:      "card-1",
		From:        "vault",
		To:          "active",
		RecallCost:  1,
		StressAfter: &stressAfter,
	}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:    campaignID,
		Seq:           seq,
		Timestamp:     time.Date(2025, 1, 10, 12, 9, 0, 0, time.UTC),
		Type:          event.Type("sys.daggerheart.action.loadout_swapped"),
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		EntityType:    "character",
		EntityID:      characterID,
		PayloadJSON:   data,
	}
}

type errorEventStore struct {
	err error
}

func (s *errorEventStore) AppendEvent(context.Context, event.Event) (event.Event, error) {
	return event.Event{}, s.err
}

func (s *errorEventStore) GetEventByHash(context.Context, string) (event.Event, error) {
	return event.Event{}, s.err
}

func (s *errorEventStore) GetEventBySeq(context.Context, string, uint64) (event.Event, error) {
	return event.Event{}, s.err
}

func (s *errorEventStore) ListEvents(context.Context, string, uint64, int) ([]event.Event, error) {
	return nil, s.err
}

func (s *errorEventStore) ListEventsBySession(context.Context, string, string, uint64, int) ([]event.Event, error) {
	return nil, s.err
}

func (s *errorEventStore) GetLatestEventSeq(context.Context, string) (uint64, error) {
	return 0, s.err
}

func (s *errorEventStore) ListEventsPage(context.Context, storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	return storage.ListEventsPageResult{}, s.err
}

func TestReplayCampaignWith_ListEventsError(t *testing.T) {
	ctx := context.Background()
	store := &errorEventStore{err: fmt.Errorf("list failed")}
	applier := Applier{Campaign: newProjectionCampaignStore()}
	_, err := ReplayCampaignWith(ctx, store, applier, "camp-1", ReplayOptions{})
	if err == nil {
		t.Fatal("expected error from ListEvents")
	}
}

func TestReplayCampaignWith_UntilSeq(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	participantStore := newProjectionParticipantStore()
	applier := Applier{Campaign: campaignStore, Participant: participantStore}
	eventStore := &projectionEventStore{
		events: []event.Event{
			newCampaignCreatedEvent("camp-1", 1),
			newParticipantJoinedEvent("camp-1", "part-1", 2),
			newParticipantJoinedEvent("camp-1", "part-2", 3),
		},
	}

	lastSeq, err := ReplayCampaignWith(ctx, eventStore, applier, "camp-1", ReplayOptions{UntilSeq: 2})
	if err != nil {
		t.Fatalf("ReplayCampaignWith: %v", err)
	}
	if lastSeq != 2 {
		t.Fatalf("lastSeq = %d, want 2", lastSeq)
	}
	c, _ := campaignStore.Get(ctx, "camp-1")
	if c.ParticipantCount != 1 {
		t.Fatalf("ParticipantCount = %d, want 1 (only first participant)", c.ParticipantCount)
	}
}

func TestReplayCampaignWith_NilEventStore(t *testing.T) {
	ctx := context.Background()
	_, err := ReplayCampaignWith(ctx, nil, Applier{}, "camp-1", ReplayOptions{})
	if err == nil {
		t.Fatal("expected error for nil event store")
	}
}

func TestReplayCampaignWith_EmptyCampaignID(t *testing.T) {
	ctx := context.Background()
	store := &projectionEventStore{}
	_, err := ReplayCampaignWith(ctx, store, Applier{}, "  ", ReplayOptions{})
	if err == nil {
		t.Fatal("expected error for empty campaign id")
	}
}

func TestReplayCampaignWith_ApplyError(t *testing.T) {
	ctx := context.Background()
	applier := Applier{}
	eventStore := &projectionEventStore{
		events: []event.Event{
			newCampaignCreatedEvent("camp-1", 1),
		},
	}
	_, err := ReplayCampaignWith(ctx, eventStore, applier, "camp-1", ReplayOptions{})
	if err == nil {
		t.Fatal("expected error from applier.Apply")
	}
}
