package daggerheart

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/internal/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestAdapterAndFolder_EventVectorParity(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)
	folder := NewFolder()

	vectors := daggerheartEventVectorsForParity()
	if got, want := len(vectors), len(daggerheartEventDefinitions); got != want {
		t.Fatalf("event vectors = %d, definitions = %d", got, want)
	}
	for _, def := range daggerheartEventDefinitions {
		if _, ok := vectors[def.Type]; !ok {
			t.Fatalf("missing parity vector for %s", def.Type)
		}
	}

	sequence := []event.Type{
		EventTypeGMFearChanged,
		EventTypeCharacterStatePatched,
		EventTypeConditionChanged,
		EventTypeLoadoutSwapped,
		EventTypeCharacterTemporaryArmorApplied,
		EventTypeRestTaken,
		EventTypeDamageApplied,
		EventTypeDowntimeMoveApplied,
		EventTypeCountdownCreated,
		EventTypeCountdownUpdated,
		EventTypeCountdownDeleted,
		EventTypeAdversaryCreated,
		EventTypeAdversaryConditionChanged,
		EventTypeAdversaryDamageApplied,
		EventTypeAdversaryUpdated,
		EventTypeAdversaryDeleted,
		EventTypeLevelUpApplied,
		EventTypeGoldUpdated,
		EventTypeDomainCardAcquired,
		EventTypeEquipmentSwapped,
		EventTypeConsumableUsed,
		EventTypeConsumableAcquired,
	}
	if got, want := len(sequence), len(daggerheartEventDefinitions); got != want {
		t.Fatalf("event sequence = %d, definitions = %d", got, want)
	}
	seen := make(map[event.Type]struct{}, len(sequence))
	for _, typ := range sequence {
		if _, dup := seen[typ]; dup {
			t.Fatalf("duplicate sequence entry for %s", typ)
		}
		seen[typ] = struct{}{}
	}
	for _, def := range daggerheartEventDefinitions {
		if _, ok := seen[def.Type]; !ok {
			t.Fatalf("event definition %s not covered by sequence", def.Type)
		}
	}

	ctx := context.Background()
	campaignID := "camp-1"
	base := time.Date(2026, 2, 28, 10, 0, 0, 0, time.UTC)
	var folded any
	for i, typ := range sequence {
		payloadJSON, err := json.Marshal(vectors[typ])
		if err != nil {
			t.Fatalf("marshal payload for %s: %v", typ, err)
		}
		evt := event.Event{
			CampaignID:    campaignID,
			Seq:           uint64(i + 1),
			Type:          typ,
			Timestamp:     base.Add(time.Duration(i) * time.Minute),
			ActorType:     event.ActorTypeSystem,
			ActorID:       "system-1",
			EntityType:    "campaign",
			EntityID:      campaignID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			PayloadJSON:   payloadJSON,
		}
		if err := adapter.Apply(ctx, evt); err != nil {
			t.Fatalf("adapter apply %s: %v", typ, err)
		}
		folded, err = folder.Fold(folded, evt)
		if err != nil {
			t.Fatalf("folder fold %s: %v", typ, err)
		}

		folderState := assertTestSnapshotState(t, folded)
		adapterState := store.snapshotState(campaignID)
		folderState = canonicalizeSnapshotForParity(folderState)
		adapterState = canonicalizeSnapshotForParity(adapterState)
		if !reflect.DeepEqual(folderState, adapterState) {
			t.Fatalf("state mismatch after %s\nfolder=%#v\nadapter=%#v", typ, folderState, adapterState)
		}
	}
}

func daggerheartEventVectorsForParity() map[event.Type]any {
	lifeStateAlive := LifeStateAlive
	return map[event.Type]any{
		EventTypeGMFearChanged: GMFearChangedPayload{
			Before: 0,
			After:  2,
		},
		EventTypeCharacterStatePatched: CharacterStatePatchedPayload{
			CharacterID:    "char-1",
			HPAfter:        intPtr(6),
			HopeAfter:      intPtr(2),
			HopeMaxAfter:   intPtr(6),
			StressAfter:    intPtr(1),
			ArmorAfter:     intPtr(0),
			LifeStateAfter: &lifeStateAlive,
		},
		EventTypeConditionChanged: ConditionChangedPayload{
			CharacterID:     "char-1",
			ConditionsAfter: []string{"hidden"},
		},
		EventTypeLoadoutSwapped: LoadoutSwappedPayload{
			CharacterID: "char-1",
			CardID:      "card-1",
			From:        "vault",
			To:          "active",
			StressAfter: intPtr(2),
		},
		EventTypeCharacterTemporaryArmorApplied: CharacterTemporaryArmorAppliedPayload{
			CharacterID: "char-1",
			Source:      "ritual",
			Duration:    "short_rest",
			Amount:      2,
			SourceID:    "tmp-1",
		},
		EventTypeRestTaken: RestTakenPayload{
			RestType:         "short",
			GMFearBefore:     2,
			GMFearAfter:      3,
			ShortRestsBefore: 0,
			ShortRestsAfter:  1,
			RefreshRest:      true,
			CharacterStates: []RestCharacterStatePatch{
				{
					CharacterID: "char-1",
					HopeAfter:   intPtr(3),
					StressAfter: intPtr(0),
				},
			},
		},
		EventTypeDamageApplied: DamageAppliedPayload{
			CharacterID: "char-1",
			HpAfter:     intPtr(5),
			ArmorAfter:  intPtr(0),
		},
		EventTypeDowntimeMoveApplied: DowntimeMoveAppliedPayload{
			CharacterID: "char-1",
			Move:        "prepare",
			HopeAfter:   intPtr(4),
		},
		EventTypeCountdownCreated: CountdownCreatedPayload{
			CountdownID:       "cd-1",
			Name:              "Doom Clock",
			Kind:              "progress",
			Current:           0,
			Max:               4,
			Direction:         "increase",
			Looping:           true,
			Variant:           "dynamic",
			TriggerEventType:  "sys.daggerheart.damage_applied",
			LinkedCountdownID: "",
		},
		EventTypeCountdownUpdated: CountdownUpdatedPayload{
			CountdownID: "cd-1",
			Before:      0,
			After:       1,
			Delta:       1,
			Looped:      false,
		},
		EventTypeCountdownDeleted: CountdownDeletedPayload{
			CountdownID: "cd-1",
		},
		EventTypeAdversaryCreated: AdversaryCreatedPayload{
			AdversaryID: "adv-1",
			Name:        "Goblin",
			Kind:        "bruiser",
			SessionID:   "sess-1",
			Notes:       "watchpost",
			HP:          6,
			HPMax:       6,
			Stress:      2,
			StressMax:   2,
			Evasion:     9,
			Major:       2,
			Severe:      4,
			Armor:       1,
		},
		EventTypeAdversaryConditionChanged: AdversaryConditionChangedPayload{
			AdversaryID:     "adv-1",
			ConditionsAfter: []string{"hidden"},
		},
		EventTypeAdversaryDamageApplied: AdversaryDamageAppliedPayload{
			AdversaryID: "adv-1",
			HpAfter:     intPtr(4),
			ArmorAfter:  intPtr(0),
		},
		EventTypeAdversaryUpdated: AdversaryUpdatedPayload{
			AdversaryID: "adv-1",
			Name:        "Goblin Captain",
			Kind:        "leader",
			SessionID:   "sess-1",
			Notes:       "reinforced",
			HP:          4,
			HPMax:       6,
			Stress:      2,
			StressMax:   2,
			Evasion:     10,
			Major:       2,
			Severe:      4,
			Armor:       1,
		},
		EventTypeAdversaryDeleted: AdversaryDeletedPayload{
			AdversaryID: "adv-1",
		},
		EventTypeLevelUpApplied: LevelUpAppliedPayload{
			CharacterID: "char-1",
			LevelBefore: 1,
			LevelAfter:  2,
			Advancements: []LevelUpAdvancementPayload{
				{Type: "add_hp_slots"},
				{Type: "add_stress_slots"},
			},
			Tier:           2,
			PreviousTier:   1,
			IsTierEntry:    true,
			ThresholdDelta: 1,
		},
		EventTypeGoldUpdated: GoldUpdatedPayload{
			CharacterID:    "char-1",
			HandfulsBefore: 0,
			HandfulsAfter:  3,
		},
		EventTypeDomainCardAcquired: DomainCardAcquiredPayload{
			CharacterID: "char-1",
			CardID:      "domain-card-new",
			CardLevel:   2,
			Destination: "vault",
		},
		EventTypeEquipmentSwapped: EquipmentSwappedPayload{
			CharacterID: "char-1",
			ItemID:      "sword-1",
			ItemType:    "weapon",
			From:        "inventory",
			To:          "active",
		},
		EventTypeConsumableUsed: ConsumableUsedPayload{
			CharacterID:    "char-1",
			ConsumableID:   "potion-1",
			QuantityBefore: 2,
			QuantityAfter:  1,
		},
		EventTypeConsumableAcquired: ConsumableAcquiredPayload{
			CharacterID:    "char-1",
			ConsumableID:   "scroll-1",
			QuantityBefore: 0,
			QuantityAfter:  1,
		},
	}
}

func intPtr(v int) *int {
	return &v
}

type parityDaggerheartStore struct {
	mu          sync.Mutex
	profiles    map[string]storage.DaggerheartCharacterProfile
	states      map[string]storage.DaggerheartCharacterState
	snapshots   map[string]storage.DaggerheartSnapshot
	countdowns  map[string]storage.DaggerheartCountdown
	adversaries map[string]storage.DaggerheartAdversary
}

func newParityDaggerheartStore() *parityDaggerheartStore {
	return &parityDaggerheartStore{
		profiles:    make(map[string]storage.DaggerheartCharacterProfile),
		states:      make(map[string]storage.DaggerheartCharacterState),
		snapshots:   make(map[string]storage.DaggerheartSnapshot),
		countdowns:  make(map[string]storage.DaggerheartCountdown),
		adversaries: make(map[string]storage.DaggerheartAdversary),
	}
}

func (m *parityDaggerheartStore) PutDaggerheartCharacterProfile(_ context.Context, profile storage.DaggerheartCharacterProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[m.characterKey(profile.CampaignID, profile.CharacterID)] = cloneCharacterProfile(profile)
	return nil
}

func (m *parityDaggerheartStore) GetDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) (storage.DaggerheartCharacterProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	profile, ok := m.profiles[m.characterKey(campaignID, characterID)]
	if !ok {
		return storage.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	return cloneCharacterProfile(profile), nil
}

func (m *parityDaggerheartStore) DeleteDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.profiles, m.characterKey(campaignID, characterID))
	return nil
}

func (m *parityDaggerheartStore) PutDaggerheartCharacterState(_ context.Context, state storage.DaggerheartCharacterState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[m.characterKey(state.CampaignID, state.CharacterID)] = cloneCharacterStateStorage(state)
	return nil
}

func (m *parityDaggerheartStore) GetDaggerheartCharacterState(_ context.Context, campaignID, characterID string) (storage.DaggerheartCharacterState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.states[m.characterKey(campaignID, characterID)]
	if !ok {
		return storage.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return cloneCharacterStateStorage(state), nil
}

func (m *parityDaggerheartStore) PutDaggerheartSnapshot(_ context.Context, snap storage.DaggerheartSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshots[snap.CampaignID] = snap
	return nil
}

func (m *parityDaggerheartStore) GetDaggerheartSnapshot(_ context.Context, campaignID string) (storage.DaggerheartSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	snap, ok := m.snapshots[campaignID]
	if !ok {
		return storage.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return snap, nil
}

func (m *parityDaggerheartStore) PutDaggerheartCountdown(_ context.Context, countdown storage.DaggerheartCountdown) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.countdowns[m.countdownKey(countdown.CampaignID, countdown.CountdownID)] = countdown
	return nil
}

func (m *parityDaggerheartStore) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (storage.DaggerheartCountdown, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	countdown, ok := m.countdowns[m.countdownKey(campaignID, countdownID)]
	if !ok {
		return storage.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return countdown, nil
}

func (m *parityDaggerheartStore) ListDaggerheartCountdowns(_ context.Context, campaignID string) ([]storage.DaggerheartCountdown, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := campaignID + "/"
	out := make([]storage.DaggerheartCountdown, 0)
	for key, countdown := range m.countdowns {
		if strings.HasPrefix(key, prefix) {
			out = append(out, countdown)
		}
	}
	return out, nil
}

func (m *parityDaggerheartStore) DeleteDaggerheartCountdown(_ context.Context, campaignID, countdownID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.countdowns, m.countdownKey(campaignID, countdownID))
	return nil
}

func (m *parityDaggerheartStore) PutDaggerheartAdversary(_ context.Context, adversary storage.DaggerheartAdversary) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.adversaries[m.adversaryKey(adversary.CampaignID, adversary.AdversaryID)] = cloneAdversary(adversary)
	return nil
}

func (m *parityDaggerheartStore) GetDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) (storage.DaggerheartAdversary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	adversary, ok := m.adversaries[m.adversaryKey(campaignID, adversaryID)]
	if !ok {
		return storage.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return cloneAdversary(adversary), nil
}

func (m *parityDaggerheartStore) ListDaggerheartAdversaries(_ context.Context, campaignID, _ string) ([]storage.DaggerheartAdversary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := campaignID + "/"
	out := make([]storage.DaggerheartAdversary, 0)
	for key, adversary := range m.adversaries {
		if strings.HasPrefix(key, prefix) {
			out = append(out, cloneAdversary(adversary))
		}
	}
	return out, nil
}

func (m *parityDaggerheartStore) DeleteDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.adversaries, m.adversaryKey(campaignID, adversaryID))
	return nil
}

func (m *parityDaggerheartStore) snapshotState(campaignID string) SnapshotState {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := SnapshotState{
		CampaignID:      campaignID,
		GMFear:          GMFearDefault,
		CharacterStates: make(map[string]CharacterState),
		AdversaryStates: make(map[string]AdversaryState),
		CountdownStates: make(map[string]CountdownState),
	}
	if snap, ok := m.snapshots[campaignID]; ok {
		state.GMFear = snap.GMFear
	}

	prefix := campaignID + "/"
	for key, stored := range m.states {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		armorMax := stored.Armor
		if profile, ok := m.profiles[key]; ok {
			armorMax = profile.ArmorMax
		}
		character := projection.CharacterStateFromStorage(stored, armorMax)
		state.CharacterStates[character.CharacterID] = character
	}
	for key, stored := range m.adversaries {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		state.AdversaryStates[stored.AdversaryID] = AdversaryState{
			CampaignID:  stored.CampaignID,
			AdversaryID: stored.AdversaryID,
			Name:        stored.Name,
			Kind:        stored.Kind,
			SessionID:   stored.SessionID,
			Notes:       stored.Notes,
			HP:          stored.HP,
			HPMax:       stored.HPMax,
			Stress:      stored.Stress,
			StressMax:   stored.StressMax,
			Evasion:     stored.Evasion,
			Major:       stored.Major,
			Severe:      stored.Severe,
			Armor:       stored.Armor,
			Conditions:  append([]string(nil), stored.Conditions...),
		}
	}
	for key, stored := range m.countdowns {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		state.CountdownStates[stored.CountdownID] = CountdownState{
			CampaignID:        stored.CampaignID,
			CountdownID:       stored.CountdownID,
			Name:              stored.Name,
			Kind:              stored.Kind,
			Current:           stored.Current,
			Max:               stored.Max,
			Direction:         stored.Direction,
			Looping:           stored.Looping,
			Variant:           stored.Variant,
			TriggerEventType:  stored.TriggerEventType,
			LinkedCountdownID: stored.LinkedCountdownID,
		}
	}
	return state
}

func (m *parityDaggerheartStore) characterKey(campaignID, characterID string) string {
	return campaignID + "/" + characterID
}

func (m *parityDaggerheartStore) countdownKey(campaignID, countdownID string) string {
	return campaignID + "/" + countdownID
}

func (m *parityDaggerheartStore) adversaryKey(campaignID, adversaryID string) string {
	return campaignID + "/" + adversaryID
}

func cloneCharacterProfile(profile storage.DaggerheartCharacterProfile) storage.DaggerheartCharacterProfile {
	out := profile
	out.Experiences = append([]storage.DaggerheartExperience(nil), profile.Experiences...)
	out.StartingWeaponIDs = append([]string(nil), profile.StartingWeaponIDs...)
	out.DomainCardIDs = append([]string(nil), profile.DomainCardIDs...)
	return out
}

func cloneCharacterStateStorage(state storage.DaggerheartCharacterState) storage.DaggerheartCharacterState {
	out := state
	out.Conditions = append([]string(nil), state.Conditions...)
	out.TemporaryArmor = append([]storage.DaggerheartTemporaryArmor(nil), state.TemporaryArmor...)
	return out
}

func cloneAdversary(adversary storage.DaggerheartAdversary) storage.DaggerheartAdversary {
	out := adversary
	out.Conditions = append([]string(nil), adversary.Conditions...)
	return out
}

func canonicalizeSnapshotForParity(state SnapshotState) SnapshotState {
	state.EnsureMaps()
	// DowntimeMovesSinceRest is fold-only state not projected to adapter storage.
	state.DowntimeMovesSinceRest = 0
	for id, character := range state.CharacterStates {
		character.HPMax = 0
		character.StressMax = 0
		character.ArmorMax = 0
		if len(character.ArmorBonus) == 0 {
			character.ArmorBonus = nil
		}
		if len(character.Conditions) == 0 {
			character.Conditions = nil
		}
		state.CharacterStates[id] = character
	}
	for id, adversary := range state.AdversaryStates {
		if len(adversary.Conditions) == 0 {
			adversary.Conditions = nil
		}
		state.AdversaryStates[id] = adversary
	}
	return state
}

var _ storage.DaggerheartStore = (*parityDaggerheartStore)(nil)
