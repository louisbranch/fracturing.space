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
		Type:          command.Type("action.gm_fear.set"),
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
	if evt.Type != event.Type("action.gm_fear_changed") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.gm_fear_changed")
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

func TestDecideGMFearSet_MissingAfterRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.gm_fear.set"),
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
		Type:          command.Type("action.gm_fear.set"),
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

func TestDecideCharacterStatePatch_EmitsCharacterStatePatched(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.character_state.patch"),
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
	if evt.Type != event.Type("action.character_state_patched") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.character_state_patched")
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

func TestDecideConditionChange_EmitsConditionChanged(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.condition.change"),
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
	if evt.Type != event.Type("action.condition_changed") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.condition_changed")
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

func TestDecideHopeSpend_EmitsHopeSpent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.hope.spend"),
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
	if evt.Type != event.Type("action.hope_spent") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.hope_spent")
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
		Amount      int    `json:"amount"`
		Before      int    `json:"before"`
		After       int    `json:"after"`
		Source      string `json:"source"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.Amount != 1 {
		t.Fatalf("amount = %d, want %d", payload.Amount, 1)
	}
	if payload.Before != 2 {
		t.Fatalf("before = %d, want %d", payload.Before, 2)
	}
	if payload.After != 1 {
		t.Fatalf("after = %d, want %d", payload.After, 1)
	}
	if payload.Source != "experience" {
		t.Fatalf("source = %s, want %s", payload.Source, "experience")
	}
}

func TestDecideStressSpend_EmitsStressSpent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.stress.spend"),
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
	if evt.Type != event.Type("action.stress_spent") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.stress_spent")
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
		Amount      int    `json:"amount"`
		Before      int    `json:"before"`
		After       int    `json:"after"`
		Source      string `json:"source"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.Amount != 1 {
		t.Fatalf("amount = %d, want %d", payload.Amount, 1)
	}
	if payload.Before != 3 {
		t.Fatalf("before = %d, want %d", payload.Before, 3)
	}
	if payload.After != 2 {
		t.Fatalf("after = %d, want %d", payload.After, 2)
	}
	if payload.Source != "loadout_swap" {
		t.Fatalf("source = %s, want %s", payload.Source, "loadout_swap")
	}
}

func TestDecideLoadoutSwap_EmitsLoadoutSwapped(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.loadout.swap"),
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
	if evt.Type != event.Type("action.loadout_swapped") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.loadout_swapped")
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
		Type:          command.Type("action.rest.take"),
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
	if evt.Type != event.Type("action.rest_taken") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.rest_taken")
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

func TestDecideDamageApply_EmitsDamageApplied(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.damage.apply"),
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
	if evt.Type != event.Type("action.damage_applied") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.damage_applied")
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

func TestDecideAdversaryDamageApply_EmitsAdversaryDamageApplied(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.adversary_damage.apply"),
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
	if evt.Type != event.Type("action.adversary_damage_applied") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.adversary_damage_applied")
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

func TestDecideAttackResolve_EmitsAttackResolved(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.attack.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "attack",
		EntityID:      "req-attack-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","roll_seq":4,"targets":["char-2"],"outcome":"success","success":true,"crit":false}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("action.attack_resolved") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.attack_resolved")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "attack" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "attack")
	}
	if evt.EntityID != "req-attack-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "req-attack-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload struct {
		CharacterID string `json:"character_id"`
		RollSeq     uint64 `json:"roll_seq"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.RollSeq != 4 {
		t.Fatalf("roll seq = %d, want %d", payload.RollSeq, 4)
	}
}

func TestDecideReactionResolve_EmitsReactionResolved(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.reaction.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "reaction",
		EntityID:      "req-reaction-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","roll_seq":5,"outcome":"success","success":true,"crit":false,"crit_negates_effects":false,"effects_negated":false}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("action.reaction_resolved") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.reaction_resolved")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "reaction" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "reaction")
	}
	if evt.EntityID != "req-reaction-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "req-reaction-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload struct {
		CharacterID string `json:"character_id"`
		RollSeq     uint64 `json:"roll_seq"`
		Outcome     string `json:"outcome"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.RollSeq != 5 {
		t.Fatalf("roll seq = %d, want %d", payload.RollSeq, 5)
	}
	if payload.Outcome != "success" {
		t.Fatalf("outcome = %s, want %s", payload.Outcome, "success")
	}
}

func TestDecideAdversaryAttackResolve_EmitsAdversaryAttackResolved(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.adversary_attack.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "attack",
		EntityID:      "req-adv-attack-1",
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","roll_seq":6,"targets":["char-1"],"roll":14,"modifier":2,"total":16,"difficulty":10,"success":true,"crit":false}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("action.adversary_attack_resolved") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.adversary_attack_resolved")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "attack" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "attack")
	}
	if evt.EntityID != "req-adv-attack-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "req-adv-attack-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload struct {
		AdversaryID string   `json:"adversary_id"`
		RollSeq     uint64   `json:"roll_seq"`
		Targets     []string `json:"targets"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.AdversaryID != "adv-1" {
		t.Fatalf("adversary id = %s, want %s", payload.AdversaryID, "adv-1")
	}
	if payload.RollSeq != 6 {
		t.Fatalf("roll seq = %d, want %d", payload.RollSeq, 6)
	}
	if len(payload.Targets) != 1 || payload.Targets[0] != "char-1" {
		t.Fatalf("targets = %v, want %v", payload.Targets, []string{"char-1"})
	}
}

func TestDecideDamageRollResolve_EmitsDamageRollResolved(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.damage_roll.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "roll",
		EntityID:      "req-damage-roll-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","roll_seq":7}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("action.damage_roll_resolved") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.damage_roll_resolved")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "roll" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "roll")
	}
	if evt.EntityID != "req-damage-roll-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "req-damage-roll-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload struct {
		CharacterID string `json:"character_id"`
		RollSeq     uint64 `json:"roll_seq"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.RollSeq != 7 {
		t.Fatalf("roll seq = %d, want %d", payload.RollSeq, 7)
	}
}

func TestDecideGroupActionResolve_EmitsGroupActionResolved(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.group_action.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "group_action",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"leader_character_id":"char-1","leader_roll_seq":1,"support_successes":1,"support_failures":0,"support_modifier":1}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("action.group_action_resolved") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.group_action_resolved")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "group_action" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "group_action")
	}
	if evt.EntityID != "char-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "char-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}
}

func TestDecideTagTeamResolve_EmitsTagTeamResolved(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.tag_team.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "tag_team",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"first_character_id":"char-1","first_roll_seq":1,"second_character_id":"char-2","second_roll_seq":2,"selected_character_id":"char-1","selected_roll_seq":1}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("action.tag_team_resolved") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.tag_team_resolved")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "tag_team" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "tag_team")
	}
	if evt.EntityID != "char-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "char-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}
}

func TestDecideCountdownCreate_EmitsCountdownCreated(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.countdown.create"),
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
	if evt.Type != event.Type("action.countdown_created") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.countdown_created")
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
		Type:          command.Type("action.countdown.update"),
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
	if evt.Type != event.Type("action.countdown_updated") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.countdown_updated")
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

func TestDecideCountdownDelete_EmitsCountdownDeleted(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.countdown.delete"),
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
	if evt.Type != event.Type("action.countdown_deleted") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.countdown_deleted")
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

func TestDecideAdversaryActionResolve_EmitsAdversaryActionResolved(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.adversary_action.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "adversary",
		EntityID:      "adv-1",
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","roll_seq":1,"difficulty":10,"dramatic":false,"auto_success":true,"success":true}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("action.adversary_action_resolved") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.adversary_action_resolved")
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
}

func TestDecideAdversaryRollResolve_EmitsAdversaryRollResolved(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.adversary_roll.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "adversary",
		EntityID:      "adv-1",
		PayloadJSON:   []byte(`{"adversary_id":"adv-1","roll_seq":1,"rolls":[12,18],"roll":18,"modifier":2,"total":20,"advantage":1,"disadvantage":0}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("action.adversary_roll_resolved") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.adversary_roll_resolved")
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
}

func TestDecideBlazeOfGloryResolve_EmitsBlazeOfGloryResolved(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.blaze_of_glory.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","life_state_before":"blaze_of_glory","life_state_after":"dead"}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("action.blaze_of_glory_resolved") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.blaze_of_glory_resolved")
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
}

func TestDecideDowntimeMoveApply_EmitsDowntimeMoveApplied(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.downtime_move.apply"),
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
	if evt.Type != event.Type("action.downtime_move_applied") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.downtime_move_applied")
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

func TestDecideDeathMoveResolve_EmitsDeathMoveResolved(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.death_move.resolve"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "character",
		EntityID:      "char-1",
		PayloadJSON:   []byte(`{"character_id":"char-1","move":"avoid_death","life_state_after":"alive","hope_die":2,"fear_die":1}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("action.death_move_resolved") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.death_move_resolved")
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
		CharacterID    string `json:"character_id"`
		Move           string `json:"move"`
		LifeStateAfter string `json:"life_state_after"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
	if payload.Move != "avoid_death" {
		t.Fatalf("move = %s, want %s", payload.Move, "avoid_death")
	}
	if payload.LifeStateAfter != "alive" {
		t.Fatalf("life_state_after = %s, want %s", payload.LifeStateAfter, "alive")
	}
}

func TestDecideGMMoveApply_EmitsGMMoveApplied(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.gm_move.apply"),
		ActorType:     command.ActorTypeSystem,
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		EntityType:    "gm_move",
		EntityID:      "camp-1",
		PayloadJSON:   []byte(`{"move":"change_environment","fear_spent":1,"severity":"soft","source":"manual"}`),
	}

	decision := Decider{}.Decide(nil, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("action.gm_move_applied") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.gm_move_applied")
	}
	if evt.SystemID != SystemID {
		t.Fatalf("system id = %s, want %s", evt.SystemID, SystemID)
	}
	if evt.SystemVersion != SystemVersion {
		t.Fatalf("system version = %s, want %s", evt.SystemVersion, SystemVersion)
	}
	if evt.EntityType != "gm_move" {
		t.Fatalf("entity type = %s, want %s", evt.EntityType, "gm_move")
	}
	if evt.EntityID != "camp-1" {
		t.Fatalf("entity id = %s, want %s", evt.EntityID, "camp-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload struct {
		Move      string `json:"move"`
		FearSpent int    `json:"fear_spent"`
		Severity  string `json:"severity"`
	}
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Move != "change_environment" {
		t.Fatalf("move = %s, want %s", payload.Move, "change_environment")
	}
	if payload.FearSpent != 1 {
		t.Fatalf("fear_spent = %d, want %d", payload.FearSpent, 1)
	}
	if payload.Severity != "soft" {
		t.Fatalf("severity = %s, want %s", payload.Severity, "soft")
	}
}

func TestDecideAdversaryConditionChange_EmitsAdversaryConditionChanged(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.adversary_condition.change"),
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
	if evt.Type != event.Type("action.adversary_condition_changed") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.adversary_condition_changed")
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

func TestDecideAdversaryCreate_EmitsAdversaryCreated(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.adversary.create"),
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
	if evt.Type != event.Type("action.adversary_created") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.adversary_created")
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

func TestDecideAdversaryUpdate_EmitsAdversaryUpdated(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:    "camp-1",
		Type:          command.Type("action.adversary.update"),
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
	if evt.Type != event.Type("action.adversary_updated") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.adversary_updated")
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
		Type:          command.Type("action.adversary.delete"),
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
	if evt.Type != event.Type("action.adversary_deleted") {
		t.Fatalf("event type = %s, want %s", evt.Type, "action.adversary_deleted")
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
