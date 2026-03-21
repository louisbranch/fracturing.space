package contenttransport

import (
	"strings"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
)

func toProtoDaggerheartClass(class contentstore.DaggerheartClass) *pb.DaggerheartClass {
	return &pb.DaggerheartClass{
		Id:              class.ID,
		Name:            class.Name,
		StartingEvasion: int32(class.StartingEvasion),
		StartingHp:      int32(class.StartingHP),
		StartingItems:   append([]string{}, class.StartingItems...),
		Features:        toProtoDaggerheartFeatures(class.Features),
		HopeFeature:     toProtoDaggerheartHopeFeature(class.HopeFeature),
		DomainIds:       append([]string{}, class.DomainIDs...),
	}
}

func toProtoDaggerheartClasses(classes []contentstore.DaggerheartClass) []*pb.DaggerheartClass {
	items := make([]*pb.DaggerheartClass, 0, len(classes))
	for _, class := range classes {
		items = append(items, toProtoDaggerheartClass(class))
	}
	return items
}

func toProtoDaggerheartSubclass(subclass contentstore.DaggerheartSubclass) *pb.DaggerheartSubclass {
	return &pb.DaggerheartSubclass{
		Id:                     subclass.ID,
		Name:                   subclass.Name,
		ClassId:                subclass.ClassID,
		SpellcastTrait:         subclass.SpellcastTrait,
		FoundationFeatures:     toProtoDaggerheartFeatures(subclass.FoundationFeatures),
		SpecializationFeatures: toProtoDaggerheartFeatures(subclass.SpecializationFeatures),
		MasteryFeatures:        toProtoDaggerheartFeatures(subclass.MasteryFeatures),
	}
}

func toProtoDaggerheartSubclasses(subclasses []contentstore.DaggerheartSubclass) []*pb.DaggerheartSubclass {
	items := make([]*pb.DaggerheartSubclass, 0, len(subclasses))
	for _, subclass := range subclasses {
		items = append(items, toProtoDaggerheartSubclass(subclass))
	}
	return items
}

func toProtoDaggerheartHeritage(heritage contentstore.DaggerheartHeritage) *pb.DaggerheartHeritage {
	return &pb.DaggerheartHeritage{
		Id:       heritage.ID,
		Name:     heritage.Name,
		Kind:     heritageKindToProto(heritage.Kind),
		Features: toProtoDaggerheartFeatures(heritage.Features),
	}
}

func toProtoDaggerheartHeritages(heritages []contentstore.DaggerheartHeritage) []*pb.DaggerheartHeritage {
	items := make([]*pb.DaggerheartHeritage, 0, len(heritages))
	for _, heritage := range heritages {
		items = append(items, toProtoDaggerheartHeritage(heritage))
	}
	return items
}

func toProtoDaggerheartExperience(experience contentstore.DaggerheartExperienceEntry) *pb.DaggerheartExperienceEntry {
	return &pb.DaggerheartExperienceEntry{
		Id:          experience.ID,
		Name:        experience.Name,
		Description: experience.Description,
	}
}

func toProtoDaggerheartExperiences(experiences []contentstore.DaggerheartExperienceEntry) []*pb.DaggerheartExperienceEntry {
	items := make([]*pb.DaggerheartExperienceEntry, 0, len(experiences))
	for _, experience := range experiences {
		items = append(items, toProtoDaggerheartExperience(experience))
	}
	return items
}

func toProtoDaggerheartFeatures(features []contentstore.DaggerheartFeature) []*pb.DaggerheartFeature {
	items := make([]*pb.DaggerheartFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, &pb.DaggerheartFeature{
			Id:          feature.ID,
			Name:        feature.Name,
			Description: feature.Description,
			Level:       int32(feature.Level),
		})
	}
	return items
}

func toProtoDaggerheartHopeFeature(feature contentstore.DaggerheartHopeFeature) *pb.DaggerheartHopeFeature {
	return &pb.DaggerheartHopeFeature{
		Name:        feature.Name,
		Description: feature.Description,
		HopeCost:    int32(feature.HopeCost),
	}
}

func toProtoDaggerheartDamageDice(dice []contentstore.DaggerheartDamageDie) []*pb.DiceSpec {
	results := make([]*pb.DiceSpec, 0, len(dice))
	for _, die := range dice {
		results = append(results, &pb.DiceSpec{
			Sides: int32(die.Sides),
			Count: int32(die.Count),
		})
	}
	return results
}

func heritageKindToProto(kind string) pb.DaggerheartHeritageKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "ancestry":
		return pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_ANCESTRY
	case "community":
		return pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_COMMUNITY
	default:
		return pb.DaggerheartHeritageKind_DAGGERHEART_HERITAGE_KIND_UNSPECIFIED
	}
}
