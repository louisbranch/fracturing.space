package adapter

import (
	"context"
	"testing"

	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func TestPutCharacterProfile_SeedsArmorStateFromNormalizedProfile(t *testing.T) {
	t.Parallel()

	store := newProfileStoreStub()
	a := NewAdapter(store, nil)

	err := a.PutCharacterProfile(context.Background(), "camp-1", "char-1", daggerheartstate.CharacterProfile{
		Level:           1,
		HpMax:           7,
		StressMax:       6,
		Evasion:         9,
		MajorThreshold:  8,
		SevereThreshold: 12,
		Proficiency:     1,
		ArmorScore:      4,
		StartingArmorID: "armor.chainmail-armor",
	})
	if err != nil {
		t.Fatalf("PutCharacterProfile returned error: %v", err)
	}

	profile, err := store.GetDaggerheartCharacterProfile(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("GetDaggerheartCharacterProfile returned error: %v", err)
	}
	if profile.EquippedArmorID != "armor.chainmail-armor" {
		t.Fatalf("equipped armor id = %q, want %q", profile.EquippedArmorID, "armor.chainmail-armor")
	}
	if profile.ArmorMax != 4 {
		t.Fatalf("armor max = %d, want 4", profile.ArmorMax)
	}

	state, err := store.GetDaggerheartCharacterState(context.Background(), "camp-1", "char-1")
	if err != nil {
		t.Fatalf("GetDaggerheartCharacterState returned error: %v", err)
	}
	if state.Armor != 4 {
		t.Fatalf("state armor = %d, want 4", state.Armor)
	}
}
