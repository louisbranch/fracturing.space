package daggerheart

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestSessionStartBootstrapSeedsFearFromPCCount(t *testing.T) {
	module := NewModule()
	now := time.Date(2026, 3, 13, 12, 0, 0, 0, time.UTC)

	events, err := module.SessionStartBootstrap(
		SnapshotState{GMFear: GMFearDefault},
		map[ids.CharacterID]character.State{
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
		},
		now,
	)
	if err != nil {
		t.Fatalf("SessionStartBootstrap returned error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	if events[0].Type != EventTypeGMFearChanged {
		t.Fatalf("event type = %s, want %s", events[0].Type, EventTypeGMFearChanged)
	}
	if events[0].Timestamp != now.UTC() {
		t.Fatalf("timestamp = %s, want %s", events[0].Timestamp, now.UTC())
	}
	var payload GMFearChangedPayload
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

	events, err := module.SessionStartBootstrap(
		SnapshotState{GMFear: 3},
		map[ids.CharacterID]character.State{
			"pc-1": {CharacterID: "pc-1", Created: true, Kind: character.KindPC},
		},
		command.Command{CampaignID: "camp-1"},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("SessionStartBootstrap returned error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("events = %d, want 0", len(events))
	}
}

func TestSessionStartBootstrapNoopsWithoutCreatedPCs(t *testing.T) {
	module := NewModule()

	events, err := module.SessionStartBootstrap(
		SnapshotState{GMFear: GMFearDefault},
		map[ids.CharacterID]character.State{
			"npc-1": {CharacterID: "npc-1", Created: true, Kind: character.KindNPC},
			"pc-1":  {CharacterID: "pc-1", Created: false, Kind: character.KindPC},
			"pc-2":  {CharacterID: "pc-2", Created: true, Deleted: true, Kind: character.KindPC},
		},
		command.Command{CampaignID: "camp-1"},
		time.Now(),
	)
	if err != nil {
		t.Fatalf("SessionStartBootstrap returned error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("events = %d, want 0", len(events))
	}
}

func TestDeciderGMMoveApplyRejectsCustomWithoutDescription(t *testing.T) {
	payloadJSON := []byte(`{"target":{"type":"direct_move","kind":"interrupt_and_move","shape":"custom"},"fear_spent":1}`)
	decision := (Decider{}).Decide(SnapshotState{GMFear: 2}, command.Command{
		CampaignID:    "camp-1",
		Type:          commandTypeGMMoveApply,
		SessionID:     "sess-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payloadJSON,
	}, func() time.Time { return time.Unix(0, 0).UTC() })

	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeGMMoveDescriptionRequired {
		t.Fatalf("rejection code = %q, want %q", decision.Rejections[0].Code, rejectionCodeGMMoveDescriptionRequired)
	}
}

func TestDeciderGMMoveApplyEmitsAuditAndFearEvents(t *testing.T) {
	now := time.Date(2026, 3, 13, 12, 30, 0, 0, time.UTC)
	payloadJSON := []byte(`{"target":{"type":"direct_move","kind":"additional_move","shape":"shift_environment"},"fear_spent":2}`)
	decision := (Decider{}).Decide(SnapshotState{GMFear: 4}, command.Command{
		CampaignID:    "camp-1",
		Type:          commandTypeGMMoveApply,
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
	if decision.Events[0].Type != EventTypeGMMoveApplied {
		t.Fatalf("first event type = %s, want %s", decision.Events[0].Type, EventTypeGMMoveApplied)
	}
	if decision.Events[1].Type != EventTypeGMFearChanged {
		t.Fatalf("second event type = %s, want %s", decision.Events[1].Type, EventTypeGMFearChanged)
	}
	if decision.Events[0].EntityType != "session" || decision.Events[0].EntityID != "sess-1" {
		t.Fatalf("gm move entity = (%q,%q), want (session,sess-1)", decision.Events[0].EntityType, decision.Events[0].EntityID)
	}
	var movePayload GMMoveAppliedPayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &movePayload); err != nil {
		t.Fatalf("decode gm move payload: %v", err)
	}
	if movePayload.Target.Type != GMMoveTargetTypeDirectMove {
		t.Fatalf("gm move target type = %q, want %q", movePayload.Target.Type, GMMoveTargetTypeDirectMove)
	}
	if movePayload.Target.Kind != GMMoveKindAdditionalMove {
		t.Fatalf("gm move kind = %q, want %q", movePayload.Target.Kind, GMMoveKindAdditionalMove)
	}
	if movePayload.Target.Shape != GMMoveShapeShiftEnvironment {
		t.Fatalf("gm move shape = %q, want %q", movePayload.Target.Shape, GMMoveShapeShiftEnvironment)
	}
	if movePayload.FearSpent != 2 {
		t.Fatalf("gm move fear_spent = %d, want 2", movePayload.FearSpent)
	}
	var fearPayload GMFearChangedPayload
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
	decision := (Decider{}).Decide(SnapshotState{GMFear: 2}, command.Command{
		CampaignID:    "camp-1",
		Type:          commandTypeGMMoveApply,
		SessionID:     "sess-1",
		SystemID:      SystemID,
		SystemVersion: SystemVersion,
		PayloadJSON:   payloadJSON,
	}, func() time.Time { return time.Unix(0, 0).UTC() })

	if len(decision.Rejections) != 1 {
		t.Fatalf("rejections = %d, want 1", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeGMMoveInsufficientFear {
		t.Fatalf("rejection code = %q, want %q", decision.Rejections[0].Code, rejectionCodeGMMoveInsufficientFear)
	}
}

func TestGMMoveAppliedEventIsAuditOnly(t *testing.T) {
	registry := event.NewRegistry()
	if err := NewModule().RegisterEvents(registry); err != nil {
		t.Fatalf("register events: %v", err)
	}
	def, ok := registry.Definition(EventTypeGMMoveApplied)
	if !ok {
		t.Fatal("expected gm_move_applied definition")
	}
	if def.Intent != event.IntentAuditOnly {
		t.Fatalf("intent = %s, want %s", def.Intent, event.IntentAuditOnly)
	}
}

func TestNormalizeGMMoveHelpers(t *testing.T) {
	t.Run("kind", func(t *testing.T) {
		cases := map[string]GMMoveKind{
			"interrupt_and_move": GMMoveKindInterruptAndMove,
			" additional_move ":  GMMoveKindAdditionalMove,
		}
		for input, want := range cases {
			got, ok := NormalizeGMMoveKind(input)
			if !ok || got != want {
				t.Fatalf("NormalizeGMMoveKind(%q) = (%q,%t), want (%q,true)", input, got, ok, want)
			}
		}
		if _, ok := NormalizeGMMoveKind("unknown"); ok {
			t.Fatal("expected unsupported kind to fail")
		}
	})

	t.Run("shape", func(t *testing.T) {
		cases := map[string]GMMoveShape{
			"show_world_reaction":      GMMoveShapeShowWorldReaction,
			"reveal_danger":            GMMoveShapeRevealDanger,
			"force_split":              GMMoveShapeForceSplit,
			"mark_stress":              GMMoveShapeMarkStress,
			"shift_environment":        GMMoveShapeShiftEnvironment,
			"spotlight_adversary":      GMMoveShapeSpotlightAdversary,
			"capture_important_target": GMMoveShapeCaptureImportantTarget,
			" custom ":                 GMMoveShapeCustom,
		}
		for input, want := range cases {
			got, ok := NormalizeGMMoveShape(input)
			if !ok || got != want {
				t.Fatalf("NormalizeGMMoveShape(%q) = (%q,%t), want (%q,true)", input, got, ok, want)
			}
		}
		if _, ok := NormalizeGMMoveShape("unknown"); ok {
			t.Fatal("expected unsupported shape to fail")
		}
	})

	t.Run("target type", func(t *testing.T) {
		cases := map[string]GMMoveTargetType{
			"direct_move":          GMMoveTargetTypeDirectMove,
			" adversary_feature ":  GMMoveTargetTypeAdversaryFeature,
			"environment_feature":  GMMoveTargetTypeEnvironmentFeature,
			"adversary_experience": GMMoveTargetTypeAdversaryExperience,
		}
		for input, want := range cases {
			got, ok := NormalizeGMMoveTargetType(input)
			if !ok || got != want {
				t.Fatalf("NormalizeGMMoveTargetType(%q) = (%q,%t), want (%q,true)", input, got, ok, want)
			}
		}
		if _, ok := NormalizeGMMoveTargetType("unknown"); ok {
			t.Fatal("expected unsupported target type to fail")
		}
	})
}

func TestValidateGMMovePayloadsForTypedTargets(t *testing.T) {
	validCases := map[string]json.RawMessage{
		"direct_move":          json.RawMessage(`{"target":{"type":"direct_move","kind":"additional_move","shape":"shift_environment"},"fear_spent":1}`),
		"adversary_feature":    json.RawMessage(`{"target":{"type":"adversary_feature","adversary_id":"adv-1","feature_id":"feature-1"},"fear_spent":1}`),
		"environment_feature":  json.RawMessage(`{"target":{"type":"environment_feature","environment_id":"env-1","feature_id":"feature-1"},"fear_spent":2}`),
		"adversary_experience": json.RawMessage(`{"target":{"type":"adversary_experience","adversary_id":"adv-1","experience_name":"Pack Hunter"},"fear_spent":1}`),
	}
	for name, raw := range validCases {
		if err := validateGMMoveApplyPayload(raw); err != nil {
			t.Fatalf("%s apply payload invalid: %v", name, err)
		}
		if err := validateGMMoveAppliedPayload(raw); err != nil {
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
		if err := validateGMMoveApplyPayload(tc.raw); err == nil || err.Error() != tc.want {
			t.Fatalf("%s apply payload error = %v, want %q", tc.name, err, tc.want)
		}
		if err := validateGMMoveAppliedPayload(tc.raw); err == nil || err.Error() != tc.want {
			t.Fatalf("%s applied payload error = %v, want %q", tc.name, err, tc.want)
		}
	}
}

func TestDeciderGMMoveApplyEmitsTypedTargets(t *testing.T) {
	now := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name     string
		payload  []byte
		validate func(t *testing.T, payload GMMoveAppliedPayload)
	}{
		{
			name:    "adversary feature",
			payload: []byte(`{"target":{"type":"adversary_feature","adversary_id":" adv-1 ","feature_id":" feature-1 ","description":" pounce now "},"fear_spent":1}`),
			validate: func(t *testing.T, payload GMMoveAppliedPayload) {
				t.Helper()
				if payload.Target.Type != GMMoveTargetTypeAdversaryFeature {
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
			validate: func(t *testing.T, payload GMMoveAppliedPayload) {
				t.Helper()
				if payload.Target.Type != GMMoveTargetTypeEnvironmentFeature {
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
			validate: func(t *testing.T, payload GMMoveAppliedPayload) {
				t.Helper()
				if payload.Target.Type != GMMoveTargetTypeAdversaryExperience {
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
			decision := (Decider{}).Decide(SnapshotState{GMFear: 3}, command.Command{
				CampaignID:    "camp-1",
				Type:          commandTypeGMMoveApply,
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
			var movePayload GMMoveAppliedPayload
			if err := json.Unmarshal(decision.Events[0].PayloadJSON, &movePayload); err != nil {
				t.Fatalf("decode payload: %v", err)
			}
			tc.validate(t, movePayload)
		})
	}
}
