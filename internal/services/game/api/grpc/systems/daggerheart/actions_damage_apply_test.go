package daggerheart

import (
	"context"
	"encoding/json"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/damagetransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"google.golang.org/grpc/codes"
)

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
	now := testTimestamp

	profile := dhStore.Profiles["camp-1:char-1"]
	state := dhStore.States["camp-1:char-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     3,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}
	payloadJSON := mustDamageAppliedPayloadJSON(t, "char-1", damage, profile, state)

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
	svc.stores.Write.Executor = domain
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
	now := testTimestamp

	profile := dhStore.Profiles["camp-1:char-1"]
	state := dhStore.States["camp-1:char-1"]
	damage := &pb.DaggerheartDamageRequest{
		Amount:     3,
		DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
	}
	payloadJSON := mustDamageAppliedPayloadJSON(t, "char-1", damage, profile, state)

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
	svc.stores.Write.Executor = domain

	ctx := grpcmeta.WithRequestID(contextWithSessionID("sess-1"), "req-apply-damage")
	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
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

	result, _, err := damagetransport.ResolveCharacterDamage(damage, profile, state, nil)
	if err != nil {
		t.Fatalf("apply daggerheart damage: %v", err)
	}
	if got.HpAfter == nil || *got.HpAfter != result.HPAfter {
		t.Fatalf("command hp after = %v, want %d", got.HpAfter, result.HPAfter)
	}
}

func TestApplyDamage_WithArmorMitigation(t *testing.T) {
	svc := newActionTestService()
	ctx := contextWithSessionID("sess-1")
	eventStore := svc.stores.Event.(*fakeEventStore)
	now := testTimestamp

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
	payloadJSON := mustDamageAppliedPayloadJSON(t, "char-1", damage, profile, state)

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
	svc.stores.Write.Executor = domain

	_, err := svc.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
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
