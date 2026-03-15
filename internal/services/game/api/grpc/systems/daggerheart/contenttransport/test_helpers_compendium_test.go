package contenttransport

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
)

func (s *fakeContentStore) GetDaggerheartAdversaryEntry(_ context.Context, id string) (contentstore.DaggerheartAdversaryEntry, error) {
	return mapGet(s.adversaries, id)
}

func (s *fakeContentStore) ListDaggerheartAdversaryEntries(_ context.Context) ([]contentstore.DaggerheartAdversaryEntry, error) {
	return mapList(s.adversaries)
}

func (s *fakeContentStore) GetDaggerheartBeastform(_ context.Context, id string) (contentstore.DaggerheartBeastformEntry, error) {
	return mapGet(s.beastforms, id)
}

func (s *fakeContentStore) ListDaggerheartBeastforms(_ context.Context) ([]contentstore.DaggerheartBeastformEntry, error) {
	return mapList(s.beastforms)
}

func (s *fakeContentStore) GetDaggerheartCompanionExperience(_ context.Context, id string) (contentstore.DaggerheartCompanionExperienceEntry, error) {
	return mapGet(s.companionExperiences, id)
}

func (s *fakeContentStore) ListDaggerheartCompanionExperiences(_ context.Context) ([]contentstore.DaggerheartCompanionExperienceEntry, error) {
	return mapList(s.companionExperiences)
}

func (s *fakeContentStore) GetDaggerheartLootEntry(_ context.Context, id string) (contentstore.DaggerheartLootEntry, error) {
	return mapGet(s.lootEntries, id)
}

func (s *fakeContentStore) ListDaggerheartLootEntries(_ context.Context) ([]contentstore.DaggerheartLootEntry, error) {
	return mapList(s.lootEntries)
}

func (s *fakeContentStore) GetDaggerheartDamageType(_ context.Context, id string) (contentstore.DaggerheartDamageTypeEntry, error) {
	return mapGet(s.damageTypes, id)
}

func (s *fakeContentStore) ListDaggerheartDamageTypes(_ context.Context) ([]contentstore.DaggerheartDamageTypeEntry, error) {
	return mapList(s.damageTypes)
}

func (s *fakeContentStore) GetDaggerheartEnvironment(_ context.Context, id string) (contentstore.DaggerheartEnvironment, error) {
	return mapGet(s.environments, id)
}

func (s *fakeContentStore) ListDaggerheartEnvironments(_ context.Context) ([]contentstore.DaggerheartEnvironment, error) {
	return mapList(s.environments)
}

func (s *fakeContentStore) GetDaggerheartDomain(_ context.Context, id string) (contentstore.DaggerheartDomain, error) {
	return mapGet(s.domains, id)
}

func (s *fakeContentStore) ListDaggerheartDomains(_ context.Context) ([]contentstore.DaggerheartDomain, error) {
	return mapList(s.domains)
}

func (s *fakeContentStore) GetDaggerheartDomainCard(_ context.Context, id string) (contentstore.DaggerheartDomainCard, error) {
	return mapGet(s.domainCards, id)
}

func (s *fakeContentStore) ListDaggerheartDomainCards(_ context.Context) ([]contentstore.DaggerheartDomainCard, error) {
	return mapList(s.domainCards)
}

func (s *fakeContentStore) ListDaggerheartDomainCardsByDomain(_ context.Context, domainID string) ([]contentstore.DaggerheartDomainCard, error) {
	cards := make([]contentstore.DaggerheartDomainCard, 0, len(s.domainCards))
	for _, card := range s.domainCards {
		if card.DomainID == domainID {
			cards = append(cards, card)
		}
	}
	return cards, nil
}

func (s *fakeContentStore) GetDaggerheartWeapon(_ context.Context, id string) (contentstore.DaggerheartWeapon, error) {
	return mapGet(s.weapons, id)
}

func (s *fakeContentStore) ListDaggerheartWeapons(_ context.Context) ([]contentstore.DaggerheartWeapon, error) {
	return mapList(s.weapons)
}

func (s *fakeContentStore) GetDaggerheartArmor(_ context.Context, id string) (contentstore.DaggerheartArmor, error) {
	return mapGet(s.armor, id)
}

func (s *fakeContentStore) ListDaggerheartArmor(_ context.Context) ([]contentstore.DaggerheartArmor, error) {
	return mapList(s.armor)
}

func (s *fakeContentStore) GetDaggerheartItem(_ context.Context, id string) (contentstore.DaggerheartItem, error) {
	return mapGet(s.items, id)
}

func (s *fakeContentStore) ListDaggerheartItems(_ context.Context) ([]contentstore.DaggerheartItem, error) {
	return mapList(s.items)
}

func (s *fakeContentStore) ListDaggerheartContentStrings(_ context.Context, contentType string, contentIDs []string, locale string) ([]contentstore.DaggerheartContentString, error) {
	idSet := make(map[string]struct{}, len(contentIDs))
	for _, id := range contentIDs {
		idSet[id] = struct{}{}
	}

	result := make([]contentstore.DaggerheartContentString, 0, len(s.contentStrings))
	for _, entry := range s.contentStrings {
		if entry.ContentType != contentType || entry.Locale != locale {
			continue
		}
		if _, ok := idSet[entry.ContentID]; !ok {
			continue
		}
		result = append(result, entry)
	}
	return result, nil
}
