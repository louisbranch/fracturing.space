package daggerheart

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"google.golang.org/grpc/codes"
)

// --- ApplyDamage tests ---

func TestApplyDamage_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyDamage(context.Background(), &pb.DaggerheartApplyDamageRequest{
		CampaignId: "c1", CharacterId: "ch1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDamage_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     3,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyDamage_MissingCampaignId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{CharacterId: "ch1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_MissingCharacterId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{CampaignId: "camp-1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_MissingSessionId(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	_, err := svc.ApplyDamage(context.Background(), &pb.DaggerheartApplyDamageRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_MissingDamage(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId: "camp-1", CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_NegativeAmount(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     -1,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_UnspecifiedType(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     2,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyDamage_Success(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	profile := dhStore.Profiles["camp-1:char-1"]
	state := dhStore.States["camp-1:char-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     3,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}
	result, mitigated, err := applyDaggerheartDamage(damage, profile, state)
	if err != nil {
		t.Fatalf("apply daggerheart damage: %v", err)
	}

	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := normalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheart.DamageAppliedPayload{
		CharacterID:        "char-1",
		HpBefore:           &hpBefore,
		HpAfter:            &hpAfter,
		ArmorBefore:        &armorBefore,
		ArmorAfter:         &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           daggerheartSeverityToString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         daggerheartDamageTypeToString(damage.DamageType),
		RollSeq:            nil,
		ResistPhysical:     damage.ResistPhysical,
		ResistMagic:        damage.ResistMagic,
		ImmunePhysical:     damage.ImmunePhysical,
		ImmuneMagic:        damage.ImmuneMagic,
		Direct:             damage.Direct,
		MassiveDamage:      damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             damage.Source,
		SourceCharacterIDs: sourceCharacterIDs,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode damage payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.damage.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.damage_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage:      damage,
	})
	if err != nil {
		t.Fatalf("ApplyDamage returned error: %v", err)
	}
	if resp.CharacterId != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterId)
	}
	if resp.State == nil {
		t.Fatal("expected state in response")
	}
	if resp.State.Hp >= 6 {
		t.Fatalf("hp = %d, expected less than 6 after 3 damage", resp.State.Hp)
	}
}

func TestApplyDamage_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	profile := dhStore.Profiles["camp-1:char-1"]
	state := dhStore.States["camp-1:char-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     3,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}
	result, mitigated, err := applyDaggerheartDamage(damage, profile, state)
	if err != nil {
		t.Fatalf("apply daggerheart damage: %v", err)
	}

	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := normalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheart.DamageAppliedPayload{
		CharacterID:        "char-1",
		HpBefore:           &hpBefore,
		HpAfter:            &hpAfter,
		ArmorBefore:        &armorBefore,
		ArmorAfter:         &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           daggerheartSeverityToString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         daggerheartDamageTypeToString(damage.DamageType),
		RollSeq:            nil,
		ResistPhysical:     damage.ResistPhysical,
		ResistMagic:        damage.ResistMagic,
		ImmunePhysical:     damage.ImmunePhysical,
		ImmuneMagic:        damage.ImmuneMagic,
		Direct:             damage.Direct,
		MassiveDamage:      damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             damage.Source,
		SourceCharacterIDs: sourceCharacterIDs,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode damage payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.damage.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.damage_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-apply-damage",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-apply-damage")
	_, err = svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage:      damage,
	})
	if err != nil {
		t.Fatalf("ApplyDamage returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.damage.apply") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.damage.apply")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got struct {
		CharacterID string `json:"character_id"`
		DamageType  string `json:"damage_type"`
		HpAfter     *int   `json:"hp_after"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode damage command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if got.DamageType != "physical" {
		t.Fatalf("command damage type = %s, want %s", got.DamageType, "physical")
	}
	if got.HpAfter == nil || *got.HpAfter != hpAfter {
		t.Fatalf("command hp after = %v, want %d", got.HpAfter, hpAfter)
	}
}

func TestApplyDamage_WithArmorMitigation(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	profile := dhStore.Profiles["camp-1:char-1"]
	profile.MajorThreshold = 3
	profile.SevereThreshold = 6
	profile.ArmorMax = 1
	dhStore.Profiles["camp-1:char-1"] = profile

	state := dhStore.States["camp-1:char-1"]
	state.Hp = 6
	state.Armor = 1
	dhStore.States["camp-1:char-1"] = state

	damage := &pb.DaggerheartDamageRequest{
		Amount:     4,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}
	result, mitigated, err := applyDaggerheartDamage(damage, profile, state)
	if err != nil {
		t.Fatalf("apply daggerheart damage: %v", err)
	}

	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := normalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheart.DamageAppliedPayload{
		CharacterID:        "char-1",
		HpBefore:           &hpBefore,
		HpAfter:            &hpAfter,
		ArmorBefore:        &armorBefore,
		ArmorAfter:         &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           daggerheartSeverityToString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         daggerheartDamageTypeToString(damage.DamageType),
		RollSeq:            nil,
		ResistPhysical:     damage.ResistPhysical,
		ResistMagic:        damage.ResistMagic,
		ImmunePhysical:     damage.ImmunePhysical,
		ImmuneMagic:        damage.ImmuneMagic,
		Direct:             damage.Direct,
		MassiveDamage:      damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             damage.Source,
		SourceCharacterIDs: sourceCharacterIDs,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode damage payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.damage.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.damage_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	_, err = svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage:      damage,
	})
	if err != nil {
		t.Fatalf("ApplyDamage returned error: %v", err)
	}

	events := eventStore.Events["camp-1"]
	if len(events) == 0 {
		t.Fatal("expected damage event")
	}
	last := events[len(events)-1]
	if last.Type != event.Type("sys.daggerheart.damage_applied") {
		t.Fatalf("last event type = %s, want %s", last.Type, event.Type("sys.daggerheart.damage_applied"))
	}
	var parsedPayload daggerheart.DamageAppliedPayload
	if err := json.Unmarshal(last.PayloadJSON, &parsedPayload); err != nil {
		t.Fatalf("decode damage payload: %v", err)
	}
	if parsedPayload.ArmorSpent != 1 {
		t.Fatalf("armor_spent = %d, want 1", parsedPayload.ArmorSpent)
	}
	if parsedPayload.Marks != 1 {
		t.Fatalf("marks = %d, want 1", parsedPayload.Marks)
	}
	if parsedPayload.Severity != "minor" {
		t.Fatalf("severity = %s, want minor", parsedPayload.Severity)
	}
}

func TestApplyDamage_RequireDamageRollWithoutSeq(t *testing.T) {
	svc := newActionTestService()
	configureNoopDomain(svc)
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     3,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
		RequireDamageRoll: true,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyConditions_LifeStateOnly(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	before := daggerheart.LifeStateAlive
	after := daggerheart.LifeStateUnconscious
	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:     "char-1",
		LifeStateBefore: &before,
		LifeStateAfter:  &after,
	}
	patchJSON, err := json.Marshal(patchPayload)
	if err != nil {
		t.Fatalf("encode patch payload: %v", err)
	}
	serviceDomain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC),
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-conditions-life",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
	}}
	svc.stores.Domain = serviceDomain
	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-conditions-life")
	resp, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if resp.State == nil {
		t.Fatal("expected state in response")
	}
	if resp.State.LifeState != pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS {
		t.Fatalf("life_state = %v, want UNCONSCIOUS", resp.State.LifeState)
	}

	events := eventStore.Events["camp-1"]
	if len(events) == 0 {
		t.Fatal("expected events")
	}
	last := events[len(events)-1]
	if last.Type != event.Type("sys.daggerheart.character_state_patched") {
		t.Fatalf("last event type = %s, want %s", last.Type, event.Type("sys.daggerheart.character_state_patched"))
	}
}

func TestApplyConditions_LifeStateNoChange(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyConditions_InvalidStoredLifeState(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	state.LifeState = "not-a-life-state"
	dhStore.States["camp-1:char-1"] = state

	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyConditions_NoConditionChanges(t *testing.T) {
	svc := newActionTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartStore)
	state := dhStore.States["camp-1:char-1"]
	state.Conditions = []string{"vulnerable"}
	dhStore.States["camp-1:char-1"] = state

	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE},
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestApplyConditions_RequiresDomainEngine(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyConditions_UsesDomainEngine(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	conditionPayload := daggerheart.ConditionChangedPayload{
		CharacterID:      "char-1",
		ConditionsBefore: []string{},
		ConditionsAfter:  []string{daggerheart.ConditionHidden},
		Added:            []string{daggerheart.ConditionHidden},
	}
	conditionJSON, err := json.Marshal(conditionPayload)
	if err != nil {
		t.Fatalf("encode condition payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.condition.change"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.condition_changed"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-apply-conditions",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   conditionJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-apply-conditions")
	_, err = svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Add:         []pb.DaggerheartCondition{pb.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN},
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.condition.change") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.condition.change")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got daggerheart.ConditionChangePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode condition command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if len(got.ConditionsAfter) != 1 || got.ConditionsAfter[0] != daggerheart.ConditionHidden {
		t.Fatalf("command conditions_after = %v, want %s", got.ConditionsAfter, daggerheart.ConditionHidden)
	}
	var foundConditionEvent bool
	for _, evt := range eventStore.Events["camp-1"] {
		if evt.Type == event.Type("sys.daggerheart.condition_changed") {
			foundConditionEvent = true
			break
		}
	}
	if !foundConditionEvent {
		t.Fatal("expected condition changed event")
	}
}

func TestApplyConditions_UsesDomainEngineForLifeState(t *testing.T) {
	svc := newActionTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	before := daggerheart.LifeStateAlive
	after := daggerheart.LifeStateUnconscious
	patchPayload := daggerheart.CharacterStatePatchedPayload{
		CharacterID:     "char-1",
		LifeStateBefore: &before,
		LifeStateAfter:  &after,
	}
	patchJSON, err := json.Marshal(patchPayload)
	if err != nil {
		t.Fatalf("encode patch payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.character_state.patch"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.character_state_patched"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-apply-conditions",
				EntityType:    "character",
				EntityID:      "char-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   patchJSON,
			}),
		},
	}}
	svc.stores.Domain = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-apply-conditions")
	_, err = svc.ApplyConditions(ctx, &pb.DaggerheartApplyConditionsRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		LifeState:   pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS,
	})
	if err != nil {
		t.Fatalf("ApplyConditions returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.character_state.patch") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.character_state.patch")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got daggerheart.CharacterStatePatchPayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode patch command payload: %v", err)
	}
	if got.CharacterID != "char-1" {
		t.Fatalf("command character id = %s, want %s", got.CharacterID, "char-1")
	}
	if got.LifeStateBefore == nil || *got.LifeStateBefore != before {
		t.Fatalf("command life_state_before = %v, want %s", got.LifeStateBefore, before)
	}
	if got.LifeStateAfter == nil || *got.LifeStateAfter != after {
		t.Fatalf("command life_state_after = %v, want %s", got.LifeStateAfter, after)
	}
	var foundStateEvent bool
	for _, evt := range eventStore.Events["camp-1"] {
		if evt.Type == event.Type("sys.daggerheart.character_state_patched") {
			foundStateEvent = true
			break
		}
	}
	if !foundStateEvent {
		t.Fatal("expected character state patched event")
	}
}
