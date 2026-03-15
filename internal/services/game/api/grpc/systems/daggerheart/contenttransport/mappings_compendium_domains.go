package contenttransport

import (
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

func toProtoDaggerheartDamageType(entry contentstore.DaggerheartDamageTypeEntry) *pb.DaggerheartDamageTypeEntry {
	return &pb.DaggerheartDamageTypeEntry{
		Id:          entry.ID,
		Name:        entry.Name,
		Description: entry.Description,
	}
}

func toProtoDaggerheartDamageTypes(entries []contentstore.DaggerheartDamageTypeEntry) []*pb.DaggerheartDamageTypeEntry {
	items := make([]*pb.DaggerheartDamageTypeEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toProtoDaggerheartDamageType(entry))
	}
	return items
}

func toProtoDaggerheartDomain(domain contentstore.DaggerheartDomain) *pb.DaggerheartDomain {
	return &pb.DaggerheartDomain{
		Id:          domain.ID,
		Name:        domain.Name,
		Description: domain.Description,
	}
}

func toProtoDaggerheartDomains(domains []contentstore.DaggerheartDomain) []*pb.DaggerheartDomain {
	items := make([]*pb.DaggerheartDomain, 0, len(domains))
	for _, domain := range domains {
		items = append(items, toProtoDaggerheartDomain(domain))
	}
	return items
}

func toProtoDaggerheartDomainCard(card contentstore.DaggerheartDomainCard) *pb.DaggerheartDomainCard {
	return &pb.DaggerheartDomainCard{
		Id:          card.ID,
		Name:        card.Name,
		DomainId:    card.DomainID,
		Level:       int32(card.Level),
		Type:        domainCardTypeToProto(card.Type),
		RecallCost:  int32(card.RecallCost),
		UsageLimit:  card.UsageLimit,
		FeatureText: card.FeatureText,
	}
}

func toProtoDaggerheartDomainCards(cards []contentstore.DaggerheartDomainCard) []*pb.DaggerheartDomainCard {
	items := make([]*pb.DaggerheartDomainCard, 0, len(cards))
	for _, card := range cards {
		items = append(items, toProtoDaggerheartDomainCard(card))
	}
	return items
}
