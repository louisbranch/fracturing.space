package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
)

func newAdversaryDamageTestService() *DaggerheartService {
	svc := newAdversaryTestService()
	dhStore := svc.stores.Daggerheart.(*fakeDaggerheartAdversaryStore)
	dhStore.adversaries["camp-1:adv-1"] = projectionstore.DaggerheartAdversary{
		AdversaryID:      "adv-1",
		AdversaryEntryID: testAdversaryEntryGoblinID,
		CampaignID:       "camp-1",
		SessionID:        "sess-1",
		Name:             "Goblin",
		HP:               8,
		HPMax:            8,
		Armor:            1,
		Major:            4,
		Severe:           7,
	}
	// Character data needed by session adversary attack flow tests.
	dhStore.Profiles["camp-1:char-1"] = projectionstore.DaggerheartCharacterProfile{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		HpMax:       6,
		StressMax:   6,
		ArmorMax:    2,
	}
	dhStore.States["camp-1:char-1"] = projectionstore.DaggerheartCharacterState{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Hp:          6,
		Hope:        2,
		Stress:      3,
		Armor:       0,
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
