package creationworkflow

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

func (s *testContentStore) GetDaggerheartAdversaryEntry(_ context.Context, id string) (contentstore.DaggerheartAdversaryEntry, error) {
	return mapGet(s.adversaries, id)
}

func (s *testContentStore) ListDaggerheartAdversaryEntries(_ context.Context) ([]contentstore.DaggerheartAdversaryEntry, error) {
	return mapList(s.adversaries)
}

func (s *testContentStore) GetDaggerheartBeastform(_ context.Context, id string) (contentstore.DaggerheartBeastformEntry, error) {
	return mapGet(s.beastforms, id)
}

func (s *testContentStore) ListDaggerheartBeastforms(_ context.Context) ([]contentstore.DaggerheartBeastformEntry, error) {
	return mapList(s.beastforms)
}

func (s *testContentStore) GetDaggerheartCompanionExperience(_ context.Context, id string) (contentstore.DaggerheartCompanionExperienceEntry, error) {
	return mapGet(s.companionExperiences, id)
}

func (s *testContentStore) ListDaggerheartCompanionExperiences(_ context.Context) ([]contentstore.DaggerheartCompanionExperienceEntry, error) {
	return mapList(s.companionExperiences)
}

func (s *testContentStore) GetDaggerheartLootEntry(_ context.Context, id string) (contentstore.DaggerheartLootEntry, error) {
	return mapGet(s.lootEntries, id)
}

func (s *testContentStore) ListDaggerheartLootEntries(_ context.Context) ([]contentstore.DaggerheartLootEntry, error) {
	return mapList(s.lootEntries)
}

func (s *testContentStore) GetDaggerheartDamageType(_ context.Context, id string) (contentstore.DaggerheartDamageTypeEntry, error) {
	return mapGet(s.damageTypes, id)
}

func (s *testContentStore) ListDaggerheartDamageTypes(_ context.Context) ([]contentstore.DaggerheartDamageTypeEntry, error) {
	return mapList(s.damageTypes)
}

func (s *testContentStore) GetDaggerheartEnvironment(_ context.Context, id string) (contentstore.DaggerheartEnvironment, error) {
	return mapGet(s.environments, id)
}

func (s *testContentStore) ListDaggerheartEnvironments(_ context.Context) ([]contentstore.DaggerheartEnvironment, error) {
	return mapList(s.environments)
}

func (s *testContentStore) GetDaggerheartDomain(_ context.Context, id string) (contentstore.DaggerheartDomain, error) {
	return mapGet(s.domains, id)
}

func (s *testContentStore) ListDaggerheartDomains(_ context.Context) ([]contentstore.DaggerheartDomain, error) {
	return mapList(s.domains)
}

func (s *testContentStore) GetDaggerheartDomainCard(_ context.Context, id string) (contentstore.DaggerheartDomainCard, error) {
	return mapGet(s.domainCards, id)
}

func (s *testContentStore) ListDaggerheartDomainCards(_ context.Context) ([]contentstore.DaggerheartDomainCard, error) {
	return mapList(s.domainCards)
}

func (s *testContentStore) ListDaggerheartDomainCardsByDomain(_ context.Context, domainID string) ([]contentstore.DaggerheartDomainCard, error) {
	cards := make([]contentstore.DaggerheartDomainCard, 0, len(s.domainCards))
	for _, card := range s.domainCards {
		if card.DomainID == domainID {
			cards = append(cards, card)
		}
	}
	return cards, nil
}

func (s *testContentStore) GetDaggerheartWeapon(_ context.Context, id string) (contentstore.DaggerheartWeapon, error) {
	return mapGet(s.weapons, id)
}

func (s *testContentStore) ListDaggerheartWeapons(_ context.Context) ([]contentstore.DaggerheartWeapon, error) {
	return mapList(s.weapons)
}

func (s *testContentStore) GetDaggerheartArmor(_ context.Context, id string) (contentstore.DaggerheartArmor, error) {
	return mapGet(s.armor, id)
}

func (s *testContentStore) ListDaggerheartArmor(_ context.Context) ([]contentstore.DaggerheartArmor, error) {
	return mapList(s.armor)
}

func (s *testContentStore) GetDaggerheartItem(_ context.Context, id string) (contentstore.DaggerheartItem, error) {
	return mapGet(s.items, id)
}

func (s *testContentStore) ListDaggerheartItems(_ context.Context) ([]contentstore.DaggerheartItem, error) {
	return mapList(s.items)
}

func (*testContentStore) ListDaggerheartContentStrings(_ context.Context, _ string, _ []string, _ string) ([]contentstore.DaggerheartContentString, error) {
	return nil, nil
}
