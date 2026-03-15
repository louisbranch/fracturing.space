package creationworkflow

import (
	"testing"

	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
)

func TestDefaultProfileForCharacter(t *testing.T) {
	profile := defaultProfileForCharacter("c1", character.KindPC)

	if profile.CampaignID != "c1" {
		t.Fatalf("CampaignID = %q, want %q", profile.CampaignID, "c1")
	}
	if profile.HpMax == 0 {
		t.Fatal("HpMax should have a default > 0")
	}
}

func TestEnsureProfileDefaults_PreservesExisting(t *testing.T) {
	profile := projectionstore.DaggerheartCharacterProfile{
		HpMax:     20,
		StressMax: 8,
		Evasion:   12,
		Level:     3,
	}
	result := ensureProfileDefaults(profile, character.KindPC)

	if result.HpMax != 20 {
		t.Fatalf("HpMax = %d, want 20 (should preserve existing)", result.HpMax)
	}
	if result.StressMax != 8 {
		t.Fatalf("StressMax = %d, want 8", result.StressMax)
	}
	if result.Evasion != 12 {
		t.Fatalf("Evasion = %d, want 12", result.Evasion)
	}
	if result.Level != 3 {
		t.Fatalf("Level = %d, want 3", result.Level)
	}
}

func TestEnsureProfileDefaults_NPC(t *testing.T) {
	profile := ensureProfileDefaults(projectionstore.DaggerheartCharacterProfile{}, character.KindNPC)

	if profile.HpMax == 0 {
		t.Fatal("NPC HpMax should have a default > 0")
	}
}

func TestSystemProfileMap_Empty(t *testing.T) {
	profile := daggerheart.CharacterProfileFromStorage(projectionstore.DaggerheartCharacterProfile{})
	if profile.Level != 0 {
		t.Fatalf("Level = %d, want 0", profile.Level)
	}
}

func TestCharacterProfileFromStorage_IncludesDescription(t *testing.T) {
	profile := daggerheart.CharacterProfileFromStorage(projectionstore.DaggerheartCharacterProfile{
		Description: "Tall, patient, and heavily armored.",
	})
	if got := profile.Description; got != "Tall, patient, and heavily armored." {
		t.Fatalf("description = %#v, want %q", got, "Tall, patient, and heavily armored.")
	}
}

func TestCreationProfileFromStorage_PreservesDescription(t *testing.T) {
	profile := daggerheart.CharacterProfileFromStorage(projectionstore.DaggerheartCharacterProfile{
		Description: "A calm veteran with a scarred shield.",
	}).CreationProfile()

	if profile.Description != "A calm veteran with a scarred shield." {
		t.Fatalf("Description = %q, want %q", profile.Description, "A calm veteran with a scarred shield.")
	}
}
