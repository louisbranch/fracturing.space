package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestDaggerheartCharacterProfilePutGet(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-dhp", now)
	seedCharacter(t, store, "camp-dhp", "char-1", "Aria", character.CharacterKindPC, now)

	expected := storage.DaggerheartCharacterProfile{
		CampaignID:      "camp-dhp",
		CharacterID:     "char-1",
		Level:           5,
		HpMax:           18,
		StressMax:       12,
		Evasion:         10,
		MajorThreshold:  7,
		SevereThreshold: 14,
		Proficiency:     3,
		ArmorScore:      2,
		ArmorMax:        4,
		Experiences: []storage.DaggerheartExperience{
			{Name: "Stealth", Modifier: 2},
			{Name: "Perception", Modifier: 1},
		},
		Agility:   3,
		Strength:  1,
		Finesse:   4,
		Instinct:  2,
		Presence:  0,
		Knowledge: -1,
	}

	if err := store.PutDaggerheartCharacterProfile(context.Background(), expected); err != nil {
		t.Fatalf("put profile: %v", err)
	}

	got, err := store.GetDaggerheartCharacterProfile(context.Background(), "camp-dhp", "char-1")
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}

	if got.CampaignID != expected.CampaignID || got.CharacterID != expected.CharacterID {
		t.Fatalf("expected identity to match")
	}
	if got.Level != expected.Level || got.HpMax != expected.HpMax || got.StressMax != expected.StressMax {
		t.Fatalf("expected level/hp/stress to match: level=%d/%d hp=%d/%d stress=%d/%d",
			got.Level, expected.Level, got.HpMax, expected.HpMax, got.StressMax, expected.StressMax)
	}
	if got.Evasion != expected.Evasion || got.MajorThreshold != expected.MajorThreshold || got.SevereThreshold != expected.SevereThreshold {
		t.Fatalf("expected defense stats to match")
	}
	if got.Proficiency != expected.Proficiency || got.ArmorScore != expected.ArmorScore || got.ArmorMax != expected.ArmorMax {
		t.Fatalf("expected proficiency/armor to match")
	}
	if got.Agility != expected.Agility || got.Strength != expected.Strength || got.Finesse != expected.Finesse {
		t.Fatalf("expected traits (agi/str/fin) to match")
	}
	if got.Instinct != expected.Instinct || got.Presence != expected.Presence || got.Knowledge != expected.Knowledge {
		t.Fatalf("expected traits (ins/pre/kno) to match")
	}
	if len(got.Experiences) != 2 {
		t.Fatalf("expected 2 experiences, got %d", len(got.Experiences))
	}
	if got.Experiences[0].Name != "Stealth" || got.Experiences[0].Modifier != 2 {
		t.Fatalf("expected first experience to match")
	}
}

func TestDaggerheartCharacterProfileNotFound(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-dhp-nf", now)

	_, err := store.GetDaggerheartCharacterProfile(context.Background(), "camp-dhp-nf", "no-char")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDaggerheartCharacterStatePutGet(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-dhs", now)
	seedCharacter(t, store, "camp-dhs", "char-1", "Brim", character.CharacterKindPC, now)

	expected := storage.DaggerheartCharacterState{
		CampaignID:  "camp-dhs",
		CharacterID: "char-1",
		Hp:          15,
		Hope:        4,
		HopeMax:     6,
		Stress:      3,
		Armor:       2,
		Conditions:  []string{"blinded", "poisoned"},
		LifeState:   "alive",
	}

	if err := store.PutDaggerheartCharacterState(context.Background(), expected); err != nil {
		t.Fatalf("put state: %v", err)
	}

	got, err := store.GetDaggerheartCharacterState(context.Background(), "camp-dhs", "char-1")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}

	if got.CampaignID != expected.CampaignID || got.CharacterID != expected.CharacterID {
		t.Fatalf("expected identity to match")
	}
	if got.Hp != expected.Hp {
		t.Fatalf("expected hp %d, got %d", expected.Hp, got.Hp)
	}
	if got.Hope != expected.Hope || got.HopeMax != expected.HopeMax {
		t.Fatalf("expected hope %d/%d, got %d/%d", expected.Hope, expected.HopeMax, got.Hope, got.HopeMax)
	}
	if got.Stress != expected.Stress {
		t.Fatalf("expected stress %d, got %d", expected.Stress, got.Stress)
	}
	if got.Armor != expected.Armor {
		t.Fatalf("expected armor %d, got %d", expected.Armor, got.Armor)
	}
	if got.LifeState != expected.LifeState {
		t.Fatalf("expected life state %q, got %q", expected.LifeState, got.LifeState)
	}
	if len(got.Conditions) != 2 || got.Conditions[0] != "blinded" || got.Conditions[1] != "poisoned" {
		t.Fatalf("expected conditions [blinded, poisoned], got %v", got.Conditions)
	}
}

func TestDaggerheartCharacterStateConditions(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-cond", now)
	seedCharacter(t, store, "camp-cond", "char-1", "Cass", character.CharacterKindPC, now)

	// Nil conditions should round-trip as empty/nil slice
	state := storage.DaggerheartCharacterState{
		CampaignID:  "camp-cond",
		CharacterID: "char-1",
		Hp:          10,
		Hope:        3,
		HopeMax:     6,
		LifeState:   "alive",
	}
	if err := store.PutDaggerheartCharacterState(context.Background(), state); err != nil {
		t.Fatalf("put state: %v", err)
	}

	got, err := store.GetDaggerheartCharacterState(context.Background(), "camp-cond", "char-1")
	if err != nil {
		t.Fatalf("get state: %v", err)
	}
	if len(got.Conditions) != 0 {
		t.Fatalf("expected empty conditions, got %v", got.Conditions)
	}
}

func TestDaggerheartCharacterStateNotFound(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-dhs-nf", now)

	_, err := store.GetDaggerheartCharacterState(context.Background(), "camp-dhs-nf", "no-char")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDaggerheartSnapshotPutGet(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-snap", now)

	expected := storage.DaggerheartSnapshot{
		CampaignID:            "camp-snap",
		GMFear:                5,
		ConsecutiveShortRests: 2,
	}

	if err := store.PutDaggerheartSnapshot(context.Background(), expected); err != nil {
		t.Fatalf("put snapshot: %v", err)
	}

	got, err := store.GetDaggerheartSnapshot(context.Background(), "camp-snap")
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if got.CampaignID != expected.CampaignID {
		t.Fatalf("expected campaign id to match")
	}
	if got.GMFear != expected.GMFear {
		t.Fatalf("expected gm fear %d, got %d", expected.GMFear, got.GMFear)
	}
	if got.ConsecutiveShortRests != expected.ConsecutiveShortRests {
		t.Fatalf("expected consecutive short rests %d, got %d", expected.ConsecutiveShortRests, got.ConsecutiveShortRests)
	}
}

func TestDaggerheartSnapshotNotFoundReturnsZero(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-snap-nf", now)

	got, err := store.GetDaggerheartSnapshot(context.Background(), "camp-snap-nf")
	if err != nil {
		t.Fatalf("expected no error for missing snapshot, got %v", err)
	}
	if got.GMFear != 0 || got.ConsecutiveShortRests != 0 {
		t.Fatalf("expected zero-value snapshot, got fear=%d rests=%d", got.GMFear, got.ConsecutiveShortRests)
	}
}

func TestDaggerheartCountdownLifecycle(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-cd", now)

	expected := storage.DaggerheartCountdown{
		CampaignID:  "camp-cd",
		CountdownID: "cd-1",
		Name:        "Dragon Approach",
		Kind:        "threat",
		Current:     3,
		Max:         8,
		Direction:   "up",
		Looping:     true,
	}

	if err := store.PutDaggerheartCountdown(context.Background(), expected); err != nil {
		t.Fatalf("put countdown: %v", err)
	}

	got, err := store.GetDaggerheartCountdown(context.Background(), "camp-cd", "cd-1")
	if err != nil {
		t.Fatalf("get countdown: %v", err)
	}
	if got.Name != expected.Name || got.Kind != expected.Kind {
		t.Fatalf("expected name/kind to match")
	}
	if got.Current != expected.Current || got.Max != expected.Max {
		t.Fatalf("expected current/max to match")
	}
	if got.Direction != expected.Direction {
		t.Fatalf("expected direction %q, got %q", expected.Direction, got.Direction)
	}
	if got.Looping != expected.Looping {
		t.Fatalf("expected looping %v, got %v", expected.Looping, got.Looping)
	}

	// Non-looping countdown
	cd2 := storage.DaggerheartCountdown{
		CampaignID:  "camp-cd",
		CountdownID: "cd-2",
		Name:        "Ritual Timer",
		Kind:        "progress",
		Current:     0,
		Max:         4,
		Direction:   "up",
		Looping:     false,
	}
	if err := store.PutDaggerheartCountdown(context.Background(), cd2); err != nil {
		t.Fatalf("put countdown 2: %v", err)
	}

	list, err := store.ListDaggerheartCountdowns(context.Background(), "camp-cd")
	if err != nil {
		t.Fatalf("list countdowns: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 countdowns, got %d", len(list))
	}

	if err := store.DeleteDaggerheartCountdown(context.Background(), "camp-cd", "cd-1"); err != nil {
		t.Fatalf("delete countdown: %v", err)
	}

	list, err = store.ListDaggerheartCountdowns(context.Background(), "camp-cd")
	if err != nil {
		t.Fatalf("list countdowns after delete: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 countdown after delete, got %d", len(list))
	}
}

func TestDaggerheartCountdownNotFound(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-cd-nf", now)

	_, err := store.GetDaggerheartCountdown(context.Background(), "camp-cd-nf", "no-cd")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestDaggerheartAdversaryLifecycle(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-adv", now)

	expected := storage.DaggerheartAdversary{
		CampaignID:  "camp-adv",
		AdversaryID: "adv-1",
		Name:        "Shadow Drake",
		Kind:        "solo",
		SessionID:   "sess-1",
		Notes:       "Dangerous",
		HP:          20,
		HPMax:       20,
		Stress:      0,
		StressMax:   8,
		Evasion:     12,
		Major:       8,
		Severe:      15,
		Armor:       3,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := store.PutDaggerheartAdversary(context.Background(), expected); err != nil {
		t.Fatalf("put adversary: %v", err)
	}

	got, err := store.GetDaggerheartAdversary(context.Background(), "camp-adv", "adv-1")
	if err != nil {
		t.Fatalf("get adversary: %v", err)
	}
	if got.Name != expected.Name || got.Kind != expected.Kind {
		t.Fatalf("expected name/kind to match")
	}
	if got.SessionID != expected.SessionID {
		t.Fatalf("expected session id %q, got %q", expected.SessionID, got.SessionID)
	}
	if got.HP != expected.HP || got.HPMax != expected.HPMax {
		t.Fatalf("expected hp to match")
	}
	if got.Stress != expected.Stress || got.StressMax != expected.StressMax {
		t.Fatalf("expected stress to match")
	}
	if got.Evasion != expected.Evasion || got.Major != expected.Major || got.Severe != expected.Severe {
		t.Fatalf("expected defense stats to match")
	}
	if got.Armor != expected.Armor {
		t.Fatalf("expected armor to match")
	}

	// Adversary without session (nullable SessionID)
	adv2 := storage.DaggerheartAdversary{
		CampaignID:  "camp-adv",
		AdversaryID: "adv-2",
		Name:        "Goblin Scout",
		Kind:        "minion",
		HP:          4,
		HPMax:       4,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.PutDaggerheartAdversary(context.Background(), adv2); err != nil {
		t.Fatalf("put adversary 2: %v", err)
	}

	// List by campaign (empty session filter)
	all, err := store.ListDaggerheartAdversaries(context.Background(), "camp-adv", "")
	if err != nil {
		t.Fatalf("list adversaries by campaign: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 adversaries, got %d", len(all))
	}

	// List by session
	bySession, err := store.ListDaggerheartAdversaries(context.Background(), "camp-adv", "sess-1")
	if err != nil {
		t.Fatalf("list adversaries by session: %v", err)
	}
	if len(bySession) != 1 || bySession[0].AdversaryID != "adv-1" {
		t.Fatalf("expected 1 adversary for session, got %d", len(bySession))
	}

	// Delete
	if err := store.DeleteDaggerheartAdversary(context.Background(), "camp-adv", "adv-1"); err != nil {
		t.Fatalf("delete adversary: %v", err)
	}
	_, err = store.GetDaggerheartAdversary(context.Background(), "camp-adv", "adv-1")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestDaggerheartAdversaryNotFound(t *testing.T) {
	store := openTestStore(t)
	now := time.Date(2026, 2, 3, 11, 0, 0, 0, time.UTC)
	seedCampaign(t, store, "camp-adv-nf", now)

	_, err := store.GetDaggerheartAdversary(context.Background(), "camp-adv-nf", "no-adv")
	if err == nil || !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
