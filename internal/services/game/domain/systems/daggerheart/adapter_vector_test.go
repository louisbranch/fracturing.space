package daggerheart

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestAdapterAndFolder_EventVectorParity(t *testing.T) {
	store := newParityDaggerheartStore()
	adapter := NewAdapter(store)
	folder := NewFolder()

	vectors := daggerheartEventVectorsForParity()
	for _, def := range daggerheartEventDefinitions {
		if def.Intent != event.IntentProjectionAndReplay {
			continue
		}
		if _, ok := vectors[def.Type]; !ok {
			t.Fatalf("missing parity vector for %s", def.Type)
		}
	}
	if got, want := len(vectors), countProjectionAndReplayDefinitions(); got != want {
		t.Fatalf("event vectors = %d, projection/replay definitions = %d", got, want)
	}

	sequence := []event.Type{
		EventTypeGMFearChanged,
		EventTypeCharacterProfileReplaced,
		EventTypeCharacterProfileDeleted,
		EventTypeCharacterStatePatched,
		EventTypeBeastformTransformed,
		EventTypeBeastformDropped,
		EventTypeCompanionExperienceBegun,
		EventTypeCompanionReturned,
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
		EventTypeEnvironmentEntityCreated,
		EventTypeEnvironmentEntityUpdated,
		EventTypeEnvironmentEntityDeleted,
		EventTypeLevelUpApplied,
		EventTypeGoldUpdated,
		EventTypeDomainCardAcquired,
		EventTypeEquipmentSwapped,
		EventTypeConsumableUsed,
		EventTypeConsumableAcquired,
	}
	if got, want := len(sequence), countProjectionAndReplayDefinitions(); got != want {
		t.Fatalf("event sequence = %d, projection/replay definitions = %d", got, want)
	}
	seen := make(map[event.Type]struct{}, len(sequence))
	for _, typ := range sequence {
		if _, dup := seen[typ]; dup {
			t.Fatalf("duplicate sequence entry for %s", typ)
		}
		seen[typ] = struct{}{}
	}
	for _, def := range daggerheartEventDefinitions {
		if def.Intent != event.IntentProjectionAndReplay {
			continue
		}
		if _, ok := seen[def.Type]; !ok {
			t.Fatalf("event definition %s not covered by sequence", def.Type)
		}
	}

	ctx := context.Background()
	campaignID := ids.CampaignID("camp-1")
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
			EntityID:      string(campaignID),
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
		adapterState := store.snapshotState(string(campaignID))
		folderState = canonicalizeSnapshotForParity(folderState)
		adapterState = canonicalizeSnapshotForParity(adapterState)
		if !reflect.DeepEqual(folderState, adapterState) {
			t.Fatalf("state mismatch after %s\nfolder=%#v\nadapter=%#v", typ, folderState, adapterState)
		}
	}
}

func countProjectionAndReplayDefinitions() int {
	count := 0
	for _, def := range daggerheartEventDefinitions {
		if def.Intent == event.IntentProjectionAndReplay {
			count++
		}
	}
	return count
}

func daggerheartEventVectorsForParity() map[event.Type]any {
	lifeStateAlive := LifeStateAlive
	return map[event.Type]any{
		EventTypeGMFearChanged: GMFearChangedPayload{
			Value: 2,
		},
		EventTypeCharacterProfileReplaced: CharacterProfileReplacedPayload{
			CharacterID: "char-1",
			Profile: CharacterProfile{
				Level:           1,
				HpMax:           6,
				StressMax:       6,
				Evasion:         10,
				MajorThreshold:  1,
				SevereThreshold: 2,
				Proficiency:     1,
				ArmorScore:      0,
				ArmorMax:        0,
				ClassID:         "class.guardian",
				SubclassID:      "subclass.stalwart",
			},
		},
		EventTypeCharacterProfileDeleted: CharacterProfileDeletedPayload{
			CharacterID: "char-1",
		},
		EventTypeCharacterStatePatched: CharacterStatePatchedPayload{
			CharacterID: "char-1",
			HP:          intPtr(6),
			Hope:        intPtr(2),
			HopeMax:     intPtr(6),
			Stress:      intPtr(1),
			Armor:       intPtr(0),
			LifeState:   &lifeStateAlive,
		},
		EventTypeBeastformTransformed: BeastformTransformedPayload{
			CharacterID: "char-1",
			BeastformID: "beastform.wolf",
			Stress:      intPtr(2),
			ActiveBeastform: &CharacterActiveBeastformState{
				BeastformID:  "beastform.wolf",
				BaseTrait:    "agility",
				AttackTrait:  "agility",
				TraitBonus:   1,
				EvasionBonus: 1,
				AttackRange:  "melee",
				DamageDice: []CharacterDamageDie{
					{Count: 1, Sides: 8},
				},
				DamageBonus: 1,
				DamageType:  "physical",
			},
			Source: "beastform.transform",
		},
		EventTypeBeastformDropped: BeastformDroppedPayload{
			CharacterID: "char-1",
			BeastformID: "beastform.wolf",
			Source:      "beastform.drop",
		},
		EventTypeCompanionExperienceBegun: CompanionExperienceBegunPayload{
			CharacterID:  "char-1",
			ExperienceID: "companion-experience.scout",
			CompanionState: &CharacterCompanionState{
				Status:             CompanionStatusAway,
				ActiveExperienceID: "companion-experience.scout",
			},
			Source: "companion.experience.begin",
		},
		EventTypeCompanionReturned: CompanionReturnedPayload{
			CharacterID: "char-1",
			Resolution:  "experience_completed",
			Stress:      intPtr(0),
			CompanionState: &CharacterCompanionState{
				Status: CompanionStatusPresent,
			},
			Source: "companion.return",
		},
		EventTypeConditionChanged: ConditionChangedPayload{
			CharacterID: "char-1",
			Conditions:  []ConditionState{mustConditionState("hidden")},
		},
		EventTypeLoadoutSwapped: LoadoutSwappedPayload{
			CharacterID: "char-1",
			CardID:      "card-1",
			From:        "vault",
			To:          "active",
			Stress:      intPtr(2),
		},
		EventTypeCharacterTemporaryArmorApplied: CharacterTemporaryArmorAppliedPayload{
			CharacterID: "char-1",
			Source:      "ritual",
			Duration:    "short_rest",
			Amount:      2,
			SourceID:    "tmp-1",
		},
		EventTypeRestTaken: RestTakenPayload{
			RestType:     "short",
			GMFear:       3,
			ShortRests:   1,
			RefreshRest:  true,
			Participants: []ids.CharacterID{"char-1"},
		},
		EventTypeDamageApplied: DamageAppliedPayload{
			CharacterID: "char-1",
			Hp:          intPtr(5),
			Armor:       intPtr(0),
		},
		EventTypeDowntimeMoveApplied: DowntimeMoveAppliedPayload{
			ActorCharacterID:  "char-1",
			TargetCharacterID: "char-1",
			Move:              "prepare",
			Hope:              intPtr(4),
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
			Value:       1,
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
			AdversaryID: "adv-1",
			Conditions:  []ConditionState{mustConditionState("hidden")},
		},
		EventTypeAdversaryDamageApplied: AdversaryDamageAppliedPayload{
			AdversaryID: "adv-1",
			Hp:          intPtr(4),
			Armor:       intPtr(0),
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
		EventTypeEnvironmentEntityCreated: EnvironmentEntityCreatedPayload{
			EnvironmentEntityID: "env-entity-1",
			EnvironmentID:       "environment.falling-ruins",
			Name:                "Falling Ruins",
			Type:                "hazard",
			Tier:                2,
			Difficulty:          15,
			SessionID:           "sess-1",
			SceneID:             "scene-1",
			Notes:               "Loose stone and dust",
		},
		EventTypeEnvironmentEntityUpdated: EnvironmentEntityUpdatedPayload{
			EnvironmentEntityID: "env-entity-1",
			EnvironmentID:       "environment.falling-ruins",
			Name:                "Falling Ruins",
			Type:                "hazard",
			Tier:                2,
			Difficulty:          16,
			SessionID:           "sess-1",
			SceneID:             "scene-2",
			Notes:               "Collapsed toward the exit",
		},
		EventTypeEnvironmentEntityDeleted: EnvironmentEntityDeletedPayload{
			EnvironmentEntityID: "env-entity-1",
			Reason:              "scene cleanup",
		},
		EventTypeLevelUpApplied: LevelUpAppliedPayload{
			CharacterID: "char-1",
			Level:       2,
			Advancements: []LevelUpAdvancementPayload{
				{Type: "add_hp_slots"},
				{Type: "add_stress_slots"},
			},
			Tier:           2,
			IsTierEntry:    true,
			ThresholdDelta: 1,
		},
		EventTypeGoldUpdated: GoldUpdatedPayload{
			CharacterID: "char-1",
			Handfuls:    3,
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
			CharacterID:  "char-1",
			ConsumableID: "potion-1",
			Quantity:     1,
		},
		EventTypeConsumableAcquired: ConsumableAcquiredPayload{
			CharacterID:  "char-1",
			ConsumableID: "scroll-1",
			Quantity:     1,
		},
	}
}

func intPtr(v int) *int {
	return &v
}

type parityDaggerheartStore struct {
	mu           sync.Mutex
	profiles     map[string]projectionstore.DaggerheartCharacterProfile
	states       map[string]projectionstore.DaggerheartCharacterState
	snapshots    map[string]projectionstore.DaggerheartSnapshot
	countdowns   map[string]projectionstore.DaggerheartCountdown
	adversaries  map[string]projectionstore.DaggerheartAdversary
	environments map[string]projectionstore.DaggerheartEnvironmentEntity
}

func newParityDaggerheartStore() *parityDaggerheartStore {
	return &parityDaggerheartStore{
		profiles:     make(map[string]projectionstore.DaggerheartCharacterProfile),
		states:       make(map[string]projectionstore.DaggerheartCharacterState),
		snapshots:    make(map[string]projectionstore.DaggerheartSnapshot),
		countdowns:   make(map[string]projectionstore.DaggerheartCountdown),
		adversaries:  make(map[string]projectionstore.DaggerheartAdversary),
		environments: make(map[string]projectionstore.DaggerheartEnvironmentEntity),
	}
}

func (m *parityDaggerheartStore) PutDaggerheartCharacterProfile(_ context.Context, profile projectionstore.DaggerheartCharacterProfile) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.profiles[m.characterKey(profile.CampaignID, profile.CharacterID)] = cloneCharacterProfile(profile)
	return nil
}

func (m *parityDaggerheartStore) GetDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	profile, ok := m.profiles[m.characterKey(campaignID, characterID)]
	if !ok {
		return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
	}
	return cloneCharacterProfile(profile), nil
}

func (m *parityDaggerheartStore) ListDaggerheartCharacterProfiles(_ context.Context, campaignID string, _ int, _ string) (projectionstore.DaggerheartCharacterProfilePage, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	page := projectionstore.DaggerheartCharacterProfilePage{
		Profiles: make([]projectionstore.DaggerheartCharacterProfile, 0),
	}
	prefix := campaignID + ":"
	for key, profile := range m.profiles {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			page.Profiles = append(page.Profiles, cloneCharacterProfile(profile))
		}
	}
	return page, nil
}

func (m *parityDaggerheartStore) DeleteDaggerheartCharacterProfile(_ context.Context, campaignID, characterID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.profiles, m.characterKey(campaignID, characterID))
	return nil
}

func (m *parityDaggerheartStore) PutDaggerheartCharacterState(_ context.Context, state projectionstore.DaggerheartCharacterState) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.states[m.characterKey(state.CampaignID, state.CharacterID)] = cloneCharacterStateStorage(state)
	return nil
}

func (m *parityDaggerheartStore) GetDaggerheartCharacterState(_ context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state, ok := m.states[m.characterKey(campaignID, characterID)]
	if !ok {
		return projectionstore.DaggerheartCharacterState{}, storage.ErrNotFound
	}
	return cloneCharacterStateStorage(state), nil
}

func (m *parityDaggerheartStore) PutDaggerheartSnapshot(_ context.Context, snap projectionstore.DaggerheartSnapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.snapshots[snap.CampaignID] = snap
	return nil
}

func (m *parityDaggerheartStore) GetDaggerheartSnapshot(_ context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	snap, ok := m.snapshots[campaignID]
	if !ok {
		return projectionstore.DaggerheartSnapshot{}, storage.ErrNotFound
	}
	return snap, nil
}

func (m *parityDaggerheartStore) PutDaggerheartCountdown(_ context.Context, countdown projectionstore.DaggerheartCountdown) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.countdowns[m.countdownKey(countdown.CampaignID, countdown.CountdownID)] = countdown
	return nil
}

func (m *parityDaggerheartStore) GetDaggerheartCountdown(_ context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	countdown, ok := m.countdowns[m.countdownKey(campaignID, countdownID)]
	if !ok {
		return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
	}
	return countdown, nil
}

func (m *parityDaggerheartStore) ListDaggerheartCountdowns(_ context.Context, campaignID string) ([]projectionstore.DaggerheartCountdown, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := campaignID + "/"
	out := make([]projectionstore.DaggerheartCountdown, 0)
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

func (m *parityDaggerheartStore) PutDaggerheartAdversary(_ context.Context, adversary projectionstore.DaggerheartAdversary) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.adversaries[m.adversaryKey(adversary.CampaignID, adversary.AdversaryID)] = cloneAdversary(adversary)
	return nil
}

func (m *parityDaggerheartStore) GetDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	adversary, ok := m.adversaries[m.adversaryKey(campaignID, adversaryID)]
	if !ok {
		return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return cloneAdversary(adversary), nil
}

func (m *parityDaggerheartStore) ListDaggerheartAdversaries(_ context.Context, campaignID, _ string) ([]projectionstore.DaggerheartAdversary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	prefix := campaignID + "/"
	out := make([]projectionstore.DaggerheartAdversary, 0)
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

func (m *parityDaggerheartStore) PutDaggerheartEnvironmentEntity(_ context.Context, environmentEntity projectionstore.DaggerheartEnvironmentEntity) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.environments[environmentEntity.CampaignID+":"+environmentEntity.EnvironmentEntityID] = environmentEntity
	return nil
}

func (m *parityDaggerheartStore) GetDaggerheartEnvironmentEntity(_ context.Context, campaignID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	environmentEntity, ok := m.environments[campaignID+":"+environmentEntityID]
	if !ok {
		return projectionstore.DaggerheartEnvironmentEntity{}, storage.ErrNotFound
	}
	return environmentEntity, nil
}

func (m *parityDaggerheartStore) ListDaggerheartEnvironmentEntities(_ context.Context, campaignID, sessionID, sceneID string) ([]projectionstore.DaggerheartEnvironmentEntity, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]projectionstore.DaggerheartEnvironmentEntity, 0)
	prefix := campaignID + ":"
	for key, environmentEntity := range m.environments {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		if sessionID != "" && environmentEntity.SessionID != sessionID {
			continue
		}
		if sceneID != "" && environmentEntity.SceneID != sceneID {
			continue
		}
		out = append(out, environmentEntity)
	}
	return out, nil
}

func (m *parityDaggerheartStore) DeleteDaggerheartEnvironmentEntity(_ context.Context, campaignID, environmentEntityID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.environments, campaignID+":"+environmentEntityID)
	return nil
}

func (m *parityDaggerheartStore) snapshotState(campaignID string) SnapshotState {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := SnapshotState{
		CampaignID:              ids.CampaignID(campaignID),
		GMFear:                  GMFearDefault,
		CharacterProfiles:       make(map[ids.CharacterID]CharacterProfile),
		CharacterStates:         make(map[ids.CharacterID]CharacterState),
		CharacterClassStates:    make(map[ids.CharacterID]CharacterClassState),
		CharacterSubclassStates: make(map[ids.CharacterID]CharacterSubclassState),
		CharacterCompanions:     make(map[ids.CharacterID]CharacterCompanionState),
		AdversaryStates:         make(map[ids.AdversaryID]AdversaryState),
		EnvironmentStates:       make(map[ids.EnvironmentEntityID]EnvironmentEntityState),
		CountdownStates:         make(map[ids.CountdownID]CountdownState),
	}
	if snap, ok := m.snapshots[campaignID]; ok {
		state.GMFear = snap.GMFear
	}

	prefix := campaignID + "/"
	for key, stored := range m.profiles {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		state.CharacterProfiles[ids.CharacterID(stored.CharacterID)] = CharacterProfileFromStorage(stored)
	}
	for key, stored := range m.states {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		armorMax := stored.Armor
		if profile, ok := m.profiles[key]; ok {
			armorMax = profile.ArmorMax
		}
		character := projection.CharacterStateFromStorage(stored, armorMax)
		state.CharacterStates[ids.CharacterID(character.CharacterID)] = character
		classState := classStateFromProjection(stored.ClassState)
		if !classState.IsZero() {
			state.CharacterClassStates[ids.CharacterID(character.CharacterID)] = classState
		}
		if companionState := normalizedCompanionStatePtr(companionStateFromProjection(stored.CompanionState)); companionState != nil && !companionState.IsZero() {
			state.CharacterCompanions[ids.CharacterID(character.CharacterID)] = *companionState
		}
		subclassState := subclassStateFromProjection(stored.SubclassState)
		if subclassState != nil && !subclassState.IsZero() {
			state.CharacterSubclassStates[ids.CharacterID(character.CharacterID)] = *subclassState
		}
	}
	for key, stored := range m.adversaries {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		state.AdversaryStates[ids.AdversaryID(stored.AdversaryID)] = AdversaryState{
			CampaignID:  ids.CampaignID(stored.CampaignID),
			AdversaryID: ids.AdversaryID(stored.AdversaryID),
			Name:        stored.Name,
			Kind:        stored.Kind,
			SessionID:   ids.SessionID(stored.SessionID),
			Notes:       stored.Notes,
			HP:          stored.HP,
			HPMax:       stored.HPMax,
			Stress:      stored.Stress,
			StressMax:   stored.StressMax,
			Evasion:     stored.Evasion,
			Major:       stored.Major,
			Severe:      stored.Severe,
			Armor:       stored.Armor,
			Conditions:  projectionConditionCodes(stored.Conditions),
		}
	}
	for key, stored := range m.countdowns {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		state.CountdownStates[ids.CountdownID(stored.CountdownID)] = CountdownState{
			CampaignID:        ids.CampaignID(stored.CampaignID),
			CountdownID:       ids.CountdownID(stored.CountdownID),
			Name:              stored.Name,
			Kind:              stored.Kind,
			Current:           stored.Current,
			Max:               stored.Max,
			Direction:         stored.Direction,
			Looping:           stored.Looping,
			Variant:           stored.Variant,
			TriggerEventType:  stored.TriggerEventType,
			LinkedCountdownID: ids.CountdownID(stored.LinkedCountdownID),
		}
	}
	environmentPrefix := campaignID + ":"
	for key, stored := range m.environments {
		if !strings.HasPrefix(key, environmentPrefix) {
			continue
		}
		state.EnvironmentStates[ids.EnvironmentEntityID(stored.EnvironmentEntityID)] = EnvironmentEntityState{
			CampaignID:          ids.CampaignID(stored.CampaignID),
			EnvironmentEntityID: ids.EnvironmentEntityID(stored.EnvironmentEntityID),
			EnvironmentID:       stored.EnvironmentID,
			Name:                stored.Name,
			Type:                stored.Type,
			Tier:                stored.Tier,
			Difficulty:          stored.Difficulty,
			SessionID:           ids.SessionID(stored.SessionID),
			SceneID:             ids.SceneID(stored.SceneID),
			Notes:               stored.Notes,
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

func cloneCharacterProfile(profile projectionstore.DaggerheartCharacterProfile) projectionstore.DaggerheartCharacterProfile {
	out := profile
	out.Experiences = append([]projectionstore.DaggerheartExperience(nil), profile.Experiences...)
	out.StartingWeaponIDs = append([]string(nil), profile.StartingWeaponIDs...)
	out.DomainCardIDs = append([]string(nil), profile.DomainCardIDs...)
	return out
}

func cloneCharacterStateStorage(state projectionstore.DaggerheartCharacterState) projectionstore.DaggerheartCharacterState {
	out := state
	out.Conditions = append([]projectionstore.DaggerheartConditionState(nil), state.Conditions...)
	out.TemporaryArmor = append([]projectionstore.DaggerheartTemporaryArmor(nil), state.TemporaryArmor...)
	return out
}

func cloneAdversary(adversary projectionstore.DaggerheartAdversary) projectionstore.DaggerheartAdversary {
	out := adversary
	out.Conditions = append([]projectionstore.DaggerheartConditionState(nil), adversary.Conditions...)
	return out
}

func projectionConditionsToDomain(states []projectionstore.DaggerheartConditionState) []ConditionState {
	out := make([]ConditionState, 0, len(states))
	for _, state := range states {
		out = append(out, ConditionState{
			ID:       state.ID,
			Class:    ConditionClass(state.Class),
			Standard: state.Standard,
			Code:     state.Code,
			Label:    state.Label,
			Source:   state.Source,
			SourceID: state.SourceID,
		})
	}
	return out
}

func projectionConditionCodes(states []projectionstore.DaggerheartConditionState) []string {
	out := make([]string, 0, len(states))
	for _, state := range states {
		out = append(out, state.Code)
	}
	return out
}

func canonicalizeSnapshotForParity(state SnapshotState) SnapshotState {
	state.EnsureMaps()
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
	for id, profile := range state.CharacterProfiles {
		if len(profile.Experiences) == 0 {
			profile.Experiences = nil
		}
		if len(profile.StartingWeaponIDs) == 0 {
			profile.StartingWeaponIDs = nil
		}
		if len(profile.DomainCardIDs) == 0 {
			profile.DomainCardIDs = nil
		}
		state.CharacterProfiles[id] = profile
	}
	for id, classState := range state.CharacterClassStates {
		if classState.IsZero() {
			delete(state.CharacterClassStates, id)
			continue
		}
		state.CharacterClassStates[id] = classState.Normalized()
	}
	for id, adversary := range state.AdversaryStates {
		if len(adversary.Conditions) == 0 {
			adversary.Conditions = nil
		}
		state.AdversaryStates[id] = adversary
	}
	return state
}

var _ projectionstore.Store = (*parityDaggerheartStore)(nil)
