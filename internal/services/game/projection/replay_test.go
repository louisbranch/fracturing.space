package projection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
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
	if storedCampaign.System != bridge.SystemIDDaggerheart {
		t.Fatalf("campaign system = %v, want %v", storedCampaign.System, bridge.SystemIDDaggerheart)
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
	daggerheartStore.states["camp-1:char-1"] = projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, Hope: 2, Stress: 1, Armor: 2}
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
	daggerheartStore.snapshots["camp-1"] = projectionstore.DaggerheartSnapshot{CampaignID: "camp-1", GMFear: 0}
	daggerheartStore.states["camp-1:char-1"] = projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, Hope: 1, Stress: 3, Armor: 0}
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
	if state.Hope != 1 || state.Stress != 3 {
		t.Fatalf("state = %+v, want hope=1 stress=3", state)
	}
}

func TestReplaySnapshot_AppliesDowntimeMove(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	daggerheartStore := newProjectionDaggerheartStore()
	applier := newProjectionApplier(campaignStore, daggerheartStore)
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{ID: "camp-1"}
	daggerheartStore.states["camp-1:char-1"] = projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, Hope: 1, Stress: 3, Armor: 0}
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
	daggerheartStore.states["camp-1:char-1"] = projectionstore.DaggerheartCharacterState{CampaignID: "camp-1", CharacterID: "char-1", Hp: 6, Hope: 1, Stress: 3, Armor: 0}
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

func (s *projectionEventStore) ListEvents(_ context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	results := make([]event.Event, 0, limit)
	for _, evt := range s.events {
		if string(evt.CampaignID) != campaignID {
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

func newCampaignCreatedEvent(campaignID string, seq uint64) event.Event {
	payload := campaign.CreatePayload{
		Name:       "Test Campaign",
		GameSystem: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:     "GM_MODE_HUMAN",
	}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:  ids.CampaignID(campaignID),
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
		ParticipantID:  ids.ParticipantID(participantID),
		Name:           "Player One",
		Role:           "PLAYER",
		Controller:     "CONTROLLER_HUMAN",
		CampaignAccess: "MEMBER",
	}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:  ids.CampaignID(campaignID),
		Seq:         seq,
		Timestamp:   time.Date(2025, 1, 10, 10, 1, 0, 0, time.UTC),
		Type:        event.Type("participant.joined"),
		EntityType:  "participant",
		EntityID:    participantID,
		PayloadJSON: data,
	}
}

func newGMFearChangedEvent(campaignID string, seq uint64, gmFear int) event.Event {
	payload := daggerheart.GMFearChangedPayload{Value: gmFear}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:    ids.CampaignID(campaignID),
		Seq:           seq,
		Timestamp:     time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC),
		Type:          event.Type("sys.daggerheart.gm_fear_changed"),
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		EntityType:    "campaign",
		EntityID:      campaignID,
		PayloadJSON:   data,
	}
}

func newCharacterStateChangedEvent(campaignID, characterID string, seq uint64, hp, hope, stress int) event.Event {
	payload := daggerheart.CharacterStatePatchedPayload{
		CharacterID: ids.CharacterID(characterID),
		HP:          &hp,
		Hope:        &hope,
		Stress:      &stress,
	}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:    ids.CampaignID(campaignID),
		Seq:           seq,
		Timestamp:     time.Date(2025, 1, 10, 12, 5, 0, 0, time.UTC),
		Type:          event.Type("sys.daggerheart.character_state_patched"),
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		EntityType:    "character",
		EntityID:      characterID,
		PayloadJSON:   data,
	}
}

func newDamageAppliedEvent(campaignID, characterID string, seq uint64, hp, armor int) event.Event {
	payload := daggerheart.DamageAppliedPayload{
		CharacterID: ids.CharacterID(characterID),
		Hp:          &hp,
		Armor:       &armor,
		Severity:    "major",
		Marks:       2,
	}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:    ids.CampaignID(campaignID),
		Seq:           seq,
		Timestamp:     time.Date(2025, 1, 10, 12, 6, 0, 0, time.UTC),
		Type:          event.Type("sys.daggerheart.damage_applied"),
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		EntityType:    "character",
		EntityID:      characterID,
		PayloadJSON:   data,
	}
}

func newRestTakenEvent(campaignID, characterID string, seq uint64) event.Event {
	payload := daggerheart.RestTakenPayload{
		RestType:     "short",
		Interrupted:  false,
		GMFear:       2,
		ShortRests:   1,
		RefreshRest:  true,
		Participants: []ids.CharacterID{ids.CharacterID(characterID)},
	}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:    ids.CampaignID(campaignID),
		Seq:           seq,
		Timestamp:     time.Date(2025, 1, 10, 12, 7, 0, 0, time.UTC),
		Type:          event.Type("sys.daggerheart.rest_taken"),
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		EntityType:    "session",
		EntityID:      campaignID,
		PayloadJSON:   data,
	}
}

func newDowntimeMoveAppliedEvent(campaignID, characterID string, seq uint64) event.Event {
	hope := 3
	payload := daggerheart.DowntimeMoveAppliedPayload{
		ActorCharacterID:  ids.CharacterID(characterID),
		TargetCharacterID: ids.CharacterID(characterID),
		Move:              "prepare",
		RestType:          "short",
		Hope:              &hope,
	}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:    ids.CampaignID(campaignID),
		Seq:           seq,
		Timestamp:     time.Date(2025, 1, 10, 12, 8, 0, 0, time.UTC),
		Type:          event.Type("sys.daggerheart.downtime_move_applied"),
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		EntityType:    "character",
		EntityID:      characterID,
		PayloadJSON:   data,
	}
}

func newLoadoutSwappedEvent(campaignID, characterID string, seq uint64) event.Event {
	stress := 2
	payload := daggerheart.LoadoutSwappedPayload{
		CharacterID: ids.CharacterID(characterID),
		CardID:      "card-1",
		From:        "vault",
		To:          "active",
		RecallCost:  1,
		Stress:      &stress,
	}
	data, _ := json.Marshal(payload)
	return event.Event{
		CampaignID:    ids.CampaignID(campaignID),
		Seq:           seq,
		Timestamp:     time.Date(2025, 1, 10, 12, 9, 0, 0, time.UTC),
		Type:          event.Type("sys.daggerheart.loadout_swapped"),
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

func (s *errorEventStore) ListEvents(context.Context, string, uint64, int) ([]event.Event, error) {
	return nil, s.err
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

func TestReplayCampaignWith_NilApplier(t *testing.T) {
	ctx := context.Background()
	store := &projectionEventStore{}
	_, err := ReplayCampaignWith(ctx, store, nil, "camp-1", ReplayOptions{})
	if err == nil {
		t.Fatal("expected error for nil applier")
	}
}

func TestReplayCampaignWith_DetectsSequenceGap(t *testing.T) {
	ctx := context.Background()
	campaignStore := newProjectionCampaignStore()
	participantStore := newProjectionParticipantStore()
	applier := Applier{Campaign: campaignStore, Participant: participantStore}
	eventStore := &projectionEventStore{
		events: []event.Event{
			newCampaignCreatedEvent("camp-1", 1),
			// Seq 2 is missing — jump straight to 3.
			newParticipantJoinedEvent("camp-1", "part-1", 3),
		},
	}

	_, err := ReplayCampaignWith(ctx, eventStore, applier, "camp-1", ReplayOptions{})
	if err == nil {
		t.Fatal("expected error for sequence gap")
	}
	if !strings.Contains(err.Error(), "sequence gap") {
		t.Fatalf("expected sequence gap error, got: %v", err)
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
