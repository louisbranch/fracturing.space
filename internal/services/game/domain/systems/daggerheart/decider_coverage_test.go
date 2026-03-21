package daggerheart

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestSnapshotEnvironmentEntityState_Branches(t *testing.T) {
	snapshot := SnapshotState{
		CampaignID: "camp-1",
		EnvironmentStates: map[ids.EnvironmentEntityID]EnvironmentEntityState{
			"ee-1": {
				EnvironmentID: "env-1",
				Name:          "Trap",
				Type:          "hazard",
				Tier:          1,
				Difficulty:    5,
			},
		},
	}

	// Valid lookup.
	got, ok := snapshotEnvironmentEntityState(snapshot, "ee-1")
	if !ok {
		t.Fatal("expected environment entity to be found")
	}
	if got.CampaignID != "camp-1" {
		t.Fatalf("campaign id = %s, want camp-1", got.CampaignID)
	}
	if got.EnvironmentEntityID != "ee-1" {
		t.Fatalf("entity id = %s, want ee-1", got.EnvironmentEntityID)
	}
	if got.Name != "Trap" {
		t.Fatalf("name = %s, want Trap", got.Name)
	}

	// Trimmed lookup.
	got, ok = snapshotEnvironmentEntityState(snapshot, " ee-1 ")
	if !ok {
		t.Fatal("expected trimmed environment entity to be found")
	}

	// Empty ID returns false.
	_, ok = snapshotEnvironmentEntityState(snapshot, "  ")
	if ok {
		t.Fatal("expected empty ID to return false")
	}

	// Missing entity returns false.
	_, ok = snapshotEnvironmentEntityState(snapshot, "ee-missing")
	if ok {
		t.Fatal("expected missing entity to return false")
	}
}

func TestIsEnvironmentEntityCreateNoMutation_Branches(t *testing.T) {
	snapshot := SnapshotState{
		CampaignID: "camp-1",
		EnvironmentStates: map[ids.EnvironmentEntityID]EnvironmentEntityState{
			"ee-1": {
				EnvironmentID: "env-1",
				Name:          "Trap",
				Type:          "hazard",
				Tier:          1,
				Difficulty:    5,
				SessionID:     "sess-1",
				SceneID:       "scene-1",
				Notes:         "Watch out",
			},
		},
	}

	// Exact match → no mutation.
	payload := EnvironmentEntityCreatePayload{
		EnvironmentEntityID: "ee-1",
		EnvironmentID:       "env-1",
		Name:                "Trap",
		Type:                "hazard",
		Tier:                1,
		Difficulty:          5,
		SessionID:           "sess-1",
		SceneID:             "scene-1",
		Notes:               "Watch out",
	}
	if !isEnvironmentEntityCreateNoMutation(snapshot, payload) {
		t.Fatal("expected no mutation for identical payload")
	}

	// Different name → mutation.
	payload.Name = "New Trap"
	if isEnvironmentEntityCreateNoMutation(snapshot, payload) {
		t.Fatal("expected mutation for different name")
	}

	// Missing entity → mutation.
	payload.EnvironmentEntityID = "ee-missing"
	if isEnvironmentEntityCreateNoMutation(snapshot, payload) {
		t.Fatal("expected mutation for missing entity")
	}
}

func TestIsAdversaryFeatureApplyNoMutation_Branches(t *testing.T) {
	xp := &AdversaryPendingExperience{Name: "xp", Modifier: 10}
	snapshot := SnapshotState{
		CampaignID: "camp-1",
		AdversaryStates: map[ids.AdversaryID]AdversaryState{
			"adv-1": {
				FeatureStates:     []AdversaryFeatureState{{FeatureID: "f1", Status: "active"}},
				PendingExperience: xp,
			},
		},
	}

	// Exact match → no mutation.
	if !isAdversaryFeatureApplyNoMutation(snapshot, AdversaryFeatureApplyPayload{
		AdversaryID:            "adv-1",
		FeatureStatesAfter:     []AdversaryFeatureState{{FeatureID: "f1", Status: "active"}},
		PendingExperienceAfter: xp,
	}) {
		t.Fatal("expected no mutation for identical state")
	}

	// Stress change → mutation.
	stressBefore, stressAfter := 0, 2
	if isAdversaryFeatureApplyNoMutation(snapshot, AdversaryFeatureApplyPayload{
		AdversaryID:        "adv-1",
		StressBefore:       &stressBefore,
		StressAfter:        &stressAfter,
		FeatureStatesAfter: []AdversaryFeatureState{{FeatureID: "f1", Status: "active"}},
	}) {
		t.Fatal("expected mutation for stress change")
	}

	// Feature state change → mutation.
	if isAdversaryFeatureApplyNoMutation(snapshot, AdversaryFeatureApplyPayload{
		AdversaryID:        "adv-1",
		FeatureStatesAfter: []AdversaryFeatureState{{FeatureID: "f1", Status: "used"}},
	}) {
		t.Fatal("expected mutation for feature state change")
	}

	// Pending experience change → mutation.
	newXP := &AdversaryPendingExperience{Name: "xp", Modifier: 20}
	if isAdversaryFeatureApplyNoMutation(snapshot, AdversaryFeatureApplyPayload{
		AdversaryID:            "adv-1",
		FeatureStatesAfter:     []AdversaryFeatureState{{FeatureID: "f1", Status: "active"}},
		PendingExperienceAfter: newXP,
	}) {
		t.Fatal("expected mutation for pending experience change")
	}

	// Missing adversary → mutation.
	if isAdversaryFeatureApplyNoMutation(snapshot, AdversaryFeatureApplyPayload{
		AdversaryID: "adv-missing",
	}) {
		t.Fatal("expected mutation for missing adversary")
	}
}

func TestCompanionStatePtrValue(t *testing.T) {
	// nil returns nil.
	if got := companionStatePtrValue(nil); got != nil {
		t.Fatal("expected nil for nil input")
	}
	// Non-nil returns normalized copy.
	state := &CharacterCompanionState{Status: " AWAY ", ActiveExperienceID: " exp-1 "}
	got := companionStatePtrValue(state)
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.Status != CompanionStatusAway || got.ActiveExperienceID != "exp-1" {
		t.Fatalf("got = %+v, want normalized away state", got)
	}
}

func TestSnapshotAdversaryState_Branches(t *testing.T) {
	snapshot := SnapshotState{
		CampaignID: "camp-1",
		AdversaryStates: map[ids.AdversaryID]AdversaryState{
			"adv-1": {Name: "Goblin"},
		},
	}

	// Valid lookup.
	got, ok := snapshotAdversaryState(snapshot, "adv-1")
	if !ok {
		t.Fatal("expected adversary to be found")
	}
	if got.CampaignID != "camp-1" || got.AdversaryID != "adv-1" {
		t.Fatalf("adversary = %+v, want camp-1/adv-1", got)
	}

	// Empty ID returns false.
	_, ok = snapshotAdversaryState(snapshot, "  ")
	if ok {
		t.Fatal("expected empty id to return false")
	}

	// Missing returns false.
	_, ok = snapshotAdversaryState(snapshot, "adv-missing")
	if ok {
		t.Fatal("expected missing to return false")
	}
}
