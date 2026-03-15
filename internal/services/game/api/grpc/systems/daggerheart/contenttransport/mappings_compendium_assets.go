package contenttransport

import (
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

func toProtoDaggerheartLootEntry(entry contentstore.DaggerheartLootEntry) *pb.DaggerheartLootEntry {
	return &pb.DaggerheartLootEntry{
		Id:          entry.ID,
		Name:        entry.Name,
		Roll:        int32(entry.Roll),
		Description: entry.Description,
	}
}

func toProtoDaggerheartLootEntries(entries []contentstore.DaggerheartLootEntry) []*pb.DaggerheartLootEntry {
	items := make([]*pb.DaggerheartLootEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toProtoDaggerheartLootEntry(entry))
	}
	return items
}

func toProtoDaggerheartWeapon(weapon contentstore.DaggerheartWeapon) *pb.DaggerheartWeapon {
	return &pb.DaggerheartWeapon{
		Id:         weapon.ID,
		Name:       weapon.Name,
		Category:   weaponCategoryToProto(weapon.Category),
		Tier:       int32(weapon.Tier),
		Trait:      weapon.Trait,
		Range:      weapon.Range,
		DamageDice: toProtoDaggerheartDamageDice(weapon.DamageDice),
		DamageType: damageTypeToProto(weapon.DamageType),
		Burden:     int32(weapon.Burden),
		Feature:    weapon.Feature,
	}
}

func toProtoDaggerheartWeapons(weapons []contentstore.DaggerheartWeapon) []*pb.DaggerheartWeapon {
	items := make([]*pb.DaggerheartWeapon, 0, len(weapons))
	for _, weapon := range weapons {
		items = append(items, toProtoDaggerheartWeapon(weapon))
	}
	return items
}

func toProtoDaggerheartArmor(armor contentstore.DaggerheartArmor) *pb.DaggerheartArmor {
	return &pb.DaggerheartArmor{
		Id:                  armor.ID,
		Name:                armor.Name,
		Tier:                int32(armor.Tier),
		BaseMajorThreshold:  int32(armor.BaseMajorThreshold),
		BaseSevereThreshold: int32(armor.BaseSevereThreshold),
		ArmorScore:          int32(armor.ArmorScore),
		Feature:             armor.Feature,
	}
}

func toProtoDaggerheartArmorList(items []contentstore.DaggerheartArmor) []*pb.DaggerheartArmor {
	armor := make([]*pb.DaggerheartArmor, 0, len(items))
	for _, item := range items {
		armor = append(armor, toProtoDaggerheartArmor(item))
	}
	return armor
}

func toProtoDaggerheartItem(item contentstore.DaggerheartItem) *pb.DaggerheartItem {
	return &pb.DaggerheartItem{
		Id:          item.ID,
		Name:        item.Name,
		Rarity:      itemRarityToProto(item.Rarity),
		Kind:        itemKindToProto(item.Kind),
		StackMax:    int32(item.StackMax),
		Description: item.Description,
		EffectText:  item.EffectText,
	}
}

func toProtoDaggerheartItems(items []contentstore.DaggerheartItem) []*pb.DaggerheartItem {
	results := make([]*pb.DaggerheartItem, 0, len(items))
	for _, item := range items {
		results = append(results, toProtoDaggerheartItem(item))
	}
	return results
}

func toProtoDaggerheartEnvironment(env contentstore.DaggerheartEnvironment) *pb.DaggerheartEnvironment {
	return &pb.DaggerheartEnvironment{
		Id:                    env.ID,
		Name:                  env.Name,
		Tier:                  int32(env.Tier),
		Type:                  environmentTypeToProto(env.Type),
		Difficulty:            int32(env.Difficulty),
		Impulses:              append([]string{}, env.Impulses...),
		PotentialAdversaryIds: append([]string{}, env.PotentialAdversaryIDs...),
		Features:              toProtoDaggerheartFeatures(env.Features),
		Prompts:               append([]string{}, env.Prompts...),
	}
}

func toProtoDaggerheartEnvironments(envs []contentstore.DaggerheartEnvironment) []*pb.DaggerheartEnvironment {
	results := make([]*pb.DaggerheartEnvironment, 0, len(envs))
	for _, env := range envs {
		results = append(results, toProtoDaggerheartEnvironment(env))
	}
	return results
}
