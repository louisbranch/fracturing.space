package daggerheart

import (
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/damagetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
)

func TestApplyAdversaryDamage_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	now := testTimestamp

	adversary := dhStore.adversaries["camp-1:adv-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     5,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}
	result, mitigated, err := damagetransport.ResolveAdversaryDamage(damage, adversary)
	if err != nil {
		t.Fatalf("apply adversary damage: %v", err)
	}
	hpAfter := result.HPAfter
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := workflowtransport.NormalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheartpayload.AdversaryDamageAppliedPayload{
		AdversaryID:        "adv-1",
		Hp:                 &hpAfter,
		Armor:              &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           damagetransport.DamageSeverityString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         damagetransport.DamageTypeString(damage.DamageType),
		RollSeq:            nil,
		ResistPhysical:     damage.ResistPhysical,
		ResistMagic:        damage.ResistMagic,
		ImmunePhysical:     damage.ImmunePhysical,
		ImmuneMagic:        damage.ImmuneMagic,
		Direct:             damage.Direct,
		MassiveDamage:      damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             damage.Source,
		SourceCharacterIDs: testStringsToCharacterIDs(sourceCharacterIDs),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary damage payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.adversary_damage.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.adversary_damage_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     5, // major damage (>=4, <7)
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryDamage returned error: %v", err)
	}
	if resp.AdversaryId != "adv-1" {
		t.Fatalf("adversary_id = %q, want adv-1", resp.AdversaryId)
	}
	if resp.Adversary == nil {
		t.Fatal("expected adversary in response")
	}
}

func TestApplyAdversaryDamage_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	now := testTimestamp

	adversary := dhStore.adversaries["camp-1:adv-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     5,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}
	result, mitigated, err := damagetransport.ResolveAdversaryDamage(damage, adversary)
	if err != nil {
		t.Fatalf("apply adversary damage: %v", err)
	}
	hpAfter := result.HPAfter
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := workflowtransport.NormalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheartpayload.AdversaryDamageAppliedPayload{
		AdversaryID:        "adv-1",
		Hp:                 &hpAfter,
		Armor:              &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           damagetransport.DamageSeverityString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         damagetransport.DamageTypeString(damage.DamageType),
		RollSeq:            nil,
		ResistPhysical:     damage.ResistPhysical,
		ResistMagic:        damage.ResistMagic,
		ImmunePhysical:     damage.ImmunePhysical,
		ImmuneMagic:        damage.ImmuneMagic,
		Direct:             damage.Direct,
		MassiveDamage:      damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             damage.Source,
		SourceCharacterIDs: testStringsToCharacterIDs(sourceCharacterIDs),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary damage payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.adversary_damage.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.adversary_damage_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				RequestID:     "req-adversary-damage",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-adversary-damage")
	_, err = svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage:      damage,
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryDamage returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 domain command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("sys.daggerheart.adversary_damage.apply") {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, "sys.daggerheart.adversary_damage.apply")
	}
	if domain.commands[0].SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", domain.commands[0].SystemID, daggerheart.SystemID)
	}
	if domain.commands[0].SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", domain.commands[0].SystemVersion, daggerheart.SystemVersion)
	}
	var got struct {
		AdversaryID string `json:"adversary_id"`
		DamageType  string `json:"damage_type"`
		HpAfter     *int   `json:"hp_after"`
	}
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &got); err != nil {
		t.Fatalf("decode adversary damage command payload: %v", err)
	}
	if got.AdversaryID != "adv-1" {
		t.Fatalf("command adversary id = %s, want %s", got.AdversaryID, "adv-1")
	}
	if got.DamageType != "physical" {
		t.Fatalf("command damage type = %s, want %s", got.DamageType, "physical")
	}
	if got.HpAfter == nil || *got.HpAfter != hpAfter {
		t.Fatalf("command hp after = %v, want %d", got.HpAfter, hpAfter)
	}
}

func TestApplyAdversaryDamage_DirectDamage(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	now := testTimestamp

	adversary := dhStore.adversaries["camp-1:adv-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     3,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		Direct:     true,
	}
	result, mitigated, err := damagetransport.ResolveAdversaryDamage(damage, adversary)
	if err != nil {
		t.Fatalf("apply adversary damage: %v", err)
	}
	hpAfter := result.HPAfter
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := workflowtransport.NormalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheartpayload.AdversaryDamageAppliedPayload{
		AdversaryID:        "adv-1",
		Hp:                 &hpAfter,
		Armor:              &armorAfter,
		ArmorSpent:         result.ArmorSpent,
		Severity:           damagetransport.DamageSeverityString(result.Result.Severity),
		Marks:              result.Result.Marks,
		DamageType:         damagetransport.DamageTypeString(damage.DamageType),
		RollSeq:            nil,
		ResistPhysical:     damage.ResistPhysical,
		ResistMagic:        damage.ResistMagic,
		ImmunePhysical:     damage.ImmunePhysical,
		ImmuneMagic:        damage.ImmuneMagic,
		Direct:             damage.Direct,
		MassiveDamage:      damage.MassiveDamage,
		Mitigated:          mitigated,
		Source:             damage.Source,
		SourceCharacterIDs: testStringsToCharacterIDs(sourceCharacterIDs),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode adversary damage payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("sys.daggerheart.adversary_damage.apply"): {
			Decision: command.Accept(event.Event{
				CampaignID:    "camp-1",
				Type:          event.Type("sys.daggerheart.adversary_damage_applied"),
				Timestamp:     now,
				ActorType:     event.ActorTypeSystem,
				SessionID:     "sess-1",
				EntityType:    "adversary",
				EntityID:      "adv-1",
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
			}),
		},
	}}
	svc.stores.Write.Executor = domain
	ctx := contextWithSessionID("sess-1")
	resp, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     3,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
			Direct:     true,
		},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryDamage returned error: %v", err)
	}
	if resp.Adversary == nil {
		t.Fatal("expected adversary in response")
	}
}
