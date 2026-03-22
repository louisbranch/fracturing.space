package daggerheart

import (
	"encoding/json"
	"testing"
	"time"

	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"

	daggerheartdecider "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/decider"

	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func TestDecideGMFearSet_EmitsGMFearChanged(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.gm_fear.set"),
		ActorType:     command.ActorTypeGM,
		ActorID:       "gm-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"after":4,"reason":"  doom "}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(daggerheartstate.SnapshotState{GMFear: 2}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.gm_fear_changed") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.gm_fear_changed")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "campaign" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "campaign")
	}
	if evt.EntityID != "camp-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "camp-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}
	if evt.ActorType != event.ActorTypeGM {
		t.Fatalf("actor type = %s, want %s", evt.ActorType, event.ActorTypeGM)
	}
	if evt.ActorID != "gm-1" {
		t.Fatalf("actor id = %s, want %s", evt.ActorID, "gm-1")
	}

	var payload daggerheartpayload.GMFearChangedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Value != 4 {
		t.Fatalf("payload value = %d, want %d", payload.Value, 4)
	}
	if payload.Reason != "doom" {
		t.Fatalf("payload reason = %s, want %s", payload.Reason, "doom")
	}
}

func TestDecideGMFearSet_RejectsLegacyActionType(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys." + SystemID + ".action.gm_fear.set"),
		ActorType:     command.ActorTypeGM,
		ActorID:       "gm-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"after":4}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(daggerheartstate.SnapshotState{GMFear: 2}, cmd, func() time.Time { return now })
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeCommandTypeUnsupported {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeCommandTypeUnsupported)
	}
}

func TestDecideUnsupportedCommandRejected(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.unknown.command"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(daggerheartstate.SnapshotState{}, cmd, func() time.Time { return now })
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeCommandTypeUnsupported {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeCommandTypeUnsupported)
	}
}

func TestDaggerheartDecisionHandlers_CoversHandledCommands(t *testing.T) {
	handled := daggerheartdecider.NewDecider(commandTypesFromDefinitions()).DeciderHandledCommands()
	if len(handled) != len(daggerheartdecider.DecisionHandlers) {
		t.Fatalf("handled command count = %d, decision handler count = %d", len(handled), len(daggerheartdecider.DecisionHandlers))
	}
	handledSet := make(map[command.Type]struct{}, len(handled))
	for _, typ := range handled {
		handledSet[typ] = struct{}{}
		if _, ok := daggerheartdecider.DecisionHandlers[typ]; !ok {
			t.Fatalf("missing decision handler for command type %s", typ)
		}
	}
	for typ := range daggerheartdecider.DecisionHandlers {
		if _, ok := handledSet[typ]; !ok {
			t.Fatalf("decision handler registered for unsupported command type %s", typ)
		}
	}
}

func TestDecideRegisteredCommandsDoNotReturnEmptyDecision(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	for _, tc := range commandValidationCases() {
		actorType := tc.actorType
		if actorType == "" {
			actorType = command.ActorTypeSystem
		}
		decision := daggerheartdecider.Decider{}.Decide(daggerheartstate.SnapshotState{}, command.Command{
			CampaignID:    "camp-1",
			Type:          tc.typ,
			ActorType:     actorType,
			ActorID:       tc.actorID,
			SystemID:      SystemID,
			SystemVersion: SystemVersion,
			PayloadJSON:   []byte(tc.validPayload),
		}, func() time.Time { return now })
		if len(decision.Events) == 0 && len(decision.Rejections) == 0 {
			t.Fatalf("registered command %s returned empty decision", tc.typ)
		}
	}
}

func TestDecideGMFearSet_MissingAfterRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.gm_fear.set"),
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"reason":""}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(daggerheartstate.SnapshotState{}, cmd, time.Now)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeGMFearAfterRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeGMFearAfterRequired)
	}
}

func TestDecideGMFearSet_AfterOutOfRangeRejected(t *testing.T) {
	after := daggerheartstate.GMFearMax + 1
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.gm_fear.set"),
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"after":13}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(daggerheartstate.SnapshotState{}, cmd, time.Now)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeGMFearOutOfRange {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeGMFearOutOfRange)
	}
	if after <= daggerheartstate.GMFearMax {
		t.Fatalf("expected out of range after, got %d", after)
	}
}

func TestDecideGMFearSet_UnchangedStateRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.gm_fear.set"),
		ActorType:     command.ActorTypeGM,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"after":4}`),
	}

	// The aggregate applier extracts system-specific state before calling
	// RouteCommand, so the decider receives daggerheartstate.SnapshotState directly.
	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		GMFear:     4,
	}

	decision := daggerheartdecider.Decider{}.Decide(state, cmd, time.Now)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeGMFearUnchanged {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeGMFearUnchanged)
	}
}

func TestDecideCharacterStatePatch_EmitsCharacterStatePatched(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.character_state.patch"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","hp_before":6,"hp_after":5}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.character_state_patched") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.character_state_patched")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "character" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "character")
	}
	if evt.EntityID != "char-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "char-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload daggerheartpayload.CharacterStatePatchedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.HP == nil || *payload.HP != 5 {
		t.Fatalf("hp = %v, want %d", payload.HP, 5)
	}
}

func TestDecideCharacterStatePatch_UnchangedStateRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.character_state.patch"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","hp_before":5,"hp_after":6}`),
	}

	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[ids.CharacterID]daggerheartstate.CharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", HP: 6},
		},
	}

	decision := daggerheartdecider.Decider{}.Decide(state, cmd, time.Now)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeCharacterStatePatchNoMutation {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeCharacterStatePatchNoMutation)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
}

func TestDecideConditionChange_EmitsConditionChanged(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.condition.change"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","conditions_before":["vulnerable"],"conditions_after":["hidden"],"added":["hidden"],"removed":["vulnerable"]}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.condition_changed") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.condition_changed")
	}
	if evt.EntityType != "character" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "character")
	}
	if evt.EntityID != "char-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "char-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload daggerheartpayload.ConditionChangedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if len(payload.Conditions) != 1 || payload.Conditions[0].Code != rules.ConditionHidden {
		t.Fatalf("conditions after = %v, want [hidden]", payload.Conditions)
	}
	if len(payload.Added) != 1 || payload.Added[0].Code != rules.ConditionHidden {
		t.Fatalf("added = %v, want [hidden]", payload.Added)
	}
	if len(payload.Removed) != 1 || payload.Removed[0].Code != rules.ConditionVulnerable {
		t.Fatalf("removed = %v, want [vulnerable]", payload.Removed)
	}
}

func TestDecideConditionChange_UnchangedStateRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.condition.change"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","conditions_before":["vulnerable"],"conditions_after":["vulnerable"],"added":[],"removed":[]}`),
	}

	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[ids.CharacterID]daggerheartstate.CharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Conditions: []string{"vulnerable"}},
		},
	}

	decision := daggerheartdecider.Decider{}.Decide(state, cmd, time.Now)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeConditionChangeNoMutation {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeConditionChangeNoMutation)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
}

func TestDecideConditionChange_RemoveMissingConditionRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.condition.change"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","conditions_before":["hidden"],"conditions_after":["hidden","vulnerable"],"added":["vulnerable"],"removed":["restrained"]}`),
	}

	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[ids.CharacterID]daggerheartstate.CharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Conditions: []string{"hidden"}},
		},
	}

	decision := daggerheartdecider.Decider{}.Decide(state, cmd, time.Now)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeConditionChangeRemoveMissing {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeConditionChangeRemoveMissing)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
}

func TestDecideHopeSpend_EmitsCharacterStatePatched(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.hope.spend"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","amount":1,"before":2,"after":1,"source":"experience"}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.character_state_patched") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.character_state_patched")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "character" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "character")
	}
	if evt.EntityID != "char-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "char-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload daggerheartpayload.CharacterStatePatchedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.Hope == nil || *payload.Hope != 1 {
		t.Fatalf("hope = %v, want %d", payload.Hope, 1)
	}
}

func TestDecideHopeSpend_PropagatesSourceToEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.hope.spend"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","amount":1,"before":2,"after":1,"source":"experience"}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Source != "hope.spend" {
		t.Fatalf("source = %q, want %q", payload.Source, "hope.spend")
	}
}

func TestDecideStressSpend_PropagatesSourceToEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.stress.spend"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","amount":1,"before":3,"after":2,"source":"loadout_swap"}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload struct {
		Source string `json:"source"`
	}
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Source != "stress.spend" {
		t.Fatalf("source = %q, want %q", payload.Source, "stress.spend")
	}
}

func TestDecideStressSpend_EmitsCharacterStatePatched(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.stress.spend"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","amount":1,"before":3,"after":2,"source":"loadout_swap"}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.character_state_patched") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.character_state_patched")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "character" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "character")
	}
	if evt.EntityID != "char-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "char-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload daggerheartpayload.CharacterStatePatchedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.Stress == nil || *payload.Stress != 2 {
		t.Fatalf("stress = %v, want %d", payload.Stress, 2)
	}
}

func TestDecideLoadoutSwap_EmitsLoadoutSwapped(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.loadout.swap"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","card_id":"card-1","from":"vault","to":"active","recall_cost":1,"stress_before":3,"stress_after":2}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.loadout_swapped") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.loadout_swapped")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "character" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "character")
	}
	if evt.EntityID != "char-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "char-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload daggerheartpayload.LoadoutSwappedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.CardID != "card-1" {
		t.Fatalf("card id = %s, want %s", payload.CardID, "card-1")
	}
	if payload.From != "vault" {
		t.Fatalf("from = %s, want %s", payload.From, "vault")
	}
	if payload.To != "active" {
		t.Fatalf("to = %s, want %s", payload.To, "active")
	}
	if payload.RecallCost != 1 {
		t.Fatalf("recall cost = %d, want %d", payload.RecallCost, 1)
	}
	if payload.Stress == nil || *payload.Stress != 2 {
		t.Fatalf("stress = %v, want %d", payload.Stress, 2)
	}
}

func TestDecideRestTake_EmitsRestTaken(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.rest.take"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "session",
		EntityID:      "camp-1",
		PayloadJSON:   []byte(`{"rest_type":"short","interrupted":false,"gm_fear_before":1,"gm_fear_after":2,"short_rests_before":0,"short_rests_after":1,"refresh_rest":true,"refresh_long_rest":false}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.rest_taken") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.rest_taken")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "session" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "session")
	}
	if evt.EntityID != "camp-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "camp-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload struct {
		RestType        string `json:"rest_type"`
		Interrupted     bool   `json:"interrupted"`
		GMFearAfter     int    `json:"gm_fear_after"`
		ShortRestsAfter int    `json:"short_rests_after"`
		RefreshRest     bool   `json:"refresh_rest"`
		RefreshLongRest bool   `json:"refresh_long_rest"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.RestType != "short" {
		t.Fatalf("rest type = %s, want %s", payload.RestType, "short")
	}
	if payload.Interrupted {
		t.Fatal("expected interrupted to be false")
	}
	if payload.GMFearAfter != 2 {
		t.Fatalf("gm fear after = %d, want %d", payload.GMFearAfter, 2)
	}
	if payload.ShortRestsAfter != 1 {
		t.Fatalf("short rests after = %d, want %d", payload.ShortRestsAfter, 1)
	}
	if !payload.RefreshRest {
		t.Fatal("expected refresh_rest true")
	}
	if payload.RefreshLongRest {
		t.Fatal("expected refresh_long_rest false")
	}
}

func TestDecideRestTake_WithCampaignCountdown_EmitsCountdownAdvanced(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.rest.take"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "session",
		EntityID:      "camp-1",
		PayloadJSON:   []byte(`{"rest_type":"long","interrupted":false,"gm_fear_before":1,"gm_fear_after":2,"short_rests_before":1,"short_rests_after":0,"refresh_rest":true,"refresh_long_rest":true,"participants":["char-1"],"campaign_countdown_advances":[{"countdown_id":"cd-1","before_remaining":2,"after_remaining":1,"advanced_by":1,"status_before":"active","status_after":"active","reason":"long_rest"}]}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(decision.Events))
	}

	restEvent := decision.Events[0]
	if restEvent.Type != event.Type("sys.daggerheart.rest_taken") {
		t.Fatalf("first event type = %s, want %s", restEvent.Type, "sys.daggerheart.rest_taken")
	}
	countdownEvent := decision.Events[1]
	if countdownEvent.Type != event.Type("sys.daggerheart.campaign_countdown_advanced") {
		t.Fatalf("second event type = %s, want %s", countdownEvent.Type, "sys.daggerheart.campaign_countdown_advanced")
	}
	if countdownEvent.EntityType != "campaign_countdown" {
		t.Fatalf("countdown event entity type = %s, want %s", countdownEvent.EntityType, "campaign_countdown")
	}
	if countdownEvent.EntityID != "cd-1" {
		t.Fatalf("countdown event entity id = %s, want %s", countdownEvent.EntityID, "cd-1")
	}
}

func TestDecideRestTake_WithCampaignCountdown_BeforeMismatchRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.rest.take"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "session",
		EntityID:      "camp-1",
		PayloadJSON:   []byte(`{"rest_type":"long","interrupted":false,"gm_fear_before":1,"gm_fear_after":2,"short_rests_before":1,"short_rests_after":0,"refresh_rest":true,"refresh_long_rest":true,"participants":["char-1"],"campaign_countdown_advances":[{"countdown_id":"cd-1","before_remaining":2,"after_remaining":1,"advanced_by":1,"status_before":"active","status_after":"active","reason":"long_rest"}]}`),
	}

	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CampaignCountdownStates: map[ids.CountdownID]daggerheartstate.CampaignCountdownState{
			"cd-1": {CountdownID: "cd-1", StartingValue: 4, RemainingValue: 3, LoopBehavior: "none", Status: "active"},
		},
	}

	decision := daggerheartdecider.Decider{}.Decide(state, cmd, time.Now)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeCountdownBeforeMismatch {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeCountdownBeforeMismatch)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
}

func TestDecideRestTake_WithCampaignCountdown_UnchangedRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.rest.take"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "session",
		EntityID:      "camp-1",
		PayloadJSON:   []byte(`{"rest_type":"long","interrupted":false,"gm_fear_before":1,"gm_fear_after":2,"short_rests_before":1,"short_rests_after":0,"refresh_rest":true,"refresh_long_rest":true,"participants":["char-1"],"campaign_countdown_advances":[{"countdown_id":"cd-1","before_remaining":3,"after_remaining":3,"advanced_by":1,"status_before":"active","status_after":"active","reason":"long_rest"}]}`),
	}

	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CampaignCountdownStates: map[ids.CountdownID]daggerheartstate.CampaignCountdownState{
			"cd-1": {CountdownID: "cd-1", StartingValue: 4, RemainingValue: 3, LoopBehavior: "reset", Status: "active"},
		},
	}

	decision := daggerheartdecider.Decider{}.Decide(state, cmd, time.Now)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeCountdownUpdateNoMutation {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeCountdownUpdateNoMutation)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
}

func TestDecideDamageApply_EmitsDamageApplied(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.damage.apply"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","hp_before":6,"hp_after":3,"damage_type":"physical","marks":1}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.damage_applied") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.damage_applied")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "character" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "character")
	}
	if evt.EntityID != "char-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "char-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload daggerheartpayload.DamageAppliedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.DamageType != "physical" {
		t.Fatalf("damage type = %s, want %s", payload.DamageType, "physical")
	}
	if payload.Hp == nil || *payload.Hp != 3 {
		t.Fatalf("hp = %v, want %d", payload.Hp, 3)
	}
}

func TestDecideDamageApply_BeforeMismatchRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.damage.apply"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","hp_before":6,"hp_after":3,"armor_before":2,"armor_after":1,"damage_type":"physical","marks":1}`),
	}

	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[ids.CharacterID]daggerheartstate.CharacterState{
			"char-1": {
				CharacterID: "char-1",
				HP:          5,
				Armor:       2,
			},
		},
	}

	decision := daggerheartdecider.Decider{}.Decide(state, cmd, time.Now)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "DAMAGE_BEFORE_MISMATCH" {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, "DAMAGE_BEFORE_MISMATCH")
	}
}

func TestDecideDamageApply_RejectsMultipleArmorSlotsSpent(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.damage.apply"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","hp_before":10,"hp_after":8,"armor_before":2,"armor_after":0,"damage_type":"physical","armor_spent":2,"marks":2}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, time.Now)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "DAMAGE_ARMOR_SPEND_LIMIT" {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, "DAMAGE_ARMOR_SPEND_LIMIT")
	}
}

func TestDecideAdversaryDamageApply_EmitsAdversaryDamageApplied(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.adversary_damage.apply"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "adversary",
		EntityID:      "adv-1",
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","hp_before":8,"hp_after":3,"damage_type":"physical"}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.adversary_damage_applied") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.adversary_damage_applied")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "adversary" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "adversary")
	}
	if evt.EntityID != "adv-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "adv-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload daggerheartpayload.AdversaryDamageAppliedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.AdversaryID != "adv-1" {
		t.Fatalf("adversary id = %s, want %s", payload.AdversaryID, "adv-1")
	}
	if payload.DamageType != "physical" {
		t.Fatalf("damage type = %s, want %s", payload.DamageType, "physical")
	}
	if payload.Hp == nil || *payload.Hp != 3 {
		t.Fatalf("hp = %v, want %d", payload.Hp, 3)
	}
}

func TestDecideAdversaryDamageApply_BeforeMismatchRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.adversary_damage.apply"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "adversary",
		EntityID:      "adv-1",
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","hp_before":8,"hp_after":3,"armor_before":1,"armor_after":0,"damage_type":"physical"}`),
	}

	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		AdversaryStates: map[ids.AdversaryID]daggerheartstate.AdversaryState{
			"adv-1": {
				AdversaryID: "adv-1",
				HP:          7,
				Armor:       1,
			},
		},
	}

	decision := daggerheartdecider.Decider{}.Decide(state, cmd, time.Now)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "ADVERSARY_DAMAGE_BEFORE_MISMATCH" {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, "ADVERSARY_DAMAGE_BEFORE_MISMATCH")
	}
}

func TestDecideSceneCountdownCreate_EmitsSceneCountdownCreated(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		SessionID:     "sess-1",
		SceneID:       "scene-1",
		Type:          command.Type("sys.daggerheart.scene_countdown.create"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "scene_countdown",
		EntityID:      "cd-1",
		PayloadJSON:   []byte(`{"session_id":"sess-1","scene_id":"scene-1","countdown_id":"cd-1","name":"Doom","kind":"progress","current":0,"max":4,"direction":"increase","looping":true}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.scene_countdown_created") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.scene_countdown_created")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "scene_countdown" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "scene_countdown")
	}
	if evt.EntityID != "cd-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "cd-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}
}

func TestDecideSceneCountdownAdvance_EmitsSceneCountdownAdvanced(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		SessionID:     "sess-1",
		SceneID:       "scene-1",
		Type:          command.Type("sys.daggerheart.scene_countdown.advance"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "scene_countdown",
		EntityID:      "cd-1",
		PayloadJSON:   []byte(`{"countdown_id":"cd-1","before_remaining":2,"after_remaining":1,"advanced_by":1,"status_before":"active","status_after":"active","reason":"advance"}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.scene_countdown_advanced") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.scene_countdown_advanced")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "scene_countdown" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "scene_countdown")
	}
	if evt.EntityID != "cd-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "cd-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}
}

func TestDecideSceneCountdownAdvance_UnchangedStateRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.scene_countdown.advance"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "scene_countdown",
		EntityID:      "cd-1",
		PayloadJSON:   []byte(`{"countdown_id":"cd-1","before_remaining":3,"after_remaining":3,"advanced_by":1,"status_before":"active","status_after":"active"}`),
	}

	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		SceneCountdownStates: map[ids.CountdownID]daggerheartstate.SceneCountdownState{
			"cd-1": {CountdownID: "cd-1", StartingValue: 4, RemainingValue: 3, LoopBehavior: "reset", Status: "active"},
		},
	}

	decision := daggerheartdecider.Decider{}.Decide(state, cmd, time.Now)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeCountdownUpdateNoMutation {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeCountdownUpdateNoMutation)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
}

func TestDecideSceneCountdownAdvance_BeforeMismatchRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.scene_countdown.advance"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "scene_countdown",
		EntityID:      "cd-1",
		PayloadJSON:   []byte(`{"countdown_id":"cd-1","before_remaining":2,"after_remaining":1,"advanced_by":1,"status_before":"active","status_after":"active"}`),
	}

	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		SceneCountdownStates: map[ids.CountdownID]daggerheartstate.SceneCountdownState{
			"cd-1": {CountdownID: "cd-1", StartingValue: 4, RemainingValue: 3, LoopBehavior: "none", Status: "active"},
		},
	}

	decision := daggerheartdecider.Decider{}.Decide(state, cmd, time.Now)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeCountdownBeforeMismatch {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeCountdownBeforeMismatch)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
}

func TestDecideSceneCountdownDelete_EmitsSceneCountdownDeleted(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		SessionID:     "sess-1",
		SceneID:       "scene-1",
		Type:          command.Type("sys.daggerheart.scene_countdown.delete"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "scene_countdown",
		EntityID:      "cd-1",
		PayloadJSON:   []byte(`{"countdown_id":"cd-1","reason":"cleanup"}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.scene_countdown_deleted") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.scene_countdown_deleted")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "scene_countdown" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "scene_countdown")
	}
	if evt.EntityID != "cd-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "cd-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}
}

func TestDecideAdversaryConditionChange_EmitsAdversaryConditionChanged(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.adversary_condition.change"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "adversary",
		EntityID:      "adv-1",
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","conditions_before":["vulnerable"],"conditions_after":["hidden"],"added":["hidden"],"removed":["vulnerable"]}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.adversary_condition_changed") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.adversary_condition_changed")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "adversary" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "adversary")
	}
	if evt.EntityID != "adv-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "adv-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload daggerheartpayload.AdversaryConditionChangedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.AdversaryID != "adv-1" {
		t.Fatalf("adversary id = %s, want %s", payload.AdversaryID, "adv-1")
	}
	if len(payload.Conditions) != 1 || payload.Conditions[0].Code != rules.ConditionHidden {
		t.Fatalf("conditions_after = %v, want [hidden]", payload.Conditions)
	}
}

func TestDecideAdversaryConditionChange_UnchangedStateRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.adversary_condition.change"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "adversary",
		EntityID:      "adv-1",
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","conditions_before":["hidden"],"conditions_after":["hidden"],"added":[],"removed":[]}`),
	}

	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		AdversaryStates: map[ids.AdversaryID]daggerheartstate.AdversaryState{
			"adv-1": {AdversaryID: "adv-1", Conditions: []string{"hidden"}},
		},
	}

	decision := daggerheartdecider.Decider{}.Decide(state, cmd, time.Now)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeAdversaryConditionNoMutation {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeAdversaryConditionNoMutation)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
}

func TestDecideAdversaryConditionChange_RemoveMissingConditionRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.adversary_condition.change"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "adversary",
		EntityID:      "adv-1",
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","conditions_before":["hidden"],"conditions_after":["hidden","vulnerable"],"added":["vulnerable"],"removed":["restrained"]}`),
	}

	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		AdversaryStates: map[ids.AdversaryID]daggerheartstate.AdversaryState{
			"adv-1": {AdversaryID: "adv-1", Conditions: []string{"hidden"}},
		},
	}

	decision := daggerheartdecider.Decider{}.Decide(state, cmd, time.Now)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeAdversaryConditionRemoveMissing {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeAdversaryConditionRemoveMissing)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
}

func TestDecideAdversaryCreate_EmitsAdversaryCreated(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.adversary.create"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "adversary",
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","name":"  Goblin ","kind":"bruiser","session_id":"sess-1","notes":" note ","hp":6,"hp_max":6,"stress":2,"stress_max":2,"evasion":1,"major_threshold":2,"severe_threshold":3,"armor":1}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.adversary_created") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.adversary_created")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "adversary" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "adversary")
	}
	if evt.EntityID != "adv-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "adv-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload struct {
		AdversaryID string `json:"adversary_id"`
		Name        string `json:"name"`
		Notes       string `json:"notes"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.AdversaryID != "adv-1" {
		t.Fatalf("adversary id = %s, want %s", payload.AdversaryID, "adv-1")
	}
	if payload.Name != "Goblin" {
		t.Fatalf("name = %s, want %s", payload.Name, "Goblin")
	}
	if payload.Notes != "note" {
		t.Fatalf("notes = %s, want %s", payload.Notes, "note")
	}
}

func TestDecideAdversaryCreate_UnchangedStateRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.adversary.create"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "adversary",
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","name":"  Goblin ","kind":"bruiser","session_id":"sess-1","notes":" note ","hp":6,"hp_max":6,"stress":2,"stress_max":2,"evasion":1,"major_threshold":2,"severe_threshold":3,"armor":1}`),
	}

	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		AdversaryStates: map[ids.AdversaryID]daggerheartstate.AdversaryState{
			"adv-1": {
				AdversaryID: "adv-1", Name: "Goblin", Kind: "bruiser", SessionID: "sess-1", Notes: "note",
				HP: 6, HPMax: 6, Stress: 2, StressMax: 2, Evasion: 1, Major: 2, Severe: 3, Armor: 1,
			},
		},
	}

	decision := daggerheartdecider.Decider{}.Decide(state, cmd, time.Now)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeAdversaryCreateNoMutation {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeAdversaryCreateNoMutation)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
}

func TestDecideAdversaryUpdate_EmitsAdversaryUpdated(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.adversary.update"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "adversary",
		PayloadJSON:   []byte(`{"adversary_id":"adv-2","name":"  Ogre ","kind":"elite","session_id":"sess-2","notes":" updated ","hp":10,"hp_max":10,"stress":3,"stress_max":3,"evasion":2,"major_threshold":3,"severe_threshold":4,"armor":2}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.adversary_updated") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.adversary_updated")
	}
	if evt.EntityType != "adversary" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "adversary")
	}
	if evt.EntityID != "adv-2" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "adv-2")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload struct {
		AdversaryID string `json:"adversary_id"`
		Name        string `json:"name"`
		Notes       string `json:"notes"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.AdversaryID != "adv-2" {
		t.Fatalf("adversary id = %s, want %s", payload.AdversaryID, "adv-2")
	}
	if payload.Name != "Ogre" {
		t.Fatalf("name = %s, want %s", payload.Name, "Ogre")
	}
	if payload.Notes != "updated" {
		t.Fatalf("notes = %s, want %s", payload.Notes, "updated")
	}
}

func TestDecideAdversaryDelete_EmitsAdversaryDeleted(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.adversary.delete"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "adversary",
		PayloadJSON:   []byte(`{"adversary_id":"adv-3","reason":" removed "}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.adversary_deleted") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.adversary_deleted")
	}
	if evt.EntityType != "adversary" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "adversary")
	}
	if evt.EntityID != "adv-3" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "adv-3")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload struct {
		AdversaryID string `json:"adversary_id"`
		Reason      string `json:"reason"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.AdversaryID != "adv-3" {
		t.Fatalf("adversary id = %s, want %s", payload.AdversaryID, "adv-3")
	}
	if payload.Reason != "removed" {
		t.Fatalf("reason = %s, want %s", payload.Reason, "removed")
	}
}

func TestDecideMultiTargetDamageApply_EmitsMultipleDamageApplied(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          daggerheartdecider.CommandTypeMultiTargetDamageApply,
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON: []byte(`{"targets":[
			{"character_id":"char-1","hp_before":10,"hp_after":7,"damage_type":"physical","marks":1},
			{"character_id":"char-2","hp_before":8,"hp_after":5,"damage_type":"physical","marks":1},
			{"character_id":"char-3","hp_before":6,"hp_after":3,"damage_type":"physical","marks":1}
		]}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d: %v", len(decision.Rejections), decision.Rejections)
	}
	if len(decision.Events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(decision.Events))
	}

	targets := []string{"char-1", "char-2", "char-3"}
	for i, target := range targets {
		evt := decision.Events[i]
		if evt.Type != daggerheartpayload.EventTypeDamageApplied {
			t.Errorf("event[%d].Type = %s, want %s", i, evt.Type, daggerheartpayload.EventTypeDamageApplied)
		}
		if evt.EntityType != "character" {
			t.Errorf("event[%d].EntityType = %s, want character", i, evt.EntityType)
		}
		if evt.EntityID != target {
			t.Errorf("event[%d].EntityID = %s, want %s", i, evt.EntityID, target)
		}

		var payload struct {
			CharacterID string `json:"character_id"`
			DamageType  string `json:"damage_type"`
		}
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			t.Fatalf("event[%d] unmarshal: %v", i, err)
		}
		if payload.CharacterID != target {
			t.Errorf("event[%d].CharacterID = %s, want %s", i, payload.CharacterID, target)
		}
	}
}

func TestDecideMultiTargetDamageApply_RejectsEmptyTargets(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          daggerheartdecider.CommandTypeMultiTargetDamageApply,
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"targets":[]}`),
	}

	decision := daggerheartdecider.Decider{}.Decide(nil, cmd, time.Now)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "MULTI_TARGET_NO_TARGETS" {
		t.Fatalf("rejection code = %s, want MULTI_TARGET_NO_TARGETS", decision.Rejections[0].Code)
	}
}

func TestDecideMultiTargetDamageApply_BeforeMismatchRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          daggerheartdecider.CommandTypeMultiTargetDamageApply,
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON: []byte(`{"targets":[
			{"character_id":"char-1","hp_before":10,"hp_after":7,"damage_type":"physical","marks":1},
			{"character_id":"char-2","hp_before":8,"hp_after":5,"damage_type":"physical","marks":1}
		]}`),
	}

	state := daggerheartstate.SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[ids.CharacterID]daggerheartstate.CharacterState{
			"char-1": {CharacterID: "char-1", HP: 10, Armor: 0},
			"char-2": {CharacterID: "char-2", HP: 999, Armor: 0}, // mismatch
		},
	}

	decision := daggerheartdecider.Decider{}.Decide(state, cmd, time.Now)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeDamageBeforeMismatch {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeDamageBeforeMismatch)
	}
}
