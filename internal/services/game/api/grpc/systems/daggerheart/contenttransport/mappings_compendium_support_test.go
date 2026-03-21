package contenttransport

import (
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func TestContentKindMappings(t *testing.T) {
	if domainCardTypeToProto("Spell") != pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_SPELL {
		t.Fatal("expected spell domain card type")
	}
	if weaponCategoryToProto("secondary") != pb.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_SECONDARY {
		t.Fatal("expected secondary weapon category")
	}
	if weaponDisplayGroupToProto("magic") != pb.DaggerheartWeaponDisplayGroup_DAGGERHEART_WEAPON_DISPLAY_GROUP_MAGIC {
		t.Fatal("expected magic weapon display group")
	}
	if itemRarityToProto("Rare") != pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_RARE {
		t.Fatal("expected rare item rarity")
	}
	if itemKindToProto("equipment") != pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_EQUIPMENT {
		t.Fatal("expected equipment item kind")
	}
	if environmentTypeToProto(" Social ") != pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_SOCIAL {
		t.Fatal("expected social environment type")
	}
	if damageTypeToProto("Mixed") != pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED {
		t.Fatal("expected mixed damage type")
	}
}

func TestContentKindMappingsExtended(t *testing.T) {
	if domainCardTypeToProto("ability") != pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_ABILITY {
		t.Fatal("expected ability domain card type")
	}
	if domainCardTypeToProto("grimoire") != pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_GRIMOIRE {
		t.Fatal("expected grimoire domain card type")
	}
	if domainCardTypeToProto("unknown") != pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_UNSPECIFIED {
		t.Fatal("expected unspecified domain card type")
	}
	if weaponCategoryToProto("unknown") != pb.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_UNSPECIFIED {
		t.Fatal("expected unspecified weapon category")
	}
	if weaponDisplayGroupToProto("unknown") != pb.DaggerheartWeaponDisplayGroup_DAGGERHEART_WEAPON_DISPLAY_GROUP_UNSPECIFIED {
		t.Fatal("expected unspecified weapon display group")
	}
	if itemRarityToProto("common") != pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_COMMON {
		t.Fatal("expected common item rarity")
	}
	if itemRarityToProto("unique") != pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_UNIQUE {
		t.Fatal("expected unique item rarity")
	}
	if itemRarityToProto("legendary") != pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_LEGENDARY {
		t.Fatal("expected legendary item rarity")
	}
	if itemRarityToProto("unknown") != pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_UNSPECIFIED {
		t.Fatal("expected unspecified item rarity")
	}
	if itemKindToProto("treasure") != pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_TREASURE {
		t.Fatal("expected treasure item kind")
	}
	if itemKindToProto("unknown") != pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_UNSPECIFIED {
		t.Fatal("expected unspecified item kind")
	}
	if environmentTypeToProto("traversal") != pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_TRAVERSAL {
		t.Fatal("expected traversal environment type")
	}
	if environmentTypeToProto("event") != pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_EVENT {
		t.Fatal("expected event environment type")
	}
	if environmentTypeToProto("unknown") != pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_UNSPECIFIED {
		t.Fatal("expected unspecified environment type")
	}
	if damageTypeToProto("magic") != pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC {
		t.Fatal("expected magic damage type")
	}
	if damageTypeToProto("unknown") != pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED {
		t.Fatal("expected unspecified damage type")
	}
}
