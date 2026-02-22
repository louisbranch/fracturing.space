package daggerheart

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
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

	decision := Decider{}.Decide(SnapshotState{GMFear: 2}, cmd, func() time.Time { return now })
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

	var payload GMFearChangedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Before != 2 {
		t.Fatalf("payload before = %d, want %d", payload.Before, 2)
	}
	if payload.After != 4 {
		t.Fatalf("payload after = %d, want %d", payload.After, 4)
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

	decision := Decider{}.Decide(SnapshotState{GMFear: 2}, cmd, func() time.Time { return now })
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCommandTypeUnsupported {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCommandTypeUnsupported)
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

	decision := Decider{}.Decide(SnapshotState{}, cmd, func() time.Time { return now })
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCommandTypeUnsupported {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCommandTypeUnsupported)
	}
}

func TestDecideRegisteredCommandsDoNotReturnEmptyDecision(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	for _, tc := range commandValidationCases() {
		actorType := tc.actorType
		if actorType == "" {
			actorType = command.ActorTypeSystem
		}
		decision := Decider{}.Decide(SnapshotState{}, command.Command{
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

	decision := Decider{}.Decide(SnapshotState{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeGMFearAfterRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeGMFearAfterRequired)
	}
}

func TestDecideGMFearSet_AfterOutOfRangeRejected(t *testing.T) {
	after := GMFearMax + 1
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.gm_fear.set"),
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   []byte(`{"after":13}`),
	}

	decision := Decider{}.Decide(SnapshotState{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeGMFearOutOfRange {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeGMFearOutOfRange)
	}
	if after <= GMFearMax {
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
	// RouteCommand, so the decider receives SnapshotState directly.
	state := SnapshotState{
		CampaignID: "camp-1",
		GMFear:     4,
	}

	decision := Decider{}.Decide(state, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeGMFearUnchanged {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeGMFearUnchanged)
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

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
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

	var payload struct {
		CharacterID string `json:"character_id"`
		HPBefore    *int   `json:"hp_before"`
		HPAfter     *int   `json:"hp_after"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.HPBefore == nil || *payload.HPBefore != 6 {
		t.Fatalf("hp before = %v, want %d", payload.HPBefore, 6)
	}
	if payload.HPAfter == nil || *payload.HPAfter != 5 {
		t.Fatalf("hp after = %v, want %d", payload.HPAfter, 5)
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

	state := SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[string]CharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", HP: 6},
		},
	}

	decision := Decider{}.Decide(state, cmd, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCharacterStatePatchNoMutation {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCharacterStatePatchNoMutation)
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
		PayloadJSON:   []byte(`{"character_id":"char-1","conditions_before":["vulnerable"],"conditions_after":["shaken"],"added":["shaken"],"removed":["vulnerable"]}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
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

	var payload struct {
		CharacterID string   `json:"character_id"`
		After       []string `json:"conditions_after"`
		Added       []string `json:"added"`
		Removed     []string `json:"removed"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if len(payload.After) != 1 || payload.After[0] != "shaken" {
		t.Fatalf("conditions after = %v, want [shaken]", payload.After)
	}
	if len(payload.Added) != 1 || payload.Added[0] != "shaken" {
		t.Fatalf("added = %v, want [shaken]", payload.Added)
	}
	if len(payload.Removed) != 1 || payload.Removed[0] != "vulnerable" {
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

	state := SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[string]CharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Conditions: []string{"vulnerable"}},
		},
	}

	decision := Decider{}.Decide(state, cmd, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeConditionChangeNoMutation {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeConditionChangeNoMutation)
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

	state := SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[string]CharacterState{
			"char-1": {CampaignID: "camp-1", CharacterID: "char-1", Conditions: []string{"hidden"}},
		},
	}

	decision := Decider{}.Decide(state, cmd, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeConditionChangeRemoveMissing {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeConditionChangeRemoveMissing)
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

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
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

	var payload struct {
		CharacterID string `json:"character_id"`
		HopeBefore  *int   `json:"hope_before"`
		HopeAfter   *int   `json:"hope_after"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.HopeBefore == nil || *payload.HopeBefore != 2 {
		t.Fatalf("hope_before = %v, want %d", payload.HopeBefore, 2)
	}
	if payload.HopeAfter == nil || *payload.HopeAfter != 1 {
		t.Fatalf("hope_after = %v, want %d", payload.HopeAfter, 1)
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

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
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

	var payload struct {
		CharacterID  string `json:"character_id"`
		StressBefore *int   `json:"stress_before"`
		StressAfter  *int   `json:"stress_after"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.StressBefore == nil || *payload.StressBefore != 3 {
		t.Fatalf("stress_before = %v, want %d", payload.StressBefore, 3)
	}
	if payload.StressAfter == nil || *payload.StressAfter != 2 {
		t.Fatalf("stress_after = %v, want %d", payload.StressAfter, 2)
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

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
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

	var payload struct {
		CharacterID  string `json:"character_id"`
		CardID       string `json:"card_id"`
		From         string `json:"from"`
		To           string `json:"to"`
		RecallCost   int    `json:"recall_cost"`
		StressBefore *int   `json:"stress_before"`
		StressAfter  *int   `json:"stress_after"`
	}
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
	if payload.StressBefore == nil || *payload.StressBefore != 3 {
		t.Fatalf("stress before = %v, want %d", payload.StressBefore, 3)
	}
	if payload.StressAfter == nil || *payload.StressAfter != 2 {
		t.Fatalf("stress after = %v, want %d", payload.StressAfter, 2)
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

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
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
		RestType         string `json:"rest_type"`
		Interrupted      bool   `json:"interrupted"`
		GMFearBefore     int    `json:"gm_fear_before"`
		GMFearAfter      int    `json:"gm_fear_after"`
		ShortRestsBefore int    `json:"short_rests_before"`
		ShortRestsAfter  int    `json:"short_rests_after"`
		RefreshRest      bool   `json:"refresh_rest"`
		RefreshLongRest  bool   `json:"refresh_long_rest"`
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
	if payload.GMFearBefore != 1 {
		t.Fatalf("gm fear before = %d, want %d", payload.GMFearBefore, 1)
	}
	if payload.GMFearAfter != 2 {
		t.Fatalf("gm fear after = %d, want %d", payload.GMFearAfter, 2)
	}
	if payload.ShortRestsBefore != 0 {
		t.Fatalf("short rests before = %d, want %d", payload.ShortRestsBefore, 0)
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

func TestDecideRestTake_WithLongTermCountdown_EmitsCountdownUpdated(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.rest.take"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "session",
		EntityID:      "camp-1",
		PayloadJSON:   []byte(`{"rest_type":"long","interrupted":false,"gm_fear_before":1,"gm_fear_after":2,"short_rests_before":1,"short_rests_after":0,"refresh_rest":true,"refresh_long_rest":true,"long_term_countdown":{"countdown_id":"cd-1","before":2,"after":3,"delta":1,"looped":false,"reason":"long_rest"}}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
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
	if countdownEvent.Type != event.Type("sys.daggerheart.countdown_updated") {
		t.Fatalf("second event type = %s, want %s", countdownEvent.Type, "sys.daggerheart.countdown_updated")
	}
	if countdownEvent.EntityType != "countdown" {
		t.Fatalf("countdown event entity type = %s, want %s", countdownEvent.EntityType, "countdown")
	}
	if countdownEvent.EntityID != "cd-1" {
		t.Fatalf("countdown event entity id = %s, want %s", countdownEvent.EntityID, "cd-1")
	}
}

func TestDecideRestTake_WithLongTermCountdown_BeforeMismatchRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.rest.take"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "session",
		EntityID:      "camp-1",
		PayloadJSON:   []byte(`{"rest_type":"long","interrupted":false,"gm_fear_before":1,"gm_fear_after":2,"short_rests_before":1,"short_rests_after":0,"refresh_rest":true,"refresh_long_rest":true,"long_term_countdown":{"countdown_id":"cd-1","before":2,"after":3,"delta":1,"looped":false,"reason":"long_rest"}}`),
	}

	state := SnapshotState{
		CampaignID: "camp-1",
		CountdownStates: map[string]CountdownState{
			"cd-1": {CountdownID: "cd-1", Current: 1, Max: 4, Direction: "increase", Looping: false},
		},
	}

	decision := Decider{}.Decide(state, cmd, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCountdownBeforeMismatch {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCountdownBeforeMismatch)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
}

func TestDecideRestTake_WithLongTermCountdown_UnchangedRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.rest.take"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "session",
		EntityID:      "camp-1",
		PayloadJSON:   []byte(`{"rest_type":"long","interrupted":false,"gm_fear_before":1,"gm_fear_after":2,"short_rests_before":1,"short_rests_after":0,"refresh_rest":true,"refresh_long_rest":true,"long_term_countdown":{"countdown_id":"cd-1","before":3,"after":3,"delta":1,"looped":false,"reason":"long_rest"}}`),
	}

	state := SnapshotState{
		CampaignID: "camp-1",
		CountdownStates: map[string]CountdownState{
			"cd-1": {CountdownID: "cd-1", Current: 3, Max: 4, Direction: "increase", Looping: true},
		},
	}

	decision := Decider{}.Decide(state, cmd, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCountdownUpdateNoMutation {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCountdownUpdateNoMutation)
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

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
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

	var payload struct {
		CharacterID string `json:"character_id"`
		DamageType  string `json:"damage_type"`
		HPAfter     *int   `json:"hp_after"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.DamageType != "physical" {
		t.Fatalf("damage type = %s, want %s", payload.DamageType, "physical")
	}
	if payload.HPAfter == nil || *payload.HPAfter != 3 {
		t.Fatalf("hp after = %v, want %d", payload.HPAfter, 3)
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

	state := SnapshotState{
		CampaignID: "camp-1",
		CharacterStates: map[string]CharacterState{
			"char-1": {
				CharacterID: "char-1",
				HP:          5,
				Armor:       2,
			},
		},
	}

	decision := Decider{}.Decide(state, cmd, nil)
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

	decision := Decider{}.Decide(nil, cmd, nil)
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

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
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

	var payload struct {
		AdversaryID string `json:"adversary_id"`
		DamageType  string `json:"damage_type"`
		HPAfter     *int   `json:"hp_after"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.AdversaryID != "adv-1" {
		t.Fatalf("adversary id = %s, want %s", payload.AdversaryID, "adv-1")
	}
	if payload.DamageType != "physical" {
		t.Fatalf("damage type = %s, want %s", payload.DamageType, "physical")
	}
	if payload.HPAfter == nil || *payload.HPAfter != 3 {
		t.Fatalf("hp after = %v, want %d", payload.HPAfter, 3)
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

	state := SnapshotState{
		CampaignID: "camp-1",
		AdversaryStates: map[string]AdversaryState{
			"adv-1": {
				AdversaryID: "adv-1",
				HP:          7,
				Armor:       1,
			},
		},
	}

	decision := Decider{}.Decide(state, cmd, nil)
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

func TestDecideCountdownCreate_EmitsCountdownCreated(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.countdown.create"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "countdown",
		EntityID:      "cd-1",
		PayloadJSON:   []byte(`{"countdown_id":"cd-1","name":"Doom","kind":"progress","current":0,"max":4,"direction":"increase","looping":true}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.countdown_created") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.countdown_created")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "countdown" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "countdown")
	}
	if evt.EntityID != "cd-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "cd-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}
}

func TestDecideCountdownUpdate_EmitsCountdownUpdated(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.countdown.update"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "countdown",
		EntityID:      "cd-1",
		PayloadJSON:   []byte(`{"countdown_id":"cd-1","before":2,"after":3,"delta":1,"looped":false,"reason":"advance"}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.countdown_updated") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.countdown_updated")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "countdown" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "countdown")
	}
	if evt.EntityID != "cd-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "cd-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}
}

func TestDecideCountdownUpdate_UnchangedStateRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.countdown.update"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "countdown",
		EntityID:      "cd-1",
		PayloadJSON:   []byte(`{"countdown_id":"cd-1","before":3,"after":3,"delta":1,"looped":false}`),
	}

	state := SnapshotState{
		CampaignID: "camp-1",
		CountdownStates: map[string]CountdownState{
			"cd-1": {CountdownID: "cd-1", Current: 3, Max: 4, Direction: "increase", Looping: true},
		},
	}

	decision := Decider{}.Decide(state, cmd, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCountdownUpdateNoMutation {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCountdownUpdateNoMutation)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
}

func TestDecideCountdownUpdate_BeforeMismatchRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.countdown.update"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "countdown",
		EntityID:      "cd-1",
		PayloadJSON:   []byte(`{"countdown_id":"cd-1","before":2,"after":3,"delta":1,"looped":false}`),
	}

	state := SnapshotState{
		CampaignID: "camp-1",
		CountdownStates: map[string]CountdownState{
			"cd-1": {CountdownID: "cd-1", Current: 1, Max: 4, Direction: "increase", Looping: false},
		},
	}

	decision := Decider{}.Decide(state, cmd, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCountdownBeforeMismatch {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCountdownBeforeMismatch)
	}
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
}

func TestDecideCountdownDelete_EmitsCountdownDeleted(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.countdown.delete"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "countdown",
		EntityID:      "cd-1",
		PayloadJSON:   []byte(`{"countdown_id":"cd-1","reason":"cleanup"}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.countdown_deleted") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.countdown_deleted")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "countdown" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "countdown")
	}
	if evt.EntityID != "cd-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "cd-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}
}

func TestDecideDowntimeMoveApply_EmitsDowntimeMoveApplied(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("sys.daggerheart.downtime_move.apply"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","move":"clear_all_stress","stress_before":3,"stress_after":0}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("sys.daggerheart.downtime_move_applied") {
		t.Fatalf("event type = %s, want %s", evt.Type, "sys.daggerheart.downtime_move_applied")
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

	var payload struct {
		CharacterID  string `json:"character_id"`
		Move         string `json:"move"`
		StressBefore *int   `json:"stress_before"`
		StressAfter  *int   `json:"stress_after"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.Move != "clear_all_stress" {
		t.Fatalf("move = %s, want %s", payload.Move, "clear_all_stress")
	}
	if payload.StressBefore == nil || *payload.StressBefore != 3 {
		t.Fatalf("stress before = %v, want %d", payload.StressBefore, 3)
	}
	if payload.StressAfter == nil || *payload.StressAfter != 0 {
		t.Fatalf("stress after = %v, want %d", payload.StressAfter, 0)
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

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
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

	var payload struct {
		AdversaryID     string   `json:"adversary_id"`
		ConditionsAfter []string `json:"conditions_after"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.AdversaryID != "adv-1" {
		t.Fatalf("adversary id = %s, want %s", payload.AdversaryID, "adv-1")
	}
	if len(payload.ConditionsAfter) != 1 || payload.ConditionsAfter[0] != "hidden" {
		t.Fatalf("conditions_after = %v, want [hidden]", payload.ConditionsAfter)
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

	state := SnapshotState{
		CampaignID: "camp-1",
		AdversaryStates: map[string]AdversaryState{
			"adv-1": {AdversaryID: "adv-1", Conditions: []string{"hidden"}},
		},
	}

	decision := Decider{}.Decide(state, cmd, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeAdversaryConditionNoMutation {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeAdversaryConditionNoMutation)
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

	state := SnapshotState{
		CampaignID: "camp-1",
		AdversaryStates: map[string]AdversaryState{
			"adv-1": {AdversaryID: "adv-1", Conditions: []string{"hidden"}},
		},
	}

	decision := Decider{}.Decide(state, cmd, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeAdversaryConditionRemoveMissing {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeAdversaryConditionRemoveMissing)
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

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
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

	state := SnapshotState{
		CampaignID: "camp-1",
		AdversaryStates: map[string]AdversaryState{
			"adv-1": {
				AdversaryID: "adv-1", Name: "Goblin", Kind: "bruiser", SessionID: "sess-1", Notes: "note",
				HP: 6, HPMax: 6, Stress: 2, StressMax: 2, Evasion: 1, Major: 2, Severe: 3, Armor: 1,
			},
		},
	}

	decision := Decider{}.Decide(state, cmd, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeAdversaryCreateNoMutation {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeAdversaryCreateNoMutation)
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

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
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

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
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
