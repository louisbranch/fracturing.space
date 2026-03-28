package gmmovetransport

import (
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGMMoveHelperFinders(t *testing.T) {
	t.Parallel()

	entry := contentstore.DaggerheartAdversaryEntry{
		Features:    []contentstore.DaggerheartAdversaryFeature{{ID: "feat-1", Name: "Cloaked"}},
		Experiences: []contentstore.DaggerheartAdversaryExperience{{Name: "Tracker"}},
	}
	if feature, ok := findAdversaryFeature(entry, "feat-1"); !ok || feature.Name != "Cloaked" {
		t.Fatalf("findAdversaryFeature() = %+v, %v", feature, ok)
	}
	if _, ok := findAdversaryFeature(entry, "missing"); ok {
		t.Fatal("findAdversaryFeature() unexpectedly found missing feature")
	}
	if experience, ok := findAdversaryExperience(entry, "tracker"); !ok || experience.Name != "Tracker" {
		t.Fatalf("findAdversaryExperience() = %+v, %v", experience, ok)
	}
	if _, ok := findAdversaryExperience(entry, "missing"); ok {
		t.Fatal("findAdversaryExperience() unexpectedly found missing experience")
	}

	env := contentstore.DaggerheartEnvironment{
		Features: []contentstore.DaggerheartFeature{{ID: "env-1", Name: "Fog"}},
	}
	if feature, ok := findEnvironmentFeature(env, "env-1"); !ok || feature.Name != "Fog" {
		t.Fatalf("findEnvironmentFeature() = %+v, %v", feature, ok)
	}
}

func TestGMMoveHelperFeatureStateHelpers(t *testing.T) {
	t.Parallel()

	current := []projectionstore.DaggerheartAdversaryFeatureState{{FeatureID: "feat-1", Status: "old"}}
	next := upsertFeatureState(current, projectionstore.DaggerheartAdversaryFeatureState{FeatureID: "feat-1", Status: "new"})
	if len(next) != 1 || next[0].Status != "new" {
		t.Fatalf("updated states = %+v", next)
	}

	appended := upsertFeatureState(current, projectionstore.DaggerheartAdversaryFeatureState{FeatureID: "feat-2", Status: "fresh"})
	if len(appended) != 2 {
		t.Fatalf("appended states = %+v", appended)
	}

	bridged := toBridgeAdversaryFeatureStates([]projectionstore.DaggerheartAdversaryFeatureState{{FeatureID: "feat-2", Status: "fresh", FocusedTargetID: "char-1"}})
	if len(bridged) != 1 || bridged[0].FeatureID != "feat-2" || bridged[0].FocusedTargetID != "char-1" {
		t.Fatalf("bridged states = %+v", bridged)
	}

	pending := toBridgeAdversaryPendingExperience(&projectionstore.DaggerheartAdversaryPendingExperience{Name: "Tracker", Modifier: 2})
	if pending == nil || pending.Name != "Tracker" || pending.Modifier != 2 {
		t.Fatalf("pending experience = %+v", pending)
	}
	if toBridgeAdversaryPendingExperience(nil) != nil {
		t.Fatal("nil pending experience should stay nil")
	}
}

func TestGMMoveHelperStageStatusAndPayload(t *testing.T) {
	t.Parallel()

	adversary := projectionstore.DaggerheartAdversary{AdversaryID: "adv-1", FeatureStates: []projectionstore.DaggerheartAdversaryFeatureState{{FeatureID: "other", Status: "active"}}}
	payload := stagedFearFeaturePayload(adversary, contentstore.DaggerheartAdversaryFeature{ID: "feat-1", Name: "Cloaked"}, "char-9")
	if payload == nil {
		t.Fatal("stagedFearFeaturePayload() returned nil for supported feature")
	}
	if payload.FeatureID != "feat-1" || len(payload.FeatureStatesAfter) != 2 {
		t.Fatalf("payload = %+v", payload)
	}
	if payload.FeatureStatesAfter[1].Status != stageStatusForRule(&rules.AdversaryFeatureRule{Kind: rules.AdversaryFeatureRuleKindHiddenUntilNextAttack}) {
		t.Fatalf("payload status = %+v", payload.FeatureStatesAfter[1])
	}
	if payload.FeatureStatesAfter[1].FocusedTargetID != "char-9" {
		t.Fatalf("focused target = %q, want char-9", payload.FeatureStatesAfter[1].FocusedTargetID)
	}

	if stagedFearFeaturePayload(adversary, contentstore.DaggerheartAdversaryFeature{Name: "Arcane Shield"}, "") != nil {
		t.Fatal("unsupported feature unexpectedly staged")
	}
}

func TestGMMoveHelperCurrentGateAndContentErrorMapping(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(Dependencies{SessionGate: testGateStore{err: storage.ErrNotFound}})
	if gate, open, err := handler.currentGMConsequenceGate(testContext(), "camp-1", "sess-1"); err != nil || open || gate.GateID != "" {
		t.Fatalf("currentGMConsequenceGate(not found) = %+v, %v, %v", gate, open, err)
	}

	handler = newTestHandler(Dependencies{SessionGate: testGateStore{gate: storage.SessionGate{GateID: "gate-1", GateType: "gm_consequence"}}})
	if gate, open, err := handler.currentGMConsequenceGate(testContext(), "camp-1", "sess-1"); err != nil || !open || gate.GateID != "gate-1" {
		t.Fatalf("currentGMConsequenceGate(open) = %+v, %v, %v", gate, open, err)
	}

	handler = newTestHandler(Dependencies{SessionGate: testGateStore{gate: storage.SessionGate{GateID: "gate-2", GateType: "scene_lock"}}})
	if _, _, err := handler.currentGMConsequenceGate(testContext(), "camp-1", "sess-1"); status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("currentGMConsequenceGate(wrong type) code = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}

	if err := mapContentErr("load thing", storage.ErrNotFound); status.Code(err) != codes.NotFound {
		t.Fatalf("mapContentErr(not found) code = %v, want %v", status.Code(err), codes.NotFound)
	}
	if err := mapContentErr("load thing", errors.New("boom")); status.Code(err) != codes.Internal {
		t.Fatalf("mapContentErr(internal) code = %v, want %v", status.Code(err), codes.Internal)
	}
}
