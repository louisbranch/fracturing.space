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
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

// --- ApplyAdversaryDamage tests ---

func newAdversaryDamageTestService() *DaggerheartService {
	svc := newAdversaryTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	dhStore.adversaries["camp-1:adv-1"] = storage.DaggerheartAdversary{
		AdversaryID: "adv-1",
		CampaignID:  "camp-1",
		SessionID:   "sess-1",
		Name:        "Goblin",
		HP:          8,
		HPMax:       8,
		Armor:       1,
		Major:       4,
		Severe:      7,
	}
	return svc
}

func TestApplyAdversaryDamage_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.ApplyAdversaryDamage(context.Background(), &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId: "c1", AdversaryId: "a1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAdversaryDamage_RequiresDomainEngine(t *testing.T) {
	svc := newAdversaryDamageTestService()
	svc.stores.Domain = nil
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     5,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestApplyAdversaryDamage_MissingCampaignId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_MissingAdversaryId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_MissingSessionId(t *testing.T) {
	svc := newAdversaryDamageTestService()
	_, err := svc.ApplyAdversaryDamage(context.Background(), &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId: "camp-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_MissingDamage(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId: "camp-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_NegativeAmount(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     -1,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_UnspecifiedType(t *testing.T) {
	svc := newAdversaryDamageTestService()
	ctx := contextWithSessionID("sess-1")
	_, err := svc.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     2,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED,
		},
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestApplyAdversaryDamage_Success(t *testing.T) {
	svc := newAdversaryDamageTestService()
	eventStore := svc.stores.Event.(*fakeEventStore)
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	adversary := dhStore.adversaries["camp-1:adv-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     5,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}
	result, mitigated, err := applyDaggerheartAdversaryDamage(damage, adversary)
	if err != nil {
		t.Fatalf("apply adversary damage: %v", err)
	}
	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := normalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheart.AdversaryDamageAppliedPayload{
		AdversaryID:        "adv-1",
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
	svc.stores.Domain = domain
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
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	adversary := dhStore.adversaries["camp-1:adv-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     5,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}
	result, mitigated, err := applyDaggerheartAdversaryDamage(damage, adversary)
	if err != nil {
		t.Fatalf("apply adversary damage: %v", err)
	}
	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := normalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheart.AdversaryDamageAppliedPayload{
		AdversaryID:        "adv-1",
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
	svc.stores.Domain = domain

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
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)

	adversary := dhStore.adversaries["camp-1:adv-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     3,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		Direct:     true,
	}
	result, mitigated, err := applyDaggerheartAdversaryDamage(damage, adversary)
	if err != nil {
		t.Fatalf("apply adversary damage: %v", err)
	}
	hpBefore := result.HPBefore
	hpAfter := result.HPAfter
	armorBefore := result.ArmorBefore
	armorAfter := result.ArmorAfter
	sourceCharacterIDs := normalizeTargets(damage.GetSourceCharacterIds())
	payload := daggerheart.AdversaryDamageAppliedPayload{
		AdversaryID:        "adv-1",
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
	svc.stores.Domain = domain
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
