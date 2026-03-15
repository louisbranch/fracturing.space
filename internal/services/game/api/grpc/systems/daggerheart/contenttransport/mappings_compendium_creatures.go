package contenttransport

import (
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

func toProtoDaggerheartAdversaryAttack(attack contentstore.DaggerheartAdversaryAttack) *pb.DaggerheartAdversaryAttack {
	return &pb.DaggerheartAdversaryAttack{
		Name:        attack.Name,
		Range:       attack.Range,
		DamageDice:  toProtoDaggerheartDamageDice(attack.DamageDice),
		DamageBonus: int32(attack.DamageBonus),
		DamageType:  damageTypeToProto(attack.DamageType),
	}
}

func toProtoDaggerheartAdversaryExperiences(experiences []contentstore.DaggerheartAdversaryExperience) []*pb.DaggerheartAdversaryExperience {
	items := make([]*pb.DaggerheartAdversaryExperience, 0, len(experiences))
	for _, experience := range experiences {
		items = append(items, &pb.DaggerheartAdversaryExperience{
			Name:     experience.Name,
			Modifier: int32(experience.Modifier),
		})
	}
	return items
}

func toProtoDaggerheartAdversaryFeatures(features []contentstore.DaggerheartAdversaryFeature) []*pb.DaggerheartAdversaryFeature {
	items := make([]*pb.DaggerheartAdversaryFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, &pb.DaggerheartAdversaryFeature{
			Id:          feature.ID,
			Name:        feature.Name,
			Kind:        feature.Kind,
			Description: feature.Description,
			CostType:    feature.CostType,
			Cost:        int32(feature.Cost),
		})
	}
	return items
}

func toProtoDaggerheartAdversaryEntry(entry contentstore.DaggerheartAdversaryEntry) *pb.DaggerheartAdversaryEntry {
	return &pb.DaggerheartAdversaryEntry{
		Id:              entry.ID,
		Name:            entry.Name,
		Tier:            int32(entry.Tier),
		Role:            entry.Role,
		Description:     entry.Description,
		Motives:         entry.Motives,
		Difficulty:      int32(entry.Difficulty),
		MajorThreshold:  int32(entry.MajorThreshold),
		SevereThreshold: int32(entry.SevereThreshold),
		Hp:              int32(entry.HP),
		Stress:          int32(entry.Stress),
		Armor:           int32(entry.Armor),
		AttackModifier:  int32(entry.AttackModifier),
		StandardAttack:  toProtoDaggerheartAdversaryAttack(entry.StandardAttack),
		Experiences:     toProtoDaggerheartAdversaryExperiences(entry.Experiences),
		Features:        toProtoDaggerheartAdversaryFeatures(entry.Features),
	}
}

func toProtoDaggerheartAdversaryEntries(entries []contentstore.DaggerheartAdversaryEntry) []*pb.DaggerheartAdversaryEntry {
	items := make([]*pb.DaggerheartAdversaryEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toProtoDaggerheartAdversaryEntry(entry))
	}
	return items
}

func toProtoDaggerheartBeastformAttack(attack contentstore.DaggerheartBeastformAttack) *pb.DaggerheartBeastformAttack {
	return &pb.DaggerheartBeastformAttack{
		Range:       attack.Range,
		Trait:       attack.Trait,
		DamageDice:  toProtoDaggerheartDamageDice(attack.DamageDice),
		DamageBonus: int32(attack.DamageBonus),
		DamageType:  damageTypeToProto(attack.DamageType),
	}
}

func toProtoDaggerheartBeastformFeatures(features []contentstore.DaggerheartBeastformFeature) []*pb.DaggerheartBeastformFeature {
	items := make([]*pb.DaggerheartBeastformFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, &pb.DaggerheartBeastformFeature{
			Id:          feature.ID,
			Name:        feature.Name,
			Description: feature.Description,
		})
	}
	return items
}

func toProtoDaggerheartBeastform(beastform contentstore.DaggerheartBeastformEntry) *pb.DaggerheartBeastformEntry {
	return &pb.DaggerheartBeastformEntry{
		Id:           beastform.ID,
		Name:         beastform.Name,
		Tier:         int32(beastform.Tier),
		Examples:     beastform.Examples,
		Trait:        beastform.Trait,
		TraitBonus:   int32(beastform.TraitBonus),
		EvasionBonus: int32(beastform.EvasionBonus),
		Attack:       toProtoDaggerheartBeastformAttack(beastform.Attack),
		Advantages:   append([]string{}, beastform.Advantages...),
		Features:     toProtoDaggerheartBeastformFeatures(beastform.Features),
	}
}

func toProtoDaggerheartBeastforms(beastforms []contentstore.DaggerheartBeastformEntry) []*pb.DaggerheartBeastformEntry {
	items := make([]*pb.DaggerheartBeastformEntry, 0, len(beastforms))
	for _, beastform := range beastforms {
		items = append(items, toProtoDaggerheartBeastform(beastform))
	}
	return items
}

func toProtoDaggerheartCompanionExperience(experience contentstore.DaggerheartCompanionExperienceEntry) *pb.DaggerheartCompanionExperienceEntry {
	return &pb.DaggerheartCompanionExperienceEntry{
		Id:          experience.ID,
		Name:        experience.Name,
		Description: experience.Description,
	}
}

func toProtoDaggerheartCompanionExperiences(experiences []contentstore.DaggerheartCompanionExperienceEntry) []*pb.DaggerheartCompanionExperienceEntry {
	items := make([]*pb.DaggerheartCompanionExperienceEntry, 0, len(experiences))
	for _, experience := range experiences {
		items = append(items, toProtoDaggerheartCompanionExperience(experience))
	}
	return items
}
