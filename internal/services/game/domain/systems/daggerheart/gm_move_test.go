package daggerheart

import (
	"encoding/json"
	"testing"
	"time"

	daggerheartvalidator "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/validator"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"

	daggerheartdecider "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/internal/decider"

	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	domainmodule "github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
)

func TestSessionStartBootstrapSeedsFearFromPCCount(t *testing.T) {
	module := NewModule()
	now := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)

	emitter, err := module.BindSessionStartBootstrap("camp-1", map[domainmodule.Key]any{
		{ID: SystemID, Version: SystemVersion}: daggerheartstate.SnapshotState{GMFear: daggerheartstate.GMFearDefault},
	})
	if err != nil {
		t.Fatalf("BindSessionStartBootstrap returned error: %v", err)
	}

	events, err := emitter.EmitSessionStartBootstrap(map[ids.CharacterID]character.State{
		"pc-1":  {CharacterID: "pc-1", Created: true, Kind: character.KindPC},
		"pc-2":  {CharacterID: "pc-2", Created: true, Kind: character.KindPC},
		"npc-1": {CharacterID: "npc-1", Created: true, Kind: character.KindNPC},
	},
		command.Command{
			CampaignID:    "camp-1",
			SessionID:     "sess-1",
			SceneID:       "scene-1",
			RequestID:     "req-1",
			InvocationID:  "inv-1",
			ActorType:     command.ActorTypeGM,
			ActorID:       "gm-1",
			CorrelationID: "corr-1",
			CausationID:   "cause-1",
		}, now)
	if err != nil {
		t.Fatalf("EmitSessionStartBootstrap returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Type != daggerheartpayload.EventTypeGMFearChanged {
		t.Fatalf("event type = %s, want %s", events[0].Type, daggerheartpayload.EventTypeGMFearChanged)
	}
	if events[0].Timestamp != now.UTC() {
		t.Fatalf("timestamp = %s, want %s", events[0].Timestamp, now.UTC())
	}
	var payload daggerheartpayload.GMFearChangedPayload
	if err := json.Unmarshal(events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.Value != 2 {
		t.Fatalf("payload value = %d, want 2", payload.Value)
	}
	if payload.Reason != "campaign_start" {
		t.Fatalf("payload reason = %q, want campaign_start", payload.Reason)
	}
}

func TestSessionStartBootstrapSkipsWhenFearAlreadySeeded(t *testing.T) {
	module := NewModule()

	emitter, err := module.BindSessionStartBootstrap("camp-1", map[domainmodule.Key]any{
		{ID: SystemID, Version: SystemVersion}: daggerheartstate.SnapshotState{GMFear: 3},
	})
	if err != nil {
		t.Fatalf("BindSessionStartBootstrap returned error: %v", err)
	}

	events, err := emitter.EmitSessionStartBootstrap(map[ids.CharacterID]character.State{
		"pc-1": {CharacterID: "pc-1", Created: true, Kind: character.KindPC},
	},
		command.Command{CampaignID: "camp-1"},
		time.Now())
	if err != nil {
		t.Fatalf("EmitSessionStartBootstrap returned error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("events = %d, want 0", len(events))
	}
}

func TestSessionStartBootstrapNoopsWithoutCreatedPCs(t *testing.T) {
	module := NewModule()

	emitter, err := module.BindSessionStartBootstrap("camp-1", map[domainmodule.Key]any{
		{ID: SystemID, Version: SystemVersion}: daggerheartstate.SnapshotState{GMFear: daggerheartstate.GMFearDefault},
	})
	if err != nil {
		t.Fatalf("BindSessionStartBootstrap returned error: %v", err)
	}

	events, err := emitter.EmitSessionStartBootstrap(map[ids.CharacterID]character.State{
		"npc-1": {CharacterID: "npc-1", Created: true, Kind: character.KindNPC},
		"pc-1":  {CharacterID: "pc-1", Created: false, Kind: character.KindPC},
		"pc-2":  {CharacterID: "pc-2", Created: true, Deleted: true, Kind: character.KindPC},
	},
		command.Command{CampaignID: "camp-1"},
		time.Now())
	if err != nil {
		t.Fatalf("EmitSessionStartBootstrap returned error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("events = %d, want 0", len(events))
	}
}

func TestSessionStartBootstrapRejectsInvalidSystemState(t *testing.T) {
	module := NewModule()

	_, err := module.BindSessionStartBootstrap("camp-1", map[domainmodule.Key]any{
		{ID: SystemID, Version: SystemVersion}: struct{}{},
	})
	if err == nil {
		t.Fatal("expected invalid system state error")
	}
}

func TestDeciderGMMoveApplyRejectsCustomWithoutDescription(t *testing.T) {
	payloadJSON := []byte(`{"target":{"type":"direct_move","kind":"interrupt_and_move","shape":"custom"},"fear_spent":1}`)
	decision := (daggerheartdecider.Decider{}).Decide(daggerheartstate.SnapshotState{GMFear: 2}, command.Command{
		CampaignID:    "camp-1",
		Type:          daggerheartdecider.CommandTypeGMMoveApply,
		SessionID:     "sess-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payloadJSON,
	}, func() time.Time { return time.Unix(0, 0).UTC() })

	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeGMMoveDescriptionRequired {
		t.Fatalf("rejection code = %q, want %q", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeGMMoveDescriptionRequired)
	}
}

func TestDeciderGMMoveApplyEmitsAuditAndFearEvents(t *testing.T) {
	now := time.Date(2026, 3, 13, 12, 30, 0, 0, time.UTC)
	payloadJSON := []byte(`{"target":{"type":"direct_move","kind":"additional_move","shape":"shift_environment"},"fear_spent":2}`)
	decision := (daggerheartdecider.Decider{}).Decide(daggerheartstate.SnapshotState{GMFear: 4}, command.Command{
		CampaignID:    "camp-1",
		Type:          daggerheartdecider.CommandTypeGMMoveApply,
		SessionID:     "sess-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payloadJSON,
	}, func() time.Time { return now })

	if len(decision.Rejections) != 0 {
		t.Fatalf("rejections = %d, want 0", len(decision.Rejections))
	}
	if len(decision.Events) != 2 {
		t.Fatalf("events = %d, want 2", len(decision.Events))
	}
	if decision.Events[0].Type != daggerheartpayload.EventTypeGMMoveApplied {
		t.Fatalf("first event type = %s, want %s", decision.Events[0].Type, daggerheartpayload.EventTypeGMMoveApplied)
	}
	if decision.Events[1].Type != daggerheartpayload.EventTypeGMFearChanged {
		t.Fatalf("second event type = %s, want %s", decision.Events[1].Type, daggerheartpayload.EventTypeGMFearChanged)
	}
	if decision.Events[0].EntityType != "session" || decision.Events[0].EntityID != "sess-1" {
		t.Fatalf("gm move entity = (%q,%q), want (session,sess-1)", decision.Events[0].EntityType, decision.Events[0].EntityID)
	}
	var movePayload daggerheartpayload.GMMoveAppliedPayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &movePayload); err != nil {
		t.Fatalf("decode gm move payload: %v", err)
	}
	if movePayload.Target.Type != rules.GMMoveTargetTypeDirectMove {
		t.Fatalf("gm move target type = %q, want %q", movePayload.Target.Type, rules.GMMoveTargetTypeDirectMove)
	}
	if movePayload.Target.Kind != rules.GMMoveKindAdditionalMove {
		t.Fatalf("gm move kind = %q, want %q", movePayload.Target.Kind, rules.GMMoveKindAdditionalMove)
	}
	if movePayload.Target.Shape != rules.GMMoveShapeShiftEnvironment {
		t.Fatalf("gm move shape = %q, want %q", movePayload.Target.Shape, rules.GMMoveShapeShiftEnvironment)
	}
	if movePayload.FearSpent != 2 {
		t.Fatalf("gm move fear_spent = %d, want 2", movePayload.FearSpent)
	}
	var fearPayload daggerheartpayload.GMFearChangedPayload
	if err := json.Unmarshal(decision.Events[1].PayloadJSON, &fearPayload); err != nil {
		t.Fatalf("decode gm fear payload: %v", err)
	}
	if fearPayload.Value != 2 {
		t.Fatalf("gm fear value = %d, want 2", fearPayload.Value)
	}
	if fearPayload.Reason != "gm_move" {
		t.Fatalf("gm fear reason = %q, want gm_move", fearPayload.Reason)
	}
}

func TestDeciderGMMoveApplyRejectsInsufficientFear(t *testing.T) {
	payloadJSON := []byte(`{"target":{"type":"direct_move","kind":"additional_move","shape":"shift_environment"},"fear_spent":3}`)
	decision := (daggerheartdecider.Decider{}).Decide(daggerheartstate.SnapshotState{GMFear: 2}, command.Command{
		CampaignID:    "camp-1",
		Type:          daggerheartdecider.CommandTypeGMMoveApply,
		SessionID:     "sess-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payloadJSON,
	}, func() time.Time { return time.Unix(0, 0).UTC() })

	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != daggerheartdecider.RejectionCodeGMMoveInsufficientFear {
		t.Fatalf("rejection code = %q, want %q", decision.Rejections[0].Code, daggerheartdecider.RejectionCodeGMMoveInsufficientFear)
	}
}

func TestGMMoveAppliedEventIsAuditOnly(t *testing.T) {
	registry := event.NewRegistry()
	if err := NewModule().RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}
	def, ok := registry.Definition(daggerheartpayload.EventTypeGMMoveApplied)
	if !ok {
		t.Fatal("expected gm_move_applied definition")
	}
	if def.Intent != event.IntentAuditOnly {
		t.Fatalf("intent = %s, want %s", def.Intent, event.IntentAuditOnly)
	}
}

func TestValidateGMMovePayloadsForTypedTargets(t *testing.T) {
	validCases := map[string]json.RawMessage{
		"direct_move":          json.RawMessage(`{"target":{"type":"direct_move","kind":"additional_move","shape":"shift_environment"},"fear_spent":1}`),
		"adversary_feature":    json.RawMessage(`{"target":{"type":"adversary_feature","adversary_id":"adv-1","feature_id":"feature-1"},"fear_spent":1}`),
		"environment_feature":  json.RawMessage(`{"target":{"type":"environment_feature","environment_id":"env-1","feature_id":"feature-1"},"fear_spent":2}`),
		"adversary_experience": json.RawMessage(`{"target":{"type":"adversary_experience","adversary_id":"adv-1","experience_name":"Pack Hunter"},"fear_spent":1}`),
	}
	for name, raw := range validCases {
		if err := daggerheartvalidator.ValidateGMMoveApplyPayload(raw); err != nil {
			t.Fatalf("%s apply payload invalid: %v", name, err)
		}
		if err := daggerheartvalidator.ValidateGMMoveAppliedPayload(raw); err != nil {
			t.Fatalf("%s applied payload invalid: %v", name, err)
		}
	}

	invalidCases := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{
			name: "unsupported target type",
			raw:  json.RawMessage(`{"target":{"type":"mystery"},"fear_spent":1}`),
			want: "target type is unsupported",
		},
		{
			name: "custom missing description",
			raw:  json.RawMessage(`{"target":{"type":"direct_move","kind":"interrupt_and_move","shape":"custom"},"fear_spent":1}`),
			want: "description is required for custom shape",
		},
		{
			name: "adversary feature missing feature id",
			raw:  json.RawMessage(`{"target":{"type":"adversary_feature","adversary_id":"adv-1"},"fear_spent":1}`),
			want: "feature_id is required",
		},
		{
			name: "environment feature missing environment id",
			raw:  json.RawMessage(`{"target":{"type":"environment_feature","feature_id":"feature-1"},"fear_spent":1}`),
			want: "environment_entity_id is required",
		},
		{
			name: "adversary experience missing name",
			raw:  json.RawMessage(`{"target":{"type":"adversary_experience","adversary_id":"adv-1"},"fear_spent":1}`),
			want: "experience_name is required",
		},
		{
			name: "fear spent required",
			raw:  json.RawMessage(`{"target":{"type":"direct_move","kind":"additional_move","shape":"shift_environment"},"fear_spent":0}`),
			want: "fear_spent must be greater than zero",
		},
	}
	for _, tc := range invalidCases {
		if err := daggerheartvalidator.ValidateGMMoveApplyPayload(tc.raw); err == nil || err.Error() != tc.want {
			t.Fatalf("%s apply payload error = %v, want %q", tc.name, err, tc.want)
		}
		if err := daggerheartvalidator.ValidateGMMoveAppliedPayload(tc.raw); err == nil || err.Error() != tc.want {
			t.Fatalf("%s applied payload error = %v, want %q", tc.name, err, tc.want)
		}
	}
}

func TestDeciderGMMoveApplyEmitsTypedTargets(t *testing.T) {
	now := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		payload  []byte
		validate func(t *testing.T, payload daggerheartpayload.GMMoveAppliedPayload)
	}{
		{
			name:    "adversary feature",
			payload: []byte(`{"target":{"type":"adversary_feature","adversary_id":" adv-1 ","feature_id":" feature-1 ","description":" pounce now "},"fear_spent":1}`),
			validate: func(t *testing.T, payload daggerheartpayload.GMMoveAppliedPayload) {
				t.Helper()
				if payload.Target.Type != rules.GMMoveTargetTypeAdversaryFeature {
					t.Fatalf("target type = %q", payload.Target.Type)
				}
				if payload.Target.AdversaryID != "adv-1" || payload.Target.FeatureID != "feature-1" || payload.Target.Description != "pounce now" {
					t.Fatalf("target = %+v", payload.Target)
				}
			},
		},
		{
			name:    "environment feature",
			payload: []byte(`{"target":{"type":"environment_feature","environment_entity_id":" env-entity-1 ","environment_id":" environment.falling-ruins ","feature_id":" feature-2 ","description":" falling stone "},"fear_spent":1}`),
			validate: func(t *testing.T, payload daggerheartpayload.GMMoveAppliedPayload) {
				t.Helper()
				if payload.Target.Type != rules.GMMoveTargetTypeEnvironmentFeature {
					t.Fatalf("target type = %q", payload.Target.Type)
				}
				if payload.Target.EnvironmentEntityID != "env-entity-1" || payload.Target.EnvironmentID != "environment.falling-ruins" || payload.Target.FeatureID != "feature-2" || payload.Target.Description != "falling stone" {
					t.Fatalf("target = %+v", payload.Target)
				}
			},
		},
		{
			name:    "adversary experience",
			payload: []byte(`{"target":{"type":"adversary_experience","adversary_id":" adv-2 ","experience_name":" Pack Hunter ","description":" coordinated strike "},"fear_spent":1}`),
			validate: func(t *testing.T, payload daggerheartpayload.GMMoveAppliedPayload) {
				t.Helper()
				if payload.Target.Type != rules.GMMoveTargetTypeAdversaryExperience {
					t.Fatalf("target type = %q", payload.Target.Type)
				}
				if payload.Target.AdversaryID != "adv-2" || payload.Target.ExperienceName != "Pack Hunter" || payload.Target.Description != "coordinated strike" {
					t.Fatalf("target = %+v", payload.Target)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			decision := (daggerheartdecider.Decider{}).Decide(daggerheartstate.SnapshotState{GMFear: 3}, command.Command{
				CampaignID:    "camp-1",
				Type:          daggerheartdecider.CommandTypeGMMoveApply,
				SessionID:     "sess-1",
				SystemID:      SystemID,
				SystemVersion: SystemVersion,
				PayloadJSON:   tc.payload,
			}, func() time.Time { return now })

			if len(decision.Rejections) != 0 {
				t.Fatalf("rejections = %d, want 0", len(decision.Rejections))
			}
			if len(decision.Events) != 2 {
				t.Fatalf("events = %d, want 2", len(decision.Events))
			}
			var movePayload daggerheartpayload.GMMoveAppliedPayload
			if err := json.Unmarshal(decision.Events[0].PayloadJSON, &movePayload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			tc.validate(t, movePayload)
		})
	}
}
