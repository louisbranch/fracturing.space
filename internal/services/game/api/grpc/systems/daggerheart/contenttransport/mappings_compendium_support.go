package contenttransport

import (
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func domainCardTypeToProto(kind string) pb.DaggerheartDomainCardType {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "ability":
		return pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_ABILITY
	case "spell":
		return pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_SPELL
	case "grimoire":
		return pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_GRIMOIRE
	default:
		return pb.DaggerheartDomainCardType_DAGGERHEART_DOMAIN_CARD_TYPE_UNSPECIFIED
	}
}

func weaponCategoryToProto(kind string) pb.DaggerheartWeaponCategory {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "primary":
		return pb.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_PRIMARY
	case "secondary":
		return pb.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_SECONDARY
	default:
		return pb.DaggerheartWeaponCategory_DAGGERHEART_WEAPON_CATEGORY_UNSPECIFIED
	}
}

func weaponDisplayGroupToProto(kind string) pb.DaggerheartWeaponDisplayGroup {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "physical":
		return pb.DaggerheartWeaponDisplayGroup_DAGGERHEART_WEAPON_DISPLAY_GROUP_PHYSICAL
	case "magic":
		return pb.DaggerheartWeaponDisplayGroup_DAGGERHEART_WEAPON_DISPLAY_GROUP_MAGIC
	case "combat_wheelchair":
		return pb.DaggerheartWeaponDisplayGroup_DAGGERHEART_WEAPON_DISPLAY_GROUP_COMBAT_WHEELCHAIR
	default:
		return pb.DaggerheartWeaponDisplayGroup_DAGGERHEART_WEAPON_DISPLAY_GROUP_UNSPECIFIED
	}
}

func itemRarityToProto(kind string) pb.DaggerheartItemRarity {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "common":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_COMMON
	case "uncommon":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_UNCOMMON
	case "rare":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_RARE
	case "unique":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_UNIQUE
	case "legendary":
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_LEGENDARY
	default:
		return pb.DaggerheartItemRarity_DAGGERHEART_ITEM_RARITY_UNSPECIFIED
	}
}

func itemKindToProto(kind string) pb.DaggerheartItemKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "consumable":
		return pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_CONSUMABLE
	case "equipment":
		return pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_EQUIPMENT
	case "treasure":
		return pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_TREASURE
	default:
		return pb.DaggerheartItemKind_DAGGERHEART_ITEM_KIND_UNSPECIFIED
	}
}

func environmentTypeToProto(kind string) pb.DaggerheartEnvironmentType {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "exploration":
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_EXPLORATION
	case "social":
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_SOCIAL
	case "traversal":
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_TRAVERSAL
	case "event":
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_EVENT
	default:
		return pb.DaggerheartEnvironmentType_DAGGERHEART_ENVIRONMENT_TYPE_UNSPECIFIED
	}
}

func damageTypeToProto(kind string) pb.DaggerheartDamageType {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "physical":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL
	case "magic":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC
	case "mixed":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED
	default:
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED
	}
}
